package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type pullRequestFetcher interface {
	FetchPullRequest(ctx context.Context, ref prRef) (pullRequestData, error)
}

type pullRequestData struct {
	Title        string
	Author       string
	FilesChanged int
	Additions    int
	Deletions    int
	Files        []pullRequestFile
}

type pullRequestFile struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch"`
}

type githubClient struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

func newGitHubClient() githubClient {
	return githubClient{
		baseURL: "https://api.github.com",
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		token: os.Getenv("GITHUB_TOKEN"),
	}
}

func (client githubClient) FetchPullRequest(ctx context.Context, ref prRef) (pullRequestData, error) {
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
		return pullRequestData{}, err
	}

	var files []pullRequestFile
	if err := client.getJSON(ctx, pullFilesURL(client.baseURL, ref), &files); err != nil {
		return pullRequestData{}, err
	}

	return pullRequestData{
		Title:        pull.Title,
		Author:       pull.User.Login,
		FilesChanged: pull.ChangedFiles,
		Additions:    pull.Additions,
		Deletions:    pull.Deletions,
		Files:        files,
	}, nil
}

func (client githubClient) getJSON(ctx context.Context, endpoint string, target any) error {
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

func pullURL(baseURL string, ref prRef) string {
	return joinGitHubURL(baseURL, ref, "pulls", strconv.Itoa(ref.Number))
}

func pullFilesURL(baseURL string, ref prRef) string {
	return joinGitHubURL(baseURL, ref, "pulls", strconv.Itoa(ref.Number), "files") + "?per_page=100"
}

func joinGitHubURL(baseURL string, ref prRef, parts ...string) string {
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
