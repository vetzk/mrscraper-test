package main

import (
	"context"
	"log"
	"os"
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

func main() {
	db, err := mmysql.NewMySQLFromEnv()
	if err != nil {
		log.Fatalf("db: connect: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1000)
	sqlDB.SetMaxIdleConns(200)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(1 * time.Minute)

	repo := mysqlrepo.NewOrderRepository(db)

	productClient := infra.NewProductClient(os.Getenv("PRODUCT_SERVICE_URL"), 2*time.Second)

	publisher, err := rabbitmq.NewPublisher(os.Getenv("RABBITMQ_URL"), "order.exchange")
	if err != nil {
		log.Fatalf("failed to init publisher: %v", err)
	}

	s := services.NewOrderService(repo, productClient, publisher)

	redisClient := redis.NewClient(&redis.Options{
		Addr:         os.Getenv("REDIS_HOST") + ":6379",
		DB:           0,
		PoolSize:     200,
		MinIdleConns: 20,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})

	s.SetRedisClient(redisClient)

	ctx := context.Background()
	go func() {
		time.Sleep(5 * time.Second) 
		if err := s.WarmupProductCache(ctx, []uint64{1, 2}); err != nil {
			log.Printf("Failed to warm up cache: %v", err)
		} else {
			log.Println("Cache warmed up successfully")
		}
	}()

	handler := http.NewHandler(s, redisClient)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	
	handler.RegisterRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting order service on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server run: %v", err)
	}
}