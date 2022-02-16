package main

import (
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/evgenspj/url-shortener/internal/app"
)

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

		longUrl := url.String()
		shortUrl := app.GenShortUrl(longUrl)
		app.SaveShortUrl(shortUrl, longUrl)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(shortUrl))

	default:
		if r.Method != http.MethodGet {
			http.Error(w, "Only GET requests are allowed", http.StatusMethodNotAllowed)
			return
		}
		shortUrl := r.URL.Path[1:]
		longUrl, exists := app.GetUrlFromShort(shortUrl)
		if !exists {
			http.Error(w, "No such short url", http.StatusNotFound)
			return
		}
		http.Redirect(w, r, longUrl, http.StatusTemporaryRedirect)
		return
	}
}
