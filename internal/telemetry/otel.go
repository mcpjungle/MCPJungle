package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// Config holds otel configuration options
type Config struct {
	ServiceName string
	Enabled     bool
}

// Providers holds the Otel configuration and metrics provider.
// Eventually, it will also hold providers for tracing and logging
type Providers struct {
	Config        *Config
	MeterProvider *sdkmetric.MeterProvider
	Meter         metric.Meter
}

// Init initializes Otel with the provided configuration
func Init(ctx context.Context, config *Config) (*Providers, error) {
	// If otel is disabled, return empty providers
	if !config.Enabled {
		return &Providers{
			Config: config,
		}, nil
	}

	// Create resource with service information
	res, err := sdkresource.New(
		ctx,
		sdkresource.WithFromEnv(),
		sdkresource.WithHost(),
		sdkresource.WithProcess(),
		sdkresource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create otel resource: %w", err)
	}

	// Create Prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	// Create meter provider with Prometheus exporter
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
		sdkmetric.WithResource(res),
	)

	// Set the global meter provider
	otel.SetMeterProvider(meterProvider)

	// Create meter for the service
	meter := meterProvider.Meter(config.ServiceName)

	providers := &Providers{
		Config:        config,
		MeterProvider: meterProvider,
		Meter:         meter,
	}
	return providers, nil
}

// Shutdown gracefully shuts down the otel providers
func (p *Providers) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}
	if p.MeterProvider != nil {
		if err := p.MeterProvider.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown meter provider: %w", err)
		}
	}
	return nil
}

// IsEnabled returns true if otel is enabled
func (p *Providers) IsEnabled() bool {
	return p.Config.Enabled
}

// ServiceName returns the service name configured for otel
func (p *Providers) ServiceName() string {
	return p.Config.ServiceName
}
