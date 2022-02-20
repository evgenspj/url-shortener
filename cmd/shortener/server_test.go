package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evgenspj/url-shortener/internal/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortenerHandler_GetByShort(t *testing.T) {
	type want struct {
		code           int
		locationHeader string
	}
	tests := []struct {
		name          string
		request       string
		requestMethod string
		storedURLs    map[string]string
		want          want
	}{
		{
			name:    "simple positive test",
			request: "/loremid",
			storedURLs: map[string]string{
				"loremid": "http://example.com",
			},
			want: want{
				code:           http.StatusTemporaryRedirect,
				locationHeader: "http://example.com",
			},
		},
		{
			name:       "wrong id",
			request:    "/no-such-id",
			storedURLs: map[string]string{},
			want: want{
				code:           http.StatusNotFound,
				locationHeader: "",
			},
		},
		{
			name:       "wrong url",
			request:    "/someid/something/else",
			storedURLs: map[string]string{},
			want: want{
				code:           http.StatusNotFound,
				locationHeader: "",
			},
		},
		{
			name:          "disallowed method",
			request:       "/loremid",
			requestMethod: http.MethodPost,
			storedURLs: map[string]string{
				"loremid": "http://example.com",
			},
			want: want{
				code:           http.StatusMethodNotAllowed,
				locationHeader: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app.Storage = tt.storedURLs
			requestMethod := tt.requestMethod
			if requestMethod == "" {
				requestMethod = http.MethodGet
			}
			request := httptest.NewRequest(requestMethod, tt.request, nil)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(ShortenerHandler)
			h.ServeHTTP(w, request)
			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.want.code, resp.StatusCode)
			assert.Equal(t, tt.want.locationHeader, resp.Header.Get("Location"))
		})
	}
}

func TestShortenerHandler_ShortenURL(t *testing.T) {
	type want struct {
		code           int
		shortURLInBody bool
	}
	tests := []struct {
		name          string
		requestBody   string
		requestMethod string
		want          want
	}{
		{
			name:        "simple positive test",
			requestBody: "http://example.com",
			want: want{
				code:           201,
				shortURLInBody: true,
			},
		},
		{
			name:        "invalid url",
			requestBody: "lorem ipsum",
			want: want{
				code:           http.StatusBadRequest,
				shortURLInBody: false,
			},
		},
		{
			name:          "disallowed method",
			requestBody:   "http://example.com",
			requestMethod: http.MethodPut,
			want: want{
				code:           http.StatusMethodNotAllowed,
				shortURLInBody: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestMethod := tt.requestMethod
			if requestMethod == "" {
				requestMethod = http.MethodPost
			}
			body := []byte(tt.requestBody)
			request := httptest.NewRequest(requestMethod, "/", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			h := http.HandlerFunc(ShortenerHandler)
			h.ServeHTTP(w, request)
			resp := w.Result()

			assert.Equal(t, tt.want.code, tt.want.code)
			assert.Equal(t, tt.want.code, resp.StatusCode)
			if tt.want.shortURLInBody {
				defer resp.Body.Close()
				respBody, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(respBody), baseServerURL)
			}
		})
	}
}
