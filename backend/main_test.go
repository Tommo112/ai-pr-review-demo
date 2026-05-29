package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReviewEndpoint(t *testing.T) {
	router := setupRouter()

	body := bytes.NewBufferString(`{"pr_url":"https://github.com/owner/repo/pull/1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/review", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestReviewEndpointRequiresPRURL(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/review", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestReviewEndpointRejectsInvalidPRURL(t *testing.T) {
	router := setupRouter()

	body := bytes.NewBufferString(`{"pr_url":"https://example.com/owner/repo/pull/1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/review", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}
