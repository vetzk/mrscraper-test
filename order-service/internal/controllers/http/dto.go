package http

type CreateOrderRequest struct {
	ProductID  uint64 `json:"productId" binding:"required"`
	TotalPrice int64  `json:"totalPrice" binding:"required,min=0"`
}

type CreateOrderResponse struct {
	ID uint64 `json:"id"`
}