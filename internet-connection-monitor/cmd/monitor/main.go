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
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/health"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/metrics"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/outputs"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/testloop"
)

const version = "1.1.0"

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

	// Initialize optional outputs
	log.Printf("DEBUG: ES_ENABLED config value: %v", cfg.Elasticsearch.Enabled)
	esOutput, err := outputs.NewElasticsearchOutput(&cfg.Elasticsearch)
	if err != nil {
		log.Fatalf("Failed to create Elasticsearch output: %v", err)
	}
	if esOutput != nil {
		dispatcher.RegisterOutput(esOutput)
		log.Println("✓ Elasticsearch output enabled")
	} else {
		log.Println("Elasticsearch output not enabled (config.Enabled=false)")
	}

	promOutput, err := outputs.NewPrometheusOutput(&cfg.Prometheus)
	if err != nil {
		log.Fatalf("Failed to create Prometheus output: %v", err)
	}
	if promOutput != nil {
		dispatcher.RegisterOutput(promOutput)
		log.Println("✓ Prometheus exporter enabled")
	}

	snmpOutput, err := outputs.NewSNMPOutput(&cfg.SNMP)
	if err != nil {
		log.Fatalf("Failed to create SNMP output: %v", err)
	}
	if snmpOutput != nil {
		dispatcher.RegisterOutput(snmpOutput)
		log.Println("✓ SNMP agent enabled")
	}

	// Initialize health check endpoint
	healthCfg := &health.Config{
		Enabled:       cfg.Advanced.HealthCheckEnabled,
		Port:          cfg.Advanced.HealthCheckPort,
		Path:          cfg.Advanced.HealthCheckPath,
		ListenAddress: "0.0.0.0",
	}
	healthServer, err := health.NewHealthServer(healthCfg)
	if err != nil {
		log.Fatalf("Failed to create health check server: %v", err)
	}
	if healthServer != nil {
		log.Println("✓ Health check endpoint enabled")
	}

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

	// Close all output modules
	if esOutput != nil {
		if err := esOutput.Close(); err != nil {
			log.Printf("Error closing Elasticsearch output: %v", err)
		} else {
			log.Println("✓ Elasticsearch output closed")
		}
	}

	if promOutput != nil {
		if err := promOutput.Close(); err != nil {
			log.Printf("Error closing Prometheus output: %v", err)
		} else {
			log.Println("✓ Prometheus exporter closed")
		}
	}

	if snmpOutput != nil {
		if err := snmpOutput.Close(); err != nil {
			log.Printf("Error closing SNMP output: %v", err)
		} else {
			log.Println("✓ SNMP agent closed")
		}
	}

	// Close health check server
	if healthServer != nil {
		if err := healthServer.Close(); err != nil {
			log.Printf("Error closing health check server: %v", err)
		} else {
			log.Println("✓ Health check server closed")
		}
	}

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
	if err := config.LoadFromEnv(cfg); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

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
