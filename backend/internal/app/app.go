package app

import (
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
	reviewHandler := handlers.NewReviewHandler(reviewService)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	routes.Register(router, reviewHandler)

	return router
}
