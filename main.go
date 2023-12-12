package OZON_Test_Task

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/wackonline/goway"
	_ "github.com/wackonline/goway"
	"log"
	"math/rand"
	"net/http"
	"strings"
)

var storage Storage

// Storage интерфейс определяет методы для работы с хранилищем
type Storage interface {
	shortenUrl(longUrl string) (string, error)
	getOriginalUrl(shortUrl string) (string, error)
}

// InMemoryStorage реализует Storage интерфейс для in-memory хранилища
type InMemoryStorage struct {
	Urls map[string]string
}

// PostgresStorage реализует Storage интерфейс для PostgreSQL хранилища
type PostgresStorage struct {
	Db *sql.DB
}

// newInMemoryStorage создает новый экземпляр InMemoryStorage
func newInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		Urls: make(map[string]string),
	}
}

// newPostgresStorage создает новый экземпляр PostgreSQLStorage
func newPostgresStorage() (*PostgresStorage, error) {
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
	return &PostgresStorage{Db: db}, nil
}

func (s *PostgresStorage) urlExists(longUrl string) bool {
	var count int
	err := s.Db.QueryRow("SELECT COUNT(*) FROM urls WHERE long_url = $1", longUrl).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

func (s *InMemoryStorage) urlExists(longUrl string) bool {
	_, exists := s.Urls[longUrl]
	return exists
}

// shortenUrl добавляет URL в хранилище и возвращает короткий URL
func (s *InMemoryStorage) shortenUrl(longUrl string) (string, error) {
	if _, exists := s.Urls[longUrl]; exists {
		return "", fmt.Errorf("this URL already exists in in-memory: %s", longUrl)
	}
	shortUrl := generateShortUrl()
	shortUrl = "https://" + shortUrl + ".com"
	s.Urls[longUrl] = shortUrl
	return shortUrl, nil
}

// getOriginalUrl возвращает оригинальный URL по короткому URL
func (s *InMemoryStorage) getOriginalUrl(shortUrl string) (string, error) {
	for key, value := range s.Urls {
		if value == "https://"+shortUrl+".com" {
			return key, nil
		}
	}
	return "", fmt.Errorf("short URL %s not found in Memory", shortUrl)
}

// shortenUrl добавляет URL в хранилище и возвращает короткий URL
func (s *PostgresStorage) shortenUrl(longUrl string) (string, error) {
	shortUrl := generateShortUrl()
	shortUrl = "https://" + shortUrl + ".com"
	if !s.urlExists(longUrl) {
		_, err := s.Db.Exec("INSERT INTO urls (long_url, short_url) VALUES ($1, $2)", longUrl, shortUrl)
		if err != nil {
			return "", err
		}
		return shortUrl, nil
	} else {
		return "", fmt.Errorf("this URL already exists in DataBase: %s", longUrl)
	}

}

// getOriginalUrl возвращает оригинальный URL по короткому URL
func (s *PostgresStorage) getOriginalUrl(shortUrl string) (string, error) {
	var longUrl string
	err := s.Db.QueryRow("SELECT long_url FROM urls WHERE short_url = $1", "https://"+shortUrl+".com").Scan(&longUrl)
	if err != nil {
		return "", fmt.Errorf("ShortUrl not found in PostgreSQL")
	}
	return longUrl, nil
}

func generateShortUrl() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
	shortUrl := ""
	for i := 0; i < 10; i++ {
		shortUrl += string(charset[rand.Intn(len(charset))])
	}
	return shortUrl
}

func shortenUrlHttp(w http.ResponseWriter, r *http.Request) {
	var Input struct {
		LongUrl string `json:"long_url"`
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&Input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	shortUrl, err := storage.shortenUrl(Input.LongUrl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]string{"short_url": shortUrl}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Println("Error:", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
}

func originalUrlHttp(w http.ResponseWriter, r *http.Request) {
	shortUrl := strings.TrimPrefix(r.URL.Path, "/short/")
	log.Println("Requested short URL:", shortUrl)

	longUrl, err := storage.getOriginalUrl(shortUrl)
	if err != nil {
		log.Println("Error:", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	log.Println("Redirecting to long URL:", longUrl)
	// Возвращаем JSON с long_url
	response := map[string]string{"long_url": longUrl}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Println("Error:", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
}

func main() {
	gm := goway.Bootstrap()
	storageType := gm.Configs.Get("storageType")
	log.Println("storageType: ", storageType)
	switch storageType {
	case "in-memory":
		storage = newInMemoryStorage()
	case "postgres":
		db, err := newPostgresStorage()
		if err != nil {
			log.Fatal(err)
		}
		storage = db
	default:
		log.Fatal("Unknown storage type")
	}
	gm.Post("/shorten", shortenUrlHttp)
	gm.Get("/short", originalUrlHttp)
	gm.Run()
}
