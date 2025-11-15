package rabbitmq

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/streadway/amqp"
)

type RabbitMQClient struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel
	QueueName  string
}

const maxRetries = 5
const retryInterval = time.Second * 5

func NewRabbitMQClient(queueName string) (*RabbitMQClient, error) {
	var conn *amqp.Connection
	var err error

	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	if rabbitMQURL == "" {
		rabbitMQURL = "amqp://guest:guest@localhost:5672/"
		log.Printf("RABBITMQ_URL not set, using default: %s", rabbitMQURL)
	}

	for i := 0; i < maxRetries; i++ {
		conn, err = amqp.Dial(rabbitMQURL)
		if err == nil {
			log.Println("Berhasil terhubung ke RabbitMQ")
			break
		}

		log.Printf("Gagal terhubung ke RabbitMQ (percobaan %d/%d): %v. Mencoba lagi dalam %v...", i+1, maxRetries, err, retryInterval)
		time.Sleep(retryInterval)
	}

	if err != nil {
		return nil, err
	}

	log.Println("Connected to RabbitMQ")

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}
	log.Println("RabbitMQ channel opened")

	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable (pesan tidak hilang saat RabbitMQ restart)
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare a queue '%s': %w", queueName, err)
	}
	log.Printf("Queue '%s' declared, containing %d messages", q.Name, q.Messages)

	return &RabbitMQClient{
		Connection: conn,
		Channel:    ch,
		QueueName:  queueName,
	}, nil
}

func (rc *RabbitMQClient) Close() {
	if rc.Channel != nil {
		log.Println("Closing RabbitMQ channel...")
		err := rc.Channel.Close()
		if err != nil {
			log.Printf("Failed to close RabbitMQ channel: %v", err)
		}
	}
	if rc.Connection != nil {
		log.Println("Closing RabbitMQ connection...")
		err := rc.Connection.Close()
		if err != nil {
			log.Printf("Failed to close RabbitMQ connection: %v", err)
		}
	}
}

func (rc *RabbitMQClient) PublishMessage(body []byte) error {
	err := rc.Channel.Publish(
		"",           // exchange (kosong berarti default exchange)
		rc.QueueName, // routing key (sama dengan queue name untuk default exchange)
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:  "application/json", // Tipe konten pesan
			DeliveryMode: amqp.Persistent,    // Pesan tahan lama (akan disimpan di disk jika RabbitMQ crash)
			Timestamp:    time.Now(),
			Body:         body,
		})
	if err != nil {
		return fmt.Errorf("failed to publish a message: %w", err)
	}
	log.Printf("Message published to queue '%s'", rc.QueueName)
	return nil
}

func (rc *RabbitMQClient) ConsumeMessages(handler func(msg amqp.Delivery) error) error {
	msgs, err := rc.Channel.Consume(
		rc.QueueName, // queue
		"",           // consumer (unique string to identify the consumer)
		false,        // auto-ack (kita akan ack secara manual setelah diproses)
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %w", err)
	}
	log.Printf("Waiting for messages in queue '%s'...", rc.QueueName)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			if err := handler(d); err != nil {
				log.Printf("Error processing message: %v", err)
				d.Nack(false, true)
			} else {
				d.Ack(false)
			}
		}
	}()

	<-forever
	return nil
}
