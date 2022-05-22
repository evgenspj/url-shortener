package main

import (
	"encoding/json"
	"errors"
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

type ShortenBatchHandlerJSONRequest []struct {
	CorrelationID string `json:"correlation_id"`
	OrginalURL    string `json:"original_url"`
}

type ShortenBatchHandlerJSONResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
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
	err = h.storage.SaveShort(r.Context(), short, longURL, userID)
	var duplicateErr *app.DuplicateError
	var respStatus int
	if err != nil {
		if errors.As(err, &duplicateErr) {
			respStatus = http.StatusConflict
		} else {
			panic(err)
		}
	} else {
		respStatus = http.StatusCreated
	}
	shortURL := strings.Join([]string{h.baseServerURL, short}, "/")
	w.WriteHeader(respStatus)
	_, err = w.Write([]byte(shortURL))
	if err != nil {
		panic(err)
	}
}

func (h *Handler) GetFromShortHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET requests are allowed", http.StatusMethodNotAllowed)
		return
	}
	short := chi.URLParam(r, "ID")
	urlData, exists := h.storage.GetURLFromShort(r.Context(), short)
	if !exists {
		http.Error(w, "No such short url", http.StatusNotFound)
		return
	}
	if urlData.Deleted {
		http.Error(w, "Url was deleted", http.StatusGone)
		return
	}
	http.Redirect(w, r, urlData.Long, http.StatusTemporaryRedirect)
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
	err = h.storage.SaveShort(r.Context(), short, longURL, userID)
	var duplicateErr *app.DuplicateError
	var respStatus int
	if err != nil {
		if errors.As(err, &duplicateErr) {
			respStatus = http.StatusConflict
		} else {
			panic(err)
		}
	} else {
		respStatus = http.StatusCreated
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(respStatus)
	shortURL := strings.Join([]string{h.baseServerURL, short}, "/")
	ret, _ := json.Marshal(ShortenHandlerJSONResponse{shortURL})
	_, err = w.Write(ret)
	if err != nil {
		panic(err)
	}
}

func (h *Handler) UserURLs(w http.ResponseWriter, r *http.Request) {
	userID := getUserTokenFromWriter(w)
	shortURLIDs := h.storage.GetURLsByUserID(r.Context(), userID)
	response := make([]UserURLsResponseStruct, 0)
	for _, shortURLId := range shortURLIDs {
		shortURL := strings.Join([]string{h.baseServerURL, shortURLId}, "/")
		urlData, _ := h.storage.GetURLFromShort(r.Context(), shortURLId)
		item := UserURLsResponseStruct{shortURL, urlData.Long}
		response = append(response, item)
	}

	encoder := json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/json")
	if len(response) > 0 {
		w.WriteHeader(http.StatusOK)
		err := encoder.Encode(response)
		if err != nil {
			panic(err)
		}
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
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

func (h *Handler) ShortenBatchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
		http.Error(w, "Bad Content-Type", http.StatusBadRequest)
		return
	}
	data := ShortenBatchHandlerJSONRequest{}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userID := getUserTokenFromWriter(w)
	correlationIDtoShort := make(map[string]string)
	shortToLong := make(map[string]string)
	for _, item := range data {
		longURL, err := url.ParseRequestURI(item.OrginalURL)
		if err != nil {
			http.Error(w, "Invalid url received", http.StatusBadRequest)
			return
		}
		shortURL := app.GenShort(longURL.String())
		correlationIDtoShort[item.CorrelationID] = shortURL
		shortToLong[shortURL] = longURL.String()
	}

	err := h.storage.SaveShortMulti(r.Context(), shortToLong, userID)
	var duplicateErr *app.DuplicateError
	var respStatus int
	if err != nil {
		if errors.As(err, &duplicateErr) {
			respStatus = http.StatusConflict
		} else {
			panic(err)
		}
	} else {
		respStatus = http.StatusCreated
	}
	respData := []ShortenBatchHandlerJSONResponse{}
	for correlationID, shortURL := range correlationIDtoShort {
		respData = append(
			respData,
			ShortenBatchHandlerJSONResponse{
				CorrelationID: correlationID,
				ShortURL:      strings.Join([]string{h.baseServerURL, shortURL}, "/"),
			},
		)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(respStatus)
	ret, _ := json.MarshalIndent(respData, "", "    ")
	_, err = w.Write(ret)
	if err != nil {
		panic(err)
	}
}

func (h *Handler) DeleteUserURLs(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
		http.Error(w, "Bad Content-Type", http.StatusBadRequest)
		return
	}
	urlsToDelete := []string{}

	if err := json.NewDecoder(r.Body).Decode(&urlsToDelete); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userID := getUserTokenFromWriter(w)
	go h.storage.DeleteUserURLs(userID, urlsToDelete)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
}
