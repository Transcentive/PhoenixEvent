package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error { return f.err }

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		db         fakeDB
		wantCode   int
		wantStatus string
	}{
		{"database answers", fakeDB{}, http.StatusOK, "ok"},
		{"database unreachable", fakeDB{err: errors.New("connection refused")}, http.StatusServiceUnavailable, "unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			New(tt.db, "abc123").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))

			if rec.Code != tt.wantCode {
				t.Fatalf("status code = %d, want %d", rec.Code, tt.wantCode)
			}
			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("body is not JSON: %v", err)
			}
			if body["status"] != tt.wantStatus || body["build"] != "abc123" {
				t.Errorf("body = %v", body)
			}
		})
	}
}

func TestUnknownRouteIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	New(fakeDB{}, "abc123").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/nothing", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want 404", rec.Code)
	}
}
