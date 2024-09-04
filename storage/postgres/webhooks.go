package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"sse/models"
)

type WebhookRepo struct {
	*Postgres
}

func (p *Postgres) NewWebhookRepo() *WebhookRepo {
	return &WebhookRepo{p}
}

func (p *WebhookRepo) AddEvent(ctx context.Context, event models.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	query := `INSERT INTO events (event_id, order_id, user_id, order_status_id, updated_at, created_at) 
		VALUES (@eventID, @orderID, @userID, @orderStatusID, @updatedAt, @createdAt)`
	args := pgx.NamedArgs{
		"eventID":       event.EventID,
		"orderID":       event.OrderID,
		"userID":        event.UserID,
		"orderStatusID": event.OrderStatusID,
		"updatedAt":     event.UpdatedAt,
		"createdAt":     event.CreatedAt,
	}

	_, err := p.db.Exec(ctx, query, args)
	if err != nil {
		return fmt.Errorf("unable to insert row: %w", err)
	}

	return nil
}

func (p *WebhookRepo) GetOrderEvents(ctx context.Context, orderID uuid.UUID) ([]models.FullEventInfo, error) {
	query := `SELECT e.event_id, e.order_id, e.user_id, e.order_status_id, e.created_at, e.updated_at, os.name AS order_status_name, os.is_final
			 FROM events e
			 JOIN order_statuses os ON e.order_status_id = os.id
			 WHERE e.order_id = @orderID
			 ORDER BY e.updated_at ASC`
	args := pgx.NamedArgs{
		"orderID": orderID,
	}

	rows, err := p.db.Query(ctx, query, args)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	var events []models.FullEventInfo
	for rows.Next() {
		var event models.FullEventInfo
		err = rows.Scan(&event.EventID, &event.OrderID, &event.UserID, &event.OrderStatusID, &event.CreatedAt,
			&event.UpdatedAt, &event.OrderStatusName, &event.IsFinal)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}

func (p *WebhookRepo) EventExists(ctx context.Context, orderID uuid.UUID, orderStatusIDs []int) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 
			FROM events 
			WHERE order_id = @orderID 
			AND order_status_id = ANY(@orderStatusIDs)
		)
	`
	args := pgx.NamedArgs{
		"orderID":        orderID,
		"orderStatusIDs": orderStatusIDs,
	}

	var exists bool

	// Execute the query
	err := p.db.QueryRow(ctx, query, args).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (p *WebhookRepo) GetEventByOrderStatus(ctx context.Context, orderID uuid.UUID, orderStatusID int) (*models.Event, error) {
	query := `
		SELECT event_id, order_id, user_id, order_status_id, updated_at, created_at
			FROM events 
			WHERE order_id = @orderID 
			AND order_status_id = @orderStatusID
		)
	`
	args := pgx.NamedArgs{
		"orderID":       orderID,
		"orderStatusID": orderStatusID,
	}

	var res models.Event

	// Execute the query
	err := p.db.QueryRow(ctx, query, args).
		Scan(&res.EventID, &res.OrderID, &res.UserID, &res.OrderStatusID, &res.UpdatedAt, &res.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (p *WebhookRepo) GetOrderStatusByName(ctx context.Context, name string) (*models.OrderStatus, error) {
	query := `
		SELECT id, name, is_final
			FROM order_statuses 
			WHERE name = @name
	`
	args := pgx.NamedArgs{
		"name": name,
	}

	var res models.OrderStatus
	err := p.db.QueryRow(ctx, query, args).Scan(&res.ID, &res.Name, &res.IsFinal)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (p *WebhookRepo) CheckCompletedOrderStatusByID(ctx context.Context, orderID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 
			FROM events 
			WHERE order_id = @orderID 
			  AND order_status_id IN (4, 5, 6, 7)
		)
	`
	args := pgx.NamedArgs{
		"orderID": orderID,
	}

	var exists bool

	// Execute the query
	err := p.db.QueryRow(ctx, query, args).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (p *WebhookRepo) AddCompletedOrder(ctx context.Context, orderID uuid.UUID) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	query := `INSERT INTO orders (id, completed) 
		VALUES (@eventID, @orderID, @completed)`
	args := pgx.NamedArgs{
		"orderID":   orderID,
		"completed": true,
	}

	_, err := p.db.Exec(ctx, query, args)
	if err != nil {
		return fmt.Errorf("unable to insert row: %w", err)
	}

	return nil
}

func (p *WebhookRepo) GetEventByID(ctx context.Context, eventID uuid.UUID) (*models.Event, error) {
	query := `
		SELECT event_id, order_id, user_id, order_status_id, updated_at, created_at
			FROM events 
			WHERE event_id = @eventID
	`
	args := pgx.NamedArgs{
		"eventID": eventID,
	}

	var res models.Event

	// Execute the query
	err := p.db.QueryRow(ctx, query, args).
		Scan(&res.EventID, &res.OrderID, &res.UserID, &res.OrderStatusID, &res.UpdatedAt, &res.CreatedAt)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &res, nil
}

func (p *WebhookRepo) GetLastUpdatedEventByOrderID(ctx context.Context, orderID uuid.UUID) (*models.Event, error) {
	query := `
		SELECT event_id, order_id, user_id, order_status_id, updated_at, created_at
			FROM events 
			WHERE order_id = @order_id AND MAX(updated_at)
	`
	args := pgx.NamedArgs{
		"orderID": orderID,
	}

	var res models.Event

	// Execute the query
	err := p.db.QueryRow(ctx, query, args).
		Scan(&res.EventID, &res.OrderID, &res.UserID, &res.OrderStatusID, &res.UpdatedAt, &res.CreatedAt)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &res, nil
}
