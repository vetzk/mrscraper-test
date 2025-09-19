package http

import (
	"context"
	"encoding/json"
	"net/http"
	"order-service/internal/services"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type Handler struct {
	service *services.OrderService
	rdb *redis.Client
}

func NewHandler(u *services.OrderService, rdb *redis.Client) *Handler {
	return &Handler{service: u, rdb: rdb}
}

func (h *Handler) RegisterRoutes(r *gin.Engine){
	r.POST("/orders", h.CreateOrder)
	r.GET("/orders/product/:productId", h.GetOrderByProduct)
}

func (h *Handler) CreateOrder(c *gin.Context) {
	var req struct {
		ProductID uint64 `json:"productId" binding:"required"`
		TotalPrice int64 `json:"totalPrice" binding:"required,min=0"`
	}
	if err := c.ShouldBindJSON(&req); err !=nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx:= c.Request.Context()

	order, err := h.service.CreateOrder(ctx, req.ProductID, req.TotalPrice)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cacheKey := "orders:product" + strconv.FormatUint(req.ProductID, 10)
	h.rdb.Del(context.Background(), cacheKey)

	c.JSON(http.StatusCreated, gin.H{"id": order.ID})
}

func (h *Handler) GetOrderByProduct(c *gin.Context) {
	productIdStr := c.Param("productId")
	if productIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "productId required"})
		return
	}
	productId, _ := strconv.ParseUint(productIdStr, 10, 64)
	cacheKey := "orders:product" + productIdStr

	ctx:= context.Background()
	b, err := h.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var orders []map[string]any
		_ = json.Unmarshal([]byte(b), &orders)
		c.JSON(http.StatusOK, orders)
		return
	}

	orders, err  := h.service.GetOrderByProductId(ctx, productId)

	 if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    data, _ := json.Marshal(orders)
    h.rdb.Set(ctx, cacheKey, data, 10*time.Second)

    c.JSON(http.StatusOK, orders)
}
