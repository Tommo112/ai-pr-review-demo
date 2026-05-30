package app

import (
	"net/http"

	"demo/backend/internal/config"
	"demo/backend/internal/handlers"
	"demo/backend/internal/routes"
	"demo/backend/internal/services"

	"github.com/gin-gonic/gin"
)

func New(cfg config.Config) *gin.Engine {
	fetcher := services.NewGitHubClient(cfg)
	analyzer := services.NewPRAnalyzer(cfg)
	reviewService := services.NewReviewService(fetcher, analyzer)
	reviewHandler := handlers.NewReviewHandler(reviewService, cfg)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), corsMiddleware())
	routes.Register(router, reviewHandler)

	return router
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Type")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
