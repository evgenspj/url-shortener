package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/evgenspj/url-shortener/internal/app"
)

const baseServerURL = "http://localhost:8080"

func ShortenerHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch path {
	case "/":
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()
		data, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println("Can't read request body")
			http.Error(w, "Can't read request body", http.StatusBadRequest)
		}
		url, err := url.ParseRequestURI(string(data))
		if err != nil {
			log.Println("Invalid url received")
			http.Error(w, "Invalid url received", http.StatusBadRequest)
		}

		longURL := url.String()
		short := app.GenShort(longURL)
		app.SaveShort(short, longURL)
		w.WriteHeader(http.StatusCreated)
		shortURL := strings.Join([]string{baseServerURL, short}, "/")
		w.Write([]byte(shortURL))

	default:
		if r.Method != http.MethodGet {
			http.Error(w, "Only GET requests are allowed", http.StatusMethodNotAllowed)
			return
		}
		short := r.URL.Path[1:]
		longURL, exists := app.GetURLFromShort(short)
		if !exists {
			http.Error(w, "No such short url", http.StatusNotFound)
			return
		}
		http.Redirect(w, r, longURL, http.StatusTemporaryRedirect)
		return
	}
}
