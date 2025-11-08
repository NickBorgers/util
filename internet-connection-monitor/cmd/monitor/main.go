package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/browser"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/config"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/metrics"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/outputs"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/testloop"
)

const version = "0.1.0-dev"

func main() {
	// Print banner
	printBanner()

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	log.Printf("Loaded configuration: %d sites to monitor", len(cfg.Sites.List))
	log.Printf("  Inter-test delay: %v", cfg.General.InterTestDelay)
	log.Printf("  Global timeout: %v", cfg.General.GlobalTimeout)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize browser controller
	browserCtrl, err := browser.NewController(&cfg.Browser)
	if err != nil {
		log.Fatalf("Failed to create browser controller: %v", err)
	}
	defer browserCtrl.Close()
	log.Println("✓ Browser controller initialized")

	// Initialize output modules
	dispatcher := metrics.NewDispatcher()

	// Always enable JSON logger
	logger, err := outputs.NewLogger(&cfg.Logging)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	dispatcher.RegisterOutput(logger)
	log.Println("✓ JSON logger enabled")

	// TODO: Initialize optional outputs
	// - Elasticsearch (if enabled)
	// - Prometheus (if enabled)
	// - SNMP (if enabled)

	// Create test loop
	testLoop, err := testloop.NewTestLoop(cfg, browserCtrl, dispatcher)
	if err != nil {
		log.Fatalf("Failed to create test loop: %v", err)
	}
	log.Println("✓ Test loop initialized")

	// Start the test loop in a goroutine
	loopDone := make(chan error, 1)
	go func() {
		loopDone <- testLoop.Run(ctx)
	}()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal or loop error
	log.Println("Internet Connection Monitor started. Press Ctrl+C to stop.")
	log.Println()

	select {
	case <-sigChan:
		log.Println("\nReceived shutdown signal...")
	case err := <-loopDone:
		if err != nil {
			log.Printf("Test loop exited with error: %v", err)
		}
	}

	// Graceful shutdown
	log.Println("Shutting down gracefully...")

	// Cancel context to stop test loop
	cancel()

	// Wait for test loop to finish (with timeout)
	shutdownTimeout := time.After(cfg.Advanced.ShutdownTimeout)
	select {
	case <-loopDone:
		log.Println("✓ Test loop stopped")
	case <-shutdownTimeout:
		log.Println("⚠ Shutdown timeout exceeded")
	}

	// Close browser
	if err := browserCtrl.Close(); err != nil {
		log.Printf("Error closing browser: %v", err)
	}
	log.Println("✓ Browser closed")

	// TODO: Close other services
	// - Flush Elasticsearch bulk processor
	// - Stop Prometheus HTTP server
	// - Stop SNMP agent

	log.Println("Shutdown complete")
}

func loadConfig() (*config.Config, error) {
	// Check for config file path in env var
	configFile := os.Getenv("CONFIG_FILE")

	var cfg *config.Config
	var err error

	if configFile != "" {
		// Load from file
		cfg, err = config.Load(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	} else {
		// Use defaults
		cfg = config.DefaultConfig()
	}

	// Override with environment variables
	// TODO: Implement env var parsing
	// This allows Docker deployment with env vars only

	// Load default sites if none configured
	if len(cfg.Sites.List) == 0 {
		cfg.Sites.List = config.DefaultSites()
	}

	return cfg, nil
}

func printBanner() {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  Internet Connection Monitor                                   ║")
	fmt.Printf("║  Version: %-52s ║\n", version)
	fmt.Println("║  Real-world Internet connectivity from a user's perspective    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
}
