package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	TimeFormat = "2006-01-02T15:04:05Z"

	CoolOrderCreated       = "cool_order_created"
	SBUVarificationPending = "sbu_varification_pending"
	ConfirmedByMayor       = "confirmed_by_mayor"
	ChangedMyMind          = "changed_my_mind"
	Failed                 = "failed"
	Chinazes               = "chinazes"
	GiveMyMoneyBack        = "give_my_money_back"
)

const (
	ChinazesID        = 6
	GiveMyMoneyBackID = 7
)

const (
	GiveMyMoneyBackTimeout time.Duration = 30 * time.Second
	InactivityTimeout      time.Duration = 1 * time.Minute
)

type EventBody struct {
	EventID     string `json:"event_id"`
	OrderID     string `json:"order_id"`
	UserID      string `json:"user_id"`
	OrderStatus string `json:"order_status"`
	UpdatedAt   string `json:"updated_at"`
	CreatedAt   string `json:"created_at"`
}

type EventMsg struct {
	EventID     uuid.UUID `json:"event_id"`
	OrderID     uuid.UUID `json:"order_id"`
	UserID      uuid.UUID `json:"user_id"`
	OrderStatus string    `json:"order_status"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type Event struct {
	EventID       uuid.UUID `json:"event_id"`
	OrderID       uuid.UUID `json:"order_id"`
	UserID        uuid.UUID `json:"user_id"`
	OrderStatusID int       `json:"order_status_id"`
	UpdatedAt     time.Time `json:"updated_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type FullEventInfo struct {
	EventID         uuid.UUID `json:"event_id"`
	OrderID         uuid.UUID `json:"order_id"`
	UserID          uuid.UUID `json:"user_id"`
	OrderStatusID   int       `json:"order_status_id"`
	UpdatedAt       time.Time `json:"updated_at"`
	CreatedAt       time.Time `json:"created_at"`
	OrderStatusName string    `json:"order_status_name"`
	IsFinal         bool      `json:"is_final"`
}
