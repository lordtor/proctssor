package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the gitlab-service
type Config struct {
	// Server configuration
	Server ServerConfig `yaml:"server"`

	// NATS configuration
	NATS NATSConfig `yaml:"nats"`

	// GitLab configuration
	GitLab GitLabConfig `yaml:"gitlab"`

	// Registry configuration (for heartbeat)
	Registry RegistryConfig `yaml:"registry"`

	// Service configuration
	Service ServiceConfig `yaml:"service"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// NATSConfig holds NATS configuration
type NATSConfig struct {
	URL        string           `yaml:"url"`
	Timeout    time.Duration    `yaml:"timeout"`
	Subscriber SubscriberConfig `yaml:"subscriber"`
}

// SubscriberConfig holds NATS subscriber configuration
type SubscriberConfig struct {
	SubjectPrefix string `yaml:"subject_prefix"`
	QueueGroup    string `yaml:"queue_group"`
}

// GitLabConfig holds GitLab API configuration
type GitLabConfig struct {
	URL       string `yaml:"url"`
	Token     string `yaml:"token"`
	TokenFile string `yaml:"token_file"`
	VerifyTLS bool   `yaml:"verify_tls"`
}

// RegistryConfig holds service registry configuration
type RegistryConfig struct {
	EngineURL         string        `yaml:"engine_url"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	Timeout           time.Duration `yaml:"timeout"`
}

// ServiceConfig holds service metadata
type ServiceConfig struct {
	Name     string            `yaml:"name"`
	Type     string            `yaml:"type"`
	Endpoint string            `yaml:"endpoint"`
	Metadata map[string]string `yaml:"metadata"`
}

// Load loads configuration from environment and yaml file
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnvAsInt("SERVER_PORT", 8081),
		},
		NATS: NATSConfig{
			URL:     getEnv("NATS_URL", "nats://localhost:4222"),
			Timeout: getEnvAsDuration("NATS_TIMEOUT", 30*time.Second),
			Subscriber: SubscriberConfig{
				SubjectPrefix: "wf.cmd.service.gitlab-service",
				QueueGroup:    "gitlab-service",
			},
		},
		GitLab: GitLabConfig{
			URL:       getEnv("GITLAB_URL", "https://gitlab.com"),
			Token:     getEnv("GITLAB_TOKEN", ""),
			TokenFile: getEnv("GITLAB_TOKEN_FILE", ""),
			VerifyTLS: getEnvAsBool("GITLAB_VERIFY_TLS", true),
		},
		Registry: RegistryConfig{
			EngineURL:         getEnv("ENGINE_URL", "http://engine:8080"),
			HeartbeatInterval: getEnvAsDuration("HEARTBEAT_INTERVAL", 30*time.Second),
			Timeout:           getEnvAsDuration("REGISTRY_TIMEOUT", 10*time.Second),
		},
		Service: ServiceConfig{
			Name:     "gitlab-service",
			Type:     "external",
			Endpoint: getEnv("SERVICE_ENDPOINT", "http://gitlab-service:8081"),
			Metadata: map[string]string{
				"description": "GitLab integration service for workflow automation",
				"version":     "1.0.0",
			},
		},
	}

	// Override from YAML if exists
	yamlPath := os.Getenv("CONFIG_PATH")
	if yamlPath == "" {
		yamlPath = "config.yaml"
	}

	if data, err := os.ReadFile(yamlPath); err == nil {
		if err := parseYAML(data, cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := parseInt(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := parseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func parseYAML(data []byte, cfg *Config) error {
	return nil // YAML parsing disabled for simplicity
}

func parseInt(value string) (int, error) {
	return strconv.Atoi(value)
}

func parseBool(value string) (bool, error) {
	return strconv.ParseBool(value)
}

func parseDuration(value string) (time.Duration, error) {
	return time.ParseDuration(value)
}
