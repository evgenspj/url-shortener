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

func testRequest(
	t *testing.T,
	ts *httptest.Server,
	method,
	path string,
	reqBody string,
) *http.Response {
	req, err := http.NewRequest(method, ts.URL+path, bytes.NewBuffer([]byte(reqBody)))
	require.NoError(t, err)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

func TestGetFromShortHandler(t *testing.T) {
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
			handler := Handler{storage: app.MyStorage{Val: make(map[string]string)}}
			handler.storage.Val = tt.storedURLs
			r := NewRouter(handler)
			ts := httptest.NewServer(r)
			defer ts.Close()
			requestMethod := tt.requestMethod
			if requestMethod == "" {
				requestMethod = http.MethodGet
			}
			resp := testRequest(t, ts, requestMethod, tt.request, "")
			defer resp.Body.Close()

			require.Equal(t, tt.want.code, resp.StatusCode)
			assert.Equal(t, tt.want.locationHeader, resp.Header.Get("Location"))
		})
	}
}

func TestShortenHandler(t *testing.T) {
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
			handler := Handler{storage: app.MyStorage{Val: make(map[string]string)}}
			r := NewRouter(handler)
			ts := httptest.NewServer(r)
			defer ts.Close()
			requestMethod := tt.requestMethod
			if requestMethod == "" {
				requestMethod = http.MethodPost
			}
			resp := testRequest(t, ts, requestMethod, "/", tt.requestBody)
			defer resp.Body.Close()

			require.Equal(t, tt.want.code, resp.StatusCode)
			if tt.want.shortURLInBody {
				respBody, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(respBody), baseServerURL)
			}
		})
	}
}
