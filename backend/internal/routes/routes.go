package routes

import (
	"demo/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

func Register(router *gin.Engine, reviewHandler *handlers.ReviewHandler) {
	router.GET("/health", reviewHandler.Health)
	router.GET("/api/status", reviewHandler.RuntimeStatus)
	router.POST("/api/review", reviewHandler.Review)
	router.GET("/api/review/stream", reviewHandler.ReviewStream)
	router.POST("/api/review/stream", reviewHandler.ReviewStream)
}
