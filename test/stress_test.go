package stress_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"
)

const instanceIDHeader = "X-Instance-ID"

type config struct {
	serviceURL        string
	expectedClientIP  string
	initialWorkers    int
	workerStep        int
	maxWorkers        int
	expectedInstances int
	rampInterval      time.Duration
	scaleTimeout      time.Duration
}

type serviceResponse struct {
	Timestamp string `json:"timestamp"`
	IP        string `json:"ip"`
}

type requestResult struct {
	instanceID string
	duration   time.Duration
	transient  bool
	err        error
}

// TestStress ramps continuous traffic until Cloud Run uses the target replica count.
func TestStress(t *testing.T) {
	if os.Getenv("RUN_STRESS_TEST") != "true" {
		t.Skip("run with make stress-test")
	}
	if testing.Short() {
		t.Skip("stress test is disabled in short mode")
	}

	cfg := loadConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.scaleTimeout)
	defer cancel()

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        cfg.maxWorkers,
			MaxIdleConnsPerHost: cfg.maxWorkers,
			IdleConnTimeout:     90 * time.Second,
		},
	}
	defer client.CloseIdleConnections()
	results := make(chan requestResult, cfg.maxWorkers)
	var workers sync.WaitGroup
	activeWorkers := 0
	startWorkers := func(count int) {
		count = min(count, cfg.maxWorkers-activeWorkers)
		workers.Add(count)
		for range count {
			go runWorker(ctx, &workers, client, cfg, results)
		}
		activeWorkers += count
	}
	startWorkers(cfg.initialWorkers)

	ticker := time.NewTicker(cfg.rampInterval)
	defer ticker.Stop()

	started := time.Now()
	totalRequests := 0
	errorsSeen := 0
	durations := make([]time.Duration, 0, cfg.maxWorkers)
	instanceRequests := make(map[string]int, cfg.expectedInstances)

	for {
		select {
		case <-ticker.C:
			// Ramp gradually while every existing worker keeps sending requests.
			if activeWorkers < cfg.maxWorkers {
				startWorkers(cfg.workerStep)
			}

		case result := <-results:
			totalRequests++
			if result.err != nil {
				errorsSeen++
				if !result.transient {
					cancel()
					workers.Wait()
					logMetrics(t, started, activeWorkers, totalRequests, errorsSeen, durations, instanceRequests)
					t.Fatalf("response validation failed: %v", result.err)
				}
				// Avoid flooding test output while retaining evidence of transient failures.
				if errorsSeen <= 5 || errorsSeen%100 == 0 {
					t.Logf("transient request error #%d: %v", errorsSeen, result.err)
				}
				continue
			}

			durations = append(durations, result.duration)
			instanceRequests[result.instanceID]++
			if len(instanceRequests) >= cfg.expectedInstances {
				// Stop immediately once responses prove the target replica count.
				cancel()
				workers.Wait()
				logMetrics(t, started, activeWorkers, totalRequests, errorsSeen, durations, instanceRequests)
				return
			}

		case <-ctx.Done():
			workers.Wait()
			logMetrics(t, started, activeWorkers, totalRequests, errorsSeen, durations, instanceRequests)
			t.Fatalf(
				"Cloud Run did not scale to %d instances within %s; observed %d",
				cfg.expectedInstances,
				cfg.scaleTimeout,
				len(instanceRequests),
			)
		}
	}
}

func runWorker(
	ctx context.Context,
	workers *sync.WaitGroup,
	client *http.Client,
	cfg config,
	results chan<- requestResult,
) {
	defer workers.Done()

	for ctx.Err() == nil {
		result := makeRequest(ctx, client, cfg)
		select {
		case results <- result:
		case <-ctx.Done():
			return
		}
	}
}

