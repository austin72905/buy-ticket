package service

import (
	"context"
	"time"

	"buy-ticket/domain"
	"buy-ticket/repository"
)

type AdminService struct {
	AdminOrderRepo repository.AdminOrderRepository
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

func NewAdminService(adminOrderRepo repository.AdminOrderRepository) *AdminService {
	return &AdminService{
		AdminOrderRepo: adminOrderRepo,
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

	if input.AdminUser == nil {
		return nil, ErrUnauthorized
	}

	if input.AdminUser.IsEventAdmin() {
		if input.AdminUser.OrganizerID == nil {
			return nil, ErrForbidden
		}
		filter.OrganizerID = input.AdminUser.OrganizerID
	}

	return s.AdminOrderRepo.ListAdminOrders(ctx, filter)
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
