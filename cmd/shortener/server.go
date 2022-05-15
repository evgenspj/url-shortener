package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/evgenspj/url-shortener/internal/app"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	storage       app.Storage
	baseServerURL string
}

type ShortenHandlerJSONRequest struct {
	URL string `json:"url"`
}

type ShortenHandlerJSONResponse struct {
	Result string `json:"result"`
}

type UserURLsResponseStruct struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"original_url"`
}

func (h *Handler) ShortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Can't read request body", http.StatusBadRequest)
		return
	}
	url, err := url.ParseRequestURI(string(data))
	if err != nil {
		http.Error(w, "Invalid url received", http.StatusBadRequest)
		return
	}

	longURL := url.String()
	short := app.GenShort(longURL)
	userID := getUserTokenFromWriter(w)
	h.storage.SaveShort(r.Context(), short, longURL, userID)
	w.WriteHeader(http.StatusCreated)
	shortURL := strings.Join([]string{h.baseServerURL, short}, "/")
	w.Write([]byte(shortURL))
}

func (h *Handler) GetFromShortHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET requests are allowed", http.StatusMethodNotAllowed)
		return
	}
	short := chi.URLParam(r, "ID")
	longURL, exists := h.storage.GetURLFromShort(r.Context(), short)
	if !exists {
		http.Error(w, "No such short url", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, longURL, http.StatusTemporaryRedirect)

}

func (h *Handler) ShortenHandlerJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
		http.Error(w, "Bad Content-Type", http.StatusBadRequest)
		return
	}
	data := ShortenHandlerJSONRequest{}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	url, err := url.ParseRequestURI(data.URL)
	if err != nil {
		http.Error(w, "Invalid url received", http.StatusBadRequest)
		return
	}

	longURL := url.String()
	short := app.GenShort(longURL)
	userID := getUserTokenFromWriter(w)
	h.storage.SaveShort(r.Context(), short, longURL, userID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	shortURL := strings.Join([]string{h.baseServerURL, short}, "/")
	ret, _ := json.Marshal(ShortenHandlerJSONResponse{shortURL})
	w.Write(ret)
}

func (h *Handler) UserURLs(w http.ResponseWriter, r *http.Request) {
	userID := getUserTokenFromWriter(w)
	shortURLIDs := h.storage.GetURLsByUserID(r.Context(), userID)
	response := make([]UserURLsResponseStruct, 0)
	for _, shortURLId := range shortURLIDs {
		shortURL := strings.Join([]string{h.baseServerURL, shortURLId}, "/")
		longURL, _ := h.storage.GetURLFromShort(r.Context(), shortURLId)
		item := UserURLsResponseStruct{shortURL, longURL}
		response = append(response, item)
	}

	encoder := json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/json")
	if len(response) > 0 {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
	encoder.Encode(response)
}

func (h *Handler) PingHandler(w http.ResponseWriter, r *http.Request) {
	if pgStorage, isPgStorage := h.storage.(*app.PostgresStorage); !isPgStorage {
		w.WriteHeader(http.StatusOK)
	} else {
		err := pgStorage.PingContext(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}
