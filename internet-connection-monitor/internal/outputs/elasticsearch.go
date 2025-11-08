package outputs

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/config"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// ElasticsearchOutput pushes test results to Elasticsearch
type ElasticsearchOutput struct {
	config        *config.ElasticsearchConfig
	client        *elasticsearch.Client
	bulkIndexer   esutil.BulkIndexer
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	resultChannel chan *models.TestResult
}

// NewElasticsearchOutput creates a new Elasticsearch output
func NewElasticsearchOutput(cfg *config.ElasticsearchConfig) (*ElasticsearchOutput, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	// Build Elasticsearch configuration
	esCfg := elasticsearch.Config{
		Addresses: []string{cfg.Endpoint},
		RetryOnStatus: []int{502, 503, 504, 429},
		MaxRetries:    cfg.MaxRetries,
	}

	// Configure authentication
	if cfg.APIKey != "" {
		esCfg.APIKey = cfg.APIKey
	} else if cfg.Username != "" && cfg.Password != "" {
		esCfg.Username = cfg.Username
		esCfg.Password = cfg.Password
	}

	// Configure TLS
	if cfg.TLSSkipVerify {
		esCfg.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	// Create Elasticsearch client
	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	// Test connection
	res, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("Elasticsearch returned error: %s", res.Status())
	}

	log.Printf("Connected to Elasticsearch at %s", cfg.Endpoint)

	// Create bulk indexer
	bulkIndexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client:        client,
		NumWorkers:    2,
		FlushBytes:    int(cfg.BulkSize) * 1024,
		FlushInterval: cfg.FlushInterval,
		OnError: func(ctx context.Context, err error) {
			log.Printf("Elasticsearch bulk indexer error: %v", err)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bulk indexer: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	e := &ElasticsearchOutput{
		config:        cfg,
		client:        client,
		bulkIndexer:   bulkIndexer,
		ctx:           ctx,
		cancel:        cancel,
		resultChannel: make(chan *models.TestResult, 100),
	}

	// Start background worker to process results
	e.wg.Add(1)
	go e.processResults()

	return e, nil
}

// processResults is a background worker that processes test results
func (e *ElasticsearchOutput) processResults() {
	defer e.wg.Done()

	for {
		select {
		case <-e.ctx.Done():
			return
		case result := <-e.resultChannel:
			if err := e.indexResult(result); err != nil {
				log.Printf("Failed to index result to Elasticsearch: %v", err)
			}
		}
	}
}

// indexResult indexes a single test result to Elasticsearch
func (e *ElasticsearchOutput) indexResult(result *models.TestResult) error {
	// Format the index name using the pattern
	// Replace %{+yyyy.MM.dd} with actual date
	indexName := e.formatIndexName(result.Timestamp)

	// Convert result to JSON
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	// Add to bulk indexer
	err = e.bulkIndexer.Add(
		e.ctx,
		esutil.BulkIndexerItem{
			Action:     "index",
			Index:      indexName,
			DocumentID: result.TestID,
			Body:       bytes.NewReader(data),
			OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
				if err != nil {
					log.Printf("Elasticsearch indexing error: %v", err)
				} else {
					log.Printf("Elasticsearch indexing failed: %s: %s", res.Error.Type, res.Error.Reason)
				}
			},
		},
	)

	return err
}

// formatIndexName formats the index name using the configured pattern
func (e *ElasticsearchOutput) formatIndexName(t time.Time) string {
	indexName := e.config.IndexPattern

	// Replace date format patterns
	// %{+yyyy.MM.dd} -> 2024.01.15
	if strings.Contains(indexName, "%{+yyyy.MM.dd}") {
		dateStr := t.Format("2006.01.02")
		indexName = strings.Replace(indexName, "%{+yyyy.MM.dd}", dateStr, -1)
	}

	// %{+yyyy.MM} -> 2024.01
	if strings.Contains(indexName, "%{+yyyy.MM}") {
		dateStr := t.Format("2006.01")
		indexName = strings.Replace(indexName, "%{+yyyy.MM}", dateStr, -1)
	}

	// %{+yyyy} -> 2024
	if strings.Contains(indexName, "%{+yyyy}") {
		dateStr := t.Format("2006")
		indexName = strings.Replace(indexName, "%{+yyyy}", dateStr, -1)
	}

	return indexName
}

// Write sends a test result to Elasticsearch
func (e *ElasticsearchOutput) Write(result *models.TestResult) error {
	if e == nil {
		return nil
	}

	// Send to channel for async processing
	select {
	case e.resultChannel <- result:
		return nil
	case <-e.ctx.Done():
		return fmt.Errorf("Elasticsearch output is shutting down")
	default:
		// Channel is full, log and drop
		log.Printf("Warning: Elasticsearch result channel is full, dropping result")
		return nil
	}
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

	log.Println("Shutting down Elasticsearch output...")

	// Stop accepting new results
	e.cancel()

	// Wait for worker to finish
	e.wg.Wait()

	// Close the bulk indexer (flushes pending documents)
	if err := e.bulkIndexer.Close(context.Background()); err != nil {
		log.Printf("Error closing Elasticsearch bulk indexer: %v", err)
		return err
	}

	// Get statistics
	stats := e.bulkIndexer.Stats()
	log.Printf("Elasticsearch indexer stats: %d indexed, %d failed", stats.NumIndexed, stats.NumFailed)

	return nil
}
