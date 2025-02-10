package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Region   struct {
		DefaultRegion string                  `mapstructure:"default_region"`
		Configs       map[string]RegionConfig `mapstructure:"region_configs"`
	} `mapstructure:"region"`
}

type ServerConfig struct {
	Port           int `mapstructure:"port"`
	TimeoutSeconds int `mapstructure:"timeoutSeconds"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"sslmode"`
}

type JWTConfig struct {
	Secret             string `mapstructure:"secret"`
	RefreshSecret      string `mapstructure:"refresh_secret"`
	ExpiryHours        int    `mapstructure:"expiry_hours"`
	RefreshExpiryHours int    `mapstructure:"refresh_expiry_hours"`
}

type RegionConfig struct {
	DataRetention struct {
		AuditLogs      int `mapstructure:"audit_logs"`
		MedicalRecords int `mapstructure:"medical_records"`
	} `mapstructure:"data_retention"`

	Compliance struct {
		GDPR  bool `mapstructure:"gdpr"`
		HIPAA bool `mapstructure:"hipaa"`
		CCPA  bool `mapstructure:"ccpa"`
	} `mapstructure:"compliance"`

	API struct {
		RateLimit    int      `mapstructure:"rate_limit"`
		MaxPageSize  int      `mapstructure:"max_page_size"`
		AllowedHosts []string `mapstructure:"allowed_hosts"`
	} `mapstructure:"api"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
