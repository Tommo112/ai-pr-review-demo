package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

	analyzer := openAICompatibleAnalyzer{
		apiKey:     "test-key",
		baseURL:    server.URL,
		model:      "test-model",
		httpClient: server.Client(),
	}

	response, err := analyzer.AnalyzePullRequest(context.Background(), prRef{
		Owner:  "owner",
		Repo:   "repo",
		Number: 1,
	}, pullRequestData{
		Title:        "Fix auth",
		Author:       "alice",
		FilesChanged: 1,
		Additions:    10,
		Deletions:    2,
		Files: []pullRequestFile{
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

func TestTrimDiffForPrompt(t *testing.T) {
	result := trimDiffForPrompt([]pullRequestFile{
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
