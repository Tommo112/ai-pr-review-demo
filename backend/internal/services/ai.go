package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"demo/backend/internal/config"
	"demo/backend/internal/models"
)

type PRAnalyzer interface {
	AnalyzePullRequest(ctx context.Context, ref models.PRRef, pr models.PullRequestData) (models.ReviewResponse, error)
}

type FallbackAnalyzer struct{}

func NewPRAnalyzer(cfg config.Config) PRAnalyzer {
	if cfg.OpenAIAPIKey == "" || cfg.OpenAIModel == "" {
		return FallbackAnalyzer{}
	}

	return OpenAICompatibleAnalyzer{
		apiKey:  cfg.OpenAIAPIKey,
		baseURL: strings.TrimRight(cfg.OpenAIBaseURL, "/"),
		model:   cfg.OpenAIModel,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func (FallbackAnalyzer) AnalyzePullRequest(_ context.Context, ref models.PRRef, pr models.PullRequestData) (models.ReviewResponse, error) {
	return newPendingAIReviewResponse(ref, pr), nil
}

type OpenAICompatibleAnalyzer struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewOpenAICompatibleAnalyzerForTest(apiKey string, baseURL string, model string, httpClient *http.Client) OpenAICompatibleAnalyzer {
	return OpenAICompatibleAnalyzer{apiKey: apiKey, baseURL: baseURL, model: model, httpClient: httpClient}
}

type aiReviewOutput struct {
	Summary        string                 `json:"summary"`
	Risks          []models.Risk          `json:"risks"`
	ReviewComments []models.ReviewComment `json:"review_comments"`
	FinalReview    string                 `json:"final_review"`
}

func (client OpenAICompatibleAnalyzer) AnalyzePullRequest(ctx context.Context, ref models.PRRef, pr models.PullRequestData) (models.ReviewResponse, error) {
	prompt := buildReviewPrompt(ref, pr)
	output, err := client.requestReview(ctx, prompt)
	if err != nil {
		return models.ReviewResponse{}, err
	}

	response := newReviewResponseFromGitHub(pr)
	response.Summary = output.Summary
	response.Risks = output.Risks
	response.ReviewComments = output.ReviewComments
	response.FinalReview = output.FinalReview
	ensureReviewDefaults(&response, ref)
	return response, nil
}

func (client OpenAICompatibleAnalyzer) requestReview(ctx context.Context, prompt string) (aiReviewOutput, error) {
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
		return aiReviewOutput{}, ServiceError{
			Kind:    ErrorKindAIUnavailable,
			Message: "无法连接 AI 服务，请检查 OPENAI_BASE_URL、OPENAI_API_KEY 或代理配置",
			Err:     err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return aiReviewOutput{}, ServiceError{
			Kind:    ErrorKindAIUnavailable,
			Message: fmt.Sprintf("AI 服务返回异常状态：%s", resp.Status),
		}
	}

	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return aiReviewOutput{}, ServiceError{
			Kind:    ErrorKindAIUnavailable,
			Message: "AI 服务响应格式无法解析",
			Err:     err,
		}
	}
	if len(completion.Choices) == 0 {
		return aiReviewOutput{}, ServiceError{
			Kind:    ErrorKindAIUnavailable,
			Message: "AI 服务没有返回可用结果",
			Err:     errors.New("ai api returned no choices"),
		}
	}

	return parseAIReviewOutput(completion.Choices[0].Message.Content)
}

func buildReviewPrompt(ref models.PRRef, pr models.PullRequestData) string {
	var builder strings.Builder
	builder.WriteString("Review this GitHub pull request and return JSON with exactly these keys: summary, risks, review_comments, final_review.\n")
	builder.WriteString("The JSON shape must be: {\"summary\":\"...\",\"risks\":[{\"level\":\"high|medium|low\",\"file\":\"path\",\"line\":1,\"title\":\"...\",\"description\":\"...\",\"suggestion\":\"...\"}],\"review_comments\":[{\"file\":\"path\",\"line\":1,\"comment\":\"...\"}],\"final_review\":\"...\"}.\n")
	builder.WriteString("Risk level must be one of high, medium, low. Use concise Chinese review language.\n\n")
	builder.WriteString("PR: ")
	builder.WriteString(ref.Owner + "/" + ref.Repo + "#" + strconv.Itoa(ref.Number) + "\n")
	builder.WriteString("Title: " + pr.Title + "\n")
	builder.WriteString("Author: " + pr.Author + "\n")
	builder.WriteString("Stats: +" + strconv.Itoa(pr.Additions) + " -" + strconv.Itoa(pr.Deletions) + ", files " + strconv.Itoa(pr.FilesChanged) + "\n\n")
	builder.WriteString("Files and patches:\n")
	builder.WriteString(TrimDiffForPrompt(pr.Files, 12000))
	return builder.String()
}

func parseAIReviewOutput(content string) (aiReviewOutput, error) {
	var output aiReviewOutput
	if err := json.Unmarshal([]byte(content), &output); err == nil {
		return output, nil
	}

	var raw struct {
		Summary        string          `json:"summary"`
		Risks          json.RawMessage `json:"risks"`
		ReviewComments json.RawMessage `json:"review_comments"`
		FinalReview    string          `json:"final_review"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return aiReviewOutput{}, fmt.Errorf("ai response is not valid review JSON: %w", err)
	}

	output.Summary = raw.Summary
	output.FinalReview = raw.FinalReview
	_ = json.Unmarshal(raw.Risks, &output.Risks)
	_ = json.Unmarshal(raw.ReviewComments, &output.ReviewComments)
	return output, nil
}

func TrimDiffForPrompt(files []models.PullRequestFile, maxChars int) string {
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
