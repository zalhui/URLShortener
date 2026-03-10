package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zalhui/URLShortener/internal/config"
	dbconfig "github.com/zalhui/URLShortener/internal/config/db"
	"github.com/zalhui/URLShortener/internal/database"
	"github.com/zalhui/URLShortener/internal/delivery/http/handler"
	"github.com/zalhui/URLShortener/internal/delivery/http/middleware"
	"github.com/zalhui/URLShortener/internal/logger"
	"github.com/zalhui/URLShortener/internal/repository"
	"github.com/zalhui/URLShortener/internal/service"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second
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

	ctx := context.Background()
	dbCfg := dbconfig.Load()

	var repo repository.URLRepository
	var db *database.DB
	if dbCfg.Enabled() {
		var err error
		db, err = database.NewDB(ctx, dbCfg)
		if err != nil {
			logger.Sugar.Fatalw("failed to create database", "error", err)
		}
		defer db.Close()

		if err := db.InitSchema(ctx); err != nil {
			logger.Sugar.Fatalw("failed to initialize schema", "error", err)
		}

		repo = repository.NewPostgresRepository(db)
	} else if cfg.Filename != "" {
		var err error
		repo, err = repository.NewFileRepository(cfg.Filename)
		if err != nil {
			logger.Sugar.Fatalw("failed to create repository", "error", err)
		}
	} else {
		repo = repository.NewMemoryRepository()
	}
	defer repo.Close()

	shortenerService := service.NewShortenerService(repo, cfg.BaseURL, db)
	shortener := handler.NewShortenHandler(shortenerService)

	r := chi.NewRouter()

	r.Use(middleware.ContextMiddleware)
	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.DecompressMiddleware)
	r.Use(middleware.CompressMiddleware)
	r.Get("/ping", shortener.PingHandler)
	r.Mount("/", shortener.URLRouter())

	logger.Sugar.Infow(
		"Starting server",
		"address", cfg.ServerAddr,
	)

	server := &http.Server{
		Addr:              cfg.ServerAddr,
		Handler:           r,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
