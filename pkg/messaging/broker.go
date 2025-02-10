package messaging

import (
	"context"
)

// Broker defines the interface for message brokers
type Broker interface {
	Publish(ctx context.Context, channel string, message interface{}) error
	Subscribe(ctx context.Context, channel string) (<-chan []byte, error)
	Close() error
}

// Publisher defines the interface for publishing messages
type Publisher interface {
	Publish(ctx context.Context, eventType string, payload interface{}) error
}

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type MessageBroker interface {
	Publish(ctx context.Context, topic string, payload []byte) error
	Subscribe(ctx context.Context, topic string, handler func([]byte) error) error
	Close() error
}
