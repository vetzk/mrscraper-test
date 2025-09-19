package domain

import "time"

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusConfirmed OrderStatus = "confirmed"
	StatusFailed    OrderStatus = "failed"
)

type Order struct {
	ID         uint64      `json:"id" gorm:"primaryKey;autoIncrement"`
	ProductId  uint64      `json:"productId" gorm:"not null;index"`
	TotalPrice int64       `json:"totalPrice" gorm:"not null"`
	Status     OrderStatus `json:"status" gorm:"type:enum('pending','confirmed','failed');default:'pending'"`
	CreatedAt  time.Time `json:"createdAt" gorm:"autoCreateTime"`
}