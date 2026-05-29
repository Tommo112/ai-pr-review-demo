package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockPullRequestFetcher struct {
	data pullRequestData
	err  error
}

func (fetcher mockPullRequestFetcher) FetchPullRequest(_ context.Context, _ prRef) (pullRequestData, error) {
	return fetcher.data, fetcher.err
}

func TestReviewEndpoint(t *testing.T) {
	router := setupRouterWithFetcher(mockPullRequestFetcher{
		data: pullRequestData{
			Title:        "Fix auth",
			Author:       "alice",
			FilesChanged: 1,
			Additions:    10,
			Deletions:    2,
			Files: []pullRequestFile{
				{Filename: "auth.go", Status: "modified", Patch: "@@ -1 +1 @@"},
			},
		},
	})

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
	router := setupRouterWithFetcher(mockPullRequestFetcher{})

	req := httptest.NewRequest(http.MethodPost, "/api/review", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestReviewEndpointRejectsInvalidPRURL(t *testing.T) {
	router := setupRouterWithFetcher(mockPullRequestFetcher{})

	body := bytes.NewBufferString(`{"pr_url":"https://example.com/owner/repo/pull/1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/review", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}
