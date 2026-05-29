package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"demo/backend/internal/config"
	"demo/backend/internal/models"
)

type PullRequestFetcher interface {
	FetchPullRequest(ctx context.Context, ref models.PRRef) (models.PullRequestData, error)
}

type GitHubClient struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

func NewGitHubClient(cfg config.Config) GitHubClient {
	return GitHubClient{
		baseURL: "https://api.github.com",
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		token: cfg.GitHubToken,
	}
}

func NewGitHubClientForTest(baseURL string, httpClient *http.Client, token string) GitHubClient {
	return GitHubClient{baseURL: baseURL, httpClient: httpClient, token: token}
}

func (client GitHubClient) FetchPullRequest(ctx context.Context, ref models.PRRef) (models.PullRequestData, error) {
	var pull struct {
		Title        string `json:"title"`
		Additions    int    `json:"additions"`
		Deletions    int    `json:"deletions"`
		ChangedFiles int    `json:"changed_files"`
		User         struct {
			Login string `json:"login"`
		} `json:"user"`
	}

	if err := client.getJSON(ctx, pullURL(client.baseURL, ref), &pull); err != nil {
		return models.PullRequestData{}, err
	}

	var files []models.PullRequestFile
	if err := client.getJSON(ctx, pullFilesURL(client.baseURL, ref), &files); err != nil {
		return models.PullRequestData{}, err
	}

	return models.PullRequestData{
		Title:        pull.Title,
		Author:       pull.User.Login,
		FilesChanged: pull.ChangedFiles,
		Additions:    pull.Additions,
		Deletions:    pull.Deletions,
		Files:        files,
	}, nil
}

func (client GitHubClient) getJSON(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ai-pr-review-demo")
	if client.token != "" {
		req.Header.Set("Authorization", "Bearer "+client.token)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("github api returned %s", resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func pullURL(baseURL string, ref models.PRRef) string {
	return joinGitHubURL(baseURL, ref, "pulls", strconv.Itoa(ref.Number))
}

func pullFilesURL(baseURL string, ref models.PRRef) string {
	return joinGitHubURL(baseURL, ref, "pulls", strconv.Itoa(ref.Number), "files") + "?per_page=100"
}

func joinGitHubURL(baseURL string, ref models.PRRef, parts ...string) string {
	escaped := []string{
		"repos",
		url.PathEscape(ref.Owner),
		url.PathEscape(ref.Repo),
	}
	for _, part := range parts {
		escaped = append(escaped, url.PathEscape(part))
	}

	return baseURL + "/" + strings.Join(escaped, "/")
}
