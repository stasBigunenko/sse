package models

type OrderStatus struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	IsFinal bool   `json:"is_final"`
}
