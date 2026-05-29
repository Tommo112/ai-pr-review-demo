package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type prAnalyzer interface {
	AnalyzePullRequest(ctx context.Context, ref prRef, pr pullRequestData) (reviewResponse, error)
}

type fallbackAnalyzer struct{}

func newPRAnalyzer() prAnalyzer {
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := os.Getenv("OPENAI_MODEL")
	if apiKey == "" || model == "" {
		return fallbackAnalyzer{}
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return openAICompatibleAnalyzer{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func (fallbackAnalyzer) AnalyzePullRequest(_ context.Context, ref prRef, pr pullRequestData) (reviewResponse, error) {
	return newPendingAIReviewResponse(ref, pr), nil
}

type openAICompatibleAnalyzer struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

type aiReviewOutput struct {
	Summary        string          `json:"summary"`
	Risks          []risk          `json:"risks"`
	ReviewComments []reviewComment `json:"review_comments"`
	FinalReview    string          `json:"final_review"`
}

func (client openAICompatibleAnalyzer) AnalyzePullRequest(ctx context.Context, ref prRef, pr pullRequestData) (reviewResponse, error) {
	prompt := buildReviewPrompt(ref, pr)
	output, err := client.requestReview(ctx, prompt)
	if err != nil {
		return reviewResponse{}, err
	}

	response := newReviewResponseFromGitHub(ref, pr)
	response.Summary = output.Summary
	response.Risks = output.Risks
	response.ReviewComments = output.ReviewComments
	response.FinalReview = output.FinalReview
	return response, nil
}

func (client openAICompatibleAnalyzer) requestReview(ctx context.Context, prompt string) (aiReviewOutput, error) {
	payload := map[string]any{
		"model": client.model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a senior code reviewer. Return only valid JSON.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.2,
		"response_format": map[string]string{
			"type": "json_object",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return aiReviewOutput{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return aiReviewOutput{}, err
	}
	req.Header.Set("Authorization", "Bearer "+client.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return aiReviewOutput{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return aiReviewOutput{}, fmt.Errorf("ai api returned %s", resp.Status)
	}

	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return aiReviewOutput{}, err
	}
	if len(completion.Choices) == 0 {
		return aiReviewOutput{}, errors.New("ai api returned no choices")
	}

	var output aiReviewOutput
	if err := json.Unmarshal([]byte(completion.Choices[0].Message.Content), &output); err != nil {
		return aiReviewOutput{}, fmt.Errorf("ai response is not valid review JSON: %w", err)
	}

	return output, nil
}

func buildReviewPrompt(ref prRef, pr pullRequestData) string {
	var builder strings.Builder
	builder.WriteString("Review this GitHub pull request and return JSON with exactly these keys: summary, risks, review_comments, final_review.\n")
	builder.WriteString("Risk level must be one of high, medium, low. Use concise Chinese review language.\n\n")
	builder.WriteString("PR: ")
	builder.WriteString(ref.Owner + "/" + ref.Repo + "#" + strconv.Itoa(ref.Number) + "\n")
	builder.WriteString("Title: " + pr.Title + "\n")
	builder.WriteString("Author: " + pr.Author + "\n")
	builder.WriteString("Stats: +" + strconv.Itoa(pr.Additions) + " -" + strconv.Itoa(pr.Deletions) + ", files " + strconv.Itoa(pr.FilesChanged) + "\n\n")
	builder.WriteString("Files and patches:\n")
	builder.WriteString(trimDiffForPrompt(pr.Files, 12000))
	return builder.String()
}

func trimDiffForPrompt(files []pullRequestFile, maxChars int) string {
	var builder strings.Builder
	for _, file := range files {
		if builder.Len() >= maxChars {
			break
		}

		builder.WriteString("\n--- ")
		builder.WriteString(file.Filename)
		builder.WriteString(" (")
		builder.WriteString(file.Status)
		builder.WriteString(", +")
		builder.WriteString(strconv.Itoa(file.Additions))
		builder.WriteString(" -")
		builder.WriteString(strconv.Itoa(file.Deletions))
		builder.WriteString(") ---\n")

		patch := file.Patch
		if patch == "" {
			patch = "(no patch available)"
		}
		remaining := maxChars - builder.Len()
		if remaining <= 0 {
			break
		}
		if len(patch) > remaining {
			builder.WriteString(patch[:remaining])
			builder.WriteString("\n[diff truncated]\n")
			break
		}
		builder.WriteString(patch)
		builder.WriteString("\n")
	}

	return builder.String()
}
