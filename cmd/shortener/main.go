package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/zalhui/URLShortener/internal/config"
	"github.com/zalhui/URLShortener/internal/handler"
)

func main() {
	cfg := config.NewConfig()
	cfg.ParseFlags()
	port := strings.Split(cfg.ServerAddr, ":")[1]

	shortener := handler.NewURLShortener(cfg.BaseURL)

	log.Printf("Server started on %s", cfg.ServerAddr)

	err := http.ListenAndServe(":"+port, shortener.URLRouter())
	if err != nil {
		log.Fatal(err)
	}
}
