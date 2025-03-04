package pubsub

import (
	"context"
)

// Event contains the information published on a topic, reflecting an event
// that happened in the system.
type Event struct {
	Name string `json:"name"`
	Data any    `json:"data"`
}

// EventPublisher is used to publish a message in a topic.
type EventPublisher interface {
	PublishEvent(_ context.Context, topic string, _ Event) error
}

// EventSubscriber is used to receive messages in a topic.
type EventSubscriber interface {
	Subscribe(_ context.Context, topic string) Subscription
}

// Subscription is used to read messages from a topic.
type Subscription interface {
	Channel() <-chan Event
	// Cleanup will clean up the subscription data, including the channel.
	Cleanup(context.Context) error
}

const workflowTopicPrefix = "workflow-"

// WorkflowStatusTopic returns the channel name for the status updates of a
// workflow.
func WorkflowStatusTopic(workflowID string) string {
	return workflowTopicPrefix + workflowID
}
