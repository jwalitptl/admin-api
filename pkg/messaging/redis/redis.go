package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jwalitptl/admin-api/pkg/circuitbreaker"
	"github.com/jwalitptl/admin-api/pkg/messaging"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type RedisBroker struct {
	client *redis.Client
	cb     *circuitbreaker.CircuitBreaker
	logger *zerolog.Logger
}

type Config struct {
	URL          string
	MaxRetries   int
	RetryBackoff time.Duration
	PoolSize     int
	MinIdleConns int
}

func NewRedisBroker(config Config, logger *zerolog.Logger) (messaging.Broker, error) {
	opts, err := redis.ParseURL(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Configure connection pooling
	opts.MaxRetries = config.MaxRetries
	opts.MinRetryBackoff = config.RetryBackoff
	opts.PoolSize = config.PoolSize
	opts.MinIdleConns = config.MinIdleConns

	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Settings{
		Name:        "redis-broker",
		MaxRequests: 100,
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
	})

	client := redis.NewClient(opts)

	// Test connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisBroker{
		client: client,
		cb:     cb,
		logger: logger,
	}, nil
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
	msgChan := make(chan []byte, 100)

	go func() {
		defer func() {
			pubsub.Close()
			close(msgChan)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := pubsub.ReceiveMessage(ctx)
				if err != nil {
					continue
				}
				msgChan <- []byte(msg.Payload)
			}
		}
	}()

	return msgChan, nil
}

func (b *RedisBroker) Close() error {
	return b.client.Close()
}
