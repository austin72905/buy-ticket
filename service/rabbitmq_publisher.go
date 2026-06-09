package service

import (
	"context"

	infraRabbitMQ "github.com/austin72905/go-infra/rabbitmq"
)

type RabbitMQPublisher struct {
	client *infraRabbitMQ.Client
}

func NewRabbitMQPublisher(url string) *RabbitMQPublisher {
	return &RabbitMQPublisher{
		client: infraRabbitMQ.New(url),
	}
}

func (p *RabbitMQPublisher) PublishJSON(ctx context.Context, exchange, routingKey string, body []byte) error {
	return p.client.Publish(
		ctx,
		exchange,
		routingKey,
		body,
		infraRabbitMQ.WithContentType("application/json"),
		infraRabbitMQ.WithPersistentDelivery(),
	)
}

func (p *RabbitMQPublisher) Close() error {
	return p.client.Close()
}
