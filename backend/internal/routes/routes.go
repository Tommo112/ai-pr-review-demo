package routes

import (
	"demo/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

func Register(router *gin.Engine, reviewHandler *handlers.ReviewHandler) {
	router.GET("/health", reviewHandler.Health)
	router.POST("/api/review", reviewHandler.Review)
}
