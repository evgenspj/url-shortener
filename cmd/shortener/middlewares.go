package main

import (
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

type gzipBody struct {
	Body   io.ReadCloser
	Reader io.Reader
}

func (b gzipBody) Close() error {
	return b.Body.Close()
}

func (b gzipBody) Read(p []byte) (int, error) {
	return b.Reader.Read(p)
}

func gzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				io.WriteString(w, err.Error())
				return
			}
			defer gz.Close()
			r.Body = gzipBody{Body: r.Body, Reader: gz}
		}

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
			if err != nil {
				io.WriteString(w, err.Error())
				return
			}
			defer gz.Close()

			w = gzipWriter{ResponseWriter: w, Writer: gz}
			w.Header().Set("Content-Encoding", "gzip")
		}
		next.ServeHTTP(w, r)
	})
}

type Middleware func(http.Handler) http.Handler

func middlewareConveyor(h http.Handler, middlewares ...Middleware) http.Handler {
	for _, middleware := range middlewares {
		h = middleware(h)
	}
	return h
}

func userTokenCookieHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("user_token")
		var userToken string
		if err != nil {
			log.Println(err)
			userToken = genUserToken()
		} else {
			if (cookie == nil) || (!isValidToken(cookie.Value)) {
				userToken = genUserToken()
			} else {
				userToken = cookie.Value
			}
		}
		cookie = &http.Cookie{
			Name:  "user_token",
			Value: userToken,
		}
		http.SetCookie(w, cookie)
		next.ServeHTTP(w, r)
	})
}

var secretKey = []byte("some secret key")

func genUserToken() string {
	userID := genUserID()
	return genUserTokenByID(userID)
}

func genUserTokenByID(userID uint32) string {
	src := make([]byte, 4)
	binary.BigEndian.PutUint32(src, userID)
	h := hmac.New(sha256.New, secretKey)
	h.Write(src)
	dst := h.Sum(nil)
	result := append(src, dst...)
	return hex.EncodeToString(result)
}

func genUserID() uint32 {
	return rand.Uint32()
}

func isValidToken(token string) bool {
	data, err := hex.DecodeString(token)
	if err != nil {
		return false
	}
	h := hmac.New(sha256.New, secretKey)
	h.Write(data[:4])
	sign := h.Sum(nil)
	return hmac.Equal(sign, data[4:])
}
