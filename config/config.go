package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jwalitptl/admin-api/pkg/messaging/redis"
	"github.com/jwalitptl/admin-api/pkg/worker"
	"github.com/spf13/viper"
)

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	SSLMode  string `yaml:"sslmode"`
}

type ServerConfig struct {
	Port           int           `yaml:"port"`
	ReadTimeout    time.Duration `yaml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
	MaxHeaderBytes int           `yaml:"max_header_bytes"`
}

type EndpointConfig struct {
	Enabled       bool     `yaml:"enabled"`
	EventType     string   `yaml:"event_type"`
	TrackChanges  bool     `yaml:"track_changes,omitempty"`
	TrackedFields []string `yaml:"tracked_fields"`
}

type ResourceConfig struct {
	Create EndpointConfig `yaml:"create,omitempty"`
	Update EndpointConfig `yaml:"update,omitempty"`
	Delete EndpointConfig `yaml:"delete,omitempty"`
}

type EventTrackingConfig struct {
	Enabled   bool                      `yaml:"enabled"`
	Endpoints map[string]ResourceConfig `yaml:"endpoints"`
}

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      struct {
		Secret        string `yaml:"secret"`
		RefreshSecret string `yaml:"refresh_secret"`
		ExpiryHours   int    `yaml:"expiry_hours"`
	} `yaml:"jwt"`
	Redis struct {
		URL string `yaml:"url"`
	} `yaml:"redis"`
	EventTracking EventTrackingConfig `yaml:"event_tracking"`
	RateLimit     struct {
		Enabled           bool
		RequestsPerSecond float64
		Burst             int
	}
	Security struct {
		AllowedOrigins []string
		AllowedMethods []string
		AllowedHeaders []string
	}
	Monitoring struct {
		PrometheusEnabled bool
		MetricsPath       string
	}
	Outbox OutboxConfig `yaml:"outbox"`
}

type OutboxConfig struct {
	BatchSize     int           `yaml:"batch_size"`
	PollInterval  time.Duration `yaml:"poll_interval"`
	RetryAttempts int           `yaml:"retry_attempts"`
	RetryDelay    time.Duration `yaml:"retry_delay"`
}

type RedisConfig struct {
	URL          string        `yaml:"url"`
	MaxRetries   int           `yaml:"max_retries"`
	RetryBackoff time.Duration `yaml:"retry_backoff"`
	PoolSize     int           `yaml:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns"`
}

func LoadConfig() (*Config, error) {
	fmt.Printf("DEBUG: CONFIG_FILE env var: %s\n", os.Getenv("CONFIG_FILE"))
	fmt.Printf("DEBUG: Current working directory: %s\n", getCurrentDir())
	fmt.Printf("DEBUG: Will look in: [. ./config /app /app/config]\n")

	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")           // current directory
	viper.AddConfigPath("./config")    // config subdirectory
	viper.AddConfigPath("/app")        // container root directory
	viper.AddConfigPath("/app/config") // container config directory

	fmt.Printf("DEBUG: Attempting to read config file\n")

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("DEBUG: Failed to read config: %v\n", err)
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	fmt.Printf("DEBUG: Using config file: %s\n", viper.ConfigFileUsed())

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		fmt.Printf("DEBUG: Failed to unmarshal config: %v\n", err)
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Explicitly bind event tracking config
	config.EventTracking.Enabled = viper.GetBool("event_tracking.enabled")
	if endpoints := viper.GetStringMap("event_tracking.endpoints"); endpoints != nil {
		config.EventTracking.Endpoints = make(map[string]ResourceConfig)
		endpointsBytes, _ := json.Marshal(endpoints)
		json.Unmarshal(endpointsBytes, &config.EventTracking.Endpoints)
	}

	fmt.Printf("DEBUG: Event tracking struct after fix: %+v\n", config.EventTracking)

	// Override with environment variables if present
	if port := os.Getenv("DB_PORT"); port != "" {
		config.Database.Port, _ = strconv.Atoi(port)
	}
	if host := os.Getenv("DB_HOST"); host != "" {
		config.Database.Host = host
	}
	// ... other env overrides

	return &config, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return err.Error()
	}
	return dir
}

// Add conversion methods to convert config types
func (c *OutboxConfig) ToWorkerConfig() worker.OutboxProcessorConfig {
	return worker.OutboxProcessorConfig{
		BatchSize:     c.BatchSize,
		PollInterval:  c.PollInterval,
		RetryAttempts: c.RetryAttempts,
		RetryDelay:    c.RetryDelay,
	}
}

func (c *RedisConfig) ToBrokerConfig() redis.Config {
	return redis.Config{
		URL:          c.URL,
		MaxRetries:   c.MaxRetries,
		RetryBackoff: c.RetryBackoff,
		PoolSize:     c.PoolSize,
		MinIdleConns: c.MinIdleConns,
	}
}
