package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddr string `env:"SERVER_ADDRESS"`
	BaseURL    string `env:"BASE_URL"`
	AuthIssuer string `env:"AUTH_ISSUER"`
	AuthSecret string `env:"AUTH_ACCESS_TOKEN_SECRET"`

	Filename string `env:"FILE_STORAGE_PATH"`
}

func NewConfig() *Config {
	return &Config{
		ServerAddr: "localhost:8080",
		BaseURL:    "http://localhost:8080/",
		AuthIssuer: "auth-service",
		Filename:   "/tmp/short-url-db.json",
	}
}

func (c *Config) ParseFlags() {
	flag.StringVar(&c.ServerAddr, "a", c.ServerAddr, "server address")
	flag.StringVar(&c.BaseURL, "b", c.BaseURL, "base url")
	flag.StringVar(&c.AuthIssuer, "auth-issuer", c.AuthIssuer, "auth token issuer")
	flag.StringVar(&c.AuthSecret, "auth-secret", c.AuthSecret, "auth access token secret")

	flag.StringVar(&c.Filename, "f", c.Filename, "file storage path")

	flag.Parse()
}

func (c *Config) LoadFromEnv() error {
	tempConfig := &Config{}
	if err := env.Parse(tempConfig); err != nil {
		return fmt.Errorf("failed to parse environment variables: %w", err)
	}

	if tempConfig.ServerAddr != "" {
		c.ServerAddr = tempConfig.ServerAddr
	}
	if tempConfig.BaseURL != "" {
		c.BaseURL = tempConfig.BaseURL
	}
	if tempConfig.AuthIssuer != "" {
		c.AuthIssuer = tempConfig.AuthIssuer
	}
	if tempConfig.AuthSecret != "" {
		c.AuthSecret = tempConfig.AuthSecret
	}
	if tempConfig.Filename != "" {
		c.Filename = tempConfig.Filename
	}

	return nil
}

func (c *Config) LoadConfig() error {
	if err := c.LoadFromEnv(); err != nil {
		return err
	}

	c.ParseFlags()

	return nil
}
