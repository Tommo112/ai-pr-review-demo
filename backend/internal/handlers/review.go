package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"demo/backend/internal/config"
	"demo/backend/internal/models"
	"demo/backend/internal/services"

	"github.com/gin-gonic/gin"
)

type ReviewHandler struct {
	service *services.ReviewService
	cfg     config.Config
}

func NewReviewHandler(service *services.ReviewService, cfg config.Config) *ReviewHandler {
	return &ReviewHandler{service: service, cfg: cfg}
}

func (handler *ReviewHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (handler *ReviewHandler) RuntimeStatus(c *gin.Context) {
	apiKeyConfigured := handler.cfg.OpenAIAPIKey != ""
	modelConfigured := handler.cfg.OpenAIModel != ""
	c.JSON(http.StatusOK, models.RuntimeStatusResponse{
		Port: handler.cfg.Port,
		GitHub: models.GitHubStatus{
			TokenConfigured: handler.cfg.GitHubToken != "",
		},
		AI: models.AIStatus{
			Enabled:          apiKeyConfigured && modelConfigured,
			APIKeyConfigured: apiKeyConfigured,
			ModelConfigured:  modelConfigured,
			BaseURL:          handler.cfg.OpenAIBaseURL,
		},
	})
}

func (handler *ReviewHandler) Review(c *gin.Context) {
	var req models.ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeJSONError(c, http.StatusBadRequest, "pr_url is required", "missing_pr_url")
		return
	}

	response, err := handler.service.Review(c.Request.Context(), req.PRURL)
	if err != nil {
		status, message, code := responseForError(err)
		writeJSONError(c, status, message, code)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (handler *ReviewHandler) ReviewStream(c *gin.Context) {
	prURL, ok := streamPRURL(c)
	if !ok {
		writeJSONError(c, http.StatusBadRequest, "pr_url is required", "missing_pr_url")
		return
	}

	setSSEHeaders(c)

	writeSSE(c, "status", models.ReviewStatusEvent{Message: "fetching_pr"})
	err := handler.service.ReviewStream(c.Request.Context(), prURL, func(event string, data any) error {
		return writeSSE(c, event, data)
	})
	if err != nil {
		_ = writeSSE(c, "error", models.ReviewErrorEvent{
			Error: messageForError(err),
			Code:  codeForError(err),
		})
		return
	}

	_ = writeSSE(c, "done", gin.H{"ok": true})
}

func streamPRURL(c *gin.Context) (string, bool) {
	if c.Request.Method == http.MethodGet {
		prURL := c.Query("pr_url")
		return prURL, prURL != ""
	}

	var req models.ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return "", false
	}
	return req.PRURL, req.PRURL != ""
}

func setSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
}

func writeJSONError(c *gin.Context, status int, message string, code string) {
	c.JSON(status, gin.H{
		"error": message,
		"code":  code,
	})
}

func responseForError(err error) (int, string, string) {
	if errors.Is(err, services.ErrInvalidPRURL) {
		return http.StatusBadRequest, "pr_url must match https://github.com/{owner}/{repo}/pull/{number}", "invalid_pr_url"
	}

	var serviceErr services.ServiceError
	if errors.As(err, &serviceErr) {
		return statusForServiceError(serviceErr), serviceErr.Message, string(serviceErr.Kind)
	}

	return http.StatusBadGateway, "Unable to complete PR review. Please retry later.", "review_failed"
}

func statusForServiceError(err services.ServiceError) int {
	switch err.Kind {
	case services.ErrorKindGitHubNotFound:
		return http.StatusNotFound
	case services.ErrorKindGitHubUnauthorized:
		return http.StatusForbidden
	case services.ErrorKindGitHubRateLimited:
		return http.StatusTooManyRequests
	case services.ErrorKindGitHubUnavailable, services.ErrorKindAIUnavailable:
		return http.StatusBadGateway
	default:
		return http.StatusBadGateway
	}
}

func writeSSE(c *gin.Context, event string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, payload); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

func messageForError(err error) string {
	_, message, _ := responseForError(err)
	return message
}

func codeForError(err error) string {
	_, _, code := responseForError(err)
	return code
}
