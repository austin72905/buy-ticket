package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"buy-ticket/domain"
	"buy-ticket/repository"
)

func TestAdminServiceCreateBackofficeResources(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	superAdmin := &domain.AdminUser{
		ID:     1,
		Name:   "Super Admin",
		Email:  "admin@example.com",
		Role:   domain.AdminRoleSuperAdmin,
		Status: domain.AdminUserStatusActive,
	}
	organizerID := int64(1)
	eventAdmin := &domain.AdminUser{
		ID:          2,
		OrganizerID: &organizerID,
		Name:        "Event Admin",
		Email:       "event-admin@example.com",
		Role:        domain.AdminRoleEventAdmin,
		Status:      domain.AdminUserStatusActive,
	}

	adminUserRepo := repository.NewMemoryAdminUserRepository([]*domain.AdminUser{superAdmin, eventAdmin})
	organizerRepo := repository.NewMemoryOrganizerRepository([]*domain.Organizer{
		{
			ID:        1,
			Name:      "Default Organizer",
			Status:    domain.OrganizerStatusActive,
			CreatedAt: now,
			UpdatedAt: now,
		},
	})
	eventRepo := repository.NewMemoryEventRepository(nil)
	adminService := NewAdminService(
		adminUserRepo,
		organizerRepo,
		repository.NewMemoryOrderRepository(),
		eventRepo,
		repository.NewMemoryAdminAuditLogRepository(),
	)

	t.Run("super admin creates organizer and admin user", func(t *testing.T) {
		organizer, err := adminService.CreateOrganizer(ctx, CreateOrganizerInput{
			AdminUser: superAdmin,
			Name:      "Second Organizer",
		})
		if err != nil {
			t.Fatalf("create organizer failed: %v", err)
		}
		if organizer.ID == 0 {
			t.Fatal("expected organizer id")
		}

		adminUser, err := adminService.CreateAdminUser(ctx, CreateAdminUserInput{
			AdminUser:   superAdmin,
			OrganizerID: &organizer.ID,
			Name:        "Second Event Admin",
			Email:       "second-admin@example.com",
			Password:    "1234",
			Role:        domain.AdminRoleEventAdmin,
		})
		if err != nil {
			t.Fatalf("create admin user failed: %v", err)
		}
		if adminUser.ID == 0 || adminUser.PasswordHash == "" {
			t.Fatal("expected admin user id and password hash")
		}
	})

	t.Run("event admin creates event and section under own organizer", func(t *testing.T) {
		event, err := adminService.CreateEvent(ctx, CreateAdminEventInput{
			AdminUser:   eventAdmin,
			Name:        "Event Admin Concert",
			Venue:       "Taipei Arena",
			Status:      domain.EventStatusPublished,
			StartAt:     now.Add(24 * time.Hour),
			EndAt:       now.Add(26 * time.Hour),
			SaleStartAt: now.Add(-time.Hour),
			SaleEndAt:   now.Add(12 * time.Hour),
		})
		if err != nil {
			t.Fatalf("create event failed: %v", err)
		}
		if event.OrganizerID != organizerID {
			t.Fatalf("expected organizer id %d, got %d", organizerID, event.OrganizerID)
		}

		section, err := adminService.CreateEventSection(ctx, CreateAdminSectionInput{
			AdminUser:     eventAdmin,
			EventID:       event.ID,
			Name:          "A Zone",
			Price:         2800,
			TotalQuantity: 100,
			PurchaseLimit: 4,
		})
		if err != nil {
			t.Fatalf("create section failed: %v", err)
		}
		if section.ID == 0 || section.Status != domain.SectionStatusActive {
			t.Fatalf("expected active section with id, got %+v", section)
		}
	})

	t.Run("event admin cannot list organizers", func(t *testing.T) {
		_, err := adminService.ListOrganizers(ctx, eventAdmin)
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})
}
