package main

import (
	_ "github.com/lib/pq"
	"testing"
)

func TestPostgreSQLStorage(t *testing.T) {

	storage, err := NewPostgreSQLStorage()
	if err != nil {
		t.Fatal(err)
	}
	defer storage.db.Close()
	storage.testShortenURL(t, storage)
}

func (s *PostgreSQLStorage) testShortenURL(t *testing.T, storage Storage) {
	longURL := "https://ozon_test_url.com"

	shortURL, err := storage.ShortenURL(longURL)
	if err != nil {
		t.Errorf("Error shortening URL: %v", err)
	}

	if shortURL == "" {
		t.Error("Shortened URL is empty")
	}

	if !s.urlExists(longURL) {
		_, err := s.db.Exec("INSERT INTO urls (long_url, short_url) VALUES ($1, $2)", longURL, shortURL)
		if err != nil {
			t.Error("The URL was not added to the database")
		}
	}
}
