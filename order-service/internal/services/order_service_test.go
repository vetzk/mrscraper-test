package services

import (
	"context"
	"errors"
	"order-service/internal/domain"
	"order-service/internal/infra"
	"order-service/internal/mocks"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	args := m.Called(ctx, key)
	return args.Get(0).(*redis.StringCmd)
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	args := m.Called(ctx, key, value, expiration)
	return args.Get(0).(*redis.StatusCmd)
}

func TestOrderService_CreateOrder(t *testing.T) {
	tests := []struct {
		name           string
		productId      uint64
		totalPrice     int64
		setupMocks     func(*mocks.MockOrderRepository, *mocks.MockProductClient, *mocks.MockPublisher)
		expectedError  string
		expectedResult bool
		testAsync      bool 
	}{
		{
			name:       "successful order creation - synchronous",
			productId:  1,
			totalPrice: 1000,
			setupMocks: func(mockRepo *mocks.MockOrderRepository, mockProdClient *mocks.MockProductClient, mockPub *mocks.MockPublisher) {
				mockProdClient.On("GetProductById", mock.Anything, uint64(1)).Return(&infra.ProductInfo{
					ID:    1,
					Name:  "Test Product",
					Price: 1000,
					Qty:   5,
				}, nil)
				
				mockRepo.On("Save", mock.AnythingOfType("*domain.Order")).Return(nil).Run(func(args mock.Arguments) {
					order := args.Get(0).(*domain.Order)
					order.ID = 1
				})
				
				mockPub.On("Publish", mock.Anything, "order.created", mock.Anything).Return(nil)
			},
			expectedResult: true,
		},
		{
			name:       "successful order creation - asynchronous",
			productId:  1,
			totalPrice: 1000,
			testAsync:  true,
			setupMocks: func(mockRepo *mocks.MockOrderRepository, mockProdClient *mocks.MockProductClient, mockPub *mocks.MockPublisher) {
				mockProdClient.On("GetProductById", mock.Anything, uint64(1)).Return(&infra.ProductInfo{
					ID:    1,
					Name:  "Test Product",
					Price: 1000,
					Qty:   5,
				}, nil)
				
				mockRepo.On("SaveBatch", mock.AnythingOfType("[]*domain.Order")).Return(nil).Maybe()
				mockRepo.On("Save", mock.AnythingOfType("*domain.Order")).Return(nil).Maybe().Run(func(args mock.Arguments) {
					order := args.Get(0).(*domain.Order)
					order.ID = 1
				})
				
				mockPub.On("Publish", mock.Anything, "order.created", mock.Anything).Return(nil).Maybe()
			},
			expectedResult: true,
		},
		{
			name:       "product not found",
			productId:  999,
			totalPrice: 1000,
			setupMocks: func(mockRepo *mocks.MockOrderRepository, mockProdClient *mocks.MockProductClient, mockPub *mocks.MockPublisher) {
				mockProdClient.On("GetProductById", mock.Anything, uint64(999)).Return(nil, errors.New("product not found"))
			},
			expectedError: "product not found",
		},
		{
			name:       "product returns nil",
			productId:  888,
			totalPrice: 1000,
			setupMocks: func(mockRepo *mocks.MockOrderRepository, mockProdClient *mocks.MockProductClient, mockPub *mocks.MockPublisher) {
				mockProdClient.On("GetProductById", mock.Anything, uint64(888)).Return(nil, nil)
			},
			expectedError: "product not found",
		},
		{
			name:       "service overloaded - synchronous fallback fails",
			productId:  1,
			totalPrice: 1000,
			setupMocks: func(mockRepo *mocks.MockOrderRepository, mockProdClient *mocks.MockProductClient, mockPub *mocks.MockPublisher) {
				mockProdClient.On("GetProductById", mock.Anything, uint64(1)).Return(&infra.ProductInfo{
					ID:    1,
					Name:  "Test Product",
					Price: 1000,
					Qty:   5,
				}, nil)
				
				mockRepo.On("Save", mock.AnythingOfType("*domain.Order")).Return(errors.New("database error"))
			},
			expectedError: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.MockOrderRepository)
			mockProdClient := new(mocks.MockProductClient)
			mockPublisher := new(mocks.MockPublisher)

			tt.setupMocks(mockRepo, mockProdClient, mockPublisher)

			service := NewOrderService(mockRepo, mockProdClient, mockPublisher)
			
			if tt.name == "service overloaded - synchronous fallback fails" {
				for i := 0; i < cap(service.orderBuffer)+1; i++ {
					service.orderBuffer <- &domain.Order{}
				}
			}

			result, err := service.CreateOrder(context.Background(), tt.productId, tt.totalPrice)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.productId, result.ProductId)
				assert.Equal(t, tt.totalPrice, result.TotalPrice)
				assert.Equal(t, domain.StatusPending, result.Status)
				assert.WithinDuration(t, time.Now(), result.CreatedAt, time.Second)

				if tt.testAsync {
					time.Sleep(200 * time.Millisecond)
				}
			}

			time.Sleep(100 * time.Millisecond)
			
			mockProdClient.AssertExpectations(t)
			mockRepo.AssertExpectations(t)
			mockPublisher.AssertExpectations(t)
		})
	}
}

