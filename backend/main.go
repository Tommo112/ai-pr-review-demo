package main

import (
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

type reviewRequest struct {
	PRURL string `json:"pr_url" binding:"required"`
}

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

		c.JSON(http.StatusOK, gin.H{
			"pr": gin.H{
				"title":         "Mock PR analysis for " + ref.Owner + "/" + ref.Repo,
				"author":        "demo-user",
				"files_changed": 3,
				"additions":     128,
				"deletions":     36,
			},
			"summary": "Backend is connected. GitHub and AI analysis will be added in the next steps.",
			"risks": []gin.H{
				{
					"level":       "medium",
					"file":        "backend/main.go",
					"line":        1,
					"title":       "Mock risk item",
					"description": "This placeholder confirms the frontend can render structured review data.",
					"suggestion":  "Use the parsed PR reference to fetch GitHub data in the next step.",
				},
			},
			"review_comments": []gin.H{
				{
					"file":    "backend/main.go",
					"line":    1,
					"comment": "Mock review comment for frontend/backend integration.",
				},
			},
			"final_review": "Mock final review for " + ref.Owner + "/" + ref.Repo + "#" + strconv.Itoa(ref.Number),
		})
	})

	return router
}
