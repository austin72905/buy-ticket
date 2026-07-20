package service

import (
	"context"

	"buy-ticket/domain"
)

// 同步「DB 裡的 section 庫存」到「Redis 快取庫存
func (s *BookingService) RebuildStock(ctx context.Context) error {
	if s.StockStore == nil {
		return nil
	}

	sections, err := s.listAllSections(ctx)
	if err != nil {
		return err
	}

	return s.StockStore.RebuildAll(ctx, sections)
}

// 定期校正 Redis 庫存。
func (s *BookingService) ReconcileStock(ctx context.Context) (StockReconcileResult, error) {
	if s.StockStore == nil {
		return StockReconcileResult{}, nil
	}

	sections, err := s.listAllSections(ctx)
	if err != nil {
		return StockReconcileResult{}, err
	}

	return s.StockStore.ReconcileAll(ctx, sections)
}

// 把所有 event 底下的 section 撈出來
func (s *BookingService) listAllSections(ctx context.Context) ([]domain.Section, error) {
	return s.SectionRepo.ListAll(ctx)
}
