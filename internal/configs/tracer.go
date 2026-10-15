package configs

type TracerConfig struct {
	ServiceName    string  `env:"OTEL_SERVICE_NAME,required"`
	ServiceVersion string  `env:"OTEL_SERVICE_VERSION,required"`
	Environment    string  `env:"ENV,required"`
	Endpoint       string  `env:"OTEL_EXPORTER_OTLP_ENDPOINT,required"`
	Insecure       bool    `env:"OTEL_EXPORTER_OTLP_INSECURE,required"`
	SamplerRatio   float64 `env:"OTEL_SAMPLER_RATIO,required"`
}
