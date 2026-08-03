package main

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRootReturnsTimestampAndVisitorIP(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Client-IP", "203.0.113.10")
	request.Header.Set("X-Forwarded-For", "198.51.100.20, 10.0.0.1")
	recorder := httptest.NewRecorder()

	routes().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}
	responseInstanceID := recorder.Header().Get(instanceIDHeader)
	if len(responseInstanceID) != 32 {
		t.Fatalf("expected 128-bit instance ID header, got %q", responseInstanceID)
	}
	if _, err := hex.DecodeString(responseInstanceID); err != nil {
		t.Fatalf("expected hexadecimal instance ID header, got %q", responseInstanceID)
	}

	var body response
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.IP != "203.0.113.10" {
		t.Errorf("expected forwarded visitor IP, got %q", body.IP)
	}
	if _, err := time.Parse(time.RFC3339Nano, body.Timestamp); err != nil {
		t.Errorf("expected RFC3339 timestamp, got %q: %v", body.Timestamp, err)
	}
}

func TestInstanceIDIsStableForProcessLifetime(t *testing.T) {
	first := httptest.NewRecorder()
	second := httptest.NewRecorder()

	routes().ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))
	routes().ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/", nil))

	if first.Header().Get(instanceIDHeader) != second.Header().Get(instanceIDHeader) {
		t.Fatal("expected the instance ID to remain stable for the process lifetime")
	}
}

func TestVisitorIPFallsBackToForwardedHeaderForLocalDevelopment(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Forwarded-For", "198.51.100.20, 10.0.0.1")

	if got := visitorIP(request); got != "198.51.100.20" {
		t.Fatalf("expected forwarded visitor IP fallback, got %q", got)
	}
}

func TestVisitorIPIgnoresInvalidClientHeaders(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "192.0.2.40:12345"
	request.Header.Set("X-Client-IP", "not-an-ip")
	request.Header.Set("X-Forwarded-For", "still-not-an-ip")

	if got := visitorIP(request); got != "192.0.2.40" {
		t.Fatalf("expected remote address fallback, got %q", got)
	}
}

func TestUnknownPathReturnsNotFound(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	recorder := httptest.NewRecorder()

	routes().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", recorder.Code)
	}
}

func TestHealthReturnsOK(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	routes().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("expected plain-text content type, got %q", got)
	}
	if got := recorder.Body.String(); got != "ok\n" {
		t.Fatalf("expected health response %q, got %q", "ok\n", got)
	}
}

func TestShutdownWaitsForActiveRequest(t *testing.T) {
	requestStarted := make(chan struct{})
	finishRequest := make(chan struct{})

	testServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(requestStarted)
		<-finishRequest
		w.WriteHeader(http.StatusNoContent)
	}))
	testServer.Start()
	t.Cleanup(testServer.Close)

	requestDone := make(chan error, 1)
	go func() {
		response, err := testServer.Client().Get(testServer.URL)
		if err == nil {
			_ = response.Body.Close()
		}
		requestDone <- err
	}()

	<-requestStarted
	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- shutdownServer(testServer.Config)
	}()

	select {
	case err := <-shutdownDone:
		t.Fatalf("shutdown returned before the active request completed: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(finishRequest)

	if err := <-requestDone; err != nil {
		t.Fatalf("active request failed during graceful shutdown: %v", err)
	}
	if err := <-shutdownDone; err != nil {
		t.Fatalf("graceful shutdown failed: %v", err)
	}
}
