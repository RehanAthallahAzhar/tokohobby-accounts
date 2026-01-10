package messaging

import (
	"context"

	messaging "github.com/RehanAthallahAzhar/tokohobby-messaging-go"
	"github.com/sirupsen/logrus"
)

type EventPublisher struct {
	publisher *messaging.Publisher
	log       *logrus.Logger
}

func NewEventPublisher(rmq *messaging.RabbitMQ, log *logrus.Logger) *EventPublisher {
	return &EventPublisher{
		publisher: messaging.NewPublisher(rmq),
		log:       log,
	}
}

// publish event user registered
func (p *EventPublisher) PublishUserRegistered(ctx context.Context, event UserRegisteredEvent) error {
	opts := messaging.PublishOptions{
		Exchange:   "user.events",
		RoutingKey: "user.registered",
		Mandatory:  false,
		Immediate:  false,
	}
	err := p.publisher.Publish(ctx, opts, event)
	if err != nil {
		p.log.Errorf("Failed to publish user.registered event: %v", err)
		return err
	}
	p.log.Debugf("Published user.registered event for user: %s", event.UserID)
	return nil
}
