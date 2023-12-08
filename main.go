package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
)

// Storage интерфейс определяет методы для работы с хранилищем
type Storage interface {
	ShortenURL(longURL string) (string, error)
	GetOriginalURL(shortURL string) (string, error)
}

// InMemoryStorage реализует Storage интерфейс для in-memory хранилища
type InMemoryStorage struct {
	urls map[string]string
}

// PostgreSQLStorage реализует Storage интерфейс для PostgreSQL хранилища
type PostgreSQLStorage struct {
	db *sql.DB
}

// NewInMemoryStorage создает новый экземпляр InMemoryStorage
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		urls: make(map[string]string),
	}
}

// NewPostgreSQLStorage создает новый экземпляр PostgreSQLStorage
func NewPostgreSQLStorage() (*PostgreSQLStorage, error) {
	db, err := sql.Open("postgres", "host=postgres user=user dbname=ozon_db port=5432 password=1234 sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	// Создаем таблицу, если ее нет
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS urls (
		id SERIAL PRIMARY KEY,
		long_url TEXT NOT NULL,
		short_url TEXT NOT NULL UNIQUE
	);`)
	if err != nil {
		log.Fatal(err)
	}
	return &PostgreSQLStorage{db: db}, nil
}

func (s *PostgreSQLStorage) urlExists(longURL string) bool {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM urls WHERE long_url = $1", longURL).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

func (s *InMemoryStorage) urlExists(longURL string) bool {
	_, exists := s.urls[longURL]
	return exists
}

// ShortenURL добавляет URL в хранилище и возвращает короткий URL
func (s *InMemoryStorage) ShortenURL(longURL string) (string, error) {

	if _, exists := s.urls[longURL]; exists {
		return "", fmt.Errorf("this URL already exists in in-memory: %s", longURL)
	}
	shortURL := generateShortURL()
	shortURL = "https://" + shortURL + ".com"
	s.urls[longURL] = shortURL
	return shortURL, nil
}

// GetOriginalURL возвращает оригинальный URL по короткому URL
func (s *InMemoryStorage) GetOriginalURL(shortURL string) (string, error) {

	for key, value := range s.urls {
		if value == "https://"+shortURL+".com" {
			return key, nil
		}
	}
	return "", fmt.Errorf("short URL %s not found in Memory", shortURL)
}

// ShortenURL добавляет URL в хранилище и возвращает короткий URL
func (s *PostgreSQLStorage) ShortenURL(longURL string) (string, error) {

	shortURL := generateShortURL()
	shortURL = "https://" + shortURL + ".com"
	if !s.urlExists(longURL) {
		_, err := s.db.Exec("INSERT INTO urls (long_url, short_url) VALUES ($1, $2)", longURL, shortURL)
		if err != nil {
			return "", err
		}
		return shortURL, nil
	} else {
		return "", fmt.Errorf("his URL already exists in DataBase: %s", longURL)
	}

}

// GetOriginalURL возвращает оригинальный URL по короткому URL
func (s *PostgreSQLStorage) GetOriginalURL(shortURL string) (string, error) {
	var longURL string
	err := s.db.QueryRow("SELECT long_url FROM urls WHERE short_url = $1", "https://"+shortURL+".com").Scan(&longURL)
	if err != nil {
		return "", fmt.Errorf("ShortURL not found in PostgreSQL")
	}
	return longURL, nil
}

func generateShortURL() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
	shortURL := ""
	for i := 0; i < 10; i++ {
		shortURL += string(charset[rand.Intn(len(charset))])
	}
	return shortURL
}

func shortenURL(w http.ResponseWriter, r *http.Request) {
	var input struct {
		LongURL string `json:"long_url"`
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	shortURL, err := storage.ShortenURL(input.LongURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]string{"short_url": shortURL}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func originalURL(w http.ResponseWriter, r *http.Request) {
	shortURL := strings.TrimPrefix(r.URL.Path, "/short/")
	log.Println("Requested short URL:", shortURL)

	longURL, err := storage.GetOriginalURL(shortURL)
	if err != nil {
		log.Println("Error:", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	log.Println("Redirecting to long URL:", longURL)
	// Возвращаем JSON с long_url
	response := map[string]string{"long_url": longURL}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

var storage Storage

func main() {
	storageType := os.Getenv("STORAGE_TYPE")
	log.Println("storageType:", storageType)
	switch storageType {
	case "in-memory":
		storage = NewInMemoryStorage()
	case "postgres":
		db, err := NewPostgreSQLStorage()
		if err != nil {
			log.Fatal(err)
		}
		storage = db
	default:
		log.Fatal("Unknown storage type")
	}

	http.HandleFunc("/shorten", shortenURL)
	http.HandleFunc("/short/", originalURL)

	port := "8080"
	addr := fmt.Sprintf(":%s", port)

	log.Println("Server is running on", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
