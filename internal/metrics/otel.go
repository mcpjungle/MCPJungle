package metrics

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	"github.com/mcpjungle/mcpjungle/internal/model"
)

const serviceName = "mcpjungle"

// Config holds otel configuration options
type Config struct {
	// Enabled determines if otel is enabled
	Enabled bool
	// ServerMode is the server mode (dev, prod)
	ServerMode string
	// MetricsPath is the path for metrics endpoint (default: /metrics)
	MetricsPath string
}

// DefaultConfig returns a default otel configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:     false,
		ServerMode:  string(model.ModeDev),
		MetricsPath: "/metrics",
	}
}

// LoadConfigFromEnv loads otel configuration from environment variables
func LoadConfigFromEnv() *Config {
	config := DefaultConfig()

	// Load server mode first
	if serverMode := os.Getenv("SERVER_MODE"); serverMode != "" {
		config.ServerMode = serverMode
		// Automatically enable OTEL for production mode
		if serverMode == string(model.ModeProd) {
			config.Enabled = true
		}
	}

	// Check if otel is explicitly enabled/disabled
	if enabled := os.Getenv("OTEL_ENABLED"); enabled != "" {
		switch enabled {
		case "true", "1":
			config.Enabled = true
		default:
			config.Enabled = false
		}
	}

	// Load metrics configuration
	if metricsPath := os.Getenv("METRICS_PATH"); metricsPath != "" {
		config.MetricsPath = metricsPath
	}

	return config
}

// Providers holds the Otel provider and HTTP handler for metrics
type Providers struct {
	Config        *Config
	MeterProvider *sdkmetric.MeterProvider
	Meter         metric.Meter
	Handler       http.Handler
}

// InitOTel initializes Otel with the provided configuration
func InitOTel(ctx context.Context, config *Config) (*Providers, error) {
	// If otel is disabled, return empty providers
	if !config.Enabled {
		return &Providers{
			Config: config,
		}, nil
	}

	// Create resource with service information
	res, err := sdkresource.New(ctx,
		sdkresource.WithFromEnv(),
		sdkresource.WithHost(),
		sdkresource.WithProcess(),
		sdkresource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.DeploymentEnvironment(config.ServerMode),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
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
	meter := meterProvider.Meter(serviceName)

	// Create HTTP handler for metrics endpoint
	handler := promhttp.Handler()

	providers := &Providers{
		Config:        config,
		MeterProvider: meterProvider,
		Meter:         meter,
		Handler:       handler,
	}

	return providers, nil
}

// RegisterMetricsHandlers registers the metrics and health endpoints with the provided mux
func (p *Providers) RegisterMetricsHandlers(mux *http.ServeMux) {
	if !p.Config.Enabled {
		return
	}

	// Register metrics endpoint
	mux.Handle(p.Config.MetricsPath, p.Handler)

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}

// Shutdown gracefully shuts down the Openotel providers
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
	return p.Config != nil && p.Config.Enabled
}
