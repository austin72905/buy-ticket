package service

import (
	"context"

	"buy-ticket/domain"
)

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

func (s *BookingService) listAllSections(ctx context.Context) ([]domain.Section, error) {
	events, err := s.EventRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	sections := make([]domain.Section, 0)
	for _, event := range events {
		eventSections, listErr := s.SectionRepo.ListByEventID(ctx, event.ID)
		if listErr != nil {
			return nil, listErr
		}
		sections = append(sections, eventSections...)
	}

	return sections, nil
}
