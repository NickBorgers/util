package testloop

import (
	"sync"
	"testing"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// TestSiteIterator_Basic tests basic round-robin iteration
func TestSiteIterator_Basic(t *testing.T) {
	sites := []models.SiteDefinition{
		{URL: "https://google.com", Name: "google"},
		{URL: "https://github.com", Name: "github"},
		{URL: "https://example.com", Name: "example"},
	}

	iter := NewSiteIterator(sites)

	// First iteration should return sites in order
	site1 := iter.Next()
	if site1.Name != "google" {
		t.Errorf("Expected first site 'google', got '%s'", site1.Name)
	}

	site2 := iter.Next()
	if site2.Name != "github" {
		t.Errorf("Expected second site 'github', got '%s'", site2.Name)
	}

	site3 := iter.Next()
	if site3.Name != "example" {
		t.Errorf("Expected third site 'example', got '%s'", site3.Name)
	}

	// Should wrap around to the beginning
	site4 := iter.Next()
	if site4.Name != "google" {
		t.Errorf("Expected wrap-around to 'google', got '%s'", site4.Name)
	}
}

// TestSiteIterator_SingleSite tests iteration with only one site
func TestSiteIterator_SingleSite(t *testing.T) {
	sites := []models.SiteDefinition{
		{URL: "https://google.com", Name: "google"},
	}

	iter := NewSiteIterator(sites)

	// Should always return the same site
	for i := 0; i < 5; i++ {
		site := iter.Next()
		if site.Name != "google" {
			t.Errorf("Iteration %d: expected 'google', got '%s'", i, site.Name)
		}
	}
}

// TestSiteIterator_EmptySites tests iteration with no sites
func TestSiteIterator_EmptySites(t *testing.T) {
	sites := []models.SiteDefinition{}
	iter := NewSiteIterator(sites)

	site := iter.Next()

	// Should return empty site definition
	if site.URL != "" {
		t.Errorf("Expected empty URL for no sites, got '%s'", site.URL)
	}
	if site.Name != "" {
		t.Errorf("Expected empty Name for no sites, got '%s'", site.Name)
	}
}

// TestSiteIterator_NilSites tests iteration with nil sites slice
func TestSiteIterator_NilSites(t *testing.T) {
	iter := NewSiteIterator(nil)

	site := iter.Next()

	// Should return empty site definition
	if site.URL != "" {
		t.Errorf("Expected empty URL for nil sites, got '%s'", site.URL)
	}
}

// TestSiteIterator_Count tests the Count method
func TestSiteIterator_Count(t *testing.T) {
	sites := []models.SiteDefinition{
		{URL: "https://google.com", Name: "google"},
		{URL: "https://github.com", Name: "github"},
		{URL: "https://example.com", Name: "example"},
	}

	iter := NewSiteIterator(sites)

	count := iter.Count()
	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}

	// Count should remain the same after calling Next()
	iter.Next()
	count = iter.Count()
	if count != 3 {
		t.Errorf("Expected count to remain 3, got %d", count)
	}
}

// TestSiteIterator_CountEmpty tests Count with empty sites
func TestSiteIterator_CountEmpty(t *testing.T) {
	iter := NewSiteIterator([]models.SiteDefinition{})

	count := iter.Count()
	if count != 0 {
		t.Errorf("Expected count 0 for empty sites, got %d", count)
	}
}

// TestSiteIterator_Reset tests the Reset method
func TestSiteIterator_Reset(t *testing.T) {
	sites := []models.SiteDefinition{
		{URL: "https://google.com", Name: "google"},
		{URL: "https://github.com", Name: "github"},
		{URL: "https://example.com", Name: "example"},
	}

	iter := NewSiteIterator(sites)

	// Iterate through some sites
	iter.Next() // google
	iter.Next() // github

	// Reset should go back to the beginning
	iter.Reset()

	site := iter.Next()
	if site.Name != "google" {
		t.Errorf("Expected reset to return 'google', got '%s'", site.Name)
	}
}

// TestSiteIterator_ResetAtEnd tests Reset when at the end of the list
func TestSiteIterator_ResetAtEnd(t *testing.T) {
	sites := []models.SiteDefinition{
		{URL: "https://google.com", Name: "google"},
		{URL: "https://github.com", Name: "github"},
	}

	iter := NewSiteIterator(sites)

	// Iterate through all sites
	iter.Next() // google
	iter.Next() // github (would wrap to google next)

	// Reset
	iter.Reset()

	site := iter.Next()
	if site.Name != "google" {
		t.Errorf("Expected reset at end to return 'google', got '%s'", site.Name)
	}
}

// TestSiteIterator_MultipleResets tests calling Reset multiple times
func TestSiteIterator_MultipleResets(t *testing.T) {
	sites := []models.SiteDefinition{
		{URL: "https://google.com", Name: "google"},
		{URL: "https://github.com", Name: "github"},
	}

	iter := NewSiteIterator(sites)

	for i := 0; i < 3; i++ {
		iter.Reset()
		site := iter.Next()
		if site.Name != "google" {
			t.Errorf("Reset %d: expected 'google', got '%s'", i, site.Name)
		}
	}
}

