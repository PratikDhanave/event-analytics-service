package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

////////////////////////////////////////////////////////////////////////////////
// INTEGRATION TEST SUITE
//
// These tests validate the service end-to-end:
//
//   Client → HTTP API → Auth → Postgres → Query → Response
//
// The service must already be running (for example via docker compose).
//
// Optional environment overrides:
//
//   BASE_URL    default http://localhost:8080
//   TENANT1_KEY default tenant-key-123
//   TENANT2_KEY default tenant-key-456
//
////////////////////////////////////////////////////////////////////////////////

func baseURL() string {
	if v := os.Getenv("BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

// tenant1Key returns the default API key for tenant1.
func tenant1Key() string {
	if v := os.Getenv("TENANT1_KEY"); v != "" {
		return v
	}
	return "tenant-key-123"
}

// tenant2Key returns the default API key for tenant2.
func tenant2Key() string {
	if v := os.Getenv("TENANT2_KEY"); v != "" {
		return v
	}
	return "tenant-key-456"
}

// unique generates a unique string so tests never collide with previous runs.
func unique(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

////////////////////////////////////////////////////////////////////////////////
// SERVICE READINESS HELPER
//
// waitReady polls /ready until DB + server are ready.
// Prevents flaky failures when containers are still booting.
////////////////////////////////////////////////////////////////////////////////

func waitReady(t *testing.T) {
	t.Helper()

	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		resp, err := client.Get(baseURL() + "/ready")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(300 * time.Millisecond)
	}

	t.Fatalf("service not ready after 30s")
}

////////////////////////////////////////////////////////////////////////////////
// GENERIC HTTP HELPERS
////////////////////////////////////////////////////////////////////////////////

// httpGet performs a GET request with optional API key.
func httpGet(t *testing.T, apiKey string, path string) (int, []byte) {
	t.Helper()

	req, _ := http.NewRequest("GET", baseURL()+path, nil)
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("GET %s failed: %v", path, err)
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

// postJSON performs a POST with JSON body and optional idempotency key.
func postJSON(t *testing.T, apiKey, idemKey, path string, payload any) (int, []byte) {
	t.Helper()

	b, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", baseURL()+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}

	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("POST %s failed: %v", path, err)
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, out
}

// postEvent is a convenience wrapper for POST /events.
func postEvent(t *testing.T, apiKey, idemKey, name string, ts time.Time) (int, []byte) {
	payload := map[string]any{
		"event_name": name,
		"timestamp":  ts.UTC().Format(time.RFC3339),
	}
	return postJSON(t, apiKey, idemKey, "/events", payload)
}

// getMetrics queries the metrics endpoint.
func getMetrics(t *testing.T, apiKey, name string, from, to time.Time) (int, []byte) {

	u, _ := url.Parse(baseURL() + "/metrics")
	q := u.Query()
	q.Set("event_name", name)
	q.Set("from", from.UTC().Format(time.RFC3339))
	q.Set("to", to.UTC().Format(time.RFC3339))
	u.RawQuery = q.Encode()

	req, _ := http.NewRequest("GET", u.String(), nil)
	req.Header.Set("X-API-Key", apiKey)

	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("GET metrics failed: %v", err)
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, out
}

// parseCount extracts the count from metrics JSON.
func parseCount(t *testing.T, b []byte) int64 {
	var r struct {
		Count int64 `json:"count"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatalf("invalid metrics JSON: %v", err)
	}
	return r.Count
}

////////////////////////////////////////////////////////////////////////////////
// HEALTH & READINESS TESTS
////////////////////////////////////////////////////////////////////////////////

// Health endpoint = liveness check (server process running).
func TestHealth_ReturnsOK(t *testing.T) {
	s, _ := httpGet(t, "", "/health")
	if s != http.StatusOK {
		t.Fatalf("health expected 200 got %d", s)
	}
}

// Ready endpoint = dependency readiness (DB reachable).
func TestReady_ReturnsOK(t *testing.T) {
	waitReady(t)
	s, _ := httpGet(t, "", "/ready")
	if s != http.StatusOK {
		t.Fatalf("ready expected 200 got %d", s)
	}
}

////////////////////////////////////////////////////////////////////////////////
// EVENTS CONTRACT TESTS
////////////////////////////////////////////////////////////////////////////////

// Request without API key must be rejected.
func TestEvents_UnauthorizedWithoutAPIKey(t *testing.T) {
	waitReady(t)

	payload := map[string]any{
		"event_name": "login",
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	s, _ := postJSON(t, "", unique("x"), "/events", payload)
	if s != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", s)
	}
}

// Missing timestamp should return 400.
func TestEvents_BadRequestOnInvalidPayload(t *testing.T) {
	waitReady(t)

	payload := map[string]any{"event_name": "login"}
	s, _ := postJSON(t, tenant1Key(), unique("x"), "/events", payload)

	if s != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", s)
	}
}

////////////////////////////////////////////////////////////////////////////////
// CORE SYSTEM BEHAVIOR TESTS
////////////////////////////////////////////////////////////////////////////////

// Duplicate events must not increase metrics.
func TestIdempotency_DuplicateDoesNotIncreaseCount(t *testing.T) {

	waitReady(t)

	name := unique("idem")
	key := unique("k")
	ts := time.Now().UTC()

	postEvent(t, tenant1Key(), key, name, ts)
	postEvent(t, tenant1Key(), key, name, ts)

	_, b := getMetrics(t, tenant1Key(), name, ts.Add(-time.Hour), ts.Add(time.Hour))

	if parseCount(t, b) != 1 {
		t.Fatal("duplicate increased count")
	}
}

// Each tenant must see only its own data.
func TestTenantIsolation_TenantsDoNotSeeEachOthersEvents(t *testing.T) {

	waitReady(t)

	name := unique("iso")
	ts := time.Now().UTC()

	postEvent(t, tenant1Key(), unique("a"), name, ts)
	postEvent(t, tenant2Key(), unique("b"), name, ts)

	_, b1 := getMetrics(t, tenant1Key(), name, ts.Add(-time.Hour), ts.Add(time.Hour))
	_, b2 := getMetrics(t, tenant2Key(), name, ts.Add(-time.Hour), ts.Add(time.Hour))

	if parseCount(t, b1) != 1 || parseCount(t, b2) != 1 {
		t.Fatal("tenant isolation failed")
	}
}
