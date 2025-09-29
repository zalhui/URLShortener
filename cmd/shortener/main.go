package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/zalhui/URLShortener/internal/config"
	"github.com/zalhui/URLShortener/internal/handler"
	"github.com/zalhui/URLShortener/internal/logger"
)

func main() {
	if err := logger.Init(); err != nil {
		log.Fatal(err)
	}
	defer logger.Sugar.Sync()

	cfg := config.NewConfig()
	if err := cfg.LoadConfig(); err != nil {
		log.Fatal(err)
	}

	port := strings.Split(cfg.ServerAddr, ":")[1]

	shortener := handler.NewURLShortener(cfg.BaseURL)

	logger.Sugar.Infow(
		"Starting server",
		"address", cfg.ServerAddr,
	)

	if err := http.ListenAndServe(":"+port, shortener.URLRouter()); err != nil {
		log.Fatal(err)
	}
}
