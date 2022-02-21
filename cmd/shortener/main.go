package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter() chi.Router {
	r := chi.NewRouter()
	r.Post("/", ShortenHandler)
	r.Get("/{ID}", GetFromShortHandler)
	return r
}

func main() {
	r := NewRouter()
	http.ListenAndServe(":8080", r)
}
