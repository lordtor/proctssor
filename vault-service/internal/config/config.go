package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the vault-service
type Config struct {
	// Server configuration
	Server ServerConfig `yaml:"server"`

	// NATS configuration
	NATS NATSConfig `yaml:"nats"`

	// Vault configuration
	Vault VaultConfig `yaml:"vault"`

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

// VaultConfig holds vault storage configuration
type VaultConfig struct {
	StorageBackend string `yaml:"storage_backend"` // memory, file, postgres
	StoragePath    string `yaml:"storage_path"`
	PostgresURL    string `yaml:"postgres_url"`
	EncryptionKey  string `yaml:"encryption_key"`
	SecretPrefix   string `yaml:"secret_prefix"`
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
			Port: getEnvAsInt("SERVER_PORT", 8083),
		},
		NATS: NATSConfig{
			URL:     getEnv("NATS_URL", "nats://localhost:4222"),
			Timeout: getEnvAsDuration("NATS_TIMEOUT", 30*time.Second),
			Subscriber: SubscriberConfig{
				SubjectPrefix: "wf.cmd.service.vault-service",
				QueueGroup:    "vault-service",
			},
		},
		Vault: VaultConfig{
			StorageBackend: getEnv("VAULT_STORAGE_BACKEND", "memory"),
			StoragePath:    getEnv("VAULT_STORAGE_PATH", "/data/vault"),
			PostgresURL:    getEnv("VAULT_POSTGRES_URL", "postgres://bpmn:bpmn_secret@postgres:5432/bpmn?sslmode=disable"),
			EncryptionKey:  getEnv("VAULT_ENCRYPTION_KEY", "your-32-byte-encryption-key-here"),
			SecretPrefix:   getEnv("VAULT_SECRET_PREFIX", "bpmn"),
		},
		Registry: RegistryConfig{
			EngineURL:         getEnv("ENGINE_URL", "http://engine:8080"),
			HeartbeatInterval: getEnvAsDuration("HEARTBEAT_INTERVAL", 30*time.Second),
			Timeout:           getEnvAsDuration("REGISTRY_TIMEOUT", 10*time.Second),
		},
		Service: ServiceConfig{
			Name:     "vault-service",
			Type:     "external",
			Endpoint: getEnv("SERVICE_ENDPOINT", "http://vault-service:8083"),
			Metadata: map[string]string{
				"description": "Secret management service for workflow automation",
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
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
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
	// YAML parsing disabled for simplicity
	return nil
}
