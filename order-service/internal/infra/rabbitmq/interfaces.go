package rabbitmq

import "context"

type PublisherInterface interface {
	Publish(ctx context.Context, routingKey string, data any) error
}

var _ PublisherInterface = (*Publisher)(nil)