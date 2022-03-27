package main

import (
	"compress/gzip"
	"io"
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
