package tracer

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Config menyimpan konfigurasi untuk OpenTelemetry TracerProvider.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Endpoint       string  // contoh: "localhost:4318" atau "jaeger:4318" (OTLP HTTP)
	Insecure       bool    // set true jika tidak menggunakan TLS/HTTPS di environment lokal/dev
	SamplerRatio   float64 // 1.0 = 100% trace (dev), 0.1 = 10% trace (prod)
}

// InitTracer menginisialisasi OpenTelemetry TracerProvider dan menyetel global TracerProvider & TextMapPropagator.
// Mengembalikan fungsi shutdown yang WAJIB dipanggil saat aplikasi berhenti (misal via defer) untuk me-flush span buffer.
func InitTracer(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if cfg.ServiceName == "" {
		cfg.ServiceName = "accounts-service"
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = "localhost:4318"
	}
	if cfg.SamplerRatio <= 0 {
		cfg.SamplerRatio = 1.0 // Default 100% untuk development
	}

	// OTLP Collector
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.Endpoint),
	}
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	// Define metadata
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create otel resource: %w", err)
	}

	// Set sampler
	var sampler sdktrace.Sampler
	if cfg.SamplerRatio >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else {
		sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SamplerRatio))
	}

	// SDK TracerProvider with BatchSpanProcessor
	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// Set global TracerProvider and W3C TraceContext Propagator
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Return shutdown callback
	shutdownFunc := func(shutdownCtx context.Context) error {
		ctxWithTimeout, cancel := context.WithTimeout(shutdownCtx, 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctxWithTimeout); err != nil {
			return fmt.Errorf("failed to shutdown TracerProvider: %w", err)
		}
		return nil
	}

	return shutdownFunc, nil
}

// GetTracer mengembalikan instance Tracer terkonfigurasi untuk package tertentu.
func GetTracer(name string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(name)
}
