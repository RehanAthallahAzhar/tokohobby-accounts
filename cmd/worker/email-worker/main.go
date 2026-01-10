package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	accountMsg "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/messaging"
	messaging "github.com/RehanAthallahAzhar/tokohobby-messaging-go"
)

func main() {
	log.Println("Starting Email Worker...")

	// Initialize RabbitMQ
	rmqConfig := messaging.DefaultConfig()
	rmq, err := messaging.NewRabbitMQ(rmqConfig)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rmq.Close()

	// Setup queue
	if err := messaging.SetupWelcomeEmailQueue(rmq); err != nil {
		log.Fatalf("Failed to setup queue: %v", err)
	}

	// Create email service (dummy untuk sekarang)
	emailService := NewEmailService()

	// Create message handler
	handler := func(ctx context.Context, body []byte) error {
		var event accountMsg.UserRegisteredEvent

		if err := messaging.UnmarshalMessage(body, &event); err != nil {
			return fmt.Errorf("failed to unmarshal: %w", err)
		}
		log.Printf("Processing welcome email for user: %s (%s)",
			event.Username, event.Email)
		// Send email
		return emailService.SendWelcomeEmail(event.Email, event.Username)
	}

	// Create consumer
	consumerOpts := messaging.ConsumerOptions{
		QueueName:   "user.welcome.email",
		WorkerCount: 3, // 3 concurrent workers
		AutoAck:     false,
	}
	consumer := messaging.NewConsumer(rmq, consumerOpts, handler)

	// Start consuming dengan context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consumer in goroutine
	go func() {
		if err := consumer.Start(ctx); err != nil {
			log.Printf("Consumer error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down worker...")
	cancel() // Cancel context untuk stop consumer
}

// EmailService dummy untuk demo
type EmailService struct{}

func NewEmailService() *EmailService {
	return &EmailService{}
}
func (s *EmailService) SendWelcomeEmail(email, username string) error {
	// TODO: Implement actual email sending (SMTP)
	log.Printf("[MOCK] Welcome email sent to %s (%s)", username, email)

	// Simulasi delay
	// time.Sleep(1 * time.Second)

	return nil
}