func loadConfig(t *testing.T) config {
	t.Helper()

	serviceURL := requiredEnv(t, "SERVICE_URL")
	expectedLBIP := parseIP(t, "EXPECTED_LB_IP", requiredEnv(t, "EXPECTED_LB_IP"))
	expectedClientIP := parseIP(t, "EXPECTED_CLIENT_IP", requiredEnv(t, "EXPECTED_CLIENT_IP"))
	parsedURL, err := url.ParseRequestURI(serviceURL)
	if err != nil {
		t.Fatalf("parse SERVICE_URL: %v", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		t.Fatalf("SERVICE_URL must use http or https, got %q", parsedURL.Scheme)
	}
	if targetIP := net.ParseIP(parsedURL.Hostname()); targetIP == nil || !targetIP.Equal(expectedLBIP) {
		t.Fatalf("SERVICE_URL host %q does not match EXPECTED_LB_IP %q", parsedURL.Hostname(), expectedLBIP)
	}
	maxWorkers := envInt(t, "STRESS_MAX_WORKERS", 1000)
	initialWorkers := envInt(t, "STRESS_INITIAL_WORKERS", 80)
	if initialWorkers > maxWorkers {
		t.Fatalf("STRESS_INITIAL_WORKERS cannot exceed STRESS_MAX_WORKERS")
	}

	return config{
		serviceURL:        parsedURL.String(),
		expectedClientIP:  expectedClientIP.String(),
		initialWorkers:    initialWorkers,
		workerStep:        envInt(t, "STRESS_WORKER_STEP", 80),
		maxWorkers:        maxWorkers,
		expectedInstances: envInt(t, "EXPECTED_INSTANCES", 3),
		rampInterval:      envDuration(t, "STRESS_RAMP_INTERVAL", 5*time.Second),
		scaleTimeout:      envDuration(t, "STRESS_SCALE_TIMEOUT", 90*time.Second),
	}
}

func makeRequest(ctx context.Context, client *http.Client, cfg config) requestResult {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.serviceURL, nil)
	if err != nil {
		return requestResult{err: fmt.Errorf("create request: %w", err)}
	}
	started := time.Now()
	response, err := client.Do(request)
	if err != nil {
		return requestResult{duration: time.Since(started), transient: true, err: fmt.Errorf("request failed: %w", err)}
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	duration := time.Since(started)
	if err != nil {
		return requestResult{duration: duration, transient: true, err: fmt.Errorf("read response: %w", err)}
	}
	if response.StatusCode != http.StatusOK {
		return requestResult{
			duration:  duration,
			transient: isTransientStatus(response.StatusCode),
			err:       fmt.Errorf("unexpected status %d: %s", response.StatusCode, body),
		}
	}

	instanceID := response.Header.Get(instanceIDHeader)
	if instanceID == "" {
		return requestResult{duration: duration, err: fmt.Errorf("response is missing %s", instanceIDHeader)}
	}

	var decoded serviceResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return requestResult{duration: duration, err: fmt.Errorf("decode response: %w", err)}
	}
	if decoded.IP != cfg.expectedClientIP {
		return requestResult{duration: duration, err: fmt.Errorf("expected client IP %q, got %q", cfg.expectedClientIP, decoded.IP)}
	}
	if _, err := time.Parse(time.RFC3339Nano, decoded.Timestamp); err != nil {
		return requestResult{duration: duration, err: fmt.Errorf("invalid timestamp %q: %w", decoded.Timestamp, err)}
	}

	return requestResult{instanceID: instanceID, duration: duration}
}

// isTransientStatus identifies responses worth retrying while Cloud Run scales.
func isTransientStatus(status int) bool {
	return status == http.StatusRequestTimeout ||
		status == http.StatusTooManyRequests ||
		status >= http.StatusInternalServerError
}

// TestMakeRequestTreatsServerErrorsAsTransient protects the scale-up retry behavior.
func TestMakeRequestTreatsServerErrorsAsTransient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "temporarily unavailable", http.StatusInternalServerError)
	}))
	defer server.Close()

	result := makeRequest(context.Background(), server.Client(), config{serviceURL: server.URL})
	if result.err == nil {
		t.Fatal("expected the server error to be reported")
	}
	if !result.transient {
		t.Fatal("expected HTTP 500 to be classified as transient")
	}
}

func logMetrics(
	t *testing.T,
	started time.Time,
	activeWorkers int,
	totalRequests int,
	errorsSeen int,
	durations []time.Duration,
	instanceRequests map[string]int,
) {
	t.Helper()
	elapsed := time.Since(started)
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	t.Logf(
		"elapsed=%s active_workers=%d requests=%d errors=%d throughput=%.1f req/s instances=%d",
		elapsed.Round(time.Millisecond), activeWorkers, totalRequests, errorsSeen,
		float64(totalRequests)/elapsed.Seconds(), len(instanceRequests),
	)
	if len(durations) > 0 {
		t.Logf(
			"latency min=%s p50=%s p95=%s p99=%s max=%s",
			durations[0], percentile(durations, 0.50), percentile(durations, 0.95),
			percentile(durations, 0.99), durations[len(durations)-1],
		)
	}

	instanceIDs := make([]string, 0, len(instanceRequests))
	for id := range instanceRequests {
		instanceIDs = append(instanceIDs, id)
	}
	sort.Strings(instanceIDs)
	for _, id := range instanceIDs {
		t.Logf("instance %s handled %d completed requests", id, instanceRequests[id])
	}
}

func percentile(sorted []time.Duration, percentile float64) time.Duration {
	index := int(float64(len(sorted)-1) * percentile)
	return sorted[index]
}

func requiredEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("%s is required", name)
	}
	return value
}

func parseIP(t *testing.T, name, value string) net.IP {
	t.Helper()
	ip := net.ParseIP(value)
	if ip == nil {
		t.Fatalf("%s must be a valid IP address, got %q", name, value)
	}
	return ip
}

func envInt(t *testing.T, name string, fallback int) int {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		t.Fatalf("%s must be a positive integer, got %q", name, value)
	}
	return parsed
}

func envDuration(t *testing.T, name string, fallback time.Duration) time.Duration {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		t.Fatalf("%s must be a positive duration, got %q", name, value)
	}
	return parsed
}
