package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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
		if r.URL.Query().Get("page") != "1" {
			t.Fatalf("unexpected page: %q", r.URL.Query().Get("page"))
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

func TestGitHubClientFetchPullRequestFilesAcrossPages(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/pulls/12", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"title": "Large PR",
			"additions": 300,
			"deletions": 40,
			"changed_files": 101,
			"user": {"login": "alice"}
		}`))
	})
	mux.HandleFunc("/repos/owner/repo/pulls/12/files", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("page") {
		case "1":
			_, _ = w.Write([]byte(filePageJSON(100, "page-one")))
		case "2":
			_, _ = w.Write([]byte(`[
				{
					"filename": "page-two.go",
					"status": "modified",
					"additions": 1,
					"deletions": 0,
					"patch": "@@ -1 +1 @@"
				}
			]`))
		default:
			t.Fatalf("unexpected files page: %s", r.URL.Query().Get("page"))
		}
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

	if len(data.Files) != 101 {
		t.Fatalf("expected 101 files, got %d", len(data.Files))
	}
	if data.Files[100].Filename != "page-two.go" {
		t.Fatalf("expected second page file, got %+v", data.Files[100])
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
	var serviceErr ServiceError
	if !errors.As(err, &serviceErr) {
		t.Fatalf("expected service error, got: %v", err)
	}
	if serviceErr.Kind != ErrorKindGitHubNotFound {
		t.Fatalf("unexpected error kind: %s", serviceErr.Kind)
	}
}

func TestGitHubClientReturnsRateLimitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		http.Error(w, "rate limited", http.StatusForbidden)
	}))
	defer server.Close()

	client := NewGitHubClientForTest(server.URL, server.Client(), "")
	_, err := client.FetchPullRequest(context.Background(), models.PRRef{
		Owner:  "owner",
		Repo:   "repo",
		Number: 12,
	})
	var serviceErr ServiceError
	if !errors.As(err, &serviceErr) {
		t.Fatalf("expected service error, got: %v", err)
	}
	if serviceErr.Kind != ErrorKindGitHubRateLimited {
		t.Fatalf("unexpected error kind: %s", serviceErr.Kind)
	}
}

func filePageJSON(count int, prefix string) string {
	var builder strings.Builder
	builder.WriteString("[")
	for i := 0; i < count; i++ {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(`{"filename":"`)
		builder.WriteString(prefix)
		builder.WriteString("-")
		builder.WriteString(strconv.Itoa(i))
		builder.WriteString(`.go","status":"modified","additions":1,"deletions":0,"patch":"@@ -1 +1 @@"}`)
	}
	builder.WriteString("]")
	return builder.String()
}
