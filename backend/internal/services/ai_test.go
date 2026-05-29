package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"demo/backend/internal/models"
)

func TestOpenAICompatibleAnalyzer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %q", r.Header.Get("Authorization"))
		}

		var requestBody struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("expected request JSON, got error: %v", err)
		}
		if requestBody.Model != "test-model" || len(requestBody.Messages) != 2 {
			t.Fatalf("unexpected request body: %+v", requestBody)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"message": {
						"content": "{\"summary\":\"AI summary\",\"risks\":[{\"level\":\"high\",\"file\":\"auth.go\",\"line\":12,\"title\":\"Missing check\",\"description\":\"Missing permission check\",\"suggestion\":\"Add scope validation\"}],\"review_comments\":[{\"file\":\"auth.go\",\"line\":12,\"comment\":\"Add scope validation here.\"}],\"final_review\":\"Review before merge.\"}"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	analyzer := NewOpenAICompatibleAnalyzerForTest("test-key", server.URL, "test-model", server.Client())
	response, err := analyzer.AnalyzePullRequest(context.Background(), models.PRRef{
		Owner:  "owner",
		Repo:   "repo",
		Number: 1,
	}, models.PullRequestData{
		Title:        "Fix auth",
		Author:       "alice",
		FilesChanged: 1,
		Additions:    10,
		Deletions:    2,
		Files: []models.PullRequestFile{
			{Filename: "auth.go", Status: "modified", Additions: 10, Deletions: 2, Patch: "@@ -1 +1 @@"},
		},
	})
	if err != nil {
		t.Fatalf("expected AI review response, got error: %v", err)
	}

	if response.Summary != "AI summary" || response.PR.Title != "Fix auth" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if len(response.Risks) != 1 || response.Risks[0].Level != "high" {
		t.Fatalf("unexpected risks: %+v", response.Risks)
	}
}

func TestOpenAICompatibleAnalyzerFillsEmptyReviewDefaults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"message": {
						"content": "{\"summary\":\"\",\"risks\":[],\"review_comments\":[],\"final_review\":\"\"}"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	analyzer := NewOpenAICompatibleAnalyzerForTest("test-key", server.URL, "test-model", server.Client())
	response, err := analyzer.AnalyzePullRequest(context.Background(), models.PRRef{
		Owner:  "owner",
		Repo:   "repo",
		Number: 2,
	}, models.PullRequestData{
		Title:        "Docs",
		Author:       "bob",
		FilesChanged: 0,
	})
	if err != nil {
		t.Fatalf("expected review defaults, got error: %v", err)
	}

	if response.Summary == "" || len(response.Risks) == 0 || len(response.ReviewComments) == 0 || response.FinalReview == "" {
		t.Fatalf("expected fallback review fields, got: %+v", response)
	}
}

func TestOpenAICompatibleAnalyzerToleratesWrongCollectionTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"message": {
						"content": "{\"summary\":\"AI summary\",\"risks\":\"none\",\"review_comments\":\"none\",\"final_review\":\"LGTM\"}"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	analyzer := NewOpenAICompatibleAnalyzerForTest("test-key", server.URL, "test-model", server.Client())
	response, err := analyzer.AnalyzePullRequest(context.Background(), models.PRRef{
		Owner:  "owner",
		Repo:   "repo",
		Number: 3,
	}, models.PullRequestData{
		Title:        "Refactor",
		Author:       "alice",
		FilesChanged: 1,
		Files: []models.PullRequestFile{
			{Filename: "main.go"},
		},
	})
	if err != nil {
		t.Fatalf("expected tolerant AI review response, got error: %v", err)
	}

	if response.Summary != "AI summary" || response.FinalReview != "LGTM" {
		t.Fatalf("unexpected response text: %+v", response)
	}
	if len(response.Risks) == 0 || len(response.ReviewComments) == 0 {
		t.Fatalf("expected default risks and comments, got: %+v", response)
	}
}

func TestOpenAICompatibleAnalyzerReturnsServiceErrorForBadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer server.Close()

	analyzer := NewOpenAICompatibleAnalyzerForTest("test-key", server.URL, "test-model", server.Client())
	_, err := analyzer.AnalyzePullRequest(context.Background(), models.PRRef{
		Owner:  "owner",
		Repo:   "repo",
		Number: 4,
	}, models.PullRequestData{})

	var serviceErr ServiceError
	if !errors.As(err, &serviceErr) {
		t.Fatalf("expected service error, got: %v", err)
	}
	if serviceErr.Kind != ErrorKindAIUnavailable {
		t.Fatalf("unexpected error kind: %s", serviceErr.Kind)
	}
}

func TestTrimDiffForPrompt(t *testing.T) {
	result := TrimDiffForPrompt([]models.PullRequestFile{
		{
			Filename: "large.go",
			Status:   "modified",
			Patch:    strings.Repeat("x", 200),
		},
	}, 80)

	if !strings.Contains(result, "[diff truncated]") {
		t.Fatalf("expected diff to be truncated, got: %s", result)
	}
}
