package main

import (
	"bytes"
	"context"
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
	t         *testing.T
	ts        *httptest.Server
	method    string
	path      string
	body      string
	headers   map[string][]string
	userToken string
}

func testRequest(args testRequestArgs) *http.Response {
	req, err := http.NewRequest(
		args.method,
		args.ts.URL+args.path,
		bytes.NewBuffer([]byte(args.body)),
	)
	require.NoError(args.t, err)
	for headerKey, headerVal := range args.headers {
		req.Header.Set(headerKey, strings.Join(headerVal, "; "))
	}
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	if len(args.userToken) > 0 {
		cookie := &http.Cookie{
			Name:  "user_token",
			Value: args.userToken,
		}
		req.AddCookie(cookie)
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
			handler := Handler{
				storage: &app.StructStorage{
					ShortToLong:   make(map[string]string),
					UserIDToShort: make(map[uint32][]string),
				},
				baseServerURL: defaultBaseURL,
			}
			for short, long := range tt.storedURLs {
				handler.storage.SaveShort(context.Background(), short, long, genUserID())
			}
			r := NewRouter(&handler)
			ts := httptest.NewServer(middlewareConveyor(r, gzipHandle, userTokenCookieHandle))
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
		urlsInDB      []string
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
		{
			name:          "duplicate url",
			requestBody:   "http://duplicate.com",
			requestMethod: http.MethodPost,
			urlsInDB:      []string{"http://duplicate.com"},
			want: want{
				code:           http.StatusConflict,
				shortURLInBody: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortToLong := make(map[string]string)
			for _, long := range tt.urlsInDB {
				short := app.GenShort(long)
				shortToLong[short] = long
			}
			handler := Handler{
				storage: &app.StructStorage{
					ShortToLong:   shortToLong,
					UserIDToShort: make(map[uint32][]string),
				},
				baseServerURL: defaultBaseURL,
			}
			r := NewRouter(&handler)
			ts := httptest.NewServer(middlewareConveyor(r, gzipHandle, userTokenCookieHandle))
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
				assert.Contains(t, string(respBody), defaultBaseURL)
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
		urlsInDB       []string
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
		{
			name:          "duplicate url",
			testURL:       "http://duplicate.com",
			requestMethod: http.MethodPost,
			urlsInDB:      []string{"http://duplicate.com"},
			want: want{
				code:           http.StatusConflict,
				shortURLInBody: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortToLong := make(map[string]string)
			for _, long := range tt.urlsInDB {
				short := app.GenShort(long)
				shortToLong[short] = long
			}
			handler := Handler{
				storage: &app.StructStorage{
					ShortToLong:   shortToLong,
					UserIDToShort: make(map[uint32][]string),
				},
				baseServerURL: defaultBaseURL,
			}
			r := NewRouter(&handler)
			ts := httptest.NewServer(middlewareConveyor(r, gzipHandle, userTokenCookieHandle))
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
				assert.Contains(t, respJSONStruct.Result, defaultBaseURL)
			}
		})
	}
}

func TestUserURLs(t *testing.T) {
	type want struct {
		code int
		urls map[string]string
	}
	userID := genUserID()
	userToken := genUserTokenByID(userID)
	wrongToken := "loremipsum"
	longURL := "http://yandex.ru"
	shortURLId := app.GenShort(longURL)
	tests := []struct {
		name          string
		userID        uint32
		userToken     string
		shortToLong   map[string]string
		userIDToShort map[uint32][]string
		want          want
	}{
		{
			name:          "simple positive test",
			userID:        userID,
			shortToLong:   map[string]string{shortURLId: longURL},
			userIDToShort: map[uint32][]string{userID: {shortURLId}},
			userToken:     userToken,
			want: want{
				code: 200,
				urls: map[string]string{shortURLId: longURL},
			},
		},
		{
			name:      "wrong token",
			userID:    userID,
			userToken: wrongToken,
			want: want{
				code: 204,
			},
		},
		{
			name:      "no data",
			userID:    userID,
			userToken: userToken,
			want: want{
				code: 204,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Handler{
				storage: &app.StructStorage{
					ShortToLong:   tt.shortToLong,
					UserIDToShort: tt.userIDToShort,
				},
				baseServerURL: defaultBaseURL,
			}
			r := NewRouter(&handler)
			ts := httptest.NewServer(middlewareConveyor(r, gzipHandle, userTokenCookieHandle))
			defer ts.Close()
			requestMethod := http.MethodGet
			reqArgs := testRequestArgs{
				t:         t,
				ts:        ts,
				method:    requestMethod,
				path:      "/api/user/urls",
				userToken: tt.userToken,
			}
			resp := testRequest(reqArgs)
			defer resp.Body.Close()

			require.Equal(t, tt.want.code, resp.StatusCode)
			if len(tt.want.urls) > 0 {
				respSchema := make([]UserURLsResponseStruct, 0)
				json.NewDecoder(resp.Body).Decode(&respSchema)
				assert.Equal(t, 1, len(respSchema))
				shortURL := strings.Join([]string{handler.baseServerURL, shortURLId}, "/")
				assert.Equal(t, shortURL, respSchema[0].ShortURL)
				assert.Equal(t, longURL, respSchema[0].LongURL)
			}
		})
	}
}

func TestShortenBatchHandler(t *testing.T) {
	type want struct {
		code     int
		response []ShortenBatchHandlerJSONResponse
	}
	tests := []struct {
		name        string
		requestData ShortenBatchHandlerJSONRequest
		urlsInDB    []string
		want        want
	}{
		{
			name: "simple positive test",
			requestData: ShortenBatchHandlerJSONRequest{
				{CorrelationID: "some id", OrginalURL: "https://yandex.ru"},
				{CorrelationID: "other id", OrginalURL: "https://google.com"},
			},
			want: want{
				code: 201,
			},
		},
		{
			name: "invalid url",
			requestData: ShortenBatchHandlerJSONRequest{
				{CorrelationID: "id", OrginalURL: "invalidurl"},
			},
			want: want{
				code: http.StatusBadRequest,
			},
		},
		{
			name: "duplicate url",
			requestData: ShortenBatchHandlerJSONRequest{
				{CorrelationID: "id", OrginalURL: "http://duplicate.com"},
			},
			urlsInDB: []string{"http://duplicate.com"},
			want: want{
				code: http.StatusConflict,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortToLong := make(map[string]string)
			for _, long := range tt.urlsInDB {
				short := app.GenShort(long)
				shortToLong[short] = long
			}
			handler := Handler{
				storage: &app.StructStorage{
					ShortToLong:   shortToLong,
					UserIDToShort: make(map[uint32][]string),
				},
				baseServerURL: defaultBaseURL,
			}
			r := NewRouter(&handler)
			ts := httptest.NewServer(middlewareConveyor(r, gzipHandle, userTokenCookieHandle))
			defer ts.Close()
			requestBody, _ := json.Marshal(tt.requestData)
			reqArgs := testRequestArgs{
				t:       t,
				ts:      ts,
				method:  http.MethodPost,
				path:    "/api/shorten/batch",
				body:    string(requestBody),
				headers: map[string][]string{"Content-Type": {"application/json"}},
			}
			resp := testRequest(reqArgs)
			defer resp.Body.Close()

			require.Equal(t, tt.want.code, resp.StatusCode)
			if tt.want.code == http.StatusOK {
				respBody, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				bodyParsed := make([]ShortenBatchHandlerJSONResponse, 0)
				err = json.Unmarshal(respBody, &bodyParsed)
				require.NoError(t, err)
				expected := make([]ShortenBatchHandlerJSONResponse, 0)
				for _, item := range tt.requestData {
					shortURL := strings.Join([]string{handler.baseServerURL, app.GenShort(item.OrginalURL)}, "/")
					expected = append(
						expected,
						ShortenBatchHandlerJSONResponse{
							CorrelationID: item.CorrelationID,
							ShortURL:      shortURL,
						},
					)
				}
				require.Equal(t, bodyParsed, expected)
			}
		})
	}
}
