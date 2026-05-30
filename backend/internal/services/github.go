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

const (
	githubFilesPerPage = 100
	maxGitHubFilePages = 3
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

	files, err := client.fetchPullRequestFiles(ctx, ref)
	if err != nil {
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

func (client GitHubClient) fetchPullRequestFiles(ctx context.Context, ref models.PRRef) ([]models.PullRequestFile, error) {
	files := make([]models.PullRequestFile, 0)
	for page := 1; page <= maxGitHubFilePages; page++ {
		var pageFiles []models.PullRequestFile
		if err := client.getJSON(ctx, pullFilesURL(client.baseURL, ref, page), &pageFiles); err != nil {
			return nil, err
		}

		files = append(files, pageFiles...)
		if len(pageFiles) < githubFilesPerPage {
			break
		}
	}

	return files, nil
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
		return ServiceError{
			Kind:    ErrorKindGitHubUnavailable,
			Message: "Unable to connect to GitHub API. Check network, proxy, or GitHub token settings.",
			Err:     err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return githubStatusError(resp)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func githubStatusError(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusNotFound:
		return ServiceError{
			Kind:    ErrorKindGitHubNotFound,
			Message: "GitHub PR was not found. Confirm the repository, PR number, and token access.",
		}
	case http.StatusUnauthorized:
		return ServiceError{
			Kind:    ErrorKindGitHubUnauthorized,
			Message: "GitHub token is missing or invalid. Check GITHUB_TOKEN.",
		}
	case http.StatusForbidden:
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			return ServiceError{
				Kind:    ErrorKindGitHubRateLimited,
				Message: "GitHub API rate limit exceeded. Retry later or configure a valid GITHUB_TOKEN.",
			}
		}
		return ServiceError{
			Kind:    ErrorKindGitHubUnauthorized,
			Message: "No permission to access this GitHub PR. Check repository or token permissions.",
		}
	default:
		return ServiceError{
			Kind:    ErrorKindGitHubUnavailable,
			Message: fmt.Sprintf("GitHub API returned an unexpected status: %s", resp.Status),
		}
	}
}

func pullURL(baseURL string, ref models.PRRef) string {
	return joinGitHubURL(baseURL, ref, "pulls", strconv.Itoa(ref.Number))
}

func pullFilesURL(baseURL string, ref models.PRRef, page int) string {
	return joinGitHubURL(baseURL, ref, "pulls", strconv.Itoa(ref.Number), "files") + "?per_page=" + strconv.Itoa(githubFilesPerPage) + "&page=" + strconv.Itoa(page)
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
