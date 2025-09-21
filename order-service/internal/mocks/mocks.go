// Add to your mocks/mock_order_repository.go file

package mocks

import (
	"context"
	"order-service/internal/domain"
	"order-service/internal/infra"

	"github.com/stretchr/testify/mock"
)

type MockOrderRepository struct {
	mock.Mock
}

type MockProductClient struct {
	mock.Mock
}

type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) Publish(ctx context.Context, topic string, message interface{}) error {
	args := m.Called(ctx, topic, message)
	return args.Error(0)
}

func (m *MockProductClient) GetProductById(ctx context.Context, productId uint64) (*infra.ProductInfo, error) {
	args := m.Called(ctx, productId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*infra.ProductInfo), args.Error(1)
}

func (m *MockOrderRepository) Save(order *domain.Order) error {
	args := m.Called(order)
	return args.Error(0)
}

func (m *MockOrderRepository) SaveBatch(orders []*domain.Order) error {
	args := m.Called(orders)
	return args.Error(0)
}

func (m *MockOrderRepository) FindByID(id uint64) (*domain.Order, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Order), args.Error(1)
}

func (m *MockOrderRepository) FindByProductId(productId uint64) ([]domain.Order, error) {
	args := m.Called(productId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Order), args.Error(1)
}