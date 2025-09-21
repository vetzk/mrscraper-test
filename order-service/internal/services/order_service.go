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
	"runtime"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"golang.org/x/sync/singleflight"
)

var ErrOrderNotFound = errors.New("order not found")

type OrderService struct {
    repo           repository.OrderRepository
    prodClient     infra.ProductClientInterface
    publisher      rabbit.PublisherInterface
    redisClient    *redis.Client
    
    // Performance optimizations
    sf             singleflight.Group
    localCache     *sync.Map
    
    // Connection pools
    dbWorkers      chan struct{}
    eventWorkers   chan struct{}
    
    stats          *ServiceStats
}

type ServiceStats struct {
    mu                  sync.RWMutex
    TotalRequests       int64
    SuccessfulOrders    int64
    FailedOrders        int64
    CacheHits           int64
    CacheMisses         int64
    AvgResponseTime     time.Duration
}

func (s *ServiceStats) IncrementTotalRequests() {
    s.mu.Lock()
    s.TotalRequests++
    s.mu.Unlock()
}

func (s *ServiceStats) IncrementSuccessfulOrders() {
    s.mu.Lock()
    s.SuccessfulOrders++
    s.mu.Unlock()
}

func (s *ServiceStats) IncrementFailedOrders() {
    s.mu.Lock()
    s.FailedOrders++
    s.mu.Unlock()
}

func (s *ServiceStats) IncrementCacheHits() {
    s.mu.Lock()
    s.CacheHits++
    s.mu.Unlock()
}

func (s *ServiceStats) IncrementCacheMisses() {
    s.mu.Lock()
    s.CacheMisses++
    s.mu.Unlock()
}

func (s *ServiceStats) GetStats() (int64, int64, int64, int64, int64) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.TotalRequests, s.SuccessfulOrders, s.FailedOrders, s.CacheHits, s.CacheMisses
}

type cachedProduct struct {
    product   interface{}
    expiresAt time.Time
}

func (cp *cachedProduct) isExpired() bool {
    return time.Now().After(cp.expiresAt)
}

func NewOrderService(r repository.OrderRepository, p infra.ProductClientInterface, pub rabbit.PublisherInterface) *OrderService {
    numCPU := runtime.NumCPU()
    
    service := &OrderService{
        repo:         r,
        prodClient:   p,
        publisher:    pub,
        localCache:   &sync.Map{},
        dbWorkers:    make(chan struct{}, numCPU*20),  // Limit concurrent DB operations
        eventWorkers: make(chan struct{}, numCPU*30),  // Separate pool for events
        stats:        &ServiceStats{},
    }
    
    go service.logStats()
    return service
}

func (u *OrderService) SetRedisClient(client *redis.Client) {
    u.redisClient = client
}

// BALANCED APPROACH: Fast response + reliable data
func (u *OrderService) CreateOrder(ctx context.Context, productId uint64, totalPrice int64) (*domain.Order, error) {
    start := time.Now()
    u.stats.IncrementTotalRequests()
    
    // FAST PATH: Parallel product validation
    productChan := make(chan interface{}, 1)
    productErrChan := make(chan error, 1)
    
    go func() {
        if u.isProductValidCached(productId) {
            productChan <- "cached"
            productErrChan <- nil
            return
        }
        
        prod, err := u.getProductWithFastCache(ctx, productId)
        if err != nil {
            productErrChan <- err
            return
        }
        if prod == nil {
            productErrChan <- errors.New("product not found")
            return
        }
        productChan <- prod
        productErrChan <- nil
    }()
    
    // Create order object immediately
    order := &domain.Order{
        ProductId:  productId,
        TotalPrice: totalPrice,
        Status:     domain.StatusPending,
        CreatedAt:  time.Now(),
    }
    
    // Wait for product validation with timeout
    select {
    case <-productChan:
        // Product is valid, continue
    case err := <-productErrChan:
        if err != nil {
            u.stats.IncrementFailedOrders()
            return nil, fmt.Errorf("product validation failed: %w", err)
        }
    case <-time.After(200 * time.Millisecond):
        u.stats.IncrementFailedOrders()
        return nil, errors.New("product validation timeout")
    }
    
    // CRITICAL: Save to database with connection pooling
    select {
    case u.dbWorkers <- struct{}{}:
        defer func() { <-u.dbWorkers }()
        
        if err := u.repo.Save(order); err != nil {
            u.stats.IncrementFailedOrders()
            return nil, fmt.Errorf("failed to save order: %w", err)
        }
        
        if order.ID == 0 {
            u.stats.IncrementFailedOrders()
            return nil, errors.New("order saved but ID not assigned")
        }
        
    case <-time.After(100 * time.Millisecond):
        u.stats.IncrementFailedOrders()
        return nil, errors.New("database connection timeout")
    }
    
    u.stats.IncrementSuccessfulOrders()
    
    // Async event publishing with separate worker pool
    select {
    case u.eventWorkers <- struct{}{}:
        go func() {
            defer func() { <-u.eventWorkers }()
            u.publishOrderCreatedEvent(context.Background(), order)
        }()
    default:
        // Event worker pool full, skip event (or log warning)
        log.Printf("Event worker pool full, skipping event for order %d", order.ID)
    }
    
    // Log response time
    elapsed := time.Since(start)
    if elapsed > 500*time.Millisecond {
        log.Printf("Slow order creation: %v for order %d", elapsed, order.ID)
    }
    
    return order, nil
}

