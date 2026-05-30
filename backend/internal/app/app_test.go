package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"demo/backend/internal/config"
)

func TestAppHandlesCORSPrefight(t *testing.T) {
	router := New(config.Config{
		Port:          "7897",
		OpenAIBaseURL: "https://api.openai.com/v1",
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/review/stream", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected CORS allow origin header, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatalf("expected CORS allow methods header")
	}
}
