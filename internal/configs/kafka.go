package configs

type KafkaConfig struct {
	Brokers string `env:"KAFKA_BROKERS" envDefault:""`
}
