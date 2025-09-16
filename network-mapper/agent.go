package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DeviceResearchAgent handles autonomous research and rule updates
type DeviceResearchAgent struct {
	config          *DeviceRulesConfig
	verbose         bool
	repositoryURL   string
	localConfigPath string
	detector        *DeviceDetector
}

// NewDeviceResearchAgent creates a new research agent
func NewDeviceResearchAgent(detector *DeviceDetector, verbose bool) *DeviceResearchAgent {
	return &DeviceResearchAgent{
		detector:        detector,
		verbose:         verbose,
		repositoryURL:   "https://raw.githubusercontent.com/NickBorgers/util/main/network-mapper/device_rules.yaml",
		localConfigPath: "device_rules_updated.yaml",
	}
}

// ResearchCandidate represents a device that needs research
type ResearchCandidate struct {
	Hostname     string
	MACVendor    string
	Services     []Service
	Ports        []int
	DeviceType   string
	Confidence   float64
	ResearchHint string
}

// UpdateFromRepository checks for and downloads updated rules from the repository
func (agent *DeviceResearchAgent) UpdateFromRepository() error {
	if agent.verbose {
		fmt.Println("🔄 Checking for updated device detection rules...")
	}

	// Download latest rules from repository
	resp, err := http.Get(agent.repositoryURL)
	if err != nil {
		return fmt.Errorf("failed to fetch updated rules: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch rules: HTTP %d", resp.StatusCode)
	}

	remoteData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read remote rules: %w", err)
	}

	// Parse remote configuration
	var remoteConfig DeviceRulesConfig
	if err := yaml.Unmarshal(remoteData, &remoteConfig); err != nil {
		return fmt.Errorf("failed to parse remote rules: %w", err)
	}

	// Compare versions/hashes to see if update is needed
	if agent.shouldUpdate(remoteData) {
		if err := agent.applyUpdate(remoteData, &remoteConfig); err != nil {
			return fmt.Errorf("failed to apply update: %w", err)
		}

		if agent.verbose {
			fmt.Printf("✅ Updated device rules to version %s (%d rules)\n",
				remoteConfig.Version, len(remoteConfig.Rules))
		}
	} else {
		if agent.verbose {
			fmt.Println("✅ Device rules are already up to date")
		}
	}

	return nil
}

// shouldUpdate determines if the local rules need updating
func (agent *DeviceResearchAgent) shouldUpdate(remoteData []byte) bool {
	// Calculate hash of remote data
	remoteHash := sha256.Sum256(remoteData)
	remoteHashStr := hex.EncodeToString(remoteHash[:])

	// Calculate hash of current embedded data
	embeddedHash := sha256.Sum256(embeddedRulesYAML)
	embeddedHashStr := hex.EncodeToString(embeddedHash[:])

	if agent.verbose {
		fmt.Printf("🔍 Remote hash: %s\n", remoteHashStr[:16]+"...")
		fmt.Printf("🔍 Local hash:  %s\n", embeddedHashStr[:16]+"...")
	}

	return remoteHashStr != embeddedHashStr
}

// applyUpdate saves the new rules and updates the detector
func (agent *DeviceResearchAgent) applyUpdate(data []byte, config *DeviceRulesConfig) error {
	// Save updated rules to local file
	if err := os.WriteFile(agent.localConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save updated rules: %w", err)
	}

	// Update the detector configuration
	agent.config = config

	if agent.verbose {
		fmt.Printf("💾 Saved updated rules to %s\n", agent.localConfigPath)
	}

	return nil
}

// ResearchUnknownDevices analyzes unknown devices and suggests new rules
func (agent *DeviceResearchAgent) ResearchUnknownDevices(devices []Device) []ResearchCandidate {
	var candidates []ResearchCandidate

	for _, device := range devices {
		if device.DeviceType == "Unknown" || device.DeviceType == "" {
			candidate := agent.analyzeDevice(device)
			if candidate.Confidence >= agent.getConfidenceThreshold() {
				candidates = append(candidates, candidate)
			}
		}
	}

	return candidates
}

