package main

import (
	"net/http"

	"github.com/evgenspj/url-shortener/internal/app"
	"github.com/go-chi/chi/v5"
)

func NewRouter(handler *Handler) chi.Router {
	r := chi.NewRouter()
	r.Post("/", handler.ShortenHandler)
	r.Post("/api/shorten", handler.ShortenHandlerJSON)
	r.Get("/{ID}", handler.GetFromShortHandler)
	return r
}

func main() {
	handler := Handler{storage: app.MyStorage{Val: make(map[string]string)}}
	r := NewRouter(&handler)
	http.ListenAndServe(":8080", r)
}
