package config

import "flag"

type Config struct {
	ServerAddr string
	BaseURL    string
}

func NewConfig() *Config {
	return &Config{
		ServerAddr: "localhost:8080",
		BaseURL:    "",
	}
}

func (c *Config) ParseFlags() {
	flag.StringVar(&c.ServerAddr, "a", "localhost:8080", "server address")
	flag.StringVar(&c.BaseURL, "b", "http://localhost:8080/", "base url")

	flag.Parse()
}
