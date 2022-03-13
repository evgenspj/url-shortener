package app

import "sync"

type Storage interface {
	SaveShort(short string, longUrl string)
	GetURLFromShort(short string) (string, bool)
}

type MyStorage struct {
	mu  sync.Mutex
	Val map[string]string
}

func (storage *MyStorage) SaveShort(short string, longURL string) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	storage.Val[short] = longURL
}

func (storage *MyStorage) GetURLFromShort(short string) (string, bool) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	longURL, exists := storage.Val[short]
	return longURL, exists
}
