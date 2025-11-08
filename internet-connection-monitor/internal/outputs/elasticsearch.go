package outputs

import (
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/config"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// ElasticsearchOutput pushes test results to Elasticsearch
type ElasticsearchOutput struct {
	config *config.ElasticsearchConfig
	// TODO: Add Elasticsearch client
	// TODO: Add bulk processor for batching
}

// NewElasticsearchOutput creates a new Elasticsearch output
func NewElasticsearchOutput(cfg *config.ElasticsearchConfig) (*ElasticsearchOutput, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	// TODO: Initialize Elasticsearch client
	// TODO: Set up bulk processor with batching

	return &ElasticsearchOutput{
		config: cfg,
	}, nil
}

// Write sends a test result to Elasticsearch
func (e *ElasticsearchOutput) Write(result *models.TestResult) error {
	if e == nil {
		return nil
	}

	// TODO: Add to bulk processor
	// The bulk processor will batch and flush according to config
	return nil
}

// Name returns the output module name
func (e *ElasticsearchOutput) Name() string {
	return "elasticsearch"
}

// Close flushes pending documents and closes the connection
func (e *ElasticsearchOutput) Close() error {
	if e == nil {
		return nil
	}

	// TODO: Flush bulk processor
	// TODO: Close ES client
	return nil
}
