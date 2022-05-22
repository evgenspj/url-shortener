package main

import (
	"net/http"
	"strings"

	"github.com/evgenspj/url-shortener/internal/app"
)

func getUserTokenFromWriter(w http.ResponseWriter) uint32 {
	cookiesToWrite := w.Header()["Set-Cookie"]
	userToken := strings.Split(cookiesToWrite[0], "=")[1]
	userID := app.GetUserIDFromToken(userToken)
	return userID
}
