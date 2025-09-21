package domain

import "time"

type OrderStatus string

const (
    StatusPending   OrderStatus = "pending"
    StatusConfirmed OrderStatus = "confirmed"
    StatusFailed    OrderStatus = "failed"
)

type Order struct {
    ID         uint64      `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
    ProductId  uint64      `json:"productId" gorm:"not null;index;column:product_id"`  // Fixed naming
    TotalPrice int64       `json:"totalPrice" gorm:"not null;column:total_price"`      // Fixed naming
    Status     OrderStatus `json:"status" gorm:"type:varchar(20);default:'pending';column:status"` // Fixed enum
    CreatedAt  time.Time   `json:"createdAt" gorm:"autoCreateTime;column:created_at"`   // Fixed naming
}

func (Order) TableName() string {
    return "orders"
}