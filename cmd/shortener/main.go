package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/zalhui/URLShortener/internal/config"
	"github.com/zalhui/URLShortener/internal/delivery/http/handler"
	"github.com/zalhui/URLShortener/internal/logger"
	"github.com/zalhui/URLShortener/internal/repository"
	"github.com/zalhui/URLShortener/internal/service"
)

func main() {
	if err := logger.Init(); err != nil {
		log.Fatal(err)
	}
	defer logger.Sugar.Sync()

	cfg := config.NewConfig()
	if err := cfg.LoadConfig(); err != nil {
		logger.Sugar.Fatalw("failed to load config", "error", err)
	}

	port := strings.Split(cfg.ServerAddr, ":")[1]

	var repo repository.URLRepository
	if cfg.Filename != "" {
		var err error
		repo, err = repository.NewFileRepository(cfg.Filename)
		if err != nil {
			logger.Sugar.Fatalw("failed to create repository", "error", err)
		}

	} else {
		repo = repository.NewMemoryRepository()
	}
	defer repo.Close()

	shortenerService := service.NewShortenerService(repo, cfg.BaseURL)
	shortener := handler.NewShortenHandler(shortenerService)

	logger.Sugar.Infow(
		"Starting server",
		"address", cfg.ServerAddr,
	)

	if err := http.ListenAndServe(":"+port, shortener.URLRouter()); err != nil {
		log.Fatal(err)
	}
}
