package config

import (
	"strings"
	"testing"
)

func env(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadFrom(t *testing.T) {
	base := map[string]string{
		"DATABASE_URL": "postgres://phoenixevent@localhost:5433/phoenixevent",
		"STORAGE_PATH": "/tmp/pe",
	}
	with := func(extra map[string]string) map[string]string {
		merged := map[string]string{}
		for k, v := range base {
			merged[k] = v
		}
		for k, v := range extra {
			merged[k] = v
		}
		return merged
	}

	tests := []struct {
		name    string
		values  map[string]string
		wantErr string
		check   func(t *testing.T, cfg Config)
	}{
		{
			name:   "defaults",
			values: base,
			check: func(t *testing.T, cfg Config) {
				if cfg.Port != 8082 || cfg.Email.Port != 587 || cfg.MigrationsPath != "migrations" {
					t.Errorf("unexpected defaults: %+v", cfg)
				}
			},
		},
		{
			name:   "explicit values",
			values: with(map[string]string{"PORT": "9000", "SMTP_HOST": "smtp.example.com", "EMAIL_FROM": "events@example.com"}),
			check: func(t *testing.T, cfg Config) {
				if cfg.Port != 9000 || cfg.Email.Host != "smtp.example.com" {
					t.Errorf("values not read: %+v", cfg)
				}
			},
		},
		{name: "database url missing", values: map[string]string{"STORAGE_PATH": "/tmp/pe"}, wantErr: "DATABASE_URL is required"},
		{name: "storage path missing", values: map[string]string{"DATABASE_URL": "x"}, wantErr: "STORAGE_PATH is required"},
		{name: "port not a number", values: with(map[string]string{"PORT": "http"}), wantErr: "PORT"},
		{name: "port out of range", values: with(map[string]string{"PORT": "70000"}), wantErr: "PORT"},
		{name: "smtp without a from address", values: with(map[string]string{"SMTP_HOST": "smtp.example.com"}), wantErr: "EMAIL_FROM is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadFrom(env(tt.values))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want one containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, cfg)
		})
	}
}

func TestLoadFromReportsEveryProblem(t *testing.T) {
	_, err := LoadFrom(env(map[string]string{"PORT": "x"}))
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"DATABASE_URL", "STORAGE_PATH", "PORT"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %s", err, want)
		}
	}
}
