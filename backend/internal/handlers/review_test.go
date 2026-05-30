package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"demo/backend/internal/config"
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

func TestReviewEndpointMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			name: "github not found",
			err: services.ServiceError{
				Kind:    services.ErrorKindGitHubNotFound,
				Message: "not found",
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "github rate limited",
			err: services.ServiceError{
				Kind:    services.ErrorKindGitHubRateLimited,
				Message: "rate limited",
			},
			wantStatus: http.StatusTooManyRequests,
		},
		{
			name: "ai unavailable",
			err: services.ServiceError{
				Kind:    services.ErrorKindAIUnavailable,
				Message: "ai unavailable",
			},
			wantStatus: http.StatusBadGateway,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := testRouter(mockPullRequestFetcher{err: tt.err})

			body := bytes.NewBufferString(`{"pr_url":"https://github.com/owner/repo/pull/1"}`)
			req := httptest.NewRequest(http.MethodPost, "/api/review", body)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestReviewStreamEndpointEmitsPRThenReview(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodPost, "/api/review/stream", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("expected event stream content type, got %q", contentType)
	}
	bodyText := rec.Body.String()
	for _, event := range []string{"event: status", "event: pr", "event: summary_delta", "event: risk", "event: review_comment", "event: final_review_delta", "event: review", "event: done"} {
		if !strings.Contains(bodyText, event) {
			t.Fatalf("expected stream to contain %q, got: %s", event, bodyText)
		}
	}
	if !strings.Contains(bodyText, "fetching_pr") || !strings.Contains(bodyText, "analyzing_ai") {
		t.Fatalf("expected stream to include status messages, got: %s", bodyText)
	}
	if !strings.Contains(bodyText, "Fix auth") || !strings.Contains(bodyText, "auth.go") {
		t.Fatalf("expected stream to include PR data, got: %s", bodyText)
	}
}

func TestReviewStreamEndpointAcceptsGETForEventSource(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/api/review/stream?pr_url=https%3A%2F%2Fgithub.com%2Fowner%2Frepo%2Fpull%2F1", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("expected event stream content type, got %q", contentType)
	}
	bodyText := rec.Body.String()
	if !strings.Contains(bodyText, "event: pr") || !strings.Contains(bodyText, "event: review") {
		t.Fatalf("expected GET stream to emit PR and review events, got: %s", bodyText)
	}
}

func TestReviewStreamEndpointEmitsInvalidPRURLError(t *testing.T) {
	router := testRouter(mockPullRequestFetcher{})

	body := bytes.NewBufferString(`{"pr_url":"https://example.com/owner/repo/pull/1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/review/stream", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected stream status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	bodyText := rec.Body.String()
	if !strings.Contains(bodyText, "event: error") {
		t.Fatalf("expected error event, got: %s", bodyText)
	}
	if !strings.Contains(bodyText, `"code":"invalid_pr_url"`) {
		t.Fatalf("expected invalid PR URL code, got: %s", bodyText)
	}
}

func TestReviewStreamEndpointEmitsServiceErrorCode(t *testing.T) {
	router := testRouter(mockPullRequestFetcher{
		err: services.ServiceError{
			Kind:    services.ErrorKindGitHubRateLimited,
			Message: "GitHub API rate limit exceeded. Try again later.",
		},
	})

	body := bytes.NewBufferString(`{"pr_url":"https://github.com/owner/repo/pull/1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/review/stream", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected stream status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	bodyText := rec.Body.String()
	if !strings.Contains(bodyText, "event: error") {
		t.Fatalf("expected error event, got: %s", bodyText)
	}
	if !strings.Contains(bodyText, `"code":"github_rate_limited"`) {
		t.Fatalf("expected GitHub rate limit code, got: %s", bodyText)
	}
	if strings.Contains(bodyText, "event: done") {
		t.Fatalf("did not expect done after error, got: %s", bodyText)
	}
}

func TestRuntimeStatusEndpointDoesNotExposeSecrets(t *testing.T) {
	router := testRouterWithConfig(mockPullRequestFetcher{}, config.Config{
		Port:          "7897",
		GitHubToken:   "github-token",
		OpenAIAPIKey:  "ai-key",
		OpenAIModel:   "demo-model",
		OpenAIBaseURL: "http://example.test/v1",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response models.RuntimeStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected runtime status JSON, got error: %v", err)
	}
	if response.Port != "7897" || !response.GitHub.TokenConfigured || !response.AI.Enabled {
		t.Fatalf("unexpected runtime status: %+v", response)
	}
	bodyText := rec.Body.String()
	if strings.Contains(bodyText, "github-token") || strings.Contains(bodyText, "ai-key") {
		t.Fatalf("runtime status must not expose secrets: %s", bodyText)
	}
}

func testRouter(fetcher services.PullRequestFetcher) *gin.Engine {
	return testRouterWithConfig(fetcher, config.Config{
		Port:          "8080",
		OpenAIBaseURL: "https://api.openai.com/v1",
	})
}

func testRouterWithConfig(fetcher services.PullRequestFetcher, cfg config.Config) *gin.Engine {
	reviewService := services.NewReviewService(fetcher, services.FallbackAnalyzer{})
	reviewHandler := NewReviewHandler(reviewService, cfg)
	router := gin.New()
	router.GET("/health", reviewHandler.Health)
	router.GET("/api/status", reviewHandler.RuntimeStatus)
	router.POST("/api/review", reviewHandler.Review)
	router.GET("/api/review/stream", reviewHandler.ReviewStream)
	router.POST("/api/review/stream", reviewHandler.ReviewStream)
	return router
}
