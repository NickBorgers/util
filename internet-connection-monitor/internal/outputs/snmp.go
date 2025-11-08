package outputs

import (
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/config"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// SNMPOutput provides an SNMP agent for polling recent results
type SNMPOutput struct {
	config *config.SNMPConfig
	cache  []*models.TestResult // Recent results for SNMP queries
	// TODO: Add SNMP agent
}

// NewSNMPOutput creates a new SNMP agent
func NewSNMPOutput(cfg *config.SNMPConfig) (*SNMPOutput, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	// TODO: Initialize SNMP agent
	// TODO: Define custom MIB for internet monitor
	// TODO: Start SNMP listener on cfg.Port

	return &SNMPOutput{
		config: cfg,
		cache:  make([]*models.TestResult, 0, 100),
	}, nil
}

// Write caches the test result for SNMP queries
func (s *SNMPOutput) Write(result *models.TestResult) error {
	if s == nil {
		return nil
	}

	// TODO: Add result to cache (circular buffer)
	// SNMP polls will read from this cache

	return nil
}

// Name returns the output module name
func (s *SNMPOutput) Name() string {
	return "snmp"
}

// Close shuts down the SNMP agent
func (s *SNMPOutput) Close() error {
	if s == nil {
		return nil
	}

	// TODO: Stop SNMP listener
	return nil
}
