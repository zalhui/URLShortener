package main

import (
	"github.com/zalhui/URLShortener/internal/handler"
)

func main() {
	shortener := handler.NewURLShortener()

	err := shortener.StartServer()
	if err != nil {
		panic(err)
	}
}
