package app

import (
	"bytes"
	"encoding/base64"
	"strings"
)

func GenShort(url string) string {
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.RawURLEncoding, &buf)
	encoder.Write([]byte(url))
	encoded := buf.String()
	return strings.ToLower(encoded[:len(encoded)/2])
}
