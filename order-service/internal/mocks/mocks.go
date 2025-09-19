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

func (m *MockOrderRepository) Save(order *domain.Order) error {
	args := m.Called(order)
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

type MockProductClient struct {
	mock.Mock
}

func (m *MockProductClient) GetProductById(ctx context.Context, id uint64) (*infra.ProductInfo, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*infra.ProductInfo), args.Error(1)
}

type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) Publish(ctx context.Context, routingKey string, data interface{}) error {
	args := m.Called(ctx, routingKey, data)
	return args.Error(0)
}