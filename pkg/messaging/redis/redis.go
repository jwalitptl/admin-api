package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jwalitptl/admin-api/pkg/circuitbreaker"
	"github.com/jwalitptl/admin-api/pkg/messaging"
	"github.com/redis/go-redis/v9"
)

type RedisBroker struct {
	client *redis.Client
	cb     *circuitbreaker.CircuitBreaker
}

func NewRedisBroker(url string) (messaging.Broker, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Settings{
		Name:        "redis-broker",
		MaxRequests: 100,
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
	})

	client := redis.NewClient(opts)
	return &RedisBroker{client: client, cb: cb}, nil
}

func (b *RedisBroker) Publish(ctx context.Context, channel string, message interface{}) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	return b.client.Publish(ctx, channel, payload).Err()
}

func (b *RedisBroker) Subscribe(ctx context.Context, channel string) (<-chan []byte, error) {
	pubsub := b.client.Subscribe(ctx, channel)
	msgChan := make(chan []byte)

	go func() {
		defer pubsub.Close()
		defer close(msgChan)

		for {
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				return
			}
			msgChan <- []byte(msg.Payload)
		}
	}()

	return msgChan, nil
}

func (b *RedisBroker) Close() error {
	return b.client.Close()
}
