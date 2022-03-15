package main

import (
	"log"
	"net/http"

	"github.com/caarlos0/env/v6"
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

const (
	defaultBaseServerURL = "http://localhost:8080"
	defaultServerAddress = "localhost:8080"
)

type EnvConfig struct {
	BaseServerURL string `env:"BASE_URL"`
	ServerAddress string `env:"SERVER_ADDRESS"`
}

func main() {
	var envCfg EnvConfig
	err := env.Parse(&envCfg)
	if err != nil {
		log.Fatal(err)
	}
	baseServerURL := envCfg.BaseServerURL
	if len(baseServerURL) == 0 {
		baseServerURL = defaultBaseServerURL
	}
	serverAddress := envCfg.ServerAddress
	if len(serverAddress) == 0 {
		serverAddress = defaultServerAddress
	}
	handler := Handler{
		storage:       app.MyStorage{Val: make(map[string]string)},
		baseServerURL: baseServerURL,
	}
	r := NewRouter(&handler)
	http.ListenAndServe(serverAddress, r)
}
