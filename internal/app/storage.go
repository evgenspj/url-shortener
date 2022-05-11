package app

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"sync"
)

type Storage interface {
	SaveShort(short string, longURL string)
	GetURLFromShort(short string) (string, bool)
	AssociateUserIDWithShort(userID uint32, short string) error
	GetURLsByUserID(userID uint32) []string
}

type StructStorage struct {
	mu            sync.Mutex          `json:-`
	ShortToLong   map[string]string   `json:"short_to_long,omitempty"`
	UserIDToShort map[uint32][]string `json:"user_id_to_short,omitempty"`
}

type JSONFileStorage struct {
	Filename string
}

func (storage *StructStorage) SaveShort(short string, longURL string) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	storage.ShortToLong[short] = longURL
}

func (storage *StructStorage) GetURLFromShort(short string) (longURL string, exists bool) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	longURL, exists = storage.ShortToLong[short]
	return longURL, exists
}

func (storage *StructStorage) AssociateUserIDWithShort(userID uint32, short string) error {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	storage.UserIDToShort[userID] = append(storage.UserIDToShort[userID], short)
	return nil
}

func (storage *StructStorage) GetURLsByUserID(userID uint32) []string {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	shortURLS := storage.UserIDToShort[userID]
	return shortURLS
}

func (storage *JSONFileStorage) SaveShort(short string, longURL string) {
	file, err := os.OpenFile(storage.Filename, os.O_RDWR|os.O_CREATE|os.O_SYNC, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	if len(data) == 0 {
		data = []byte("{}")
	}
	savedURLs := StructStorage{}
	json.Unmarshal(data, &savedURLs)
	savedURLs.ShortToLong[short] = longURL
	updatedURLsJSON, err := json.MarshalIndent(savedURLs, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	file.Seek(0, 0)
	file.Write(updatedURLsJSON)
}

func (storage *JSONFileStorage) GetURLFromShort(short string) (longURL string, exists bool) {
	file, err := os.OpenFile(storage.Filename, os.O_RDONLY|os.O_CREATE|os.O_SYNC, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	savedURLs := StructStorage{}
	if len(data) == 0 {
		data = []byte("{}")
	}
	json.Unmarshal(data, &savedURLs)
	longURL, exists = savedURLs.ShortToLong[short]
	return longURL, exists
}

func (storage *JSONFileStorage) AssociateUserIDWithShort(userID uint32, short string) error {
	file, err := os.OpenFile(storage.Filename, os.O_RDONLY|os.O_CREATE|os.O_SYNC, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	savedURLs := StructStorage{}
	if len(data) == 0 {
		data = []byte("{}")
	}
	json.Unmarshal(data, &savedURLs)
	savedURLs.UserIDToShort[userID] = append(savedURLs.UserIDToShort[userID], short)
	updatedURLsJSON, err := json.MarshalIndent(savedURLs, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	file.Seek(0, 0)
	file.Write(updatedURLsJSON)
	return nil
}

func (storage *JSONFileStorage) GetURLsByUserID(userID uint32) []string {
	file, err := os.OpenFile(storage.Filename, os.O_RDONLY|os.O_CREATE|os.O_SYNC, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	savedURLs := StructStorage{}
	if len(data) == 0 {
		data = []byte("{}")
	}
	json.Unmarshal(data, &savedURLs)
	shortURLs := savedURLs.UserIDToShort[userID]
	return shortURLs
}
