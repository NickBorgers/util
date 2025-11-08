package health

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// HealthServer provides a health check endpoint
type HealthServer struct {
	config         *Config
	server         *http.Server
	mu             sync.RWMutex
	lastTestTime   time.Time
	testCount      int64
	successCount   int64
	failureCount   int64
	isHealthy      bool
}

// Config contains health check server configuration
type Config struct {
	Enabled       bool
	Port          int
	Path          string
	ListenAddress string
}

// HealthResponse is the JSON response structure
type HealthResponse struct {
	Status       string    `json:"status"`
	Timestamp    time.Time `json:"timestamp"`
	LastTestTime time.Time `json:"last_test_time,omitempty"`
	TestCount    int64     `json:"test_count"`
	SuccessCount int64     `json:"success_count"`
	FailureCount int64     `json:"failure_count"`
	Uptime       string    `json:"uptime"`
}

var startTime = time.Now()

// NewHealthServer creates a new health check server
func NewHealthServer(cfg *Config) (*HealthServer, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	h := &HealthServer{
		config:    cfg,
		isHealthy: true,
	}

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc(cfg.Path, h.handleHealth)

	addr := fmt.Sprintf("%s:%d", cfg.ListenAddress, cfg.Port)
	h.server = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Health check endpoint started on %s%s", addr, cfg.Path)
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Health check server error: %v", err)
		}
	}()

	return h, nil
}

// handleHealth handles health check requests
func (h *HealthServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Determine health status
	status := "healthy"
	statusCode := http.StatusOK

	// Check if we've received any tests recently (within 5 minutes)
	if h.testCount > 0 && time.Since(h.lastTestTime) > 5*time.Minute {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	if !h.isHealthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	// Build response
	response := HealthResponse{
		Status:       status,
		Timestamp:    time.Now(),
		LastTestTime: h.lastTestTime,
		TestCount:    h.testCount,
		SuccessCount: h.successCount,
		FailureCount: h.failureCount,
		Uptime:       time.Since(startTime).String(),
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Write JSON response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding health response: %v", err)
	}
}

// RecordTest records a test execution
func (h *HealthServer) RecordTest(success bool) {
	if h == nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.lastTestTime = time.Now()
	h.testCount++

	if success {
		h.successCount++
	} else {
		h.failureCount++
	}
}

// SetHealthy sets the health status
func (h *HealthServer) SetHealthy(healthy bool) {
	if h == nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.isHealthy = healthy
}

// GetStats returns current health statistics
func (h *HealthServer) GetStats() (testCount, successCount, failureCount int64, lastTestTime time.Time) {
	if h == nil {
		return 0, 0, 0, time.Time{}
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.testCount, h.successCount, h.failureCount, h.lastTestTime
}

// Close shuts down the health check server
func (h *HealthServer) Close() error {
	if h == nil || h.server == nil {
		return nil
	}

	log.Println("Shutting down health check server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return h.server.Shutdown(ctx)
}
