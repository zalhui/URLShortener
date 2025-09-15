package main

import (
	"log"
	"net/http"

	"github.com/zalhui/URLShortener/internal/handler"
)

func main() {
	shortener := handler.NewURLShortener()

	log.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", shortener.URLRouter())
	if err != nil {
		log.Fatal(err)
	}
}
