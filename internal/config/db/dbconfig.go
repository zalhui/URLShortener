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
		DSN:      os.Getenv("DATABASE_DSN"),
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "postgres"),
		SSLMode:  getEnv("DB_SSL_MODE", "disable"),
	}
}

func (c *DBConfig) GetDBConnString() string {
	if c.DSN != "" {
		return c.DSN
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode)
}

func (c *DBConfig) Enabled() bool {
	if c.DSN != "" {
		return true
	}

	keys := []string{
		"DB_HOST",
		"DB_PORT",
		"DB_USER",
		"DB_PASSWORD",
		"DB_NAME",
		"DB_SSL_MODE",
	}
	for _, key := range keys {
		if _, ok := os.LookupEnv(key); ok {
			return true
		}
	}

	return false
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue

}
