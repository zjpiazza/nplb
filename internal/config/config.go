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

	// Cloudflare
	CloudflareAPIToken string `mapstructure:"CLOUDFLARE_API_TOKEN" validate:"required"`

	// Cloudflare R2
	R2AccountID       string `mapstructure:"R2_ACCOUNT_ID" validate:"required"`
	R2AccessKeyID     string `mapstructure:"R2_ACCESS_KEY_ID" validate:"required"`
	R2SecretAccessKey string `mapstructure:"R2_SECRET_ACCESS_KEY" validate:"required"`
	R2BucketName      string `mapstructure:"R2_BUCKET_NAME" validate:"required"`
	R2PublicURL       string `mapstructure:"R2_PUBLIC_URL"` // Optional custom domain

	// Cloudflare Queues
	QueueName string `mapstructure:"QUEUE_NAME" validate:"required"`

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

	viper.SetDefault("SERVER_HOST", "0.0.0.0")
	viper.SetDefault("SERVER_PORT", 8080)
	viper.SetDefault("GPG_HOME", "keys")
	viper.SetDefault("OUTPUT_DIR", "build")
	viper.SetDefault("DEFAULT_CODENAME", "stable")
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("LOG_FORMAT", "console")
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

	// TODO: Add validation for the config struct
	// validate := validator.New()
	// if err := validate.Struct(&cfg); err != nil {
	// 	return nil, err
	// }

	return &cfg, nil
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
