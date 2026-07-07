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

var ErrInvalidAdminRevealReason = errors.New("admin reveal reason is required")
var ErrInvalidAdminInput = errors.New("invalid admin input")

type AdminService struct {
	AdminUserRepo     repository.AdminUserRepository
	OrganizerRepo     repository.OrganizerRepository
	AdminOrderRepo    repository.AdminOrderRepository
	AdminEventRepo    repository.AdminEventRepository
	AdminAuditLogRepo repository.AdminAuditLogRepository
}

type ListAdminOrdersInput struct {
	AdminUser       *domain.AdminUser
	EventID         int64
	UserID          int64
	Status          domain.OrderStatus
	CursorCreatedAt *time.Time
	CursorID        int64
	Limit           int
}

type ListAdminAuditLogsInput struct {
	AdminUser       *domain.AdminUser
	CursorCreatedAt *time.Time
	CursorID        int64
	Limit           int
}

type RevealOrderSensitiveInput struct {
	AdminUser *domain.AdminUser
	OrderID   int64
	Reason    string
	IPAddress *string
	UserAgent *string
}

type CreateOrganizerInput struct {
	AdminUser *domain.AdminUser
	Name      string
}

type CreateAdminUserInput struct {
	AdminUser   *domain.AdminUser
	OrganizerID *int64
	Name        string
	Email       string
	Password    string
	Role        domain.AdminRole
}

type CreateAdminEventInput struct {
	AdminUser   *domain.AdminUser
	OrganizerID int64
	Name        string
	Venue       string
	Status      domain.EventStatus
	StartAt     time.Time
	EndAt       time.Time
	SaleStartAt time.Time
	SaleEndAt   time.Time
}

type CreateAdminSectionInput struct {
	AdminUser     *domain.AdminUser
	EventID       int64
	Name          string
	Price         int64
	TotalQuantity int
	PurchaseLimit int
	Status        domain.SectionStatus
}

func NewAdminService(
	adminUserRepo repository.AdminUserRepository,
	organizerRepo repository.OrganizerRepository,
	adminOrderRepo repository.AdminOrderRepository,
	adminEventRepo repository.AdminEventRepository,
	adminAuditLogRepo repository.AdminAuditLogRepository,
) *AdminService {
	return &AdminService{
		AdminUserRepo:     adminUserRepo,
		OrganizerRepo:     organizerRepo,
		AdminOrderRepo:    adminOrderRepo,
		AdminEventRepo:    adminEventRepo,
		AdminAuditLogRepo: adminAuditLogRepo,
	}
}

func (s *AdminService) ListAdminUsers(ctx context.Context, adminUser *domain.AdminUser) ([]domain.AdminUser, error) {
	if adminUser == nil {
		return nil, ErrUnauthorized
	}
	if !adminUser.IsSuperAdmin() {
		return nil, ErrForbidden
	}
	return s.AdminUserRepo.List(ctx)
}

