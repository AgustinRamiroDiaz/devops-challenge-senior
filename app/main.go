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
)

const defaultPort = "8080"

const gracefulShutdownTimeout = 8 * time.Second

type response struct {
	Timestamp string `json:"timestamp"`
	IP        string `json:"ip"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
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
	return mux
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
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if ip := strings.TrimSpace(strings.Split(forwarded, ",")[0]); ip != "" {
			return ip
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
