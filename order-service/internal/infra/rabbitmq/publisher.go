package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/streadway/amqp"
)

type Publisher struct {
    conn     *amqp.Connection
    channel  *amqp.Channel
    exchange string
}

type NestJSMessage struct {
    Pattern string      `json:"pattern"`
    Data    interface{} `json:"data"`
    ID      string      `json:"id,omitempty"`
}

func NewPublisher(amqpURL, exchange string) (*Publisher, error) {
    conn, err := amqp.Dial(amqpURL)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
    }

    channel, err := conn.Channel()
    if err != nil {
        conn.Close()
        return nil, fmt.Errorf("failed to open channel: %v", err)
    }

    err = channel.ExchangeDeclare(
        exchange, 
        "topic",  
        true,     
        false,    
        false,    
        false,    
        nil,      
    )
    if err != nil {
        channel.Close()
        conn.Close()
        return nil, fmt.Errorf("failed to declare exchange: %v", err)
    }

    return &Publisher{
        conn:     conn,
        channel:  channel,
        exchange: exchange,
    }, nil
}

func (p *Publisher) Publish(ctx context.Context, pattern string, data interface{}) error {
    message := NestJSMessage{
        Pattern: pattern,
        Data:    data,
    }

    body, err := json.Marshal(message)
    if err != nil {
        return fmt.Errorf("failed to marshal message: %v", err)
    }

    log.Printf("Publishing message with pattern '%s' to exchange '%s'", pattern, p.exchange)
    log.Printf("Message body: %s", string(body))

    err = p.channel.Publish(
        p.exchange,
        pattern,    
        false,      
        false,      
        amqp.Publishing{
            ContentType: "application/json",
            Body:        body,
        },
    )
    if err != nil {
        return fmt.Errorf("failed to publish message: %v", err)
    }

    return nil
}

func (p *Publisher) Close() {
    if p.channel != nil {
        p.channel.Close()
    }
    if p.conn != nil {
        p.conn.Close()
    }
}