package main

import (
	"context"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"

	"order-service/internal/controllers/http"
	"order-service/internal/infra"
	mmysql "order-service/internal/infra/mysql"
	"order-service/internal/infra/rabbitmq"
	mysqlrepo "order-service/internal/repository/mysql"
	"order-service/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			return duration
		}
	}
	return defaultVal
}

func main() {
	// Set optimal Go runtime settings
	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)
	
	// Database connection with optimized settings
	db, err := mmysql.NewMySQLFromEnv()
	if err != nil {
		log.Fatalf("db: connect: %v", err)
	}

	sqlDB, _ := db.DB()
	// Optimize database connection pool for high throughput
	maxOpenConns := getEnvInt("DB_MAX_OPEN_CONNS", numCPU*50) // Scale with CPU
	maxIdleConns := getEnvInt("DB_MAX_IDLE_CONNS", numCPU*10)
	
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(getEnvDuration("DB_CONN_MAX_LIFETIME", 3*time.Minute))
	sqlDB.SetConnMaxIdleTime(getEnvDuration("DB_CONN_MAX_IDLE_TIME", 30*time.Second))

	log.Printf("Database pool: MaxOpen=%d, MaxIdle=%d", maxOpenConns, maxIdleConns)

	repo := mysqlrepo.NewOrderRepository(db)

	// Product client with optimized timeouts
	productTimeout := getEnvDuration("PRODUCT_CLIENT_TIMEOUT", 500*time.Millisecond)
	productClient := infra.NewProductClient(os.Getenv("PRODUCT_SERVICE_URL"), productTimeout)

	publisher, err := rabbitmq.NewPublisher(os.Getenv("RABBITMQ_URL"), "order.exchange")
	if err != nil {
		log.Fatalf("failed to init publisher: %v", err)
	}

	s := services.NewOrderService(repo, productClient, publisher)

	// Redis with optimized connection pool
	redisPoolSize := getEnvInt("REDIS_POOL_SIZE", numCPU*50)
	redisMinIdle := getEnvInt("REDIS_MIN_IDLE", numCPU*5)
	
	redisClient := redis.NewClient(&redis.Options{
		Addr:         os.Getenv("REDIS_HOST") + ":6379",
		DB:           0,
		PoolSize:     redisPoolSize,
		MinIdleConns: redisMinIdle,
		DialTimeout:  getEnvDuration("REDIS_DIAL_TIMEOUT", 1*time.Second),
		ReadTimeout:  getEnvDuration("REDIS_READ_TIMEOUT", 200*time.Millisecond),
		WriteTimeout: getEnvDuration("REDIS_WRITE_TIMEOUT", 200*time.Millisecond),
		PoolTimeout:  getEnvDuration("REDIS_POOL_TIMEOUT", 1*time.Second),
		IdleTimeout:  getEnvDuration("REDIS_IDLE_TIMEOUT", 5*time.Minute),
		MaxRetries:   3,
		MaxRetryBackoff: 100 * time.Millisecond,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Redis connection failed: %v", err)
	} else {
		log.Printf("Redis connected: Pool=%d, MinIdle=%d", redisPoolSize, redisMinIdle)
	}

	s.SetRedisClient(redisClient)

	// Aggressive cache warmup
	go func() {
		time.Sleep(2 * time.Second) // Reduced warmup delay
		warmupProducts := []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10} // More products
		
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		if err := s.WarmupProductCache(ctx, warmupProducts); err != nil {
			log.Printf("Failed to warm up cache: %v", err)
		} else {
			log.Println("Cache warmed up successfully")
		}
	}()

	// Service stats monitoring
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		
		for range ticker.C {
			stats := s.GetServiceStats()
			log.Printf("Service Performance: %+v", stats)
		}
	}()

	handler := http.NewHandler(s, redisClient)

	// Optimize Gin for production
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	
	// Use optimized middleware
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		// Basic performance headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Next()
	})
	
	handler.RegisterRoutes(r)

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		stats := s.GetServiceStats()
		c.JSON(200, gin.H{
			"status": "healthy",
			"stats":  stats,
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting order service on port %s with %d CPU cores", port, numCPU)
	log.Printf("Service configuration: BufferSize=5000, WorkerPool=%d", numCPU*100)
	
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server run: %v", err)
	}
}