package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type OUIEntry struct {
	MAC        string `json:"mac"`
	Vendor     string `json:"vendor"`
	Country    string `json:"country,omitempty"`
	BlockType  string `json:"blockType,omitempty"`
}

type MACVendorLookup struct {
	ouiMap map[string]string
	mu     sync.RWMutex
	cache  map[string]string
}

func NewMACVendorLookup() *MACVendorLookup {
	return &MACVendorLookup{
		ouiMap: make(map[string]string),
		cache:  make(map[string]string),
	}
}

func (mvl *MACVendorLookup) Initialize() error {
	cacheFile := filepath.Join(os.TempDir(), "oui_cache.json")

	if mvl.loadFromCache(cacheFile) {
		fmt.Println("   📋 Loaded MAC vendor database from cache")
		return nil
	}

	fmt.Println("   🌐 Downloading MAC vendor database...")

	err := mvl.downloadOUIDatabase()
	if err != nil {
		fmt.Printf("   ⚠️  Failed to download OUI database: %v\n", err)
		mvl.loadBuiltinVendors()
		return nil
	}

	mvl.saveToCache(cacheFile)
	fmt.Printf("   ✅ Loaded %d MAC vendor entries\n", len(mvl.ouiMap))
	return nil
}

func (mvl *MACVendorLookup) downloadOUIDatabase() error {
	url := "http://standards-oui.ieee.org/oui/oui.txt"

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return mvl.parseOUIFile(resp.Body)
}

func (mvl *MACVendorLookup) parseOUIFile(reader io.Reader) error {
	mvl.mu.Lock()
	defer mvl.mu.Unlock()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.Contains(line, "(hex)") {
			parts := strings.Split(line, "\t")
			if len(parts) >= 3 {
				mac := strings.ToUpper(strings.TrimSpace(parts[0]))
				vendor := strings.TrimSpace(parts[2])

				mac = strings.ReplaceAll(mac, "-", ":")

				mvl.ouiMap[mac] = vendor
			}
		}
	}

	return scanner.Err()
}

func (mvl *MACVendorLookup) loadBuiltinVendors() {
	mvl.mu.Lock()
	defer mvl.mu.Unlock()

	builtinVendors := map[string]string{
		"00:50:56": "VMware",
		"08:00:27": "PCS Systemtechnik GmbH (VirtualBox)",
		"52:54:00": "QEMU/KVM Virtual Machine",
		"00:16:3E": "Xen Virtual Machine",
		"00:0C:29": "VMware",
		"00:05:69": "VMware",
		"00:1C:42": "Parallels",
		"08:00:20": "Sun Microsystems",
		"00:90:27": "Intel Corporation",
		"00:E0:4C": "Realtek",
		"D4:3D:7E": "Apple",
		"AC:DE:48": "Apple",
		"28:CF:E9": "Apple",
		"A4:83:E7": "Apple",
		"68:96:7B": "Apple",
		"3C:07:54": "Apple",
		"B8:E8:56": "Apple",
		"DC:A6:32": "Raspberry Pi Foundation",
		"B8:27:EB": "Raspberry Pi Foundation",
		"E4:5F:01": "Raspberry Pi Foundation",
		"00:1B:63": "Apple",
		"00:26:08": "Apple",
		"04:0C:CE": "Apple",
		"18:E7:F4": "Google",
		"B0:7F:B9": "Dell",
		"D0:67:E5": "Dell",
		"B8:CA:3A": "Dell",
		"6C:2B:59": "Dell",
		"78:2B:CB": "Microsoft",
		"00:12:5A": "Microsoft",
		"7C:ED:8D": "Microsoft",
		"E0:CB:4E": "Intel Corporation",
		"AC:1F:6B": "Intel Corporation",
		"00:21:6A": "Intel Corporation",
		"34:02:86": "Intel Corporation",
		"F0:76:1C": "Samsung",
		"5C:F3:70": "Samsung",
		"78:67:D7": "Samsung",
		"E8:50:8B": "Samsung",
	}

	for mac, vendor := range builtinVendors {
		mvl.ouiMap[mac] = vendor
	}

	fmt.Printf("   📋 Loaded %d built-in MAC vendors\n", len(mvl.ouiMap))
}

func (mvl *MACVendorLookup) loadFromCache(filename string) bool {
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	var ouiEntries []OUIEntry
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&ouiEntries); err != nil {
		return false
	}

	mvl.mu.Lock()
	defer mvl.mu.Unlock()

	for _, entry := range ouiEntries {
		mvl.ouiMap[entry.MAC] = entry.Vendor
	}

	return len(mvl.ouiMap) > 0
}

func (mvl *MACVendorLookup) saveToCache(filename string) error {
	mvl.mu.RLock()
	defer mvl.mu.RUnlock()

	var ouiEntries []OUIEntry
	for mac, vendor := range mvl.ouiMap {
		ouiEntries = append(ouiEntries, OUIEntry{
			MAC:    mac,
			Vendor: vendor,
		})
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(ouiEntries)
}

func (mvl *MACVendorLookup) LookupVendor(macAddress string) string {
	if macAddress == "" || macAddress == "unknown" {
		return ""
	}

	mvl.mu.RLock()
	if vendor, exists := mvl.cache[macAddress]; exists {
		mvl.mu.RUnlock()
		return vendor
	}
	mvl.mu.RUnlock()

	oui := mvl.extractOUI(macAddress)
	if oui == "" {
		return ""
	}

	mvl.mu.RLock()
	vendor, exists := mvl.ouiMap[oui]
	mvl.mu.RUnlock()

	if !exists {
		vendor = mvl.lookupOnlineVendor(macAddress)
	}

	mvl.mu.Lock()
	mvl.cache[macAddress] = vendor
	mvl.mu.Unlock()

	return vendor
}

func (mvl *MACVendorLookup) extractOUI(macAddress string) string {
	mac := strings.ToUpper(strings.ReplaceAll(macAddress, "-", ":"))

	parts := strings.Split(mac, ":")
	if len(parts) < 3 {
		return ""
	}

	return strings.Join(parts[:3], ":")
}

func (mvl *MACVendorLookup) lookupOnlineVendor(macAddress string) string {
	apis := []string{
		fmt.Sprintf("https://api.macvendors.com/%s", macAddress),
		fmt.Sprintf("https://macvendors.co/api/%s/json", macAddress),
	}

	for _, apiURL := range apis {
		if vendor := mvl.queryAPI(apiURL, macAddress); vendor != "" {
			return vendor
		}
	}

	return ""
}

func (mvl *MACVendorLookup) queryAPI(apiURL, macAddress string) string {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	vendor := strings.TrimSpace(string(body))

	if strings.Contains(apiURL, "macvendors.co") {
		var result struct {
			Result struct {
				Company string `json:"company"`
			} `json:"result"`
		}
		if json.Unmarshal(body, &result) == nil {
			vendor = result.Result.Company
		}
	}

	if vendor != "" && !strings.Contains(vendor, "error") && !strings.Contains(vendor, "Not found") {
		return vendor
	}

	return ""
}

func (ns *NetworkScanner) enhanceDevicesWithVendorInfo(devices []Device, mvl *MACVendorLookup) {
	for i := range devices {
		if devices[i].MAC != "" && devices[i].MAC != "unknown" {
			vendor := mvl.LookupVendor(devices[i].MAC)
			devices[i].MACVendor = vendor
		}
	}
}