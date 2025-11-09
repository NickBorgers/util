package health

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestNewHealthServer_Disabled tests that nil is returned when disabled
func TestNewHealthServer_Disabled(t *testing.T) {
	cfg := &Config{
		Enabled: false,
	}

	server, err := NewHealthServer(cfg)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if server != nil {
		t.Error("Expected nil server when disabled")
	}
}

// TestNewHealthServer_DefaultConfig tests creating health server with default config
func TestNewHealthServer_DefaultConfig(t *testing.T) {
	cfg := &Config{
		Enabled:       true,
		Port:          18080, // Use non-standard port to avoid conflicts
		Path:          "/health",
		ListenAddress: "127.0.0.1",
	}

	server, err := NewHealthServer(cfg)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if server == nil {
		t.Fatal("Expected server to be created")
	}

	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Verify server is running
	resp, err := http.Get("http://127.0.0.1:18080/health")
	if err != nil {
		t.Fatalf("Failed to connect to health server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Parse response
	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if healthResp.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", healthResp.Status)
	}

	// Cleanup
	server.Shutdown()
}

// TestNewHealthServer_CustomPath tests custom health check path
func TestNewHealthServer_CustomPath(t *testing.T) {
	cfg := &Config{
		Enabled:       true,
		Port:          18081,
		Path:          "/custom/healthz",
		ListenAddress: "127.0.0.1",
	}

	server, err := NewHealthServer(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer server.Shutdown()

	time.Sleep(100 * time.Millisecond)

	// Test custom path
	resp, err := http.Get("http://127.0.0.1:18081/custom/healthz")
	if err != nil {
		t.Fatalf("Failed to connect to health server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// TestNewHealthServer_ListenAddress tests different listen addresses
func TestNewHealthServer_ListenAddress(t *testing.T) {
	tests := []struct {
		name          string
		listenAddress string
		testAddress   string
	}{
		{
			name:          "localhost binding",
			listenAddress: "127.0.0.1",
			testAddress:   "127.0.0.1",
		},
		// Note: We can't easily test 0.0.0.0 binding in unit tests as it would
		// bind to all interfaces including external ones. In practice, 0.0.0.0
		// is tested via integration tests.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Enabled:       true,
				Port:          18082,
				Path:          "/health",
				ListenAddress: tt.listenAddress,
			}

			server, err := NewHealthServer(cfg)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			defer server.Shutdown()

			time.Sleep(100 * time.Millisecond)

			// Verify server is accessible at the expected address
			resp, err := http.Get("http://" + tt.testAddress + ":18082/health")
			if err != nil {
				t.Fatalf("Failed to connect to health server at %s: %v", tt.testAddress, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		})
	}
}

// TestHealthServer_RecordTest tests recording test results
func TestHealthServer_RecordTest(t *testing.T) {
	cfg := &Config{
		Enabled:       true,
		Port:          18083,
		Path:          "/health",
		ListenAddress: "127.0.0.1",
	}

	server, err := NewHealthServer(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer server.Shutdown()

	time.Sleep(100 * time.Millisecond)

	// Record some test results
	server.RecordTest(true)
	server.RecordTest(true)
	server.RecordTest(false)

	// Check health response
	resp, err := http.Get("http://127.0.0.1:18083/health")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer resp.Body.Close()

	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if healthResp.TestCount != 3 {
		t.Errorf("Expected TestCount 3, got %d", healthResp.TestCount)
	}

	if healthResp.SuccessCount != 2 {
		t.Errorf("Expected SuccessCount 2, got %d", healthResp.SuccessCount)
	}

	if healthResp.FailureCount != 1 {
		t.Errorf("Expected FailureCount 1, got %d", healthResp.FailureCount)
	}
}

// TestHealthServer_Shutdown tests graceful shutdown
func TestHealthServer_Shutdown(t *testing.T) {
	cfg := &Config{
		Enabled:       true,
		Port:          18084,
		Path:          "/health",
		ListenAddress: "127.0.0.1",
	}

	server, err := NewHealthServer(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify server is running
	_, err = http.Get("http://127.0.0.1:18084/health")
	if err != nil {
		t.Fatalf("Server should be running: %v", err)
	}

	// Shutdown server
	server.Shutdown()

	// Give it time to shutdown
	time.Sleep(200 * time.Millisecond)

	// Verify server is no longer accessible
	_, err = http.Get("http://127.0.0.1:18084/health")
	if err == nil {
		t.Error("Expected error connecting to shutdown server")
	}
}

// TestHealthServer_StaleTests tests unhealthy status when tests are stale
func TestHealthServer_StaleTests(t *testing.T) {
	cfg := &Config{
		Enabled:       true,
		Port:          18085,
		Path:          "/health",
		ListenAddress: "127.0.0.1",
	}

	server, err := NewHealthServer(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer server.Shutdown()

	time.Sleep(100 * time.Millisecond)

	// Record a test
	server.RecordTest(true)

	// Manually set last test time to be old (more than 5 minutes ago)
	server.mu.Lock()
	server.lastTestTime = time.Now().Add(-6 * time.Minute)
	server.mu.Unlock()

	// Check health - should be unhealthy due to stale tests
	resp, err := http.Get("http://127.0.0.1:18085/health")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", resp.StatusCode)
	}

	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if healthResp.Status != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got '%s'", healthResp.Status)
	}
}

// TestHealthServer_ConcurrentRequests tests handling concurrent health check requests
func TestHealthServer_ConcurrentRequests(t *testing.T) {
	cfg := &Config{
		Enabled:       true,
		Port:          18086,
		Path:          "/health",
		ListenAddress: "127.0.0.1",
	}

	server, err := NewHealthServer(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer server.Shutdown()

	time.Sleep(100 * time.Millisecond)

	// Record some tests
	for i := 0; i < 10; i++ {
		server.RecordTest(true)
	}

	// Make concurrent requests
	done := make(chan bool)
	errors := make(chan error, 5)

	for i := 0; i < 5; i++ {
		go func() {
			resp, err := http.Get("http://127.0.0.1:18086/health")
			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				errors <- nil
			}
			done <- true
		}()
	}

	// Wait for all requests
	successCount := 0
	for i := 0; i < 5; i++ {
		select {
		case err := <-errors:
			if err != nil {
				t.Errorf("Request failed: %v", err)
			}
		case <-done:
			successCount++
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}

	if successCount != 5 {
		t.Errorf("Expected 5 successful requests, got %d", successCount)
	}
}
