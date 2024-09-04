package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"sse/models"
)

type OrdersRepo struct {
	*Postgres
}

func (p *Postgres) NewOrdersRepo() *OrdersRepo {
	return &OrdersRepo{p}
}

func (p *OrdersRepo) GetOrdersByFilter(ctx context.Context, filters *models.OrderFilter) ([]models.FullEventInfo, error) {
	query, params := buildQuery(filters)

	rows, err := p.db.Query(context.Background(), query, params...)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	defer rows.Close()

	var res []models.FullEventInfo
	for rows.Next() {
		var event models.FullEventInfo
		err = rows.Scan(&event.EventID, &event.OrderID, &event.UserID, &event.OrderStatusID, &event.CreatedAt,
			&event.UpdatedAt, &event.OrderStatusName, &event.IsFinal)
		if err != nil {
			return nil, err
		}
		res = append(res, event)
	}

	return res, nil
}

func buildQuery(filter *models.OrderFilter) (string, []interface{}) {
	baseQuery := `SELECT e.event_id, e.order_id, e.user_id, e.order_status_id, e.created_at, e.updated_at, os.name AS order_status_name, os.is_final
			 FROM events e
			 JOIN order_statuses os ON e.order_status_id = os.id
			 WHERE 1=1`

	var (
		paramIndex int
		params     []interface{}
	)

	if filter.Status != nil && len(filter.Status) != 0 {
		baseQuery += fmt.Sprintf(" AND os.name IN (")
		placeholders := []string{}
		for i := range filter.Status {
			placeholders = append(placeholders, fmt.Sprintf("$%d", paramIndex+1+i))
			params = append(params, filter.Status[i])
		}
		baseQuery += strings.Join(placeholders, ", ") + ")"
		paramIndex += len(filter.Status)
	}

	if filter.UserID != uuid.Nil {
		paramIndex++
		baseQuery += fmt.Sprintf(" AND user_id = $%d", paramIndex)
		params = append(params, filter.UserID)
	}

	if filter.IsFinal != nil {
		paramIndex++
		baseQuery += fmt.Sprintf(" AND is_final = $%d", paramIndex)
		params = append(params, filter.IsFinal)
	}

	baseQuery += fmt.Sprintf(" ORDER BY %s %s", filter.SortBy, filter.SortOrder)
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", paramIndex+1, paramIndex+2)
	params = append(params, filter.Limit, filter.Offset)

	return baseQuery, params
}
