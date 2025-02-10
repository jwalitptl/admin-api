package messaging

import (
	"context"
	"encoding/json"
)

type BrokerAdapter struct {
	broker Broker
}

func NewBrokerAdapter(broker Broker) MessageBroker {
	return &BrokerAdapter{broker: broker}
}

func (a *BrokerAdapter) Publish(ctx context.Context, topic string, payload []byte) error {
	var msg interface{}
	if err := json.Unmarshal(payload, &msg); err != nil {
		return err
	}
	return a.broker.Publish(ctx, topic, msg)
}

func (a *BrokerAdapter) Close() error {
	return a.broker.Close()
}

func (a *BrokerAdapter) Subscribe(ctx context.Context, topic string, handler func([]byte) error) error {
	msgChan, err := a.broker.Subscribe(ctx, topic)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgChan {
			if err := handler(msg); err != nil {
				// Log error but continue processing
				continue
			}
		}
	}()

	return nil
}
