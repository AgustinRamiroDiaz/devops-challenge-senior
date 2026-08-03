package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
)

const defaultPort = "8080"

const gracefulShutdownTimeout = 8 * time.Second
const telemetryShutdownTimeout = 5 * time.Second

type response struct {
	Timestamp string `json:"timestamp"`
	IP        string `json:"ip"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	meterProvider, err := initMeterProvider(context.Background())
	if err != nil {
		slog.Error("failed to initialize telemetry", "error", err)
		os.Exit(1)
	}
	if meterProvider != nil {
		defer shutdownMeterProvider(meterProvider)
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           routes(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownSignal, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("SimpleTimeService listening", "port", port)
	serverError := make(chan error, 1)
	go func() {
		serverError <- server.ListenAndServe()
	}()

	select {
	case err := <-serverError:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return
		}
		slog.Error("server stopped unexpectedly", "error", err)
		os.Exit(1)
	case <-shutdownSignal.Done():
		slog.Info("shutdown signal received")
	}

	if err := shutdownServer(server); err != nil {
		slog.Error("graceful shutdown failed; forced active connections closed", "error", err)
	}

	if err := <-serverError; err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped unexpectedly", "error", err)
		os.Exit(1)
	}
}

func shutdownServer(server *http.Server) error {
	shutdownContext, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		return errors.Join(err, server.Close())
	}
	return nil
}

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

func routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/", handleTime)
	return instrumentRequests(mux)
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func handleTime(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		IP:        visitorIP(r),
	}); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func visitorIP(r *http.Request) string {
	// The ALB overwrites this header with its sanitized client_ip_address value.
	if ip := validHeaderIP(r.Header.Get("X-Client-IP")); ip != "" {
		return ip
	}

	// Local development fallbacks only; client-supplied proxy headers are spoofable.
	if ip := firstForwardedIP(r.Header.Get("X-Forwarded-For")); ip != "" {
		return ip
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func validHeaderIP(value string) string {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return ""
	}
	return ip.String()
}

func firstForwardedIP(value string) string {
	if value == "" {
		return ""
	}
	return validHeaderIP(strings.Split(value, ",")[0])
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
