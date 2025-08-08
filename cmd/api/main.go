package main

import (
	"fmt"
	"net/http"

	"github.com/ChiaYuChang/weathercock/internal/global"
	"github.com/ChiaYuChang/weathercock/internal/router"
	"github.com/ChiaYuChang/weathercock/internal/storage"
)

func main() {
	// Initialize the logger
	global.LoadConfigs(".env", "env", []string{"."})

	host := "localhost"
	port := 8080

	global.Logger.Info().
		Str("host", host).
		Int("port", port).
		Msg("Hello, World! This is a simple API server.")

	bind := fmt.Sprintf("%s:%d", host, port)
	mux := router.NewRouter(storage.Storage{})

	err := http.ListenAndServe(bind, mux)
	if err != nil {
		global.Logger.Fatal().
			Err(err).
			Str("bind", bind).
			Msg("Failed to start server")
	}
}
