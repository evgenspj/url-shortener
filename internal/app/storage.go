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
}

type StructStorage struct {
	mu  sync.Mutex
	Val map[string]string
}

type JSONFileStorage struct {
	Filename string
}

func (storage *StructStorage) SaveShort(short string, longURL string) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	storage.Val[short] = longURL
}

func (storage *StructStorage) GetURLFromShort(short string) (longURL string, exists bool) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	longURL, exists = storage.Val[short]
	return longURL, exists
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
	savedURLs := make(map[string]string)
	if len(data) == 0 {
		data = []byte("{}")
	}
	json.Unmarshal(data, &savedURLs)
	savedURLs[short] = longURL
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
	savedURLs := make(map[string]string)
	if len(data) == 0 {
		data = []byte("{}")
	}
	json.Unmarshal(data, &savedURLs)
	longURL, exists = savedURLs[short]
	return longURL, exists
}
