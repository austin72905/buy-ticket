package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"buy-ticket/domain"
	"buy-ticket/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserEmailExists        = errors.New("email already registered")
	ErrInvalidCredentials     = errors.New("invalid email or password")
	ErrInvalidRegisterInput   = errors.New("name, email, and password are required")
	ErrInvalidLoginInput      = errors.New("email and password are required")
	ErrUnauthorized           = errors.New("unauthorized")
)

type AuthService struct {
	UserRepo repository.UserRepository
}

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

func NewAuthService(userRepo repository.UserRepository) *AuthService {
	return &AuthService{UserRepo: userRepo}
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*domain.User, error) {
	name := strings.TrimSpace(input.Name)
	email := strings.TrimSpace(strings.ToLower(input.Email))
	password := strings.TrimSpace(input.Password)

	if name == "" || email == "" || password == "" {
		return nil, ErrInvalidRegisterInput
	}

	if _, err := s.UserRepo.FindByEmail(ctx, email); err == nil {
		return nil, ErrUserEmailExists
	} else if !errors.Is(err, repository.ErrUserNotFound) {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &domain.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(passwordHash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.UserRepo.Save(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*domain.User, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))
	password := strings.TrimSpace(input.Password)

	if email == "" || password == "" {
		return nil, ErrInvalidLoginInput
	}

	user, err := s.UserRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

func (s *AuthService) GetMe(ctx context.Context, userID int64) (*domain.User, error) {
	return s.UserRepo.FindByID(ctx, userID)
}
