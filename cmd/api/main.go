package main

import (
	"log"
	"net/http"

	"github.com/ChiaYuChang/weathercock/internal/router"
)

func main() {
	log.Println("Hello, World! This is a simple API server.")

	bind := "localhost:8080"
	mux := router.NewRouter()

	log.Printf("Starting server on %s\n", bind)
	http.ListenAndServe(bind, mux)
}
