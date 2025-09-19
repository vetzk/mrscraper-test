package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"order-service/internal/domain"
	"order-service/internal/infra"
	rabbit "order-service/internal/infra/rabbitmq"
	"order-service/internal/repository"
	"time"

	"github.com/go-redis/redis/v8"
)

var ErrOrderNotFound = errors.New("order not found")

type OrderService struct {
	repo        repository.OrderRepository
	prodClient  infra.ProductClientInterface
	publisher   rabbit.PublisherInterface
	redisClient *redis.Client
}

func NewOrderService(r repository.OrderRepository, p infra.ProductClientInterface, pub rabbit.PublisherInterface) *OrderService {
	return &OrderService{
		repo:       r,
		prodClient: p,
		publisher:  pub,
	}
}

func (u *OrderService) SetRedisClient(client *redis.Client) {
	u.redisClient = client
}

func (u *OrderService) CreateOrder(ctx context.Context, productId uint64, totalPrice int64) (*domain.Order, error) {
	prod, err := u.getProductWithCache(ctx, productId)
	if err != nil {
		return nil, err
	}

	if prod == nil {
		return nil, errors.New("product not found")
	}



	order := &domain.Order{
		ProductId:  productId,
		TotalPrice: totalPrice,
		Status:     domain.StatusPending,
		CreatedAt:  time.Now(),
	}

	if err := u.repo.Save(order); err != nil {
		return nil, err
	}

	go u.publishOrderCreatedEvent(context.Background(), order)

	return order, nil
}

func (u *OrderService) getProductWithCache(ctx context.Context, productId uint64) (interface{}, error) {
	cacheKey := fmt.Sprintf("product:%d", productId)

	if u.redisClient != nil {
		cached, err := u.redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			var prod interface{}
			if err := json.Unmarshal([]byte(cached), &prod); err == nil {
				return prod, nil
			}
		}
	}

	prod, err := u.prodClient.GetProductById(ctx, productId)
	if err != nil {
		return nil, err
	}

	if u.redisClient != nil && prod != nil {
		if data, err := json.Marshal(prod); err == nil {
			u.redisClient.Set(ctx, cacheKey, data, time.Minute)
		}
	}

	return prod, nil
}

func (u *OrderService) publishOrderCreatedEvent(ctx context.Context, order *domain.Order) {
	evt := map[string]any{
		"orderId":    order.ID,
		"productId":  order.ProductId,
		"totalPrice": order.TotalPrice,
		"createdAt":  order.CreatedAt,
	}

	log.Printf("Publishing order.created event: %+v", evt)
	if err := u.publisher.Publish(ctx, "order.created", evt); err != nil {
		log.Printf("Failed to publish event: %v", err)
	} else {
		log.Printf("Successfully published order.created event")
	}
}

func (u *OrderService) GetOrderById(id uint64) (*domain.Order, error) {
	o, err := u.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if o == nil {
		return nil, ErrOrderNotFound
	}
	return o, nil
}

func (u *OrderService) GetOrderByProductId(ctx context.Context, id uint64) ([]domain.Order, error) {
	o, err := u.repo.FindByProductId(id)
	if err != nil {
		return nil, err
	}

	if o == nil {
		return nil, ErrOrderNotFound
	}
	return o, nil
}

func (u *OrderService) WarmupProductCache(ctx context.Context, productIds []uint64) error {
	if u.redisClient == nil {
		return nil
	}

	for _, id := range productIds {
		prod, err := u.prodClient.GetProductById(ctx, id)
		if err != nil {
			log.Printf("Failed to warm up cache for product %d: %v", id, err)
			continue
		}

		if prod != nil {
			cacheKey := fmt.Sprintf("product:%d", id)
			if data, err := json.Marshal(prod); err == nil {
				u.redisClient.Set(ctx, cacheKey, data, 5*time.Minute)
			}
		}
	}

	return nil
}