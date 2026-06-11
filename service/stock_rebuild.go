package service

import (
	"context"

	"buy-ticket/domain"
)

func (s *BookingService) RebuildStock(ctx context.Context) error {
	if s.StockStore == nil {
		return nil
	}

	events, err := s.EventRepo.List(ctx)
	if err != nil {
		return err
	}

	sections := make([]domain.Section, 0)
	for _, event := range events {
		eventSections, listErr := s.SectionRepo.ListByEventID(ctx, event.ID)
		if listErr != nil {
			return listErr
		}
		sections = append(sections, eventSections...)
	}

	return s.StockStore.RebuildAll(ctx, sections)
}
