package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"sse/models"
)

func (s *Service) AddEvent(ctx context.Context, event models.Event, statusName string) error {
	orderStatus, err := s.WebhookRepo.GetOrderStatusByName(ctx, statusName)
	if err != nil {
		return err
	}

	if orderStatus.IsFinal {
		if valid, err := s.isValidEventOrderStatus(ctx, event); err != nil || !valid {
			if err != nil {
				return err
			}

			return models.ErrInternalServer
		}
	}

	event.OrderStatusID = orderStatus.ID

	if err = s.WebhookRepo.AddEvent(ctx, event); err != nil {
		return err
	}

	return nil
}

func (s *Service) GetEventHistory(ctx context.Context, orderID uuid.UUID) ([]models.EventBody, error) {
	eventHistory, err := s.WebhookRepo.GetOrderEvents(ctx, orderID)
	if err != nil {
		return nil, err
	}

	res := make([]models.EventBody, 0, len(eventHistory))
	for i := range eventHistory {
		res = append(res, models.EventBody{
			EventID:     eventHistory[i].EventID.String(),
			OrderID:     eventHistory[i].OrderID.String(),
			UserID:      eventHistory[i].UserID.String(),
			OrderStatus: eventHistory[i].OrderStatusName,
			UpdatedAt:   eventHistory[i].UpdatedAt.String(),
			CreatedAt:   eventHistory[i].CreatedAt.String(),
		})
	}

	return res, err
}

func (s *Service) isValidEventOrderStatus(ctx context.Context, event models.Event) (bool, error) {
	var (
		exists bool
		err    error
	)

	switch event.OrderStatusID {
	case models.FailedID, models.ChangedMyMindID:
		exists, err = s.EventExists(ctx, event.OrderID, []int{models.ChinazesID, models.ChangedMyMindID, models.GiveMyMoneyBackID})
		if err != nil || exists {
			return false, err
		}
	case models.ChinazesID:
		exists, err = s.EventExists(ctx, event.OrderID, []int{models.ConfirmedByMayorID})
		if err != nil || !exists {
			return false, err
		}
	case models.GiveMyMoneyBackID:
		eventChinazes, err := s.GetEventByOrderStatus(ctx, event.OrderID, models.ChinazesID)
		if err != nil {
			return false, err
		}

		if event.UpdatedAt.Sub(eventChinazes.UpdatedAt) > 30*time.Second {
			return false, err
		}
	default:
		return false, models.ErrInternalServer
	}

	return true, nil
}

func (s *Service) StreamStatus(ctx context.Context, orderID uuid.UUID) (bool, error) {
	completed, err := s.WebhookRepo.CheckCompletedOrderStatusByID(ctx, orderID)
	if err != nil {
		return false, err
	}

	return completed, nil
}

func (s *Service) AddCompletedOrder(ctx context.Context, orderID uuid.UUID) error {
	if err := s.WebhookRepo.AddCompletedOrder(ctx, orderID); err != nil {
		return err
	}

	return nil
}
