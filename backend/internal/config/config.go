// Package config reads the server's settings from the environment.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

// Config is everything the server needs to start. Secrets arrive through the
// environment file and are never written into the repository.
type Config struct {
	DatabaseURL    string
	Port           int
	MigrationsPath string
	StoragePath    string
	Email          Email
}

// Email selects and configures the email adapter. An empty Host means no SMTP
// server is configured, and messages are written to the server log instead.
type Email struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Load reads the configuration from the process environment.
func Load() (Config, error) {
	return LoadFrom(os.Getenv)
}

// LoadFrom reads the configuration through getenv, and reports every problem
// it finds at once, so a new host is fixed in one pass.
func LoadFrom(getenv func(string) string) (Config, error) {
	var errs []error

	cfg := Config{
		DatabaseURL:    getenv("DATABASE_URL"),
		MigrationsPath: orDefault(getenv("MIGRATIONS_PATH"), "migrations"),
		StoragePath:    getenv("STORAGE_PATH"),
		Email: Email{
			Host:     getenv("SMTP_HOST"),
			Username: getenv("SMTP_USERNAME"),
			Password: getenv("SMTP_PASSWORD"),
			From:     getenv("EMAIL_FROM"),
		},
	}

	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if cfg.StoragePath == "" {
		errs = append(errs, errors.New("STORAGE_PATH is required"))
	}

	port, err := portFrom(getenv("PORT"), 8082)
	if err != nil {
		errs = append(errs, fmt.Errorf("PORT: %w", err))
	}
	cfg.Port = port

	smtpPort, err := portFrom(getenv("SMTP_PORT"), 587)
	if err != nil {
		errs = append(errs, fmt.Errorf("SMTP_PORT: %w", err))
	}
	cfg.Email.Port = smtpPort

	if cfg.Email.Host != "" && cfg.Email.From == "" {
		errs = append(errs, errors.New("EMAIL_FROM is required when SMTP_HOST is set"))
	}

	return cfg, errors.Join(errs...)
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func portFrom(value string, fallback int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("%q is not a port number", value)
	}
	return port, nil
}
