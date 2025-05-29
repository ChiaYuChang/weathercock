package router

import "net/http"

func NewRouter() *http.ServeMux {
	mux := http.NewServeMux()

	// file server
	mux.Handle("/", http.FileServer(http.Dir("./static")))

	return mux
}
