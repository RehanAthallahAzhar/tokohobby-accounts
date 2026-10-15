package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/configs"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/messaging/rabbitmq"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/logger"
	rabbitmqpkg "github.com/RehanAthallahAzhar/tokohobby-messaging/rabbitmq"
)

func main() {
	log := logger.NewLogger()

	log.Info("Starting Email Worker")

	cfg, err := configs.LoadConfig(log)
	if err != nil {
		log.Error("Failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}

	rmqConfig := &rabbitmqpkg.RabbitMQConfig{
		URL:            cfg.RabbitMQ.URL,
		MaxRetries:     cfg.RabbitMQ.MaxRetries,
		RetryDelay:     cfg.RabbitMQ.RetryDelay,
		PrefetchCount:  cfg.RabbitMQ.PrefetchCount,
		ReconnectDelay: cfg.RabbitMQ.ReconnectDelay,
	}

	log.Info("Connecting to RabbitMQ", slog.String("url", cfg.RabbitMQ.URL))

	// Connect to RabbitMQ
	rmq, err := rabbitmqpkg.NewRabbitMQ(rmqConfig)
	if err != nil {
		log.Error("Failed to connect to RabbitMQ", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := rmq.Close(); err != nil {
			log.Error("Failed to close RabbitMQ connection cleanly", slog.Any("error", err))
		}
	}()

	log.Info("Connected to RabbitMQ successfully")

	// Setup user exchange (ensure it exists)
	if err := rabbitmqpkg.SetupUserExchange(rmq); err != nil {
		log.Error("Failed to setup user exchange", slog.Any("error", err))
		os.Exit(1)
	}

	log.Info("User exchange setup complete")

	// Create email service
	emailService := NewEmailService(log)

	// Create message handler
	handler := func(ctx context.Context, body []byte) error {
		var event rabbitmq.UserRegisteredEvent

		if err := rabbitmqpkg.UnmarshalMessage(body, &event); err != nil {
			return fmt.Errorf("failed to unmarshal: %w", err)
		}

		log.Info("Processing welcome email for user",
			slog.String("username", event.Username),
			slog.String("email", event.Email),
		)

		// Send email
		return emailService.SendWelcomeEmail(event.Email, event.Username, event.UserID)
	}

	// Create consumer with options
	consumerOpts := rabbitmqpkg.ConsumerOptions{
		QueueName:   "email.user.welcome",
		WorkerCount: 3, // 3 concurrent workers
		AutoAck:     false,
	}

	consumer := rabbitmqpkg.NewConsumer(rmq, consumerOpts, handler)

	// Declare queue
	if err := consumer.DeclareQueue(true, false); err != nil {
		log.Error("Failed to declare queue", slog.Any("error", err))
		os.Exit(1)
	}

	log.Info("Queue declared: email.user.welcome")

	// Bind queue to exchange
	if err := consumer.BindQueue("user.events", "user.registered"); err != nil {
		log.Error("Failed to bind queue", slog.Any("error", err))
		os.Exit(1)
	}

	log.Info("Queue bound to exchange: user.events (routing key: user.registered)")

	// Start consuming with context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consumer in goroutine
	go func() {
		if err := consumer.Start(ctx); err != nil {
			log.Warn("Consumer error", slog.Any("error", err))
		}
	}()

	log.Info("Email worker is running. Waiting for messages... (Press Ctrl+C to exit)")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down email worker...")
	cancel() // Cancel context to stop consumer

	// Give workers time to finish processing
	time.Sleep(2 * time.Second)

	log.Info("Email worker stopped gracefully")
}

// EmailService handles email sending logic
type EmailService struct {
	log *slog.Logger
}

func NewEmailService(log *slog.Logger) *EmailService {
	return &EmailService{log: log}
}

func (s *EmailService) SendWelcomeEmail(email, username, userID string) error {
	// TODO: Implement actual SMTP email sending
	s.log.Info("Sending welcome email",
		slog.String("email", email),
		slog.String("username", username),
		slog.String("user_id", userID),
	)

	// Simulate email sending delay
	time.Sleep(500 * time.Millisecond)

	s.log.Info("[MOCK] Welcome email sent", slog.String("username", username))
	return nil
}
