package repository

import (
	"order-service/internal/domain"
)

type OrderRepository interface {
	Save(order *domain.Order) error
	FindByID(id uint64) (*domain.Order, error)
	FindByProductId(id uint64) ([]domain.Order, error)
}