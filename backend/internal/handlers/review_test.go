package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"demo/backend/internal/models"
	"demo/backend/internal/services"

	"github.com/gin-gonic/gin"
)

type mockPullRequestFetcher struct {
	data models.PullRequestData
	err  error
}

func (fetcher mockPullRequestFetcher) FetchPullRequest(_ context.Context, _ models.PRRef) (models.PullRequestData, error) {
	return fetcher.data, fetcher.err
}

func TestReviewEndpoint(t *testing.T) {
	router := testRouter(mockPullRequestFetcher{
		data: models.PullRequestData{
			Title:        "Fix auth",
			Author:       "alice",
			FilesChanged: 1,
			Additions:    10,
			Deletions:    2,
			Files: []models.PullRequestFile{
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

	var response models.ReviewResponse
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
	router := testRouter(mockPullRequestFetcher{})

	req := httptest.NewRequest(http.MethodPost, "/api/review", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestReviewEndpointRejectsInvalidPRURL(t *testing.T) {
	router := testRouter(mockPullRequestFetcher{})

	body := bytes.NewBufferString(`{"pr_url":"https://example.com/owner/repo/pull/1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/review", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func testRouter(fetcher services.PullRequestFetcher) *gin.Engine {
	reviewService := services.NewReviewService(fetcher, services.FallbackAnalyzer{})
	reviewHandler := NewReviewHandler(reviewService)
	router := gin.New()
	router.GET("/health", reviewHandler.Health)
	router.POST("/api/review", reviewHandler.Review)
	return router
}
