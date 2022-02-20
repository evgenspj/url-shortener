package app

import (
	"bytes"
	"encoding/base64"
	"strings"
)

var urlMap = make(map[string]string)

func GenShort(url string) string {
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.RawURLEncoding, &buf)
	encoder.Write([]byte(url))
	encoded := buf.String()
	return strings.ToLower(encoded[:len(encoded)/2])
}

func SaveShort(short string, longURL string) {
	urlMap[short] = longURL
}

func GetURLFromShort(short string) (string, bool) {
	longURL, exists := urlMap[short]
	return longURL, exists
}
