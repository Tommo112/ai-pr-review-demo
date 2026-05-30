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

func TestOpenAICompatibleAnalyzerNormalizesIncompleteReviewFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"message": {
						"content": "{\"summary\":\"AI summary\",\"risks\":[{\"level\":\"critical\",\"file\":\"\",\"line\":0,\"title\":\"\",\"description\":\"\",\"suggestion\":\"\"}],\"review_comments\":[{\"file\":\"\",\"line\":0,\"comment\":\"\"}],\"final_review\":\"LGTM\"}"
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
		Number: 8,
	}, models.PullRequestData{
		Title:        "Normalize",
		Author:       "alice",
		FilesChanged: 1,
		Files: []models.PullRequestFile{
			{Filename: "main.go"},
		},
	})
	if err != nil {
		t.Fatalf("expected normalized AI review response, got error: %v", err)
	}

	if response.Risks[0].Level != "low" || response.Risks[0].File != "main.go" || response.Risks[0].Line != 1 {
		t.Fatalf("unexpected normalized risk: %+v", response.Risks[0])
	}
	if response.Risks[0].Title == "" || response.Risks[0].Description == "" || response.Risks[0].Suggestion == "" {
		t.Fatalf("expected risk text defaults, got: %+v", response.Risks[0])
	}
	if response.ReviewComments[0].File != "main.go" || response.ReviewComments[0].Line != 1 || response.ReviewComments[0].Comment == "" {
		t.Fatalf("unexpected normalized comment: %+v", response.ReviewComments[0])
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

func TestOpenAICompatibleAnalyzerStreamsReview(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var requestBody struct {
			Stream bool `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("expected request JSON, got error: %v", err)
		}
		if !requestBody.Stream {
			t.Fatalf("expected streaming request")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"content":"{\"type\":\"summary_delta\",\"text\":\"AI summary\"}\n"}}]}` + "\n\n"))
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"content":"{\"type\":\"risk\",\"risk\":{\"level\":\"critical\",\"file\":\"\",\"line\":0,\"title\":\"Missing auth\",\"description\":\"Permission check is missing.\",\"suggestion\":\"Validate scope.\"}}\n"}}]}` + "\n\n"))
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"content":"{\"type\":\"review_comment\",\"comment\":{\"file\":\"\",\"line\":0,\"comment\":\"Add scope validation here.\"}}\n"}}]}` + "\n\n"))
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"content":"{\"type\":\"final_review_delta\",\"text\":\"LGTM after auth fix.\"}\n"}}]}` + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	analyzer := NewOpenAICompatibleAnalyzerForTest("test-key", server.URL, "test-model", server.Client())
	var events []string
	response, err := analyzer.AnalyzePullRequestStream(context.Background(), models.PRRef{
		Owner:  "owner",
		Repo:   "repo",
		Number: 6,
	}, models.PullRequestData{
		Title:        "Stream",
		Author:       "alice",
		FilesChanged: 1,
		Files: []models.PullRequestFile{
			{Filename: "auth.go"},
		},
	}, func(event string, _ any) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatalf("expected streaming AI review response, got error: %v", err)
	}

	if strings.Join(events, ",") != "summary_delta,risk,review_comment,final_review_delta" {
		t.Fatalf("unexpected streamed events: %+v", events)
	}
	if response.Summary != "AI summary" || response.FinalReview != "LGTM after auth fix." {
		t.Fatalf("unexpected streaming response: %+v", response)
	}
	if len(response.Risks) != 1 || response.Risks[0].File != "auth.go" {
		t.Fatalf("unexpected streaming risks: %+v", response.Risks)
	}
	if response.Risks[0].Level != "low" || response.Risks[0].Line != 1 {
		t.Fatalf("expected normalized streaming risk, got: %+v", response.Risks[0])
	}
	if len(response.ReviewComments) != 1 || response.ReviewComments[0].File != "auth.go" {
		t.Fatalf("unexpected streaming comments: %+v", response.ReviewComments)
	}
	if response.ReviewComments[0].Line != 1 {
		t.Fatalf("expected normalized streaming comment, got: %+v", response.ReviewComments[0])
	}
}

func TestBuildReviewPromptIncludesReviewGuidance(t *testing.T) {
	prompt := buildReviewPrompt(models.PRRef{
		Owner:  "owner",
		Repo:   "repo",
		Number: 5,
	}, models.PullRequestData{
		Title:        "Update auth",
		Author:       "alice",
		FilesChanged: 1,
		Additions:    8,
		Deletions:    2,
		Files: []models.PullRequestFile{
			{
				Filename: "auth.go",
				Status:   "modified",
				Patch:    "@@ -1 +1 @@\n-old\n+new",
			},
		},
	})

	expectedParts := []string{
		"Return only JSON",
		"Security or permission risks",
		"Missing error handling",
		"Missing tests",
		"overall overview",
		"concrete changes",
		"affected areas",
		"PR comment",
		"owner/repo#5",
		"auth.go",
	}
	for _, part := range expectedParts {
		if !strings.Contains(prompt, part) {
			t.Fatalf("expected prompt to contain %q, got: %s", part, prompt)
		}
	}
}

func TestBuildStreamingReviewPromptAsksForDetailedSummary(t *testing.T) {
	prompt := buildStreamingReviewPrompt(models.PRRef{
		Owner:  "owner",
		Repo:   "repo",
		Number: 7,
	}, models.PullRequestData{
		Title:        "Update service",
		Author:       "alice",
		FilesChanged: 1,
		Files: []models.PullRequestFile{
			{
				Filename: "service.go",
				Status:   "modified",
				Patch:    "@@ -1 +1 @@\n-old\n+new",
			},
		},
	})

	expectedParts := []string{
		"summary_delta",
		"overall overview",
		"concrete changes",
		"affected areas",
		"key points to verify",
		"PR-comment-ready conclusion",
	}
	for _, part := range expectedParts {
		if !strings.Contains(prompt, part) {
			t.Fatalf("expected streaming prompt to contain %q, got: %s", part, prompt)
		}
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

func TestTrimDiffForPromptSkipsLowValueFiles(t *testing.T) {
	result := TrimDiffForPrompt([]models.PullRequestFile{
		{
			Filename: "bun.lock",
			Status:   "modified",
			Patch:    "lock diff",
		},
		{
			Filename: "src/review.go",
			Status:   "modified",
			Patch:    "code diff",
		},
		{
			Filename: "public/logo.svg",
			Status:   "modified",
			Patch:    "svg diff",
		},
	}, 1000)

	if strings.Contains(result, "bun.lock") || strings.Contains(result, "logo.svg") {
		t.Fatalf("expected low-value files to be skipped, got: %s", result)
	}
	if !strings.Contains(result, "src/review.go") || !strings.Contains(result, "code diff") {
		t.Fatalf("expected source file to remain, got: %s", result)
	}
}
