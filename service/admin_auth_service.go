package service

import (
	"context"
	"errors"
	"strings"

	"buy-ticket/domain"
	"buy-ticket/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidAdminCredentials = errors.New("invalid admin email or password")
	ErrAdminDisabled           = errors.New("admin user is disabled")
	ErrForbidden               = errors.New("forbidden")
)

type AdminAuthService struct {
	AdminUserRepo repository.AdminUserRepository
}

type AdminLoginInput struct {
	Email    string
	Password string
}

func NewAdminAuthService(adminUserRepo repository.AdminUserRepository) *AdminAuthService {
	return &AdminAuthService{AdminUserRepo: adminUserRepo}
}

func (s *AdminAuthService) Login(ctx context.Context, input AdminLoginInput) (*domain.AdminUser, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))
	password := strings.TrimSpace(input.Password)

	if email == "" || password == "" {
		return nil, ErrInvalidAdminCredentials
	}

	adminUser, err := s.AdminUserRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrAdminUserNotFound) {
			return nil, ErrInvalidAdminCredentials
		}
		return nil, err
	}

	if !adminUser.IsActive() {
		return nil, ErrAdminDisabled
	}

	if err := bcrypt.CompareHashAndPassword([]byte(adminUser.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidAdminCredentials
	}

	return adminUser, nil
}

func (s *AdminAuthService) GetMe(ctx context.Context, adminUserID int64) (*domain.AdminUser, error) {
	adminUser, err := s.AdminUserRepo.FindByID(ctx, adminUserID)
	if err != nil {
		return nil, err
	}
	if !adminUser.IsActive() {
		return nil, ErrAdminDisabled
	}
	return adminUser, nil
}
