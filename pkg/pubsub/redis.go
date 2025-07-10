package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	logx "github.com/instill-ai/x/log"
)

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

// PublishEvent publishes an event on a Redis topic.
func (r *RedisPubSub) PublishEvent(ctx context.Context, topic string, ev Event) error {
	b, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshalling event: %w", err)
	}

	return r.client.Publish(ctx, topic, b).Err()
}

// Subscribe creates a subscription on a Redis channel.
func (r *RedisPubSub) Subscribe(ctx context.Context, topic string) Subscription {
	log, _ := logx.GetZapLogger(ctx)
	log.Info("Subscribe", zap.String("topic", topic))

	return &redisSubscription{
		topic:  topic,
		pubsub: r.client.Subscribe(ctx, topic),
		logger: log,
	}
}

type redisSubscription struct {
	topic  string
	pubsub *redis.PubSub
	logger *zap.Logger
}

func (rs *redisSubscription) Cleanup(ctx context.Context) error {
	return errors.Join(
		rs.pubsub.Unsubscribe(ctx, rs.topic),
		rs.pubsub.Close(),
	)
}

func (rs *redisSubscription) Channel() <-chan Event {
	redisChannel := rs.pubsub.Channel()
	eventChannel := make(chan Event)

	go func() {
		defer close(eventChannel)
		for msg := range redisChannel {
			var event Event
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				rs.logger.Error("Couldn't unmarshal Event message", zap.Error(err))
				continue
			}
			eventChannel <- event
		}
	}()

	return eventChannel
}
