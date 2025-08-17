package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server           ServerConfig           `yaml:"server"`
	Database         DatabaseConfig         `yaml:"database"`
	ExternalServices ExternalServicesConfig `yaml:"external_services"`
	Verification     VerificationConfig     `yaml:"verification"`
	Security         SecurityConfig         `yaml:"security"`
	WebSocket        WebSocketConfig        `yaml:"websocket"`
}

type DatabaseConfig struct {
	Host            string `yaml:"host"`
	Port            string `yaml:"port"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	DBName          string `yaml:"dbname"`
	SSLMode         string `yaml:"sslmode"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime string `yaml:"conn_max_lifetime"`
}

type ServerConfig struct {
	Host        string `yaml:"host"`
	Port        string `yaml:"port"`
	Environment string `yaml:"environment"`
}

type ExternalServicesConfig struct {
	IndexingEngine IndexingEngineConfig `yaml:"indexing_engine"`
	PDM            PDMConfig            `yaml:"pdm"`
}

type IndexingEngineConfig struct {
	BaseURL    string        `yaml:"base_url"`
	APIKey     string        `yaml:"api_key"`
	Timeout    time.Duration `yaml:"timeout"`
	MaxRetries int           `yaml:"max_retries"`
	RetryDelay time.Duration `yaml:"retry_delay"`
}

type PDMConfig struct {
	BaseURL    string        `yaml:"base_url"`
	APIKey     string        `yaml:"api_key"`
	Timeout    time.Duration `yaml:"timeout"`
	MaxRetries int           `yaml:"max_retries"`
	RetryDelay time.Duration `yaml:"retry_delay"`
}

type VerificationConfig struct {
	Timeout           time.Duration `yaml:"timeout"`
	ConcurrentWorkers int           `yaml:"concurrent_workers"`
	CacheEnabled      bool          `yaml:"cache_enabled"`
	CacheTTL          time.Duration `yaml:"cache_ttl"`
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

	expandEnvVars(&config)

	return &config, nil
}

func expandEnvVars(config *Config) {
	config.Database.Host = os.ExpandEnv(config.Database.Host)
	config.Database.Port = os.ExpandEnv(config.Database.Port)
	config.Database.User = os.ExpandEnv(config.Database.User)
	config.Database.Password = os.ExpandEnv(config.Database.Password)
	config.Database.DBName = os.ExpandEnv(config.Database.DBName)
	config.Database.SSLMode = os.ExpandEnv(config.Database.SSLMode)

	config.Server.Host = os.ExpandEnv(config.Server.Host)
	config.Server.Port = os.ExpandEnv(config.Server.Port)
	config.Server.Environment = os.ExpandEnv(config.Server.Environment)

	config.ExternalServices.IndexingEngine.BaseURL = os.ExpandEnv(config.ExternalServices.IndexingEngine.BaseURL)
	config.ExternalServices.IndexingEngine.APIKey = os.ExpandEnv(config.ExternalServices.IndexingEngine.APIKey)
	config.ExternalServices.PDM.BaseURL = os.ExpandEnv(config.ExternalServices.PDM.BaseURL)
	config.ExternalServices.PDM.APIKey = os.ExpandEnv(config.ExternalServices.PDM.APIKey)

	config.Security.APIKey = os.ExpandEnv(config.Security.APIKey)
	config.Security.EncryptionKey = os.ExpandEnv(config.Security.EncryptionKey)
	config.Security.TLSCertPath = os.ExpandEnv(config.Security.TLSCertPath)
	config.Security.TLSKeyPath = os.ExpandEnv(config.Security.TLSKeyPath)
}
