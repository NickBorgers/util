package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "1.0.0"

var (
	disableServiceDiscovery bool
	disableDNSLookup        bool
	scanTimeout             int
	verbose                 bool
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
		scanner := NewNetworkScanner()
		scanner.SetOptions(disableServiceDiscovery, disableDNSLookup, scanTimeout, verbose)
		scanner.Run()
	},
}

func init() {
	rootCmd.Flags().BoolVar(&disableServiceDiscovery, "no-services", false, "Disable advanced service discovery (mDNS, SSDP, etc.)")
	rootCmd.Flags().BoolVar(&disableDNSLookup, "no-dns", false, "Disable reverse DNS lookups for faster scanning")
	rootCmd.Flags().IntVar(&scanTimeout, "timeout", 5, "Service discovery timeout in seconds")
	rootCmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}