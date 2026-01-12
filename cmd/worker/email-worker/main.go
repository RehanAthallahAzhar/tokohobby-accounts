package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	accountsMsg "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/messaging"
	messaging "github.com/RehanAthallahAzhar/tokohobby-messaging-go"
)

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)

	logrus.Info("Starting Email Worker")
	// RabbitMQ configuration from environment
	rmqURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/tokohobby")

	rmqConfig := &messaging.RabbitMQConfig{
		URL:            rmqURL,
		MaxRetries:     3,
		RetryDelay:     5 * time.Second,
		PrefetchCount:  5,
		ReconnectDelay: 10 * time.Second,
	}

	logrus.Infof("Connecting to RabbitMQ: %s", rmqURL)

	// Connect to RabbitMQ
	rmq, err := messaging.NewRabbitMQ(rmqConfig)
	if err != nil {
		logrus.Fatalf("❌ Failed to connect to RabbitMQ: %v", err)
	}
	defer rmq.Close()

	logrus.Info("Connected to RabbitMQ successfully")

	// Setup user exchange (ensure it exists)
	if err := messaging.SetupUserExchange(rmq); err != nil {
		logrus.Fatalf("Failed to setup user exchange: %v", err)
	}

	logrus.Info("User exchange setup complete")

	// Create email service
	emailService := NewEmailService()

	// Create message handler
	handler := func(ctx context.Context, body []byte) error {
		var event accountsMsg.UserRegisteredEvent

		if err := messaging.UnmarshalMessage(body, &event); err != nil {
			return fmt.Errorf("failed to unmarshal: %w", err)
		}

		logrus.Infof("Processing welcome email for user: %s (%s)",
			event.Username, event.Email)

		// Send email
		return emailService.SendWelcomeEmail(event.Email, event.Username, event.UserID)
	}

	// Create consumer with options
	consumerOpts := messaging.ConsumerOptions{
		QueueName:   "email.user.welcome",
		WorkerCount: 3, // 3 concurrent workers
		AutoAck:     false,
	}

	consumer := messaging.NewConsumer(rmq, consumerOpts, handler)

	// Declare queue
	if err := consumer.DeclareQueue(true, false); err != nil {
		logrus.Fatalf("Failed to declare queue: %v", err)
	}

	logrus.Info("Queue declared: email.user.welcome")

	// Bind queue to exchange
	if err := consumer.BindQueue("user.events", "user.registered"); err != nil {
		logrus.Fatalf("Failed to bind queue: %v", err)
	}

	logrus.Info("Queue bound to exchange: user.events (routing key: user.registered)")

	// Start consuming with context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consumer in goroutine
	go func() {
		if err := consumer.Start(ctx); err != nil {
			logrus.Warnf("Consumer error: %v", err)
		}
	}()

	logrus.Info("Email worker is running. Waiting for messages... (Press Ctrl+C to exit)")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down email worker...")
	cancel() // Cancel context to stop consumer

	// Give workers time to finish processing
	time.Sleep(2 * time.Second)

	logrus.Info("Email worker stopped gracefully")
}

// EmailService handles email sending logic
type EmailService struct{}

func NewEmailService() *EmailService {
	return &EmailService{}
}

func (s *EmailService) SendWelcomeEmail(email, username, userID string) error {
	// TODO: Implement actual SMTP email sending
	// For now, just log the email

	logrus.Infof("[📨 EMAIL] Sending welcome email to: %s", email)
	logrus.Infof("   To: %s", email)
	logrus.Infof("   Username: %s", username)
	logrus.Infof("   User ID: %s", userID)

	// Simulate email sending delay
	time.Sleep(500 * time.Millisecond)

	// Example SMTP implementation (uncomment when ready):
	/*
		import "gopkg.in/gomail.v2"

		msg := gomail.NewMessage()
		msg.SetHeader("From", "noreply@tokohobby.shop")
		msg.SetHeader("To", email)
		msg.SetHeader("Subject", "Welcome to TokoHobby!")
		msg.SetBody("text/html", fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head>
				<style>
					body { font-family: Arial, sans-serif; line-height: 1.6; }
					.container { max-width: 600px; margin: 0 auto; padding: 20px; }
					.header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center; }
					.content { padding: 30px; background: #f9f9f9; }
					.button { display: inline-block; padding: 12px 30px; background: #667eea; color: white; text-decoration: none; border-radius: 5px; margin-top: 20px; }
				</style>
			</head>
			<body>
				<div class="container">
					<div class="header">
						<h1>Welcome to TokoHobby! 🎉</h1>
					</div>
					<div class="content">
						<h2>Hi %s!</h2>
						<p>Thank you for joining TokoHobby, your one-stop shop for all hobby needs.</p>
						<p>Your account is now active and ready to use. Start exploring our amazing collection of products!</p>
						<a href="https://tokohobby.shop" class="button">Start Shopping</a>
						<p style="margin-top: 30px; color: #666; font-size: 14px;">
							If you didn't create this account, please ignore this email.
						</p>
					</div>
				</div>
			</body>
			</html>
		`, username))

		d := gomail.NewDialer(
			os.Getenv("SMTP_HOST"),
			587,
			os.Getenv("SMTP_USER"),
			os.Getenv("SMTP_PASS"),
		)

		if err := d.DialAndSend(msg); err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}
	*/

	logrus.Infof("[MOCK] Welcome email sent to %s", username)
	return nil
}

// getEnv gets environment variable with default fallback
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