// analyzeDevice performs heuristic analysis of an unknown device
func (agent *DeviceResearchAgent) analyzeDevice(device Device) ResearchCandidate {
	candidate := ResearchCandidate{
		Hostname:   device.Hostname,
		MACVendor:  device.MACVendor,
		Services:   device.Services,
		Ports:      device.Ports,
		Confidence: 0.0,
	}

	// Analyze hostname patterns
	hostname := strings.ToLower(device.Hostname)
	vendor := strings.ToLower(device.MACVendor)

	// High-confidence patterns
	if agent.matchesPatterns(hostname, []string{"camera", "cam", "security"}) {
		candidate.DeviceType = "Security Camera"
		candidate.Confidence = 0.9
		candidate.ResearchHint = "Hostname suggests security camera"
	} else if agent.matchesPatterns(hostname, []string{"thermostat", "hvac", "climate"}) {
		candidate.DeviceType = "Smart Thermostat"
		candidate.Confidence = 0.9
		candidate.ResearchHint = "Hostname suggests smart thermostat"
	} else if agent.matchesPatterns(hostname, []string{"doorbell", "bell", "door"}) {
		candidate.DeviceType = "Smart Doorbell"
		candidate.Confidence = 0.85
		candidate.ResearchHint = "Hostname suggests smart doorbell"
	} else if agent.matchesPatterns(vendor, []string{"tesla", "bmw", "ford", "toyota"}) {
		candidate.DeviceType = "Connected Vehicle"
		candidate.Confidence = 0.95
		candidate.ResearchHint = "MAC vendor suggests connected vehicle"
	}

	// Service-based analysis
	for _, service := range device.Services {
		serviceType := strings.ToLower(service.Type)
		serviceName := strings.ToLower(service.Name)

		if strings.Contains(serviceType, "camera") || strings.Contains(serviceName, "camera") {
			candidate.DeviceType = "Security Camera"
			candidate.Confidence = max(candidate.Confidence, 0.8)
			candidate.ResearchHint = "Service suggests camera functionality"
		} else if strings.Contains(serviceType, "music") || strings.Contains(serviceName, "music") {
			candidate.DeviceType = "Music Player"
			candidate.Confidence = max(candidate.Confidence, 0.75)
			candidate.ResearchHint = "Service suggests music player"
		}
	}

	// Port-based analysis
	if agent.containsPort(device.Ports, 554) { // RTSP
		candidate.DeviceType = "IP Camera"
		candidate.Confidence = max(candidate.Confidence, 0.85)
		candidate.ResearchHint = "RTSP port suggests IP camera"
	} else if agent.containsPort(device.Ports, 1883) { // MQTT
		candidate.DeviceType = "IoT Device"
		candidate.Confidence = max(candidate.Confidence, 0.7)
		candidate.ResearchHint = "MQTT port suggests IoT device"
	}

	return candidate
}

// matchesPatterns checks if text matches any of the given patterns
func (agent *DeviceResearchAgent) matchesPatterns(text string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

// containsPort checks if a port is in the list
func (agent *DeviceResearchAgent) containsPort(ports []int, port int) bool {
	for _, p := range ports {
		if p == port {
			return true
		}
	}
	return false
}

// getConfidenceThreshold returns the minimum confidence for rule suggestions
func (agent *DeviceResearchAgent) getConfidenceThreshold() float64 {
	if agent.config != nil && agent.config.AgentConfig.ConfidenceThreshold > 0 {
		return agent.config.AgentConfig.ConfidenceThreshold
	}
	return 0.8 // Default threshold
}

// GenerateRuleSuggestions creates YAML rule suggestions for research candidates
func (agent *DeviceResearchAgent) GenerateRuleSuggestions(candidates []ResearchCandidate) string {
	if len(candidates) == 0 {
		return "# No rule suggestions generated\n"
	}

	var yaml strings.Builder
	yaml.WriteString("# Suggested device detection rules\n")
	yaml.WriteString("# Generated by Network Mapper research agent\n")
	yaml.WriteString(fmt.Sprintf("# Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	for i, candidate := range candidates {
		if i < agent.getMaxRulesPerUpdate() {
			yaml.WriteString(agent.generateRuleYAML(candidate, i))
			yaml.WriteString("\n")
		}
	}

	return yaml.String()
}

// generateRuleYAML creates YAML for a single rule suggestion
func (agent *DeviceResearchAgent) generateRuleYAML(candidate ResearchCandidate, index int) string {
	ruleName := fmt.Sprintf("Auto_%s_%d", strings.ReplaceAll(candidate.DeviceType, " ", "_"), index)
	priority := 90 + index // Lower priority for auto-generated rules

	var yaml strings.Builder
	yaml.WriteString(fmt.Sprintf("  - name: \"%s\"\n", ruleName))
	yaml.WriteString(fmt.Sprintf("    priority: %d\n", priority))
	yaml.WriteString(fmt.Sprintf("    device_type: \"%s\"\n", candidate.DeviceType))
	yaml.WriteString("    icon: \"🤖\"  # Auto-generated rule\n")
	yaml.WriteString(fmt.Sprintf("    description: \"Auto-detected %s (confidence: %.1f%%)\"\n",
		candidate.DeviceType, candidate.Confidence*100))
	yaml.WriteString("    conditions:\n")
	yaml.WriteString("      any_of:\n")

	if candidate.Hostname != "" {
		hostname := strings.ToLower(candidate.Hostname)
		yaml.WriteString(fmt.Sprintf("        - hostname_contains: [\"%s\"]\n", hostname))
	}

	if candidate.MACVendor != "" {
		vendor := strings.ToLower(candidate.MACVendor)
		yaml.WriteString(fmt.Sprintf("        - mac_vendor_contains: [\"%s\"]\n", vendor))
	}

	return yaml.String()
}

// getMaxRulesPerUpdate returns the maximum number of rules to generate per update
func (agent *DeviceResearchAgent) getMaxRulesPerUpdate() int {
	if agent.config != nil && agent.config.AgentConfig.MaxRulesPerUpdate > 0 {
		return agent.config.AgentConfig.MaxRulesPerUpdate
	}
	return 10 // Default limit
}

// max returns the maximum of two float64 values
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}