// TestSiteIterator_LongIteration tests many iterations
func TestSiteIterator_LongIteration(t *testing.T) {
	sites := []models.SiteDefinition{
		{URL: "https://google.com", Name: "google"},
		{URL: "https://github.com", Name: "github"},
		{URL: "https://example.com", Name: "example"},
	}

	iter := NewSiteIterator(sites)

	expectedNames := []string{"google", "github", "example"}

	// Iterate 100 times and verify pattern
	for i := 0; i < 100; i++ {
		site := iter.Next()
		expectedName := expectedNames[i%3]
		if site.Name != expectedName {
			t.Errorf("Iteration %d: expected '%s', got '%s'", i, expectedName, site.Name)
		}
	}
}

// TestSiteIterator_ConcurrentAccess tests thread-safe concurrent access
func TestSiteIterator_ConcurrentAccess(t *testing.T) {
	sites := []models.SiteDefinition{
		{URL: "https://google.com", Name: "google"},
		{URL: "https://github.com", Name: "github"},
		{URL: "https://example.com", Name: "example"},
	}

	iter := NewSiteIterator(sites)

	// Track how many times each site was returned
	counts := make(map[string]int)
	var mu sync.Mutex

	// Launch multiple goroutines calling Next() concurrently
	var wg sync.WaitGroup
	numGoroutines := 10
	iterationsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterationsPerGoroutine; j++ {
				site := iter.Next()
				mu.Lock()
				counts[site.Name]++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Total calls should be correct
	totalCalls := numGoroutines * iterationsPerGoroutine
	actualTotal := counts["google"] + counts["github"] + counts["example"]

	if actualTotal != totalCalls {
		t.Errorf("Expected %d total calls, got %d", totalCalls, actualTotal)
	}

	// Each site should be called roughly the same number of times
	// (Within 20% tolerance due to race conditions)
	expectedPerSite := totalCalls / 3
	tolerance := expectedPerSite / 5 // 20% tolerance

	for name, count := range counts {
		if count < expectedPerSite-tolerance || count > expectedPerSite+tolerance {
			t.Errorf("Site '%s' called %d times, expected around %d (±%d)",
				name, count, expectedPerSite, tolerance)
		}
	}
}

// TestSiteIterator_ConcurrentReset tests concurrent access with Reset
func TestSiteIterator_ConcurrentReset(t *testing.T) {
	sites := []models.SiteDefinition{
		{URL: "https://google.com", Name: "google"},
		{URL: "https://github.com", Name: "github"},
	}

	iter := NewSiteIterator(sites)

	var wg sync.WaitGroup

	// One goroutine calling Next()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			iter.Next()
		}
	}()

	// Another goroutine calling Reset()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			iter.Reset()
		}
	}()

	// Should not panic or deadlock
	wg.Wait()
}

// TestSiteIterator_ConcurrentCount tests concurrent Count calls
func TestSiteIterator_ConcurrentCount(t *testing.T) {
	sites := []models.SiteDefinition{
		{URL: "https://google.com", Name: "google"},
		{URL: "https://github.com", Name: "github"},
		{URL: "https://example.com", Name: "example"},
	}

	iter := NewSiteIterator(sites)

	var wg sync.WaitGroup

	// Multiple goroutines calling Count()
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				count := iter.Count()
				if count != 3 {
					t.Errorf("Expected count 3, got %d", count)
				}
			}
		}()
	}

	wg.Wait()
}

// TestSiteIterator_PreserveSiteData tests that site data is not modified
func TestSiteIterator_PreserveSiteData(t *testing.T) {
	originalURL := "https://google.com"
	originalName := "google"

	sites := []models.SiteDefinition{
		{
			URL:                originalURL,
			Name:               originalName,
			TimeoutSeconds:     30,
			WaitForNetworkIdle: true,
		},
	}

	iter := NewSiteIterator(sites)

	// Call Next multiple times
	for i := 0; i < 5; i++ {
		site := iter.Next()

		// Verify data is preserved
		if site.URL != originalURL {
			t.Errorf("Iteration %d: URL changed from '%s' to '%s'", i, originalURL, site.URL)
		}
		if site.Name != originalName {
			t.Errorf("Iteration %d: Name changed from '%s' to '%s'", i, originalName, site.Name)
		}
		if site.TimeoutSeconds != 30 {
			t.Errorf("Iteration %d: TimeoutSeconds changed to %d", i, site.TimeoutSeconds)
		}
		if !site.WaitForNetworkIdle {
			t.Errorf("Iteration %d: WaitForNetworkIdle changed to false", i)
		}
	}
}

// TestSiteIterator_TwoSites tests iteration with exactly two sites
func TestSiteIterator_TwoSites(t *testing.T) {
	sites := []models.SiteDefinition{
		{URL: "https://google.com", Name: "google"},
		{URL: "https://github.com", Name: "github"},
	}

	iter := NewSiteIterator(sites)

	// Should alternate between the two
	for i := 0; i < 10; i++ {
		site := iter.Next()
		expectedName := "google"
		if i%2 == 1 {
			expectedName = "github"
		}
		if site.Name != expectedName {
			t.Errorf("Iteration %d: expected '%s', got '%s'", i, expectedName, site.Name)
		}
	}
}
