package service

import (
	"context"
	"sse/models"
)

func (s *Service) GetOrders(ctx context.Context, filters *models.OrderFilter) ([]models.EventBody, error) {
	res, err := s.OrderRepo.GetOrdersByFilter(ctx, filters)
	if err != nil {
		return nil, err
	}

	return buildResponse(res), nil
}

func buildResponse(events []models.FullEventInfo) []models.EventBody {
	if events == nil || len(events) == 0 {
		return nil
	}

	res := make([]models.EventBody, 0, len(events))
	for i := range events {
		eventBody := buildEventBody(&events[i])
		res = append(res, *eventBody)
	}

	return res
}

func buildEventBody(event *models.FullEventInfo) *models.EventBody {
	return &models.EventBody{
		EventID:     event.EventID.String(),
		OrderID:     event.OrderID.String(),
		UserID:      event.UserID.String(),
		OrderStatus: event.OrderStatusName,
		UpdatedAt:   event.UpdatedAt.String(),
		CreatedAt:   event.CreatedAt.String(),
	}
}
