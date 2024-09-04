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
	AddEvent(ctx context.Context, event models.Event, isFinal bool) error
	GetOrderEvents(ctx context.Context, orderID uuid.UUID) ([]models.FullEventInfo, error)
	EventExists(ctx context.Context, orderID uuid.UUID, orderStatusIDs []int) (bool, error)
	GetEventByOrderStatus(ctx context.Context, orderID uuid.UUID, orderStatusID int) (*models.Event, error)
	GetOrderStatusByName(ctx context.Context, name string) (*models.OrderStatus, error)
	CheckCompletedOrderStatusByID(ctx context.Context, orderID uuid.UUID) (bool, error)
	AddCompletedOrder(ctx context.Context, orderID uuid.UUID) error
	GetEventByID(ctx context.Context, eventID uuid.UUID) (*models.Event, error)
	GetLastUpdatedEventByOrderID(ctx context.Context, orderID uuid.UUID) (*models.Event, error)
}

type OrderRepo interface {
	GetOrdersByFilter(ctx context.Context, filters *models.OrderFilter) ([]models.FullEventInfo, error)
}
