package main

import (
	_ "embed"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed device_rules.yaml
var embeddedRulesYAML []byte

// DeviceRule represents a single device detection rule
type DeviceRule struct {
	Name        string     `yaml:"name"`
	Priority    int        `yaml:"priority"`
	DeviceType  string     `yaml:"device_type"`
	Icon        string     `yaml:"icon"`
	Description string     `yaml:"description"`
	Conditions  Conditions `yaml:"conditions"`
}

// Conditions define when a rule matches
type Conditions struct {
	AllOf        []Condition `yaml:"all_of,omitempty"`
	AnyOf        []Condition `yaml:"any_of,omitempty"`
	NotConditions []Condition `yaml:"not_conditions,omitempty"`
}

// Condition represents a single matching condition
type Condition struct {
	HostnameContains     []string `yaml:"hostname_contains,omitempty"`
	ServiceNameContains  []string `yaml:"service_name_contains,omitempty"`
	ServiceTypeContains  []string `yaml:"service_type_contains,omitempty"`
	MACVendorContains    []string `yaml:"mac_vendor_contains,omitempty"`
	OpenPorts            []int    `yaml:"open_ports,omitempty"`
	HostnameMatches      []string `yaml:"hostname_matches,omitempty"`
	ServiceNameMatches   []string `yaml:"service_name_matches,omitempty"`
}

// DeviceRulesConfig represents the entire YAML configuration
type DeviceRulesConfig struct {
	Version     string       `yaml:"version"`
	Updated     string       `yaml:"updated"`
	Rules       []DeviceRule `yaml:"rules"`
	AgentConfig AgentConfig  `yaml:"agent_config"`
}

// AgentConfig contains configuration for the background agent
type AgentConfig struct {
	UpdateInterval       string   `yaml:"update_interval"`
	ConfidenceThreshold  float64  `yaml:"confidence_threshold"`
	MaxRulesPerUpdate    int      `yaml:"max_rules_per_update"`
	ResearchSources      []string `yaml:"research_sources"`
}

// DeviceDetector handles YAML-based device detection
type DeviceDetector struct {
	config    *DeviceRulesConfig
	verbose   bool
	configPath string
}

// NewDeviceDetector creates a new device detector
func NewDeviceDetector(verbose bool, configPath string) (*DeviceDetector, error) {
	detector := &DeviceDetector{
		verbose:    verbose,
		configPath: configPath,
	}

	if err := detector.loadConfig(); err != nil {
		return nil, fmt.Errorf("failed to load device detection config: %w", err)
	}

	return detector, nil
}

// loadConfig loads the YAML configuration with fallback hierarchy
func (dd *DeviceDetector) loadConfig() error {
	var yamlData []byte
	var err error
	var source string

	// Priority order: custom config path -> local file -> embedded
	if dd.configPath != "" {
		// Use custom config path if provided
		yamlData, err = ioutil.ReadFile(dd.configPath)
		if err != nil {
			return fmt.Errorf("failed to read custom config at %s: %w", dd.configPath, err)
		}
		source = dd.configPath
	} else if _, err := os.Stat("device_rules.yaml"); err == nil {
		// Use local device_rules.yaml if it exists
		yamlData, err = ioutil.ReadFile("device_rules.yaml")
		if err != nil {
			return fmt.Errorf("failed to read local device_rules.yaml: %w", err)
		}
		source = "device_rules.yaml (local)"
	} else {
		// Fall back to embedded configuration
		yamlData = embeddedRulesYAML
		source = "embedded"
	}

	config := &DeviceRulesConfig{}
	if err := yaml.Unmarshal(yamlData, config); err != nil {
		return fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Sort rules by priority (lower number = higher priority)
	sort.Slice(config.Rules, func(i, j int) bool {
		return config.Rules[i].Priority < config.Rules[j].Priority
	})

	dd.config = config

	if dd.verbose {
		fmt.Printf("📋 Loaded %d device detection rules from %s (v%s)\n",
			len(config.Rules), source, config.Version)
	}

	return nil
}

// DetectDeviceType determines device type using YAML rules
func (dd *DeviceDetector) DetectDeviceType(device *Device) string {
	if dd.config == nil {
		return "Unknown"
	}

	// Evaluate rules in priority order
	for _, rule := range dd.config.Rules {
		if dd.evaluateRule(rule, device) {
			if dd.verbose {
				fmt.Printf("🔍 Device %s matched rule: %s -> %s\n",
					device.IP.String(), rule.Name, rule.DeviceType)
			}
			return rule.DeviceType
		}
	}

	return "Unknown"
}

// GetDeviceIcon returns the icon for a device type
func (dd *DeviceDetector) GetDeviceIcon(deviceType string) string {
	for _, rule := range dd.config.Rules {
		if rule.DeviceType == deviceType && rule.Icon != "" {
			return rule.Icon
		}
	}
	return "❓" // Default fallback
}

// evaluateRule checks if a device matches a rule
func (dd *DeviceDetector) evaluateRule(rule DeviceRule, device *Device) bool {
	conditions := rule.Conditions

	// Evaluate NOT conditions first - if any match, rule fails
	for _, notCondition := range conditions.NotConditions {
		if dd.evaluateCondition(notCondition, device) {
			return false
		}
	}

	// Evaluate ALL_OF conditions - all must match
	if len(conditions.AllOf) > 0 {
		for _, condition := range conditions.AllOf {
			if !dd.evaluateCondition(condition, device) {
				return false
			}
		}
		return true
	}

	// Evaluate ANY_OF conditions - at least one must match
	if len(conditions.AnyOf) > 0 {
		for _, condition := range conditions.AnyOf {
			if dd.evaluateCondition(condition, device) {
				return true
			}
		}
		return false
	}

	return false
}

// evaluateCondition checks if a single condition matches
func (dd *DeviceDetector) evaluateCondition(condition Condition, device *Device) bool {
	hostname := strings.ToLower(device.Hostname)
	macVendor := strings.ToLower(device.MACVendor)

	// Check hostname conditions
	for _, pattern := range condition.HostnameContains {
		if strings.Contains(hostname, strings.ToLower(pattern)) {
			return true
		}
	}

	// Check MAC vendor conditions
	for _, pattern := range condition.MACVendorContains {
		if strings.Contains(macVendor, strings.ToLower(pattern)) {
			return true
		}
	}

	// Check service name conditions
	for _, service := range device.Services {
		serviceName := strings.ToLower(service.Name)
		serviceType := strings.ToLower(service.Type)

		for _, pattern := range condition.ServiceNameContains {
			if strings.Contains(serviceName, strings.ToLower(pattern)) {
				return true
			}
		}

		for _, pattern := range condition.ServiceTypeContains {
			if strings.Contains(serviceType, strings.ToLower(pattern)) {
				return true
			}
		}
	}

	// Check open ports conditions
	if len(condition.OpenPorts) > 0 {
		for _, requiredPort := range condition.OpenPorts {
			found := false
			for _, devicePort := range device.Ports {
				if devicePort == requiredPort {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}

	return false
}

// ExportEmbeddedConfig writes the embedded configuration to a file
func (dd *DeviceDetector) ExportEmbeddedConfig(outputPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write embedded config to file
	if err := ioutil.WriteFile(outputPath, embeddedRulesYAML, 0644); err != nil {
		return fmt.Errorf("failed to write config to %s: %w", outputPath, err)
	}

	fmt.Printf("📄 Exported embedded device detection rules to: %s\n", outputPath)
	return nil
}

// GetRulesInfo returns information about loaded rules
func (dd *DeviceDetector) GetRulesInfo() (int, string) {
	if dd.config == nil {
		return 0, "unknown"
	}
	return len(dd.config.Rules), dd.config.Version
}

// ValidateConfig validates the YAML configuration
func (dd *DeviceDetector) ValidateConfig() error {
	if dd.config == nil {
		return fmt.Errorf("no configuration loaded")
	}

	for i, rule := range dd.config.Rules {
		if rule.Name == "" {
			return fmt.Errorf("rule %d has empty name", i)
		}
		if rule.DeviceType == "" {
			return fmt.Errorf("rule '%s' has empty device_type", rule.Name)
		}
		if rule.Priority < 1 || rule.Priority > 100 {
			return fmt.Errorf("rule '%s' has invalid priority %d (must be 1-100)", rule.Name, rule.Priority)
		}

		// Validate that rule has at least one condition
		conditions := rule.Conditions
		if len(conditions.AllOf) == 0 && len(conditions.AnyOf) == 0 {
			return fmt.Errorf("rule '%s' has no conditions", rule.Name)
		}
	}

	return nil
}