// FAST CACHE: Optimized for speed
func (u *OrderService) isProductValidCached(productId uint64) bool {
    if val, ok := u.localCache.Load(productId); ok {
        if cached, ok := val.(*cachedProduct); ok && !cached.isExpired() {
            u.stats.IncrementCacheHits()
            return true
        }
    }
    u.stats.IncrementCacheMisses()
    return false
}

func (u *OrderService) getProductWithFastCache(ctx context.Context, productId uint64) (interface{}, error) {
    cacheKey := fmt.Sprintf("product:%d", productId)
    
    // Use singleflight to prevent thundering herd
    result, err, _ := u.sf.Do(cacheKey, func() (interface{}, error) {
        // Level 1: Local cache (fastest)
        if val, ok := u.localCache.Load(productId); ok {
            if cached, ok := val.(*cachedProduct); ok && !cached.isExpired() {
                return cached.product, nil
            }
        }

        // Level 2: Redis with very short timeout
        if u.redisClient != nil {
            ctx, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
            defer cancel()
            
            cached, err := u.redisClient.Get(ctx, cacheKey).Result()
            if err == nil {
                var prod interface{}
                if err := json.Unmarshal([]byte(cached), &prod); err == nil {
                    // Update local cache
                    u.localCache.Store(productId, &cachedProduct{
                        product:   prod,
                        expiresAt: time.Now().Add(30 * time.Second),
                    })
                    return prod, nil
                }
            }
        }

        // Level 3: Product service with short timeout
        ctx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
        defer cancel()
        
        prod, err := u.prodClient.GetProductById(ctx, productId)
        if err != nil {
            return nil, fmt.Errorf("product service error: %w", err)
        }

        if prod != nil {
            // Cache immediately (synchronous for consistency)
            u.localCache.Store(productId, &cachedProduct{
                product:   prod,
                expiresAt: time.Now().Add(30 * time.Second),
            })
            
            // Redis cache async
            if u.redisClient != nil {
                go func() {
                    if data, err := json.Marshal(prod); err == nil {
                        ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
                        defer cancel()
                        u.redisClient.Set(ctx, cacheKey, data, 5*time.Minute)
                    }
                }()
            }
        }

        return prod, nil
    })

    return result, err
}

func (u *OrderService) publishOrderCreatedEvent(ctx context.Context, order *domain.Order) {
    evt := map[string]any{
        "orderId":    order.ID,
        "productId":  order.ProductId,
        "totalPrice": order.TotalPrice,
        "createdAt":  order.CreatedAt,
    }

    // Single attempt with timeout for performance
    ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
    defer cancel()
    
    if err := u.publisher.Publish(ctx, "order.created", evt); err != nil {
        log.Printf("Failed to publish event for order %d: %v", order.ID, err)
    }
}

func (u *OrderService) logStats() {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        total, success, failed, hits, misses := u.stats.GetStats()
        
        if total > 0 {
            successRate := float64(success) / float64(total) * 100
            hitRate := float64(0)
            if hits+misses > 0 {
                hitRate = float64(hits) / float64(hits+misses) * 100
            }
            
            log.Printf("OrderService: Total=%d, Success=%.1f%%, Failed=%d, Cache=%.1f%%, DBPool=%d/%d, EventPool=%d/%d",
                total, successRate, failed, hitRate, 
                len(u.dbWorkers), cap(u.dbWorkers),
                len(u.eventWorkers), cap(u.eventWorkers))
        }
    }
}

func (u *OrderService) GetOrderById(ctx context.Context, id uint64) (*domain.Order, error) {
    ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
    defer cancel()
    
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
    ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()
    
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

    // Parallel warmup with limited concurrency
    sem := make(chan struct{}, 10)
    var wg sync.WaitGroup
    
    for _, id := range productIds {
        wg.Add(1)
        go func(productId uint64) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()
            
            ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
            defer cancel()
            
            if _, err := u.getProductWithFastCache(ctx, productId); err != nil {
                log.Printf("Cache warmup failed for product %d: %v", productId, err)
            }
        }(id)
    }
    
    wg.Wait()
    return nil
}

func (u *OrderService) GetServiceStats() map[string]interface{} {
    total, success, failed, hits, misses := u.stats.GetStats()
    
    successRate := float64(0)
    if total > 0 {
        successRate = float64(success) / float64(total) * 100
    }
    
    hitRate := float64(0)
    if hits+misses > 0 {
        hitRate = float64(hits) / float64(hits+misses) * 100
    }
    
    return map[string]interface{}{
        "total_requests":     total,
        "successful_orders":  success,
        "failed_orders":      failed,
        "success_rate":       successRate,
        "cache_hit_rate":     hitRate,
        "db_pool_usage":      float64(len(u.dbWorkers)) / float64(cap(u.dbWorkers)) * 100,
        "event_pool_usage":   float64(len(u.eventWorkers)) / float64(cap(u.eventWorkers)) * 100,
    }
}