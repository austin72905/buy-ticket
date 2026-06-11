package service

import (
	"context"
	"errors"

	"buy-ticket/domain"
)

var ErrInsufficientStock = errors.New("insufficient stock")

type StockStore interface {
	Reserve(ctx context.Context, section domain.Section, quantity int) error
	Release(ctx context.Context, section domain.Section, quantity int) error
	RebuildAll(ctx context.Context, sections []domain.Section) error
}
