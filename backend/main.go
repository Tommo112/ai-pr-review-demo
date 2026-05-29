package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	router := setupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := router.Run(":" + port); err != nil {
		panic(err)
	}
}

func setupRouter() *gin.Engine {
	return setupRouterWithServices(newGitHubClient(), newPRAnalyzer())
}

func setupRouterWithFetcher(fetcher pullRequestFetcher) *gin.Engine {
	return setupRouterWithServices(fetcher, fallbackAnalyzer{})
}

func setupRouterWithServices(fetcher pullRequestFetcher, analyzer prAnalyzer) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.POST("/api/review", func(c *gin.Context) {
		var req reviewRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pr_url is required"})
			return
		}

		ref, err := parsePRURL(req.PRURL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		pr, err := fetcher.FetchPullRequest(c.Request.Context(), ref)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch GitHub PR data: " + err.Error()})
			return
		}

		review, err := analyzer.AnalyzePullRequest(c.Request.Context(), ref, pr)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to analyze PR: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, review)
	})

	return router
}
