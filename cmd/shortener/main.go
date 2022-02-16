package main

import (
	"net/http"
)

func main() {
	http.HandleFunc("/", ShortenerHandler)
	http.ListenAndServe(":8080", nil)
}
