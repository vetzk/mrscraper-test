package domain

import "time"

type OrderCreatedEvent struct {
	OrderID    uint64    `json:"orderId"`
	ProductId  uint64    `json:"productId"`
	TotalPrice int64     `json:"totalPrice"`
	CreatedAt  time.Time `json:"createdAt"`
}