package service

import (
	"context"

	"github.com/google/uuid"

	"sse/models"
)

type Service struct {
	WebhookRepo
	OrderRepo
}

func New(webhookRepo WebhookRepo, orderRepo OrderRepo) *Service {
	return &Service{
		webhookRepo,
		orderRepo,
	}
}

type WebhookRepo interface {
	AddEvent(ctx context.Context, event models.Event) error
	GetOrderEvents(ctx context.Context, orderID uuid.UUID) ([]models.FullEventInfo, error)
	GetOrderStatusByName(ctx context.Context, name string) (*models.OrderStatus, error)
	GetEventByID(ctx context.Context, eventID uuid.UUID) (*models.Event, error)
	GetLastUpdatedEventByOrderID(ctx context.Context, orderID uuid.UUID) (*models.FullEventInfo, error)
}

type OrderRepo interface {
	GetOrdersByFilter(ctx context.Context, filters *models.OrderFilter) ([]models.FullEventInfo, error)
}