func (s *AdminService) CreateAdminUser(ctx context.Context, input CreateAdminUserInput) (*domain.AdminUser, error) {
	if input.AdminUser == nil {
		return nil, ErrUnauthorized
	}
	if !input.AdminUser.IsSuperAdmin() {
		return nil, ErrForbidden
	}

	name := strings.TrimSpace(input.Name)
	email := strings.TrimSpace(strings.ToLower(input.Email))
	password := strings.TrimSpace(input.Password)
	if name == "" || email == "" || password == "" || input.Role == "" {
		return nil, ErrInvalidAdminInput
	}
	if input.Role != domain.AdminRoleSuperAdmin && input.Role != domain.AdminRoleEventAdmin {
		return nil, ErrInvalidAdminInput
	}
	if input.Role == domain.AdminRoleEventAdmin {
		if input.OrganizerID == nil || *input.OrganizerID == 0 {
			return nil, ErrInvalidAdminInput
		}
		if _, err := s.OrganizerRepo.FindByID(ctx, *input.OrganizerID); err != nil {
			return nil, err
		}
	}

	if _, err := s.AdminUserRepo.FindByEmail(ctx, email); err == nil {
		return nil, ErrInvalidAdminInput
	} else if !errors.Is(err, repository.ErrAdminUserNotFound) {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	created := &domain.AdminUser{
		OrganizerID:  input.OrganizerID,
		Name:         name,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         input.Role,
		Status:       domain.AdminUserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.AdminUserRepo.Save(ctx, created); err != nil {
		return nil, err
	}
	return created, nil
}

func (s *AdminService) ListOrganizers(ctx context.Context, adminUser *domain.AdminUser) ([]domain.Organizer, error) {
	if adminUser == nil {
		return nil, ErrUnauthorized
	}
	if !adminUser.IsSuperAdmin() {
		return nil, ErrForbidden
	}
	return s.OrganizerRepo.List(ctx)
}

func (s *AdminService) CreateOrganizer(ctx context.Context, input CreateOrganizerInput) (*domain.Organizer, error) {
	if input.AdminUser == nil {
		return nil, ErrUnauthorized
	}
	if !input.AdminUser.IsSuperAdmin() {
		return nil, ErrForbidden
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidAdminInput
	}

	now := time.Now()
	organizer := &domain.Organizer{
		Name:      name,
		Status:    domain.OrganizerStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.OrganizerRepo.Save(ctx, organizer); err != nil {
		return nil, err
	}
	return organizer, nil
}

func (s *AdminService) ListOrders(ctx context.Context, input ListAdminOrdersInput) ([]domain.AdminOrder, error) {
	filter := domain.AdminOrderListFilter{
		EventID: input.EventID,
		UserID:  input.UserID,
		Status:  input.Status,
		Cursor: domain.AdminOrderListCursor{
			CreatedAt: input.CursorCreatedAt,
			ID:        input.CursorID,
		},
		Limit: normalizeAdminListLimit(input.Limit),
	}

	organizerID, err := adminOrganizerScope(input.AdminUser)
	if err != nil {
		return nil, err
	}
	filter.OrganizerID = organizerID

	return s.AdminOrderRepo.ListAdminOrders(ctx, filter)
}

func (s *AdminService) ListEvents(ctx context.Context, adminUser *domain.AdminUser) ([]domain.Event, error) {
	organizerID, err := adminOrganizerScope(adminUser)
	if err != nil {
		return nil, err
	}

	return s.AdminEventRepo.ListAdminEvents(ctx, organizerID)
}

func (s *AdminService) CreateEvent(ctx context.Context, input CreateAdminEventInput) (*domain.Event, error) {
	organizerID, err := adminOrganizerScope(input.AdminUser)
	if err != nil {
		return nil, err
	}

	targetOrganizerID := input.OrganizerID
	if organizerID != nil {
		targetOrganizerID = *organizerID
	} else if targetOrganizerID == 0 {
		return nil, ErrInvalidAdminInput
	}
	if _, err := s.OrganizerRepo.FindByID(ctx, targetOrganizerID); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	venue := strings.TrimSpace(input.Venue)
	status := input.Status
	if status == 0 {
		status = domain.EventStatusDraft
	}
	if name == "" || venue == "" || input.StartAt.IsZero() || input.EndAt.IsZero() || input.SaleStartAt.IsZero() || input.SaleEndAt.IsZero() {
		return nil, ErrInvalidAdminInput
	}
	if !input.StartAt.Before(input.EndAt) || !input.SaleStartAt.Before(input.SaleEndAt) {
		return nil, ErrInvalidAdminInput
	}
	if status < domain.EventStatusDraft || status > domain.EventStatusEnded {
		return nil, ErrInvalidAdminInput
	}

	event := &domain.Event{
		OrganizerID: targetOrganizerID,
		Name:        name,
		Venue:       venue,
		Status:      status,
		StartAt:     input.StartAt,
		EndAt:       input.EndAt,
		SaleStartAt: input.SaleStartAt,
		SaleEndAt:   input.SaleEndAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.AdminEventRepo.CreateAdminEvent(ctx, event); err != nil {
		return nil, err
	}
	return event, nil
}

func (s *AdminService) GetEvent(ctx context.Context, adminUser *domain.AdminUser, eventID int64) (*domain.Event, error) {
	organizerID, err := adminOrganizerScope(adminUser)
	if err != nil {
		return nil, err
	}

	return s.AdminEventRepo.FindAdminEventByID(ctx, eventID, organizerID)
}

func (s *AdminService) CreateEventSection(ctx context.Context, input CreateAdminSectionInput) (*domain.Section, error) {
	organizerID, err := adminOrganizerScope(input.AdminUser)
	if err != nil {
		return nil, err
	}

	status := input.Status
	if status == 0 {
		status = domain.SectionStatusActive
	}
	name := strings.TrimSpace(input.Name)
	if input.EventID == 0 || name == "" || input.Price < 0 || input.TotalQuantity <= 0 || input.PurchaseLimit <= 0 {
		return nil, ErrInvalidAdminInput
	}
	if input.PurchaseLimit > input.TotalQuantity {
		return nil, ErrInvalidAdminInput
	}
	if status < domain.SectionStatusActive || status > domain.SectionStatusSoldOut {
		return nil, ErrInvalidAdminInput
	}

	section := &domain.Section{
		EventID:       input.EventID,
		Name:          name,
		Price:         input.Price,
		TotalQuantity: input.TotalQuantity,
		PurchaseLimit: input.PurchaseLimit,
		Status:        status,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := s.AdminEventRepo.CreateAdminEventSection(ctx, input.EventID, organizerID, section); err != nil {
		return nil, err
	}
	return section, nil
}

func (s *AdminService) ListEventSections(ctx context.Context, adminUser *domain.AdminUser, eventID int64) ([]domain.Section, error) {
	organizerID, err := adminOrganizerScope(adminUser)
	if err != nil {
		return nil, err
	}

	return s.AdminEventRepo.ListAdminEventSections(ctx, eventID, organizerID)
}

func (s *AdminService) ListAuditLogs(ctx context.Context, input ListAdminAuditLogsInput) ([]domain.AdminAuditLog, error) {
	if input.AdminUser == nil {
		return nil, ErrUnauthorized
	}

	adminUserID := int64(0)
	if input.AdminUser.IsEventAdmin() {
		adminUserID = input.AdminUser.ID
	}

	return s.AdminAuditLogRepo.List(ctx, domain.AdminAuditLogListFilter{
		AdminUserID: adminUserID,
		Cursor: domain.AdminAuditLogListCursor{
			CreatedAt: input.CursorCreatedAt,
			ID:        input.CursorID,
		},
		Limit: normalizeAdminListLimit(input.Limit),
	})
}

func (s *AdminService) RevealOrderSensitive(ctx context.Context, input RevealOrderSensitiveInput) (*domain.AdminOrder, error) {
	if input.AdminUser == nil {
		return nil, ErrUnauthorized
	}
	if !input.AdminUser.IsSuperAdmin() {
		return nil, ErrForbidden
	}

	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return nil, ErrInvalidAdminRevealReason
	}

	order, err := s.AdminOrderRepo.FindAdminOrderSensitiveByID(ctx, input.OrderID)
	if err != nil {
		return nil, err
	}

	auditLog := &domain.AdminAuditLog{
		AdminUserID: input.AdminUser.ID,
		Action:      "REVEAL_ORDER_SENSITIVE",
		TargetType:  "ORDER",
		TargetID:    input.OrderID,
		Reason:      &reason,
		IPAddress:   input.IPAddress,
		UserAgent:   input.UserAgent,
	}
	if err := s.AdminAuditLogRepo.Create(ctx, auditLog); err != nil {
		return nil, err
	}

	return order, nil
}

func normalizeAdminListLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 101 {
		return 101
	}
	return limit
}

func adminOrganizerScope(adminUser *domain.AdminUser) (*int64, error) {
	if adminUser == nil {
		return nil, ErrUnauthorized
	}

	if adminUser.IsEventAdmin() {
		if adminUser.OrganizerID == nil {
			return nil, ErrForbidden
		}
		return adminUser.OrganizerID, nil
	}

	return nil, nil
}
