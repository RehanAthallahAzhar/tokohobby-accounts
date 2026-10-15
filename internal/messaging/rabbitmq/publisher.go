package rabbitmq

import (
	"context"
	"log/slog"

	"github.com/RehanAthallahAzhar/tokohobby-messaging/rabbitmq"
)

type EventPublisher struct {
	rabbitmq *rabbitmq.Publisher
	log      *slog.Logger
}

func NewEventPublisher(rmq *rabbitmq.RabbitMQ, log *slog.Logger) *EventPublisher {
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
		p.log.ErrorContext(ctx, "failed to publish user.registered event",
			slog.String("user_id", event.UserID),
			slog.Any("error", err),
		)
		return err
	}
	p.log.DebugContext(ctx, "published user.registered event", slog.String("user_id", event.UserID))
	return nil
}
