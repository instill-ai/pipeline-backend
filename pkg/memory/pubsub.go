package memory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Event contains the information published on a channel, reflecting an event
// that happened in the system.
type Event struct {
	Name string `json:"name"`
	Data any    `json:"data"`
}

// EventPublisher is used to publish a message in a channel.
type EventPublisher interface {
	PublishEvent(_ context.Context, channel string, _ Event) error
}

// EventSubscriber is used to receive messages in a channel.
type EventSubscriber interface {
	Subscribe(_ context.Context, channel string) Subscription
}

// Subscription is used to read messages from a channel.
type Subscription interface {
	Receive(context.Context) (*Event, error)
	Unsubscribe(_ context.Context, channels ...string) error
}

// RedisPubSub is a Redis-based event publisher and subscriber.
type RedisPubSub struct {
	client *redis.Client
}

// NewRedisPubSub returns an initialized RedisPubSub.
func NewRedisPubSub(client *redis.Client) *RedisPubSub {
	return &RedisPubSub{
		client: client,
	}
}

// PublishEvent publishes an event on a Redis channel.
func (r *RedisPubSub) PublishEvent(ctx context.Context, channel string, ev Event) error {
	b, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshalling event: %w", err)
	}

	return r.client.Publish(ctx, channel, b).Err()
}

// Subscribe creates a subscription on a Redis channel.
func (r *RedisPubSub) Subscribe(ctx context.Context, channel string) Subscription {
	return &redisSubscription{
		pubsub: r.client.Subscribe(ctx, channel),
	}
}

type redisSubscription struct {
	pubsub *redis.PubSub
}

func (rs *redisSubscription) Receive(ctx context.Context) (*Event, error) {
	msg, err := rs.pubsub.ReceiveMessage(ctx)
	if err != nil {
		return nil, fmt.Errorf("receiving message: %w", err)
	}

	event := new(Event)
	if err := json.Unmarshal([]byte(msg.Payload), event); err != nil {
		return nil, fmt.Errorf("unmarshalling message: %w", err)
	}

	return event, nil
}

func (rs *redisSubscription) Unsubscribe(ctx context.Context, channels ...string) error {
	return rs.pubsub.Unsubscribe(ctx, channels...)
}
