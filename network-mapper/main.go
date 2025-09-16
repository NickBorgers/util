package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "2.7.0"

var (
	disableServiceDiscovery bool
	disableDNSLookup        bool
	scanTimeout             int
	verbose                 bool
	scanMode                string
	thoroughness            int
	deviceRulesPath         string
	exportDeviceRules       string
	updateDeviceRules       bool
)

var rootCmd = &cobra.Command{
	Use:     "network-mapper",
	Short:   "A cross-platform network discovery and visualization tool",
	Version: version,
	Long: `Network Mapper is a CLI tool that scans your local network,
discovers devices, and presents a pictographic representation of the network topology.
It works across macOS, Linux, and Windows platforms.

Features include mDNS/Bonjour discovery, SSDP/UPnP scanning, MAC vendor lookup,
DHCP lease analysis, and comprehensive service identification.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Handle export device rules flag
		if exportDeviceRules != "" {
			detector, err := NewDeviceDetector(verbose, "")
			if err != nil {
				fmt.Printf("❌ Error initializing device detector: %v\n", err)
				os.Exit(1)
			}
			if err := detector.ExportEmbeddedConfig(exportDeviceRules); err != nil {
				fmt.Printf("❌ Error exporting device rules: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Handle update device rules flag
		if updateDeviceRules {
			detector, err := NewDeviceDetector(verbose, deviceRulesPath)
			if err != nil {
				fmt.Printf("❌ Error initializing device detector: %v\n", err)
				os.Exit(1)
			}
			agent := NewDeviceResearchAgent(detector, verbose)
			if err := agent.UpdateFromRepository(); err != nil {
				fmt.Printf("❌ Error updating device rules: %v\n", err)
				os.Exit(1)
			}
			return
		}

		scanner := NewNetworkScanner()
		scanner.SetOptions(disableServiceDiscovery, disableDNSLookup, scanTimeout, verbose)
		scanner.SetThoroughness(thoroughness)
		scanner.SetDeviceRulesPath(deviceRulesPath)

		// Parse and set scan mode
		var mode ScanMode
		switch scanMode {
		case "intelligent":
			mode = ScanModeIntelligent
			// Validate thoroughness level
			if thoroughness < 1 || thoroughness > 5 {
				fmt.Printf("Warning: thoroughness level %d out of range (1-5), using 3\n", thoroughness)
				thoroughness = 3
			}
		case "quick":
			mode = ScanModeQuick
		case "brute-expanded":
			mode = ScanModeExpanded
		case "brute-comprehensive":
			mode = ScanModeComprehensive
		case "brute-firewall":
			mode = ScanModeFirewallTest
		default:
			fmt.Printf("Invalid scan mode: %s. Using 'intelligent' mode.\n", scanMode)
			mode = ScanModeIntelligent
		}

		scanner.SetScanMode(mode)
		scanner.Run()
	},
}

func init() {
	rootCmd.Flags().BoolVar(&disableServiceDiscovery, "no-services", false, "Disable advanced service discovery (mDNS, SSDP, etc.)")
	rootCmd.Flags().BoolVar(&disableDNSLookup, "no-dns", false, "Disable reverse DNS lookups for faster scanning")
	rootCmd.Flags().IntVar(&scanTimeout, "timeout", 5, "Service discovery timeout in seconds")
	rootCmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose output")
	rootCmd.Flags().StringVar(&scanMode, "scan-mode", "intelligent", "Scan mode: intelligent, quick, brute-expanded, brute-comprehensive, brute-firewall")
	rootCmd.Flags().IntVar(&thoroughness, "thoroughness", 3, "Thoroughness level for intelligent mode (1-5, higher=more thorough)")
	rootCmd.Flags().StringVar(&deviceRulesPath, "device-rules", "", "Path to custom device detection rules YAML file")
	rootCmd.Flags().StringVar(&exportDeviceRules, "export-device-rules", "", "Export embedded device rules to specified file and exit")
	rootCmd.Flags().BoolVar(&updateDeviceRules, "update-device-rules", false, "Update device detection rules from repository and exit")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
