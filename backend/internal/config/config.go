package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for the InfraForge server.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Temporal TemporalConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host string
	Port string
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// TemporalConfig holds Temporal server connection settings.
type TemporalConfig struct {
	Host      string
	Port      string
	Namespace string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnv("SERVER_PORT", "8081"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5435"),
			User:     getEnv("DB_USER", "infraforge"),
			Password: getEnv("DB_PASSWORD", "infraforge"),
			DBName:   getEnv("DB_NAME", "infraforge"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Temporal: TemporalConfig{
			Host:      getEnv("TEMPORAL_HOST", "localhost"),
			Port:      getEnv("TEMPORAL_PORT", "7233"),
			Namespace: getEnv("TEMPORAL_NAMESPACE", "default"),
		},
	}
}

// DSN returns the PostgreSQL connection string.
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode,
	)
}

// Address returns the Temporal server address.
func (t *TemporalConfig) Address() string {
	return fmt.Sprintf("%s:%s", t.Host, t.Port)
}

// Address returns the HTTP listen address.
func (s *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%s", s.Host, s.Port)
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
