package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/evgenspj/url-shortener/internal/app"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	storage       app.MyStorage
	baseServerURL string
}

type ShortenHandlerJSONRequest struct {
	Url string `json:"url"`
}

type ShortenHandlerJSONResponse struct {
	Result string `json:"result"`
}

func (h *Handler) ShortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("Can't read request body")
		http.Error(w, "Can't read request body", http.StatusBadRequest)
		return
	}
	url, err := url.ParseRequestURI(string(data))
	if err != nil {
		log.Println("Invalid url received")
		http.Error(w, "Invalid url received", http.StatusBadRequest)
		return
	}

	longURL := url.String()
	short := app.GenShort(longURL)
	h.storage.SaveShort(short, longURL)
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
	longURL, exists := h.storage.GetURLFromShort(short)
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
		log.Println("Bad Content-Type")
		http.Error(w, "Bad Content-Type", http.StatusBadRequest)
		return
	}
	data := ShortenHandlerJSONRequest{}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	url, err := url.ParseRequestURI(data.Url)
	if err != nil {
		log.Println("Invalid url received")
		http.Error(w, "Invalid url received", http.StatusBadRequest)
		return
	}

	longURL := url.String()
	short := app.GenShort(longURL)
	h.storage.SaveShort(short, longURL)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	shortURL := strings.Join([]string{h.baseServerURL, short}, "/")
	ret, _ := json.Marshal(ShortenHandlerJSONResponse{shortURL})
	w.Write(ret)
}
