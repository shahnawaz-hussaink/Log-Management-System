// Package config manages configuration loading from environment variables
package config

import (
	"os"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
}

// Load reads and parses configuration fields
func Load() *Config {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "host=localhost user=postgres password=postgres dbname=office_files port=5432 sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "my-office-secret-key-12345"
	}

	return &Config{
		DatabaseURL: dbURL,
		JWTSecret:   jwtSecret,
	}
}
