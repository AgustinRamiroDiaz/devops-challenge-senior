package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRootReturnsTimestampAndVisitorIP(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Forwarded-For", "203.0.113.10, 10.0.0.1")
	recorder := httptest.NewRecorder()

	routes().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
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
