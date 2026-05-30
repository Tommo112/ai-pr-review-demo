package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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

func (FallbackAnalyzer) AnalyzePullRequestStream(_ context.Context, ref models.PRRef, pr models.PullRequestData, emit StreamEmitter) (models.ReviewResponse, error) {
	response := newPendingAIReviewResponse(ref, pr)
	if err := emit("summary_delta", models.ReviewTextDeltaEvent{Text: response.Summary}); err != nil {
		return models.ReviewResponse{}, err
	}
	for _, risk := range response.Risks {
		if err := emit("risk", models.ReviewRiskEvent{Risk: risk}); err != nil {
			return models.ReviewResponse{}, err
		}
	}
	for _, comment := range response.ReviewComments {
		if err := emit("review_comment", models.ReviewCommentEvent{Comment: comment}); err != nil {
			return models.ReviewResponse{}, err
		}
	}
	if err := emit("final_review_delta", models.ReviewTextDeltaEvent{Text: response.FinalReview}); err != nil {
		return models.ReviewResponse{}, err
	}
	return response, nil
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

func (client OpenAICompatibleAnalyzer) AnalyzePullRequestStream(ctx context.Context, ref models.PRRef, pr models.PullRequestData, emit StreamEmitter) (models.ReviewResponse, error) {
	prompt := buildStreamingReviewPrompt(ref, pr)
	output, err := client.requestReviewStream(ctx, prompt, emit, fallbackReviewFile(pr.Files))
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
				"content": "You are a senior code reviewer. Return only valid JSON without markdown fences.",
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
			Message: "Unable to connect to the AI service. Check OPENAI_BASE_URL, OPENAI_API_KEY, or proxy settings.",
			Err:     err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return aiReviewOutput{}, ServiceError{
			Kind:    ErrorKindAIUnavailable,
			Message: fmt.Sprintf("AI service returned an unexpected status: %s", resp.Status),
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
			Message: "Unable to parse AI service response.",
			Err:     err,
		}
	}
	if len(completion.Choices) == 0 {
		return aiReviewOutput{}, ServiceError{
			Kind:    ErrorKindAIUnavailable,
			Message: "AI service returned no usable choices.",
			Err:     errors.New("ai api returned no choices"),
		}
	}

	return parseAIReviewOutput(completion.Choices[0].Message.Content)
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
