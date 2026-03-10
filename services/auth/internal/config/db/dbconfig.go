package db

import (
	"fmt"
	"os"
)

type DBConfig struct {
	DSN      string
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func Load() *DBConfig {
	return &DBConfig{
		DSN:      os.Getenv("AUTH_DATABASE_DSN"),
		Host:     getEnv("AUTH_DB_HOST", "localhost"),
		Port:     getEnv("AUTH_DB_PORT", "5432"),
		User:     getEnv("AUTH_DB_USER", "postgres"),
		Password: getEnv("AUTH_DB_PASSWORD", "postgres"),
		DBName:   getEnv("AUTH_DB_NAME", "postgres"),
		SSLMode:  getEnv("AUTH_DB_SSL_MODE", "disable"),
	}
}

func (c *DBConfig) GetDBConnString() string {
	if c.DSN != "" {
		return c.DSN
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode)
}

func (c *DBConfig) Validate() error {
	if c.GetDBConnString() == "" {
		return fmt.Errorf("database DSN is empty")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
