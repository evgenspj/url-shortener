package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"net/http"

	"github.com/caarlos0/env/v6"
	"github.com/evgenspj/url-shortener/internal/app"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/stdlib"
)

func NewRouter(handler *Handler) chi.Router {
	r := chi.NewRouter()
	r.Post("/", handler.ShortenHandler)
	r.Post("/api/shorten", handler.ShortenHandlerJSON)
	r.Get("/api/user/urls", handler.UserURLs)
	r.Get("/{ID}", handler.GetFromShortHandler)
	r.Get("/ping", handler.PingHandler)
	r.Post("/api/shorten/batch", handler.ShortenBatchHandler)
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
	PostgresConStr  string `env:"DATABASE_DSN"`
}

func main() {
	// comand line args
	argServerAddress := flag.String("a", "", "usage")
	argBaseURL := flag.String("b", "", "usage")
	argFileStoragePath := flag.String("f", "", "usage")
	argPostgresConStr := flag.String("d", "", "usage")
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
	case len(*argPostgresConStr) > 0:
		db, err := sql.Open("pgx", *argPostgresConStr)
		if err != nil {
			panic(err)
		}
		defer db.Close()
		dbStorage := &app.PostgresStorage{DB: db}
		if err := dbStorage.Init(context.Background()); err != nil {
			panic(err)
		}
		storage = dbStorage
	case len(*argFileStoragePath) > 0:
		storage = &app.JSONFileStorage{Filename: *argFileStoragePath}
	case len(envCfg.PostgresConStr) > 0:
		db, err := sql.Open("pgx", envCfg.PostgresConStr)
		if err != nil {
			panic(err)
		}
		defer db.Close()
		dbStorage := &app.PostgresStorage{DB: db}
		if err := dbStorage.Init(context.Background()); err != nil {
			panic(err)
		}
		storage = dbStorage
	case len(envCfg.FileStoragePath) > 0:
		storage = &app.JSONFileStorage{Filename: envCfg.FileStoragePath}
	default:
		storage = &app.StructStorage{
			ShortToLong:   make(map[string]string),
			UserIDToShort: make(map[uint32][]string),
		}
	}

	handler := Handler{
		storage:       storage,
		baseServerURL: baseURL,
	}
	r := NewRouter(&handler)
	http.ListenAndServe(serverAddress, middlewareConveyor(r, gzipHandle, userTokenCookieHandle))
}