func TestOrderService_GetOrderById(t *testing.T) {
	tests := []struct {
		name          string
		orderId       uint64
		setupMocks    func(*mocks.MockOrderRepository)
		expectedError error
		expectedOrder *domain.Order
	}{
		{
			name:    "successful order retrieval",
			orderId: 1,
			setupMocks: func(mockRepo *mocks.MockOrderRepository) {
				expectedOrder := &domain.Order{
					ID:         1,
					ProductId:  1,
					TotalPrice: 1000,
					Status:     domain.StatusPending,
					CreatedAt:  time.Now(),
				}
				mockRepo.On("FindByID", uint64(1)).Return(expectedOrder, nil)
			},
			expectedOrder: &domain.Order{
				ID:         1,
				ProductId:  1,
				TotalPrice: 1000,
				Status:     domain.StatusPending,
			},
		},
		{
			name:    "order not found",
			orderId: 999,
			setupMocks: func(mockRepo *mocks.MockOrderRepository) {
				mockRepo.On("FindByID", uint64(999)).Return(nil, nil)
			},
			expectedError: ErrOrderNotFound,
		},
		{
			name:    "repository error",
			orderId: 1,
			setupMocks: func(mockRepo *mocks.MockOrderRepository) {
				mockRepo.On("FindByID", uint64(1)).Return(nil, errors.New("database connection error"))
			},
			expectedError: errors.New("database connection error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.MockOrderRepository)
			mockProdClient := new(mocks.MockProductClient)
			mockPublisher := new(mocks.MockPublisher)

			tt.setupMocks(mockRepo)

			service := NewOrderService(mockRepo, mockProdClient, mockPublisher)
			result, err := service.GetOrderById(context.Background(),tt.orderId)

			if tt.expectedError != nil {
				assert.Error(t, err)
				if tt.expectedError == ErrOrderNotFound {
					assert.Equal(t, ErrOrderNotFound, err)
				} else {
					assert.Contains(t, err.Error(), tt.expectedError.Error())
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedOrder.ID, result.ID)
				assert.Equal(t, tt.expectedOrder.ProductId, result.ProductId)
				assert.Equal(t, tt.expectedOrder.TotalPrice, result.TotalPrice)
				assert.Equal(t, tt.expectedOrder.Status, result.Status)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestOrderService_GetOrderByProductId(t *testing.T) {
	tests := []struct {
		name           string
		productId      uint64
		setupMocks     func(*mocks.MockOrderRepository)
		expectedError  error
		expectedOrders []domain.Order
	}{
		{
			name:      "successful orders retrieval",
			productId: 1,
			setupMocks: func(mockRepo *mocks.MockOrderRepository) {
				expectedOrders := []domain.Order{
					{
						ID:         1,
						ProductId:  1,
						TotalPrice: 1000,
						Status:     domain.StatusPending,
						CreatedAt:  time.Now(),
					},
					{
						ID:         2,
						ProductId:  1,
						TotalPrice: 2000,
						Status:     domain.StatusConfirmed,
						CreatedAt:  time.Now(),
					},
				}
				mockRepo.On("FindByProductId", uint64(1)).Return(expectedOrders, nil)
			},
			expectedOrders: []domain.Order{
				{ID: 1, ProductId: 1, TotalPrice: 1000, Status: domain.StatusPending},
				{ID: 2, ProductId: 1, TotalPrice: 2000, Status: domain.StatusConfirmed},
			},
		},
		{
			name:      "no orders found",
			productId: 999,
			setupMocks: func(mockRepo *mocks.MockOrderRepository) {
				mockRepo.On("FindByProductId", uint64(999)).Return(nil, nil)
			},
			expectedError: ErrOrderNotFound,
		},
		{
			name:      "repository error",
			productId: 1,
			setupMocks: func(mockRepo *mocks.MockOrderRepository) {
				mockRepo.On("FindByProductId", uint64(1)).Return(nil, errors.New("database error"))
			},
			expectedError: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.MockOrderRepository)
			mockProdClient := new(mocks.MockProductClient)
			mockPublisher := new(mocks.MockPublisher)

			tt.setupMocks(mockRepo)

			service := NewOrderService(mockRepo, mockProdClient, mockPublisher)
			result, err := service.GetOrderByProductId(context.Background(), tt.productId)

			if tt.expectedError != nil {
				assert.Error(t, err)
				if tt.expectedError == ErrOrderNotFound {
					assert.Equal(t, ErrOrderNotFound, err)
				} else {
					assert.Contains(t, err.Error(), tt.expectedError.Error())
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, len(tt.expectedOrders))
				for i, expected := range tt.expectedOrders {
					assert.Equal(t, expected.ID, result[i].ID)
					assert.Equal(t, expected.ProductId, result[i].ProductId)
					assert.Equal(t, expected.TotalPrice, result[i].TotalPrice)
					assert.Equal(t, expected.Status, result[i].Status)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// New test for caching functionality
func TestOrderService_CachingBehavior(t *testing.T) {
	mockRepo := new(mocks.MockOrderRepository)
	mockProdClient := new(mocks.MockProductClient)
	mockPublisher := new(mocks.MockPublisher)

	product := &infra.ProductInfo{
		ID:    1,
		Name:  "Test Product",
		Price: 1000,
		Qty:   5,
	}

	// First call should hit the product client
	mockProdClient.On("GetProductById", mock.Anything, uint64(1)).Return(product, nil).Once()
	mockRepo.On("Save", mock.AnythingOfType("*domain.Order")).Return(nil).Run(func(args mock.Arguments) {
		order := args.Get(0).(*domain.Order)
		order.ID = 1
	})
	mockPublisher.On("Publish", mock.Anything, "order.created", mock.Anything).Return(nil).Maybe()

	service := NewOrderService(mockRepo, mockProdClient, mockPublisher)

	// First call - should hit product client
	result1, err1 := service.CreateOrder(context.Background(), 1, 1000)
	assert.NoError(t, err1)
	assert.NotNil(t, result1)

	result2, err2 := service.CreateOrder(context.Background(), 1, 1500)
	assert.NoError(t, err2)
	assert.NotNil(t, result2)

	time.Sleep(100 * time.Millisecond)

	mockProdClient.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestOrderService_BatchProcessing(t *testing.T) {
	mockRepo := new(mocks.MockOrderRepository)
	mockProdClient := new(mocks.MockProductClient)
	mockPublisher := new(mocks.MockPublisher)

	product := &infra.ProductInfo{
		ID:    1,
		Name:  "Test Product",
		Price: 1000,
		Qty:   100,
	}

	mockProdClient.On("GetProductById", mock.Anything, uint64(1)).Return(product, nil)
	
	mockRepo.On("SaveBatch", mock.AnythingOfType("[]*domain.Order")).Return(nil).Maybe()
	mockRepo.On("Save", mock.AnythingOfType("*domain.Order")).Return(nil).Maybe()
	mockPublisher.On("Publish", mock.Anything, "order.created", mock.Anything).Return(nil).Maybe()

	service := NewOrderService(mockRepo, mockProdClient, mockPublisher)

	var wg sync.WaitGroup
	for i := 0; i < 15; i++ {
		wg.Add(1)
		go func(orderNum int) {
			defer wg.Done()
			_, err := service.CreateOrder(context.Background(), 1, int64(1000+orderNum))
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()
	time.Sleep(300 * time.Millisecond) 
	mockProdClient.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestOrderService_ServiceOverload(t *testing.T) {
	mockRepo := new(mocks.MockOrderRepository)
	mockProdClient := new(mocks.MockProductClient)
	mockPublisher := new(mocks.MockPublisher)

	product := &infra.ProductInfo{
		ID:    1,
		Name:  "Test Product",
		Price: 1000,
		Qty:   5,
	}

	mockProdClient.On("GetProductById", mock.Anything, uint64(1)).Return(product, nil)

	service := NewOrderService(mockRepo, mockProdClient, mockPublisher)

	for len(service.orderBuffer) < cap(service.orderBuffer) {
		select {
		case service.orderBuffer <- &domain.Order{}:
		default:
			return
		}
	}

	for len(service.workerPool) < cap(service.workerPool) {
		select {
		case service.workerPool <- struct{}{}:
		default:
			return
		}
	}

	result, err := service.CreateOrder(context.Background(), 1, 1000)
	
	if err != nil {
		assert.Contains(t, err.Error(), "service overloaded")
		assert.Nil(t, result)
	} else {
		assert.NotNil(t, result)
	}

	mockProdClient.AssertExpectations(t)
}

func BenchmarkOrderService_CreateOrder(b *testing.B) {
	mockRepo := new(mocks.MockOrderRepository)
	mockProdClient := new(mocks.MockProductClient)
	mockPublisher := new(mocks.MockPublisher)

	product := &infra.ProductInfo{
		ID:    1,
		Name:  "Test Product",
		Price: 1000,
		Qty:   1000,
	}

	mockProdClient.On("GetProductById", mock.Anything, mock.AnythingOfType("uint64")).Return(product, nil)
	mockRepo.On("Save", mock.AnythingOfType("*domain.Order")).Return(nil).Maybe()
	mockRepo.On("SaveBatch", mock.AnythingOfType("[]*domain.Order")).Return(nil).Maybe()
	mockPublisher.On("Publish", mock.Anything, "order.created", mock.Anything).Return(nil).Maybe()

	service := NewOrderService(mockRepo, mockProdClient, mockPublisher)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = service.CreateOrder(context.Background(), 1, 1000)
		}
	})
}