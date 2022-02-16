package app

import (
	"bytes"
	"encoding/base64"
	"strings"
)

var urlMap = make(map[string]string)

func GenShortUrl(url string) string {
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.RawURLEncoding, &buf)
	encoder.Write([]byte(url))
	encoded := buf.String()
	return strings.ToLower(encoded[:len(encoded)/2])
}

func SaveShortUrl(shortUrl string, longUrl string) {
	urlMap[shortUrl] = longUrl
}

func GetUrlFromShort(shortUrl string) (string, bool) {
	urlMap["abcd"] = "https://google.com"
	longUrl, exists := urlMap[shortUrl]
	return longUrl, exists
}
