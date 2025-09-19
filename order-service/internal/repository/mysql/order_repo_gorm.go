package mysql

import (
	"errors"
	"order-service/internal/domain"
	"order-service/internal/repository"

	"gorm.io/gorm"
)

type orderRepo struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) repository.OrderRepository {
	return &orderRepo{db:db}
}

func (r *orderRepo) Save(order *domain.Order) error {
	return r.db.Create(order).Error
}

func (r *orderRepo) FindByID(id uint64) (*domain.Order, error) {
	var o domain.Order
	if err := r.db.First(&o, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound){
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

func (r *orderRepo) FindByProductId(productId uint64) ([]domain.Order, error) {
	var out []domain.Order
	if err := r.db.Where("product_id = ?", productId).Order("created_at_desc").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}