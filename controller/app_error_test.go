package controller

import (
	"errors"
	"net/http"
	"testing"

	"buy-ticket/repository"
	"buy-ticket/service"
)

func TestNewAppErrorUsesDomainErrorCode(t *testing.T) {
	t.Run("purchase token not found", func(t *testing.T) {
		appErr := newAppError(http.StatusBadRequest, service.ErrPurchaseTokenNotFound)

		if appErr.HTTPStatus != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, appErr.HTTPStatus)
		}
		if appErr.Code != "PURCHASE_TOKEN_NOT_FOUND" {
			t.Fatalf("expected code PURCHASE_TOKEN_NOT_FOUND, got %s", appErr.Code)
		}
		if appErr.Message != service.ErrPurchaseTokenNotFound.Error() {
			t.Fatalf("expected message %q, got %q", service.ErrPurchaseTokenNotFound.Error(), appErr.Message)
		}
	})

	t.Run("repository not found", func(t *testing.T) {
		appErr := newAppError(http.StatusNotFound, repository.ErrOrderNotFound)

		if appErr.Code != "ORDER_NOT_FOUND" {
			t.Fatalf("expected code ORDER_NOT_FOUND, got %s", appErr.Code)
		}
	})

	t.Run("authorization follows fallback status", func(t *testing.T) {
		unauthorizedErr := newAppError(http.StatusUnauthorized, service.ErrUnauthorized)
		if unauthorizedErr.Code != "UNAUTHORIZED" {
			t.Fatalf("expected code UNAUTHORIZED, got %s", unauthorizedErr.Code)
		}

		forbiddenErr := newAppError(http.StatusForbidden, service.ErrUnauthorized)
		if forbiddenErr.Code != "FORBIDDEN" {
			t.Fatalf("expected code FORBIDDEN, got %s", forbiddenErr.Code)
		}
	})

	t.Run("unknown error uses fallback status code", func(t *testing.T) {
		appErr := newAppError(http.StatusBadRequest, errors.New("invalid input"))

		if appErr.Code != "BAD_REQUEST" {
			t.Fatalf("expected code BAD_REQUEST, got %s", appErr.Code)
		}
	})
}
