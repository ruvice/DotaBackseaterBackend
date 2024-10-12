package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/hello", basicHandler)
	http.ListenAndServe(":3000", r)
}

func basicHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello world2!"))
}
