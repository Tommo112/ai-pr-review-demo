package handlers

import (
	"errors"
	"net/http"

	"demo/backend/internal/models"
	"demo/backend/internal/services"

	"github.com/gin-gonic/gin"
)

type ReviewHandler struct {
	service *services.ReviewService
}

func NewReviewHandler(service *services.ReviewService) *ReviewHandler {
	return &ReviewHandler{service: service}
}

func (handler *ReviewHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (handler *ReviewHandler) Review(c *gin.Context) {
	var req models.ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pr_url is required"})
		return
	}

	response, err := handler.service.Review(c.Request.Context(), req.PRURL)
	if err != nil {
		if errors.Is(err, services.ErrInvalidPRURL) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pr_url must match https://github.com/{owner}/{repo}/pull/{number}"})
			return
		}

		var serviceErr services.ServiceError
		if errors.As(err, &serviceErr) {
			c.JSON(statusForServiceError(serviceErr), gin.H{"error": serviceErr.Message})
			return
		}

		c.JSON(http.StatusBadGateway, gin.H{"error": "服务暂时无法完成 PR Review，请稍后重试"})
		return
	}

	c.JSON(http.StatusOK, response)
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
