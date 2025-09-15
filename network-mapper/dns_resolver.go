package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type DNSResolver struct {
	cache   map[string]string
	mu      sync.RWMutex
	timeout time.Duration
	verbose bool
}

func NewDNSResolver(timeout time.Duration, verbose bool) *DNSResolver {
	return &DNSResolver{
		cache:   make(map[string]string),
		timeout: timeout,
		verbose: verbose,
	}
}

func (dr *DNSResolver) LookupHostname(ip net.IP) string {
	ipStr := ip.String()

	// Check cache first
	dr.mu.RLock()
	if hostname, exists := dr.cache[ipStr]; exists {
		dr.mu.RUnlock()
		return hostname
	}
	dr.mu.RUnlock()

	if dr.verbose {
		fmt.Printf("   🔍 Looking up hostname for %s...\n", ipStr)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), dr.timeout)
	defer cancel()

	// Use system's default resolver for better compatibility with Tailscale and other VPNs
	resolver := net.DefaultResolver

	names, err := resolver.LookupAddr(ctx, ipStr)
	hostname := ""

	if err != nil || len(names) == 0 {
		if dr.verbose {
			if err != nil {
				fmt.Printf("   ❌ DNS lookup failed for %s: %v\n", ipStr, err)
			} else {
				fmt.Printf("   ❌ No reverse DNS records found for %s\n", ipStr)
			}
		}
	} else {
		// Clean up the hostname (remove trailing dots)
		hostname = strings.TrimSuffix(names[0], ".")

		// Validate hostname
		if dr.isValidHostname(hostname) {
			if dr.verbose {
				fmt.Printf("   ✅ Found hostname: %s for %s\n", hostname, ipStr)
			}
		} else {
			if dr.verbose {
				fmt.Printf("   ⚠️  Rejected invalid hostname: '%s' for %s\n", hostname, ipStr)
			}
			hostname = ""
		}
	}

	// Cache the result (even if empty to avoid repeated lookups)
	dr.mu.Lock()
	dr.cache[ipStr] = hostname
	dr.mu.Unlock()

	return hostname
}

func (dr *DNSResolver) BulkLookup(ips []net.IP) map[string]string {
	results := make(map[string]string)
	var wg sync.WaitGroup
	var mu sync.Mutex

	if dr.verbose {
		fmt.Printf("   🔍 Starting bulk DNS lookup for %d addresses...\n", len(ips))
	}

	// Limit concurrent lookups to avoid overwhelming DNS servers
	semaphore := make(chan struct{}, 10)

	for _, ip := range ips {
		wg.Add(1)
		go func(ip net.IP) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			hostname := dr.LookupHostname(ip)

			mu.Lock()
			results[ip.String()] = hostname
			mu.Unlock()
		}(ip)
	}

	wg.Wait()

	if dr.verbose {
		successCount := 0
		for _, hostname := range results {
			if hostname != "" {
				successCount++
			}
		}
		fmt.Printf("   ✅ DNS lookup complete: %d/%d successful\n", successCount, len(ips))
	}

	return results
}

func (dr *DNSResolver) isValidHostname(hostname string) bool {
	if hostname == "" {
		return false
	}

	// Check for obviously invalid hostnames
	invalidPatterns := []string{
		"localhost",
		"_",
		".",
		"unknown",
	}

	hostname = strings.ToLower(hostname)
	for _, pattern := range invalidPatterns {
		if hostname == pattern || strings.HasPrefix(hostname, pattern) {
			return false
		}
	}

	// Basic hostname validation
	if len(hostname) > 253 {
		return false
	}

	// Check if it contains valid characters
	for _, char := range hostname {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '.') {
			return false
		}
	}

	return true
}

func (dr *DNSResolver) GetCacheStats() (int, int) {
	dr.mu.RLock()
	defer dr.mu.RUnlock()

	total := len(dr.cache)
	successful := 0

	for _, hostname := range dr.cache {
		if hostname != "" {
			successful++
		}
	}

	return successful, total
}

func (dr *DNSResolver) ClearCache() {
	dr.mu.Lock()
	defer dr.mu.Unlock()
	dr.cache = make(map[string]string)
}
