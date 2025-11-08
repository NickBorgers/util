package config

import (
	"time"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// Config represents the complete application configuration
type Config struct {
	General       GeneralConfig       `yaml:"general"`
	Sites         SitesConfig         `yaml:"sites"`
	Browser       BrowserConfig       `yaml:"browser"`
	Logging       LoggingConfig       `yaml:"logging"`
	Elasticsearch ElasticsearchConfig `yaml:"elasticsearch"`
	SNMP          SNMPConfig          `yaml:"snmp"`
	Prometheus    PrometheusConfig    `yaml:"prometheus"`
	Advanced      AdvancedConfig      `yaml:"advanced"`
}

// GeneralConfig contains general application settings
type GeneralConfig struct {
	InterTestDelay  time.Duration `yaml:"inter_test_delay"`
	GlobalTimeout   time.Duration `yaml:"global_timeout"`
	CacheSize       int           `yaml:"cache_size"`
}

// SitesConfig contains the list of sites to monitor
type SitesConfig struct {
	List []models.SiteDefinition `yaml:"list"`
}

// BrowserConfig contains browser-specific settings
type BrowserConfig struct {
	Headless          bool   `yaml:"headless"`
	UserAgent         string `yaml:"user_agent"`
	WindowWidth       int    `yaml:"window_width"`
	WindowHeight      int    `yaml:"window_height"`
	DisableImages     bool   `yaml:"disable_images"`
	DisableJavaScript bool   `yaml:"disable_javascript"`
	ClearCookies      bool   `yaml:"clear_cookies"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level                 string `yaml:"level"`
	Format                string `yaml:"format"`
	IncludeBrowserConsole bool   `yaml:"include_browser_console"`
}

// ElasticsearchConfig contains Elasticsearch output settings
type ElasticsearchConfig struct {
	Enabled       bool          `yaml:"enabled"`
	Endpoint      string        `yaml:"endpoint"`
	IndexPattern  string        `yaml:"index_pattern"`
	Username      string        `yaml:"username"`
	Password      string        `yaml:"password"`
	APIKey        string        `yaml:"api_key"`
	BulkSize      int           `yaml:"bulk_size"`
	FlushInterval time.Duration `yaml:"flush_interval"`
	MaxRetries    int           `yaml:"max_retries"`
	RetryBackoff  time.Duration `yaml:"retry_backoff"`
	TLSEnabled    bool          `yaml:"tls_enabled"`
	TLSSkipVerify bool          `yaml:"tls_skip_verify"`
	TLSCertFile   string        `yaml:"tls_cert_file"`
	TLSKeyFile    string        `yaml:"tls_key_file"`
	TLSCAFile     string        `yaml:"tls_ca_file"`
}

// SNMPConfig contains SNMP agent settings
type SNMPConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Port           int    `yaml:"port"`
	Community      string `yaml:"community"`
	ListenAddress  string `yaml:"listen_address"`
	EnterpriseOID  string `yaml:"enterprise_oid"`
}

// PrometheusConfig contains Prometheus exporter settings
type PrometheusConfig struct {
	Enabled          bool    `yaml:"enabled"`
	Port             int     `yaml:"port"`
	Path             string  `yaml:"path"`
	ListenAddress    string  `yaml:"listen_address"`
	IncludeGoMetrics bool    `yaml:"include_go_metrics"`
	LatencyBuckets   []float64 `yaml:"latency_buckets"`
}

// AdvancedConfig contains advanced/debugging settings
type AdvancedConfig struct {
	PProfEnabled          bool          `yaml:"pprof_enabled"`
	PProfPort             int           `yaml:"pprof_port"`
	HealthCheckEnabled    bool          `yaml:"health_check_enabled"`
	HealthCheckPort       int           `yaml:"health_check_port"`
	HealthCheckPath       string        `yaml:"health_check_path"`
	ShutdownTimeout       time.Duration `yaml:"shutdown_timeout"`
	MaxConcurrentBrowsers int           `yaml:"max_concurrent_browsers"`
	CaptureScreenshots    bool          `yaml:"capture_screenshots"`
	ScreenshotPath        string        `yaml:"screenshot_path"`
	DNSServers            []string      `yaml:"dns_servers"`
}

// Load loads configuration from file and environment variables
func Load(configFile string) (*Config, error) {
	// Start with defaults
	cfg := DefaultConfig()

	// TODO: Load from YAML file if provided
	// if configFile != "" {
	//     err := loadFromYAML(configFile, cfg)
	//     if err != nil {
	//         return nil, err
	//     }
	// }

	// Override with environment variables
	if err := LoadFromEnv(cfg); err != nil {
		return nil, err
	}

	// Load default sites if none configured
	if len(cfg.Sites.List) == 0 {
		cfg.Sites.List = DefaultSites()
	}

	return cfg, nil
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		General: GeneralConfig{
			InterTestDelay: 2 * time.Second,
			GlobalTimeout:  30 * time.Second,
			CacheSize:      100,
		},
		Browser: BrowserConfig{
			Headless:     true,
			UserAgent:    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			WindowWidth:  1920,
			WindowHeight: 1080,
			ClearCookies: true,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Elasticsearch: ElasticsearchConfig{
			Enabled:       false,
			IndexPattern:  "internet-connection-monitor-%{+yyyy.MM.dd}",
			BulkSize:      50,
			FlushInterval: 10 * time.Second,
			MaxRetries:    3,
			RetryBackoff:  1 * time.Second,
		},
		SNMP: SNMPConfig{
			Enabled:       true,
			Port:          161,
			Community:     "public",
			ListenAddress: "0.0.0.0",
			EnterpriseOID: ".1.3.6.1.4.1.99999",
		},
		Prometheus: PrometheusConfig{
			Enabled:          true,
			Port:             9090,
			Path:             "/metrics",
			ListenAddress:    "0.0.0.0",
			IncludeGoMetrics: true,
			LatencyBuckets:   []float64{10, 50, 100, 250, 500, 1000, 2500, 5000, 10000},
		},
		Advanced: AdvancedConfig{
			HealthCheckEnabled:    true,
			HealthCheckPort:       8080,
			HealthCheckPath:       "/health",
			ShutdownTimeout:       30 * time.Second,
			MaxConcurrentBrowsers: 1,
			ScreenshotPath:        "/tmp/screenshots",
		},
	}
}
