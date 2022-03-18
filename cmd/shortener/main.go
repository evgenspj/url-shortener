package main

import (
	"flag"
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
	defaultBaseURL       = "http://localhost:8080"
	defaultServerAddress = "localhost:8080"
)

type EnvConfig struct {
	BaseURL         string `env:"BASE_URL"`
	ServerAddress   string `env:"SERVER_ADDRESS"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
}

func main() {
	// comand line args
	argServerAddress := flag.String("a", "", "usage")
	argBaseURL := flag.String("b", "", "usage")
	argFileStoragePath := flag.String("f", "", "usage")
	flag.Parse()

	// environment variables
	var envCfg EnvConfig
	err := env.Parse(&envCfg)
	if err != nil {
		log.Fatal(err)
	}

	var serverAddress string
	switch {
	case len(*argServerAddress) > 0:
		serverAddress = *argServerAddress
	case len(envCfg.ServerAddress) > 0:
		serverAddress = envCfg.ServerAddress
	default:
		serverAddress = defaultServerAddress
	}

	var baseURL string
	switch {
	case len(*argBaseURL) > 0:
		baseURL = *argBaseURL
	case len(envCfg.BaseURL) > 0:
		baseURL = envCfg.BaseURL
	default:
		baseURL = defaultBaseURL
	}

	var storage app.Storage
	switch {
	case len(*argFileStoragePath) > 0:
		storage = &app.JSONFileStorage{Filename: *argFileStoragePath}
	case len(envCfg.FileStoragePath) > 0:
		storage = &app.JSONFileStorage{Filename: envCfg.FileStoragePath}
	default:
		storage = &app.StructStorage{Val: make(map[string]string)}
	}

	handler := Handler{
		storage:       storage,
		baseServerURL: baseURL,
	}
	r := NewRouter(&handler)
	http.ListenAndServe(serverAddress, r)
}
