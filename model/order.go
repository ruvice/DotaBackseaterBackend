package model

import (
	"time"

	"github.com/google/uuid"
)

type Order struct {
	OrderID     uint64     `json:"order_id,omitempty"`
	CustomerID  uuid.UUID  `json:"customer_id,omitempty"`
	LineItems   []LineItem `json:"line_items,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	ShippedAt   *time.Time `json:"shipped_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type LineItem struct {
	ItemID   uuid.UUID `json:"item_id,omitempty"`
	Quantity uint      `json:"quantity,omitempty"`
	Price    uint      `json:"price,omitempty"`
}
