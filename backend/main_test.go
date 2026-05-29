package main

import (
	"bytes"
	"context"
	"encoding/json"
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

	var response reviewResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected review response JSON, got error: %v", err)
	}

	if response.PR.Title != "Fix auth" || response.PR.Author != "alice" {
		t.Fatalf("unexpected PR info: %+v", response.PR)
	}

	if len(response.Files) != 1 || response.Files[0].Filename != "auth.go" {
		t.Fatalf("unexpected files: %+v", response.Files)
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
