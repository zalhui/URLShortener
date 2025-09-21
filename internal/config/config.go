package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddr string `env:"SERVER_ADDRESS"`
	BaseURL    string `env:"BASE_URL"`
}

func NewConfig() *Config {
	return &Config{
		ServerAddr: "localhost:8080",
		BaseURL:    "http://localhost:8080",
	}
}

func (c *Config) ParseFlags() {
	flag.StringVar(&c.ServerAddr, "a", c.ServerAddr, "server address")
	flag.StringVar(&c.BaseURL, "b", c.BaseURL, "base url")

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

	return nil
}

func (c *Config) LoadConfig() error {
	c.ParseFlags()

	if err := c.LoadFromEnv(); err != nil {
		return err
	}

	return nil
}
