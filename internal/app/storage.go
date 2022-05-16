package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"os"
	"sync"
)

type Storage interface {
	SaveShort(ctx context.Context, short string, longURL string, userID uint32)
	GetURLFromShort(ctx context.Context, short string) (string, bool)
	GetURLsByUserID(ctx context.Context, userID uint32) []string
	SaveShortMulti(ctx context.Context, shortToLong map[string]string, userID uint32)
}

type StructStorage struct {
	mu            sync.Mutex
	ShortToLong   map[string]string
	UserIDToShort map[uint32][]string
}

type JSONStructure struct {
	ShortToLong   map[string]string   `json:"short_to_long,omitempty"`
	UserIDToShort map[uint32][]string `json:"user_id_to_short,omitempty"`
}

type JSONFileStorage struct {
	Filename string
}

type PostgresStorage struct {
	DB *sql.DB
}

func (storage *StructStorage) SaveShort(ctx context.Context, short string, longURL string, userID uint32) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	storage.ShortToLong[short] = longURL
	userIDToShort, exists := storage.UserIDToShort[userID]
	if !exists {
		userIDToShort = make([]string, 0)
	}
	storage.UserIDToShort[userID] = append(userIDToShort, short)
}

func (storage *StructStorage) GetURLFromShort(ctx context.Context, short string) (longURL string, exists bool) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	longURL, exists = storage.ShortToLong[short]
	return longURL, exists
}

func (storage *StructStorage) GetURLsByUserID(ctx context.Context, userID uint32) []string {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	shortURLS := storage.UserIDToShort[userID]
	return shortURLS
}

func (storage *StructStorage) SaveShortMulti(ctx context.Context, shortToLong map[string]string, userID uint32) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	userIDToShort, exists := storage.UserIDToShort[userID]
	if !exists {
		userIDToShort = make([]string, 0)
	}
	for short, long := range shortToLong {
		storage.ShortToLong[short] = long
		storage.UserIDToShort[userID] = append(userIDToShort, short)
	}
}

func (storage *JSONFileStorage) SaveShort(ctx context.Context, short string, longURL string, userID uint32) {
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
	savedURLs := JSONStructure{}
	json.Unmarshal(data, &savedURLs)
	if savedURLs.ShortToLong == nil {
		savedURLs.ShortToLong = make(map[string]string)
	}
	savedURLs.ShortToLong[short] = longURL
	if savedURLs.UserIDToShort == nil {
		savedURLs.UserIDToShort = make(map[uint32][]string)
	}
	userIDToShort, exists := savedURLs.UserIDToShort[userID]
	if !exists {
		userIDToShort = make([]string, 0)
	}
	savedURLs.UserIDToShort[userID] = append(userIDToShort, short)
	updatedURLsJSON, err := json.MarshalIndent(savedURLs, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	file.Seek(0, 0)
	file.Write(updatedURLsJSON)
}

func (storage *JSONFileStorage) GetURLFromShort(ctx context.Context, short string) (longURL string, exists bool) {
	file, err := os.OpenFile(storage.Filename, os.O_RDONLY|os.O_CREATE|os.O_SYNC, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	savedURLs := JSONStructure{}
	if len(data) == 0 {
		data = []byte("{}")
	}
	json.Unmarshal(data, &savedURLs)
	longURL, exists = savedURLs.ShortToLong[short]
	return longURL, exists
}

func (storage *JSONFileStorage) GetURLsByUserID(ctx context.Context, userID uint32) []string {
	file, err := os.OpenFile(storage.Filename, os.O_RDONLY|os.O_CREATE|os.O_SYNC, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	savedURLs := JSONStructure{}
	if len(data) == 0 {
		data = []byte("{}")
	}
	json.Unmarshal(data, &savedURLs)
	shortURLs := savedURLs.UserIDToShort[userID]
	return shortURLs
}

func (storage *JSONFileStorage) SaveShortMulti(ctx context.Context, shortToLong map[string]string, userID uint32) {
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
	savedURLs := JSONStructure{}
	json.Unmarshal(data, &savedURLs)
	if savedURLs.ShortToLong == nil {
		savedURLs.ShortToLong = make(map[string]string)
	}
	if savedURLs.UserIDToShort == nil {
		savedURLs.UserIDToShort = make(map[uint32][]string)
	}
	userIDToShort, exists := savedURLs.UserIDToShort[userID]
	if !exists {
		userIDToShort = make([]string, 0)
	}
	for short, long := range shortToLong {
		savedURLs.ShortToLong[short] = long
		userIDToShort = append(userIDToShort, short)
	}
	savedURLs.UserIDToShort[userID] = userIDToShort
	updatedURLsJSON, err := json.MarshalIndent(savedURLs, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	file.Seek(0, 0)
	file.Write(updatedURLsJSON)
}

func (storage *PostgresStorage) SaveShort(ctx context.Context, short string, longURL string, userID uint32) {
	_, err := storage.DB.ExecContext(
		ctx,
		"INSERT INTO short_urls (short_url, long_url, user_id) VALUES($1, $2, $3)",
		short,
		longURL,
		userID,
	)
	if err != nil {
		panic(err)
	}
}

func (storage *PostgresStorage) GetURLFromShort(ctx context.Context, short string) (longURL string, exists bool) {
	row := storage.DB.QueryRowContext(
		ctx,
		"SELECT long_url FROM short_urls WHERE short_url = $1",
		short,
	)
	err := row.Scan(&longURL)
	if err != nil {
		return "", false
	}
	return longURL, true
}

func (storage *PostgresStorage) GetURLsByUserID(ctx context.Context, userID uint32) []string {
	rows, err := storage.DB.QueryContext(
		ctx,
		"SELECT short_url FROM short_urls WHERE user_id = $1",
		userID,
	)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	shorts := []string{}
	for rows.Next() {
		var short string
		err = rows.Scan(&short)
		if err != nil {
			panic(err)
		}
		shorts = append(shorts, short)
	}
	err = rows.Err()
	if err != nil {
		panic(err)
	}
	return shorts
}

func (storage *PostgresStorage) PingContext(ctx context.Context) error {
	return storage.DB.PingContext(ctx)
}

func (storage *PostgresStorage) SaveShortMulti(ctx context.Context, shortToLong map[string]string, userID uint32) {
	tx, err := storage.DB.Begin()
	if err != nil {
		panic(err)
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO short_urls(short_url, long_url, user_id) VALUES($1, $2, $3)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()
	for short, long := range shortToLong {
		if _, err := stmt.ExecContext(ctx, short, long, userID); err != nil {
			panic(err)
		}
	}
	tx.Commit()
}

func (storage *PostgresStorage) Init(ctx context.Context) error {
	_, err := storage.DB.ExecContext(
		ctx,
		"CREATE TABLE IF NOT EXISTS short_urls (short_url CHAR(32) NOT NULL, long_url TEXT NOT NULL, user_id BIGINT)",
	)
	return err
}
