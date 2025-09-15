package main

import (
	"fmt"
	"sync"
	"time"
)

type ScanProgress struct {
	totalRanges     int
	completedRanges int
	totalIPs        uint32
	scannedIPs      uint32
	activeScans     int
	startTime       time.Time
	rangeStartTimes map[int]time.Time
	verbose         bool
	mu              sync.RWMutex
	lastUpdate      time.Time
	updateInterval  time.Duration
}

type RangeProgress struct {
	rangeIndex    int
	rangeName     string
	totalIPs      uint32
	scannedIPs    uint32
	devicesFound  int
	startTime     time.Time
	estimatedEnd  time.Time
}

func NewScanProgress(totalRanges int, totalIPs uint32, verbose bool) *ScanProgress {
	return &ScanProgress{
		totalRanges:     totalRanges,
		totalIPs:        totalIPs,
		startTime:       time.Now(),
		rangeStartTimes: make(map[int]time.Time),
		verbose:         verbose,
		updateInterval:  time.Second * 2, // Update every 2 seconds
	}
}

func (sp *ScanProgress) StartRange(rangeIndex int, rangeName string, rangeIPs uint32) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.rangeStartTimes[rangeIndex] = time.Now()

	if sp.verbose {
		fmt.Printf("🎯 [%d/%d] Starting scan of %s (%d IPs)\n",
			rangeIndex+1, sp.totalRanges, rangeName, rangeIPs)
	}
}

func (sp *ScanProgress) CompleteRange(rangeIndex int, rangeName string, devicesFound int) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.completedRanges++

	if startTime, exists := sp.rangeStartTimes[rangeIndex]; exists {
		duration := time.Since(startTime)
		if sp.verbose {
			fmt.Printf("✅ [%d/%d] Completed %s in %v (found %d devices)\n",
				rangeIndex+1, sp.totalRanges, rangeName, duration.Round(time.Second), devicesFound)
		}
		delete(sp.rangeStartTimes, rangeIndex)
	}

	// Show overall progress
	sp.showProgress()
}

func (sp *ScanProgress) IncrementScanned(count uint32) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.scannedIPs += count

	// Only update display every updateInterval to avoid spam
	if time.Since(sp.lastUpdate) >= sp.updateInterval {
		sp.showProgress()
		sp.lastUpdate = time.Now()
	}
}

func (sp *ScanProgress) SetActiveScans(count int) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.activeScans = count
}

func (sp *ScanProgress) showProgress() {
	if sp.totalIPs == 0 {
		return
	}

	elapsed := time.Since(sp.startTime)
	percentage := float64(sp.scannedIPs) / float64(sp.totalIPs) * 100

	// Calculate ETA based on current progress
	var eta string
	if sp.scannedIPs > 0 && percentage < 100 {
		rate := float64(sp.scannedIPs) / elapsed.Seconds() // IPs per second
		remaining := sp.totalIPs - sp.scannedIPs
		etaSeconds := float64(remaining) / rate
		etaDuration := time.Duration(etaSeconds) * time.Second
		eta = fmt.Sprintf(" (ETA: %v)", etaDuration.Round(time.Second))
	} else if percentage >= 100 {
		eta = " (Complete)"
	}

	// Create progress bar
	progressBar := sp.createProgressBar(percentage)

	fmt.Printf("\r📊 Progress: %s %.1f%% (%d/%d IPs)%s - Active: %d scans",
		progressBar, percentage, sp.scannedIPs, sp.totalIPs, eta, sp.activeScans)

	if percentage >= 100 || sp.verbose {
		fmt.Println() // New line for verbose or completion
	}
}

func (sp *ScanProgress) createProgressBar(percentage float64) string {
	const barWidth = 20
	filled := int(percentage * barWidth / 100)

	bar := "["
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else if i == filled && percentage > float64(filled)*100/barWidth {
			bar += "▒"
		} else {
			bar += "░"
		}
	}
	bar += "]"

	return bar
}

func (sp *ScanProgress) ShowFinalSummary() {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	totalDuration := time.Since(sp.startTime)
	rate := float64(sp.scannedIPs) / totalDuration.Seconds()

	fmt.Printf("\n📈 Scan Summary:\n")
	fmt.Printf("   ⏱️  Total time: %v\n", totalDuration.Round(time.Second))
	fmt.Printf("   🎯 Ranges scanned: %d\n", sp.completedRanges)
	fmt.Printf("   📡 IPs scanned: %d\n", sp.scannedIPs)
	fmt.Printf("   ⚡ Scan rate: %.1f IPs/second\n", rate)
}

func (sp *ScanProgress) GetEstimatedCompletion() time.Duration {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	if sp.scannedIPs == 0 {
		return 0
	}

	elapsed := time.Since(sp.startTime)
	rate := float64(sp.scannedIPs) / elapsed.Seconds()
	remaining := sp.totalIPs - sp.scannedIPs

	if remaining <= 0 {
		return 0
	}

	return time.Duration(float64(remaining)/rate) * time.Second
}

func (sp *ScanProgress) ForceUpdate() {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.showProgress()
}

// ScanEstimator helps estimate scan duration before starting
type ScanEstimator struct {
	baseIPsPerSecond float64 // Base scanning rate
}

func NewScanEstimator() *ScanEstimator {
	return &ScanEstimator{
		baseIPsPerSecond: 50.0, // Conservative estimate: 50 IPs per second
	}
}

func (se *ScanEstimator) EstimateDuration(totalIPs uint32, mode ScanMode) time.Duration {
	rate := se.baseIPsPerSecond

	// Adjust rate based on scan mode (more comprehensive = slower due to more service discovery)
	switch mode {
	case ScanModeQuick:
		rate *= 1.5 // Faster due to less service discovery
	case ScanModeNormal:
		rate *= 1.0 // Base rate
	case ScanModeComprehensive:
		rate *= 0.7 // Slower due to more thorough scanning
	case ScanModeFirewallTest:
		rate *= 0.8 // Slightly slower due to targeted scanning
	}

	seconds := float64(totalIPs) / rate
	return time.Duration(seconds) * time.Second
}

func (se *ScanEstimator) GetEstimateDescription(totalIPs uint32, mode ScanMode) string {
	if totalIPs == 0 {
		return "Unknown"
	}

	duration := se.EstimateDuration(totalIPs, mode)

	if duration < time.Minute {
		return fmt.Sprintf("~%d seconds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("~%d minutes", int(duration.Minutes()))
	} else {
		return fmt.Sprintf("~%.1f hours", duration.Hours())
	}
}