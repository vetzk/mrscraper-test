package infra

import "context"

type ProductClientInterface interface {
	GetProductById(ctx context.Context, id uint64) (*ProductInfo, error)
}

var _ ProductClientInterface = (*ProductClient)(nil)