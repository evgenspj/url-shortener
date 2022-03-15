package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evgenspj/url-shortener/internal/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testRequestArgs struct {
	t       *testing.T
	ts      *httptest.Server
	method  string
	path    string
	body    string
	headers map[string][]string
}

func testRequest(args testRequestArgs) *http.Response {
	req, err := http.NewRequest(
		args.method,
		args.ts.URL+args.path,
		bytes.NewBuffer([]byte(args.body)),
	)
	require.NoError(args.t, err)
	for header_key, header_val := range args.headers {
		req.Header.Set(header_key, strings.Join(header_val, "; "))
	}
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	require.NoError(args.t, err)
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
			r := NewRouter(&handler)
			ts := httptest.NewServer(r)
			defer ts.Close()
			requestMethod := tt.requestMethod
			if requestMethod == "" {
				requestMethod = http.MethodGet
			}
			reqArgs := testRequestArgs{
				t:      t,
				ts:     ts,
				method: requestMethod,
				path:   tt.request,
				body:   "",
			}
			resp := testRequest(reqArgs)
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
			r := NewRouter(&handler)
			ts := httptest.NewServer(r)
			defer ts.Close()
			requestMethod := tt.requestMethod
			if requestMethod == "" {
				requestMethod = http.MethodPost
			}
			reqArgs := testRequestArgs{
				t:      t,
				ts:     ts,
				method: requestMethod,
				path:   "/",
				body:   tt.requestBody,
			}
			resp := testRequest(reqArgs)
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

func TestShortenHandlerJSON(t *testing.T) {
	type want struct {
		code           int
		shortURLInBody bool
	}
	tests := []struct {
		name           string
		testURL        string
		requestMethod  string
		requestHeaders map[string][]string
		want           want
	}{
		{
			name:    "simple positive test",
			testURL: "http://example.com",
			want: want{
				code:           201,
				shortURLInBody: true,
			},
		},
		{
			name:    "invalid url",
			testURL: "lorem ipsum",
			want: want{
				code:           http.StatusBadRequest,
				shortURLInBody: false,
			},
		},
		{
			name:          "disallowed method",
			testURL:       "http://example.com",
			requestMethod: http.MethodPut,
			want: want{
				code:           http.StatusMethodNotAllowed,
				shortURLInBody: false,
			},
		},
		{
			name:    "wrong Content-Type header",
			testURL: "http://example.com",
			requestHeaders: map[string][]string{
				"Content-Type": {"text/plain"},
			},
			requestMethod: http.MethodPost,
			want: want{
				code:           http.StatusBadRequest,
				shortURLInBody: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Handler{storage: app.MyStorage{Val: make(map[string]string)}}
			r := NewRouter(&handler)
			ts := httptest.NewServer(r)
			defer ts.Close()
			requestMethod := tt.requestMethod
			if requestMethod == "" {
				requestMethod = http.MethodPost
			}
			requestHeaders := tt.requestHeaders
			if len(requestHeaders) == 0 {
				requestHeaders = map[string][]string{"Content-Type": {"application/json"}}
			}
			requestBody, _ := json.Marshal(ShortenHandlerJSONRequest{tt.testURL})
			reqArgs := testRequestArgs{
				t:       t,
				ts:      ts,
				method:  requestMethod,
				path:    "/api/shorten",
				body:    string(requestBody),
				headers: requestHeaders,
			}
			resp := testRequest(reqArgs)
			defer resp.Body.Close()

			require.Equal(t, tt.want.code, resp.StatusCode)
			if tt.want.shortURLInBody {
				respJSONStruct := ShortenHandlerJSONResponse{}
				json.NewDecoder(resp.Body).Decode(&respJSONStruct)
				assert.Contains(t, respJSONStruct.Result, baseServerURL)
			}
		})
	}
}
