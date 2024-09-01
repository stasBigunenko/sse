package models

import "github.com/google/uuid"

const (
	SortByCreatedAt = "created_at"
	SortByUpdatedAt = "updated_at"

	OrderASC  = "ASC"
	OrderDESC = "DESC"
)

type OrderFilter struct {
	Status    []string  `json:"status"`
	UserID    uuid.UUID `json:"user_id"`
	Limit     int       `json:"limit"`
	Offset    int       `json:"offset"`
	IsFinal   *bool     `json:"is_final"`
	SortBy    string    `json:"sort_by"`
	SortOrder string    `json:"sort_order"`
}
