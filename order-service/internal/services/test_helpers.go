package services

import (
	"order-service/internal/domain"
	"order-service/internal/infra"
	"time"
)


func CreateMockOrder(id uint64, productId uint64, totalPrice int64, status domain.OrderStatus) *domain.Order {
	return &domain.Order{
		ID:         id,
		ProductId:  productId,
		TotalPrice: totalPrice,
		Status:     status,
		CreatedAt:  time.Now(),
	}
}

func CreateMockProduct(id uint64, name string, price int64, qty int64) *infra.ProductInfo {
	return &infra.ProductInfo{
		ID:    id,
		Name:  name,
		Price: price,
		Qty:   qty,
	}
}

const (
	TestProductID    = uint64(1)
	TestOrderID      = uint64(1)
	TestTotalPrice   = int64(1000)
	TestProductName  = "Test Product"
	TestProductPrice = int64(1000)
	TestProductQty   = 5
)