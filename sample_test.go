package OZON_Test_Task

import (
	"database/sql"
	_ "github.com/lib/pq"
	"log"
	"testing"
)

func TestPostgresStorage(t *testing.T) {
	storage, err := newPostgresStorage()
	if err != nil {
		t.Fatal(err)
	}
	defer func(Db *sql.DB) {
		err := Db.Close()
		if err != nil {
			log.Println("Error:", err)
		}
	}(storage.Db)
	storage.TestShortenUrl(t, storage)
}

func (s *PostgresStorage) TestShortenUrl(t *testing.T, storage Storage) {
	longUrl := "https://ozon_test_url.com"
	shortUrl, err := storage.shortenUrl(longUrl)
	if err != nil {
		t.Errorf("Error shortening URL: %v", err)
	}
	if shortUrl == "" {
		t.Error("Shortened URL is empty")
	}
	if !s.urlExists(longUrl) {
		_, err := s.Db.Exec("INSERT INTO urls (long_url, short_url) VALUES ($1, $2)", longUrl, shortUrl)
		if err != nil {
			t.Error("The URL was not added to the database")
		}
	}
}
