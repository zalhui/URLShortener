package config

import (
	"flag"
	"fmt"
	"time"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddr        string        `env:"AUTH_SERVER_ADDRESS"`
	AccessTokenSecret string        `env:"AUTH_ACCESS_TOKEN_SECRET"`
	AccessTokenTTL    time.Duration `env:"AUTH_ACCESS_TOKEN_TTL"`
	RefreshTokenTTL   time.Duration `env:"AUTH_REFRESH_TOKEN_TTL"`
	Issuer            string        `env:"AUTH_ISSUER"`
	CookieName        string        `env:"AUTH_COOKIE_NAME"`
	CookieDomain      string        `env:"AUTH_COOKIE_DOMAIN"`
	CookieSecure      bool          `env:"AUTH_COOKIE_SECURE"`
}

type CookieConfig struct {
	Name     string
	Domain   string
	Secure   bool
	MaxAge   time.Duration
	HTTPOnly bool
}

func NewConfig() *Config {
	return &Config{
		ServerAddr:      "localhost:8081",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "auth-service",
		CookieName:      "refresh_token",
		CookieSecure:    false,
	}
}

func (c *Config) ParseFlags() {
	flag.StringVar(&c.ServerAddr, "a", c.ServerAddr, "auth service address")
	flag.DurationVar(&c.AccessTokenTTL, "access-ttl", c.AccessTokenTTL, "access token TTL")
	flag.DurationVar(&c.RefreshTokenTTL, "refresh-ttl", c.RefreshTokenTTL, "refresh token TTL")
	flag.BoolVar(&c.CookieSecure, "cookie-secure", c.CookieSecure, "set secure auth cookies")
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
	if tempConfig.AccessTokenSecret != "" {
		c.AccessTokenSecret = tempConfig.AccessTokenSecret
	}
	if tempConfig.AccessTokenTTL != 0 {
		c.AccessTokenTTL = tempConfig.AccessTokenTTL
	}
	if tempConfig.RefreshTokenTTL != 0 {
		c.RefreshTokenTTL = tempConfig.RefreshTokenTTL
	}
	if tempConfig.Issuer != "" {
		c.Issuer = tempConfig.Issuer
	}
	if tempConfig.CookieName != "" {
		c.CookieName = tempConfig.CookieName
	}
	if tempConfig.CookieDomain != "" {
		c.CookieDomain = tempConfig.CookieDomain
	}
	if tempConfig.CookieSecure {
		c.CookieSecure = tempConfig.CookieSecure
	}

	return nil
}

func (c *Config) Validate() error {
	if len(c.AccessTokenSecret) < 32 {
		return fmt.Errorf("AUTH_ACCESS_TOKEN_SECRET must be at least 32 characters")
	}
	if c.AccessTokenTTL <= 0 {
		return fmt.Errorf("AUTH_ACCESS_TOKEN_TTL must be greater than zero")
	}
	if c.RefreshTokenTTL <= 0 {
		return fmt.Errorf("AUTH_REFRESH_TOKEN_TTL must be greater than zero")
	}
	if c.CookieName == "" {
		return fmt.Errorf("AUTH_COOKIE_NAME must not be empty")
	}

	return nil
}

func (c *Config) LoadConfig() error {
	if err := c.LoadFromEnv(); err != nil {
		return err
	}
	c.ParseFlags()

	return c.Validate()
}

func (c *Config) CookieConfig() CookieConfig {
	return CookieConfig{
		Name:     c.CookieName,
		Domain:   c.CookieDomain,
		Secure:   c.CookieSecure,
		MaxAge:   c.RefreshTokenTTL,
		HTTPOnly: true,
	}
}
