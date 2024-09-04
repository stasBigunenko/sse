package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"sse/models"
)

func (s *Service) AddEvent(ctx context.Context, event models.Event, statusName string) error {
	eventFromDB, err := s.WebhookRepo.GetEventByID(ctx, event.EventID)
	if err != nil {
		return err
	}

	if eventFromDB != nil {
		return models.ErrAlreadyProcessed
	}

	lastEvent, err := s.WebhookRepo.GetLastUpdatedEventByOrderID(ctx, event.OrderID)
	if err != nil {
		return err
	}

	if lastEvent != nil {
		if err = s.validateEvent(event, *lastEvent); err != nil {
			return err
		}
	}

	orderStatus, err := s.WebhookRepo.GetOrderStatusByName(ctx, statusName)
	if err != nil {
		return err
	}

	event.OrderStatusID = orderStatus.ID

	if err = s.WebhookRepo.AddEvent(ctx, event, orderStatus.IsFinal); err != nil {
		return err
	}

	return nil
}

func (s *Service) GetEventHistory(ctx context.Context, orderID uuid.UUID) ([]models.EventMsg, error) {
	eventHistory, err := s.WebhookRepo.GetOrderEvents(ctx, orderID)
	if err != nil {
		return nil, err
	}

	res := make([]models.EventMsg, 0, len(eventHistory))
	for i := range eventHistory {
		res = append(res, models.EventMsg{
			EventID:     eventHistory[i].EventID,
			OrderID:     eventHistory[i].OrderID,
			UserID:      eventHistory[i].UserID,
			OrderStatus: eventHistory[i].OrderStatusName,
			UpdatedAt:   eventHistory[i].UpdatedAt,
			CreatedAt:   eventHistory[i].CreatedAt,
		})
	}

	return res, err
}

func (s *Service) validateEvent(event, lastEvent models.Event) error {
	if event.OrderStatusID == models.GiveMyMoneyBackID &&
		lastEvent.OrderStatusID == models.ChinazesID &&
		event.UpdatedAt.Sub(lastEvent.UpdatedAt) < 30*time.Second {
		return nil
	}

	if lastEvent.OrderStatusID > 3 {
		return models.ErrAlreadyProcessed
	}

	return nil
}
