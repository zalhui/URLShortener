package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zalhui/URLShortener/auth/internal/config"
	dbconfig "github.com/zalhui/URLShortener/auth/internal/config/db"
	"github.com/zalhui/URLShortener/auth/internal/database"
	"github.com/zalhui/URLShortener/auth/internal/delivery/http/handler"
	httpmiddleware "github.com/zalhui/URLShortener/auth/internal/delivery/http/middleware"
	"github.com/zalhui/URLShortener/auth/internal/logger"
	"github.com/zalhui/URLShortener/auth/internal/repository"
	"github.com/zalhui/URLShortener/auth/internal/service"
	"github.com/zalhui/URLShortener/auth/internal/token"
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

	dbCfg := dbconfig.Load()
	if err := dbCfg.Validate(); err != nil {
		logger.Sugar.Fatalw("failed to load database config", "error", err)
	}

	ctx := context.Background()
	db, err := database.NewDB(ctx, dbCfg)
	if err != nil {
		logger.Sugar.Fatalw("failed to create database", "error", err)
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		logger.Sugar.Fatalw("failed to run migrations", "error", err)
	}

	tokenManager, err := token.NewManager(cfg.AccessTokenSecret, cfg.Issuer, cfg.AccessTokenTTL)
	if err != nil {
		logger.Sugar.Fatalw("failed to initialize token manager", "error", err)
	}

	repo := repository.NewPostgresRepository(db)
	authService := service.NewAuthService(repo, tokenManager, db, cfg.RefreshTokenTTL)
	authHandler := handler.NewAuthHandler(authService, cfg.CookieConfig())

	r := chi.NewRouter()
	r.Use(httpmiddleware.ContextMiddleware)
	r.Use(httpmiddleware.RecoverMiddleware)
	r.Use(httpmiddleware.LoggingMiddleware)
	r.Use(httpmiddleware.DecompressMiddleware)
	r.Use(httpmiddleware.CompressMiddleware)
	r.Get("/healthz", authHandler.HealthHandler)
	r.Mount("/", authHandler.Router())

	server := &http.Server{
		Addr:              cfg.ServerAddr,
		Handler:           r,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	logger.Sugar.Infow("starting auth service", "address", cfg.ServerAddr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
