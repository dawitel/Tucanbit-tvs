package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server            ServerConfig                 `yaml:"server"`
	Database          DatabaseConfig               `yaml:"database"`
	PDM               PDMConfig                    `yaml:"pdm"`
	Verification      VerificationConfig           `yaml:"verification"`
	Security          SecurityConfig               `yaml:"security"`
	ExchangeAPIConfig ExchangeAPIConfig            `yaml:"exchange_api_config"`
	WebSocket         WebSocketConfig              `yaml:"websocket"`
	Helius            HeliusConfig                 `yaml:"helius"`
	MintAddresses     map[string]map[string]string `yaml:"mint_addresses"` // cluster_type -> token_type -> mint_address
	JWT               JWTConfig                    `yaml:"jwt"`
}

type JWTConfig struct {
	Secret string `yaml:"secret"`
}

type ExchangeAPIConfig struct {
	BaseURL          string `yaml:"base_url"`
	Timeout          int    `yaml:"timeout"`
	MaxRetries       int    `yaml:"max_retries"`
	RetryBackoffBase int    `yaml:"retry_backoff_base"`
	APIKey           string `yaml:"api_key"`
	Version          string `yaml:"version"`
}

type HeliusConfig struct {
	APIKey     string            `yaml:"api_key"`
	BaseURLs   map[string]string `yaml:"base_urls"` // cluster_type -> base_url
	Timeout    time.Duration     `yaml:"timeout"`
	MaxRetries int               `yaml:"max_retries"`
}

type DatabaseConfig struct {
	Host            string `yaml:"host"`
	Port            string `yaml:"port"`
	User            string `yaml:"user"`
	DBName          string `yaml:"name"`
	Password        string `yaml:"password"`
	SSLMode         string `yaml:"ssl_mode"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime string `yaml:"conn_max_lifetime"`
}

type ServerConfig struct {
	Host        string `yaml:"host"`
	Port        string `yaml:"port"`
	Environment string `yaml:"environment"`
}

type PDMConfig struct {
	BaseURL          string `yaml:"base_url"`
	Timeout          int    `yaml:"timeout"`
	MaxRetries       int    `yaml:"max_retries"`
	RetryBackoffBase int    `yaml:"retry_backoff_base"`
	APIKey           string `yaml:"api_key"`
	Version          string `yaml:"version"`
}

type VerificationConfig struct {
	PollingInterval     int           `yaml:"polling_interval"`
	SessionTimeoutHours int           `yaml:"session_timeout_hours"`
	ConcurrentWorkers   int           `yaml:"concurrent_workers"`
	CacheEnabled        bool          `yaml:"cache_enabled"`
	CacheTTL            time.Duration `yaml:"cache_ttl"`
}

type SecurityConfig struct {
	APIKey        string `yaml:"api_key"`
	EncryptionKey string `yaml:"encryption_key"`
	TLSCertPath   string `yaml:"tls_cert_path"`
	TLSKeyPath    string `yaml:"tls_key_path"`
}

type WebSocketConfig struct {
	ReadBufferSize  int           `yaml:"read_buffer_size"`
	WriteBufferSize int           `yaml:"write_buffer_size"`
	CheckOrigin     bool          `yaml:"check_origin"`
	PingPeriod      time.Duration `yaml:"ping_period"`
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	var config Config
	configData, err := os.ReadFile("./config.yaml")
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(configData, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
