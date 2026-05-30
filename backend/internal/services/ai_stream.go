package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"demo/backend/internal/models"
)

type aiReviewStreamEvent struct {
	Type    string               `json:"type"`
	Text    string               `json:"text"`
	Risk    models.Risk          `json:"risk"`
	Comment models.ReviewComment `json:"comment"`
}

func (client OpenAICompatibleAnalyzer) requestReviewStream(ctx context.Context, prompt string, emit StreamEmitter, fallbackFile string) (aiReviewOutput, error) {
	payload := map[string]any{
		"model": client.model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a senior code reviewer. Return only newline-delimited JSON events without markdown fences.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.2,
		"stream":      true,
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
	req.Header.Set("Accept", "text/event-stream")

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

	return readReviewStream(resp, emit, fallbackFile)
}

func readReviewStream(resp *http.Response, emit StreamEmitter, fallbackFile string) (aiReviewOutput, error) {
	var content strings.Builder
	var lineBuffer strings.Builder
	var output aiReviewOutput
	var summaryBuilder strings.Builder
	var finalReviewBuilder strings.Builder
	parsedEvents := 0

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}

		delta, err := parseStreamDelta(data)
		if err != nil {
			return aiReviewOutput{}, err
		}
		if delta == "" {
			continue
		}

		content.WriteString(delta)
		parsed, err := handleStreamDelta(delta, emit, &output, &summaryBuilder, &finalReviewBuilder, &lineBuffer, fallbackFile)
		if err != nil {
			return aiReviewOutput{}, err
		}
		parsedEvents += parsed
	}
	if err := scanner.Err(); err != nil {
		return aiReviewOutput{}, ServiceError{
			Kind:    ErrorKindAIUnavailable,
			Message: "Unable to read AI streaming response.",
			Err:     err,
		}
	}
	if content.Len() == 0 {
		return aiReviewOutput{}, ServiceError{
			Kind:    ErrorKindAIUnavailable,
			Message: "AI service returned an empty streaming response.",
			Err:     errors.New("ai stream returned no content"),
		}
	}

	parsed, err := flushStreamLineBuffer(&lineBuffer, emit, &output, &summaryBuilder, &finalReviewBuilder, fallbackFile)
	if err != nil {
		return aiReviewOutput{}, err
	}
	parsedEvents += parsed

	if parsedEvents == 0 {
		output, err := parseAIReviewOutput(content.String())
		if err != nil {
			return aiReviewOutput{}, err
		}
		if err := emitWholeReviewAsEvents(output, emit, fallbackFile); err != nil {
			return aiReviewOutput{}, err
		}
		return output, nil
	}

	output.Summary = summaryBuilder.String()
	output.FinalReview = finalReviewBuilder.String()
	return output, nil
}

func handleStreamDelta(delta string, emit StreamEmitter, output *aiReviewOutput, summaryBuilder *strings.Builder, finalReviewBuilder *strings.Builder, lineBuffer *strings.Builder, fallbackFile string) (int, error) {
	lineBuffer.WriteString(delta)
	completeLines := strings.Split(lineBuffer.String(), "\n")
	lineBuffer.Reset()
	lineBuffer.WriteString(completeLines[len(completeLines)-1])

	parsedEvents := 0
	for _, eventLine := range completeLines[:len(completeLines)-1] {
		parsed, handled, err := handleReviewStreamEventLine(eventLine, emit, output, summaryBuilder, finalReviewBuilder, fallbackFile)
		if err != nil {
			return 0, err
		}
		if handled {
			parsedEvents += parsed
		}
	}
	return parsedEvents, nil
}

func flushStreamLineBuffer(lineBuffer *strings.Builder, emit StreamEmitter, output *aiReviewOutput, summaryBuilder *strings.Builder, finalReviewBuilder *strings.Builder, fallbackFile string) (int, error) {
	if strings.TrimSpace(lineBuffer.String()) == "" {
		return 0, nil
	}

	parsed, handled, err := handleReviewStreamEventLine(lineBuffer.String(), emit, output, summaryBuilder, finalReviewBuilder, fallbackFile)
	if err != nil {
		return 0, err
	}
	if !handled {
		return 0, nil
	}
	return parsed, nil
}

func handleReviewStreamEventLine(line string, emit StreamEmitter, output *aiReviewOutput, summaryBuilder *strings.Builder, finalReviewBuilder *strings.Builder, fallbackFile string) (int, bool, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return 0, false, nil
	}

	event, err := parseReviewStreamEvent(line)
	if err != nil {
		return 0, false, err
	}
	switch event.Type {
	case "summary_delta":
		if event.Text == "" {
			return 0, true, nil
		}
		summaryBuilder.WriteString(event.Text)
		return 1, true, emit("summary_delta", models.ReviewTextDeltaEvent{Text: event.Text})
	case "risk":
		risk := normalizeRisk(event.Risk, fallbackFile)
		output.Risks = append(output.Risks, risk)
		return 1, true, emit("risk", models.ReviewRiskEvent{Risk: risk})
	case "review_comment":
		comment := normalizeReviewComment(event.Comment, fallbackFile)
		output.ReviewComments = append(output.ReviewComments, comment)
		return 1, true, emit("review_comment", models.ReviewCommentEvent{Comment: comment})
	case "final_review_delta":
		if event.Text == "" {
			return 0, true, nil
		}
		finalReviewBuilder.WriteString(event.Text)
		return 1, true, emit("final_review_delta", models.ReviewTextDeltaEvent{Text: event.Text})
	default:
		return 0, true, nil
	}
}

func parseReviewStreamEvent(line string) (aiReviewStreamEvent, error) {
	var event aiReviewStreamEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return aiReviewStreamEvent{}, ServiceError{
			Kind:    ErrorKindAIUnavailable,
			Message: "Unable to parse AI review event.",
			Err:     err,
		}
	}
	return event, nil
}

func emitWholeReviewAsEvents(output aiReviewOutput, emit StreamEmitter, fallbackFile string) error {
	if output.Summary != "" {
		if err := emit("summary_delta", models.ReviewTextDeltaEvent{Text: output.Summary}); err != nil {
			return err
		}
	}
	for _, risk := range output.Risks {
		risk = normalizeRisk(risk, fallbackFile)
		if err := emit("risk", models.ReviewRiskEvent{Risk: risk}); err != nil {
			return err
		}
	}
	for _, comment := range output.ReviewComments {
		comment = normalizeReviewComment(comment, fallbackFile)
		if err := emit("review_comment", models.ReviewCommentEvent{Comment: comment}); err != nil {
			return err
		}
	}
	if output.FinalReview != "" {
		if err := emit("final_review_delta", models.ReviewTextDeltaEvent{Text: output.FinalReview}); err != nil {
			return err
		}
	}
	return nil
}

func parseStreamDelta(data string) (string, error) {
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return "", ServiceError{
			Kind:    ErrorKindAIUnavailable,
			Message: "Unable to parse AI streaming chunk.",
			Err:     err,
		}
	}
	if len(chunk.Choices) == 0 {
		return "", nil
	}
	return chunk.Choices[0].Delta.Content, nil
}
