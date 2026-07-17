// Package config manages configuration loading from environment variables
package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL  string
	JWTSecret    string
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
}

// LoadEnv reads .env files manually if they exist
func LoadEnv() {
	// Try loading from .env in current directory, or parent directories (handles backend root and cmd/api runs)
	files := []string{".env", "../.env", "../../.env"}
	for _, filename := range files {
		file, err := os.Open(filename)
		if err == nil {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) != 2 {
					continue
				}
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				// Remove surrounding quotes
				if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
					val = val[1 : len(val)-1]
				}
				os.Setenv(key, val)
			}
			break
		}
	}
}

// Load reads and parses configuration fields
func Load() *Config {
	LoadEnv()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "host=localhost user=postgres password=postgres dbname=office_files port=5432 sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "my-office-secret-key-12345"
	}

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "587"
	}

	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	smtpFrom := os.Getenv("SMTP_FROM")
	if smtpFrom == "" {
		smtpFrom = "no-reply@school.edu"
	}

	return &Config{
		DatabaseURL:  dbURL,
		JWTSecret:    jwtSecret,
		SMTPHost:     smtpHost,
		SMTPPort:     smtpPort,
		SMTPUser:     smtpUser,
		SMTPPassword: smtpPass,
		SMTPFrom:     smtpFrom,
	}
}
