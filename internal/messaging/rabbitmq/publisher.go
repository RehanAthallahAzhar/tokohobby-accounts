package rabbitmq

import (
	"context"

	"github.com/RehanAthallahAzhar/tokohobby-messaging/rabbitmq"
	"github.com/sirupsen/logrus"
)

type EventPublisher struct {
	rabbitmq *rabbitmq.Publisher
	log      *logrus.Logger
}

func NewEventPublisher(rmq *rabbitmq.RabbitMQ, log *logrus.Logger) *EventPublisher {
	return &EventPublisher{
		rabbitmq: rabbitmq.NewPublisher(rmq),
		log:      log,
	}
}

// publish event user registered
func (p *EventPublisher) PublishUserRegistered(ctx context.Context, event UserRegisteredEvent) error {
	opts := rabbitmq.PublishOptions{
		Exchange:   "user.events",
		RoutingKey: "user.registered",
		Mandatory:  false,
		Immediate:  false,
	}
	err := p.rabbitmq.Publish(ctx, opts, event)
	if err != nil {
		p.log.Errorf("Failed to publish user.registered event: %v", err)
		return err
	}
	p.log.Debugf("Published user.registered event for user: %s", event.UserID)
	return nil
}
