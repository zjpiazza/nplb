package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	// Server
	ServerHost string `mapstructure:"SERVER_HOST"`
	ServerPort int    `mapstructure:"SERVER_PORT"`
	AppEnv     string `mapstructure:"APP_ENV"`
	LogFormat  string `mapstructure:"LOG_FORMAT"`

	// GitHub
	GitHubToken string `mapstructure:"GITHUB_TOKEN"`

	// Redis (for Asynq worker)
	RedisAddr     string `mapstructure:"REDIS_ADDR"`
	RedisPassword string `mapstructure:"REDIS_PASSWORD"`
	RedisDB       int    `mapstructure:"REDIS_DB"`

	// Cloudflare
	CloudflareAPIToken string `mapstructure:"CLOUDFLARE_API_TOKEN"`

	// Cloudflare R2
	R2AccountID       string `mapstructure:"R2_ACCOUNT_ID"`
	R2AccessKeyID     string `mapstructure:"R2_ACCESS_KEY_ID"`
	R2SecretAccessKey string `mapstructure:"R2_SECRET_ACCESS_KEY"`
	R2BucketName      string `mapstructure:"R2_BUCKET_NAME"`
	R2PublicURL       string `mapstructure:"R2_PUBLIC_URL"` // Optional custom domain

	// Cloudflare Queues
	QueueName string `mapstructure:"QUEUE_NAME"`

	// GPG
	GPGHome     string `mapstructure:"GPG_HOME"`
	GPGKeyEmail string `mapstructure:"GPG_KEY_EMAIL"`

	// Repository
	OutputDir       string `mapstructure:"OUTPUT_DIR"`
	DefaultCodename string `mapstructure:"DEFAULT_CODENAME"`
}

// Load loads the configuration from a .env file and environment variables.
func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	// Server defaults
	viper.SetDefault("SERVER_HOST", "0.0.0.0")
	viper.SetDefault("SERVER_PORT", 8080)
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("LOG_FORMAT", "console")

	// Redis defaults
	viper.SetDefault("REDIS_ADDR", "localhost:6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)

	// Repository defaults
	viper.SetDefault("GPG_HOME", "keys")
	viper.SetDefault("OUTPUT_DIR", "build")
	viper.SetDefault("DEFAULT_CODENAME", "stable")
	viper.SetDefault("QUEUE_NAME", "nplb-queue")

	if err := viper.ReadInConfig(); err != nil {
		// Config file not required if env vars are set
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate validates the configuration and returns an error if required fields are missing.
func (c *Config) Validate() error {
	// For API server, we need queue configuration
	if c.CloudflareAPIToken == "" && c.RedisAddr == "" {
		return fmt.Errorf("either CLOUDFLARE_API_TOKEN (for Cloudflare Queues) or REDIS_ADDR (for Asynq) is required")
	}

	return nil
}

// ValidateWorker validates configuration required for the worker.
func (c *Config) ValidateWorker() error {
	if c.RedisAddr == "" {
		return fmt.Errorf("REDIS_ADDR is required for worker")
	}
	if c.R2AccountID == "" {
		return fmt.Errorf("R2_ACCOUNT_ID is required")
	}
	if c.R2AccessKeyID == "" {
		return fmt.Errorf("R2_ACCESS_KEY_ID is required")
	}
	if c.R2SecretAccessKey == "" {
		return fmt.Errorf("R2_SECRET_ACCESS_KEY is required")
	}
	if c.R2BucketName == "" {
		return fmt.Errorf("R2_BUCKET_NAME is required")
	}
	return nil
}

// StorageURL returns the public URL for the storage bucket.
func (c *Config) StorageURL() string {
	if c.R2PublicURL != "" {
		return c.R2PublicURL
	}
	// Default R2 public URL format
	return fmt.Sprintf("https://%s.r2.dev", c.R2BucketName)
}

// R2Endpoint returns the R2 API endpoint.
func (c *Config) R2Endpoint() string {
	return fmt.Sprintf("https://%s.r2.cloudflarestorage.com", c.R2AccountID)
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}
