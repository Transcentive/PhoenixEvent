// Package config reads the server's settings from the environment.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config is everything the server needs to start. Secrets arrive through the
// environment file and are never written into the repository.
type Config struct {
	DatabaseURL    string
	Port           int
	MigrationsPath string
	StoragePath    string
	Email          Email
	Site           Site
	// JWTSecret signs access tokens (2A.3). At least 32 characters.
	JWTSecret string
}

// Site describes the addresses people use. Each organization is at
// <slug>.<OrgDomain> (4A.5), and OrgDomain alone is the general address,
// screen A1.
type Site struct {
	// OrgDomain is phoenixevent.tomsherlock.space during testing and
	// phoenixevent.localhost on a development Mac.
	OrgDomain string
	// Scheme and Port build the links in emails: https and no port on the
	// droplet; http and 5174, the web client's port, on a development Mac.
	Scheme string
	Port   string
	// CookieSecure marks cookies HTTPS-only. True everywhere but a
	// development Mac.
	CookieSecure bool
	// LANAddress and LANOrganization are for testing on a phone, on a
	// development Mac only. A phone cannot look up the .localhost names, so
	// the Mac's Wi-Fi address, such as 192.168.1.189:5174, stands for one
	// organization, named by its address name. scripts/run-local.sh sets
	// LAN_ADDRESS; LAN_ORGANIZATION is in backend/.env. Both empty on the
	// droplet.
	LANAddress      string
	LANOrganization string
}

// OrgURL is the address of one organization's site, ending in /.
func (s Site) OrgURL(slug string) string {
	return s.Scheme + "://" + slug + "." + s.OrgDomain + s.portSuffix() + "/"
}

// GeneralURL is the general address, screen A1.
func (s Site) GeneralURL() string {
	return s.Scheme + "://" + s.OrgDomain + s.portSuffix() + "/"
}

func (s Site) portSuffix() string {
	if s.Port == "" {
		return ""
	}
	return ":" + s.Port
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

	cfg.Site = Site{
		OrgDomain:       strings.ToLower(getenv("ORG_DOMAIN")),
		Scheme:          orDefault(getenv("SITE_SCHEME"), "https"),
		Port:            getenv("SITE_PORT"),
		CookieSecure:    getenv("COOKIE_SECURE") != "false",
		LANAddress:      strings.ToLower(getenv("LAN_ADDRESS")),
		LANOrganization: strings.ToLower(getenv("LAN_ORGANIZATION")),
	}
	cfg.JWTSecret = getenv("JWT_SECRET")

	if cfg.Site.OrgDomain == "" {
		errs = append(errs, errors.New("ORG_DOMAIN is required, such as phoenixevent.localhost"))
	}
	if cfg.Site.Scheme != "http" && cfg.Site.Scheme != "https" {
		errs = append(errs, fmt.Errorf("SITE_SCHEME: %q is neither http nor https", cfg.Site.Scheme))
	}
	if len(cfg.JWTSecret) < 32 {
		errs = append(errs, errors.New("JWT_SECRET is required, at least 32 characters"))
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
