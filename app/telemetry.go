package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
)

const telemetryShutdownTimeout = 5 * time.Second

func initMeterProvider(ctx context.Context) (*sdkmetric.MeterProvider, error) {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return nil, nil
	}

	exporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("simple-time-service"),
		)),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(
			exporter,
			sdkmetric.WithInterval(30*time.Second),
			sdkmetric.WithTimeout(5*time.Second),
		)),
	)
	otel.SetMeterProvider(meterProvider)

	return meterProvider, nil
}

func shutdownMeterProvider(meterProvider *sdkmetric.MeterProvider) {
	shutdownContext, cancel := context.WithTimeout(context.Background(), telemetryShutdownTimeout)
	defer cancel()

	if err := meterProvider.Shutdown(shutdownContext); err != nil {
		slog.Error("failed to flush telemetry", "error", err)
	}
}

func instrumentRequests(next http.Handler) http.Handler {
	meter := otel.Meter("simple-time-service")
	requestCount, err := meter.Int64Counter(
		"simple_time_requests_total",
		metric.WithDescription("Total HTTP requests handled by SimpleTimeService."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		slog.Error("failed to create request counter", "error", err)
	}

	requestDuration, err := meter.Float64Histogram(
		"simple_time_request_duration_ms",
		metric.WithDescription("HTTP request duration in milliseconds."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		slog.Error("failed to create request duration histogram", "error", err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		attrs := metric.WithAttributes(
			attribute.String("http.request.method", r.Method),
			attribute.String("http.route", routeName(r.URL.Path)),
			attribute.Int("http.response.status_code", recorder.statusCode),
		)
		if requestCount != nil {
			requestCount.Add(r.Context(), 1, attrs)
		}
		if requestDuration != nil {
			requestDuration.Record(r.Context(), float64(time.Since(start).Milliseconds()), attrs)
		}
	})
}

func routeName(path string) string {
	switch path {
	case "/":
		return "/"
	case "/healthz":
		return "/healthz"
	default:
		return "unknown"
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
