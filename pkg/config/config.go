package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Environment string        `mapstructure:"environment"`
	Version     string        `mapstructure:"version"`
	Server      ServerConfig  `mapstructure:"server"`
	Database    DatabaseConfig `mapstructure:"database"`
	Redis       RedisConfig   `mapstructure:"redis"`
	Kafka       KafkaConfig   `mapstructure:"kafka"`
	RabbitMQ    RabbitMQConfig `mapstructure:"rabbitmq"`
	Auth        AuthConfig    `mapstructure:"auth"`
	Logger      LoggerConfig  `mapstructure:"logger"`
	Metrics     MetricsConfig `mapstructure:"metrics"`
	Tracing     TracingConfig `mapstructure:"tracing"`
	Vault       VaultConfig   `mapstructure:"vault"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	Host         string        `mapstructure:"host"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	TLS          TLSConfig     `mapstructure:"tls"`
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	User         string        `mapstructure:"user"`
	Password     string        `mapstructure:"password"`
	Database     string        `mapstructure:"database"`
	SSLMode      string        `mapstructure:"ssl_mode"`
	MaxOpenConns int           `mapstructure:"max_open_conns"`
	MaxIdleConns int           `mapstructure:"max_idle_conns"`
	MaxLifetime  time.Duration `mapstructure:"max_lifetime"`
	MaxIdleTime  time.Duration `mapstructure:"max_idle_time"`
}

// DSN returns the database connection string
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Database, d.SSLMode)
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	Database     int           `mapstructure:"database"`
	MaxRetries   int           `mapstructure:"max_retries"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	PoolTimeout  time.Duration `mapstructure:"pool_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// Address returns the Redis address
func (r RedisConfig) Address() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers       []string      `mapstructure:"brokers"`
	ConsumerGroup string        `mapstructure:"consumer_group"`
	BatchSize     int           `mapstructure:"batch_size"`
	BatchTimeout  time.Duration `mapstructure:"batch_timeout"`
	RetryMax      int           `mapstructure:"retry_max"`
	Topics        TopicsConfig  `mapstructure:"topics"`
}

// TopicsConfig holds Kafka topics configuration
type TopicsConfig struct {
	UserEvents      string `mapstructure:"user_events"`
	ProductEvents   string `mapstructure:"product_events"`
	OrderEvents     string `mapstructure:"order_events"`
	PaymentEvents   string `mapstructure:"payment_events"`
	InventoryEvents string `mapstructure:"inventory_events"`
}

// RabbitMQConfig holds RabbitMQ configuration
type RabbitMQConfig struct {
	URL          string        `mapstructure:"url"`
	Exchange     string        `mapstructure:"exchange"`
	ExchangeType string        `mapstructure:"exchange_type"`
	RetryMax     int           `mapstructure:"retry_max"`
	RetryDelay   time.Duration `mapstructure:"retry_delay"`
	Queues       QueuesConfig  `mapstructure:"queues"`
}

// QueuesConfig holds RabbitMQ queues configuration
type QueuesConfig struct {
	EmailNotifications string `mapstructure:"email_notifications"`
	SMSNotifications   string `mapstructure:"sms_notifications"`
	PaymentProcessing  string `mapstructure:"payment_processing"`
	OrderProcessing    string `mapstructure:"order_processing"`
	InventoryUpdates   string `mapstructure:"inventory_updates"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWT    JWTConfig    `mapstructure:"jwt"`
	OAuth2 OAuth2Config `mapstructure:"oauth2"`
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	SecretKey      string        `mapstructure:"secret_key"`
	Issuer         string        `mapstructure:"issuer"`
	Expiration     time.Duration `mapstructure:"expiration"`
	RefreshExpiration time.Duration `mapstructure:"refresh_expiration"`
}

// OAuth2Config holds OAuth2 configuration
type OAuth2Config struct {
	Enabled      bool   `mapstructure:"enabled"`
	GoogleID     string `mapstructure:"google_client_id"`
	GoogleSecret string `mapstructure:"google_client_secret"`
	GitHubID     string `mapstructure:"github_client_id"`
	GitHubSecret string `mapstructure:"github_client_secret"`
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Path      string `mapstructure:"path"`
	Namespace string `mapstructure:"namespace"`
	Subsystem string `mapstructure:"subsystem"`
}

// TracingConfig holds tracing configuration
type TracingConfig struct {
	Enabled     bool    `mapstructure:"enabled"`
	ServiceName string  `mapstructure:"service_name"`
	Endpoint    string  `mapstructure:"endpoint"`
	SampleRate  float64 `mapstructure:"sample_rate"`
}

// VaultConfig holds Vault configuration
type VaultConfig struct {
	Address   string `mapstructure:"address"`
	Token     string `mapstructure:"token"`
	Namespace string `mapstructure:"namespace"`
	AuthMethod string `mapstructure:"auth_method"`
	Role      string `mapstructure:"role"`
	SecretPath string `mapstructure:"secret_path"`
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	config := &Config{}

	// Set configuration file name and paths
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Enable reading from environment variables
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read configuration file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal configuration
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Set default values if not provided
	setDefaults(config)

	// Validate configuration
	if err := validate(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// setDefaults sets default values for configuration
func setDefaults(config *Config) {
	if config.Environment == "" {
		config.Environment = "development"
	}
	
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	
	if config.Server.ReadTimeout == 0 {
		config.Server.ReadTimeout = 10 * time.Second
	}
	
	if config.Server.WriteTimeout == 0 {
		config.Server.WriteTimeout = 10 * time.Second
	}
	
	if config.Server.IdleTimeout == 0 {
		config.Server.IdleTimeout = 60 * time.Second
	}
	
	if config.Logger.Level == "" {
		config.Logger.Level = "info"
	}
	
	if config.Logger.Format == "" {
		config.Logger.Format = "json"
	}
	
	if config.Metrics.Path == "" {
		config.Metrics.Path = "/metrics"
	}
	
	if config.Tracing.SampleRate == 0 {
		config.Tracing.SampleRate = 0.1
	}
}

// validate validates the configuration
func validate(config *Config) error {
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}
	
	// Only validate required services if they are configured
	// For Phase 1, we'll allow empty configurations for non-essential services
	
	return nil
}
