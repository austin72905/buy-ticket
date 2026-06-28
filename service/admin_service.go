package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"buy-ticket/domain"
	"buy-ticket/repository"
)

var ErrInvalidAdminRevealReason = errors.New("admin reveal reason is required")

type AdminService struct {
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

type RevealOrderSensitiveInput struct {
	AdminUser *domain.AdminUser
	OrderID   int64
	Reason    string
	IPAddress *string
	UserAgent *string
}

func NewAdminService(
	adminOrderRepo repository.AdminOrderRepository,
	adminEventRepo repository.AdminEventRepository,
	adminAuditLogRepo repository.AdminAuditLogRepository,
) *AdminService {
	return &AdminService{
		AdminOrderRepo:    adminOrderRepo,
		AdminEventRepo:    adminEventRepo,
		AdminAuditLogRepo: adminAuditLogRepo,
	}
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

func (s *AdminService) GetEvent(ctx context.Context, adminUser *domain.AdminUser, eventID int64) (*domain.Event, error) {
	organizerID, err := adminOrganizerScope(adminUser)
	if err != nil {
		return nil, err
	}

	return s.AdminEventRepo.FindAdminEventByID(ctx, eventID, organizerID)
}

func (s *AdminService) ListEventSections(ctx context.Context, adminUser *domain.AdminUser, eventID int64) ([]domain.Section, error) {
	organizerID, err := adminOrganizerScope(adminUser)
	if err != nil {
		return nil, err
	}

	return s.AdminEventRepo.ListAdminEventSections(ctx, eventID, organizerID)
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
