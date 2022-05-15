package main

import (
	"net/http"
	"strings"

	"github.com/evgenspj/url-shortener/internal/app"
)

func getUserTokenFromWriter(w http.ResponseWriter) (userID uint32, exists bool) {
	cookiesToWrite, exists := w.Header()["Set-Cookie"]
	if exists {
		userToken := strings.Split(cookiesToWrite[0], "=")[1]
		userID = app.GetUserIDFromToken(userToken)
		return userID, exists
	}
	return userID, exists
}
