package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
)

const defaultPort = "8080"

const gracefulShutdownTimeout = 8 * time.Second

const instanceIDHeader = "X-Instance-ID"

var instanceID = newInstanceID()

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

	// This process-scoped value lets load tests distinguish Cloud Run instances.
	w.Header().Set(instanceIDHeader, instanceID)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		IP:        visitorIP(r),
	}); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func newInstanceID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		panic("generate instance ID: " + err.Error())
	}
	return hex.EncodeToString(value)
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
