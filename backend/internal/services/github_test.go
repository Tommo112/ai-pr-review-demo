package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"demo/backend/internal/models"
)

func TestGitHubClientFetchPullRequest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/pulls/12", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Fatalf("unexpected accept header: %q", r.Header.Get("Accept"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"title": "Improve parser",
			"additions": 20,
			"deletions": 4,
			"changed_files": 2,
			"user": {"login": "alice"}
		}`))
	})
	mux.HandleFunc("/repos/owner/repo/pulls/12/files", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("per_page") != "100" {
			t.Fatalf("unexpected per_page: %q", r.URL.Query().Get("per_page"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"filename": "parser.go",
				"status": "modified",
				"additions": 12,
				"deletions": 1,
				"patch": "@@ -1 +1 @@"
			}
		]`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewGitHubClientForTest(server.URL, server.Client(), "")
	data, err := client.FetchPullRequest(context.Background(), models.PRRef{
		Owner:  "owner",
		Repo:   "repo",
		Number: 12,
	})
	if err != nil {
		t.Fatalf("expected GitHub data, got error: %v", err)
	}

	if data.Title != "Improve parser" || data.Author != "alice" || data.FilesChanged != 2 {
		t.Fatalf("unexpected PR data: %+v", data)
	}

	if len(data.Files) != 1 || data.Files[0].Filename != "parser.go" {
		t.Fatalf("unexpected file data: %+v", data.Files)
	}
}

func TestGitHubClientReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client := NewGitHubClientForTest(server.URL, server.Client(), "")
	_, err := client.FetchPullRequest(context.Background(), models.PRRef{
		Owner:  "owner",
		Repo:   "repo",
		Number: 12,
	})
	if err == nil {
		t.Fatal("expected status error")
	}
}
