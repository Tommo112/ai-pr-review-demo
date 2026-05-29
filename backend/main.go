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
	return setupRouterWithFetcher(newGitHubClient())
}

func setupRouterWithFetcher(fetcher pullRequestFetcher) *gin.Engine {
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

		riskFile := "unknown"
		if len(pr.Files) > 0 {
			riskFile = pr.Files[0].Filename
		}

		c.JSON(http.StatusOK, gin.H{
			"pr": gin.H{
				"title":         pr.Title,
				"author":        pr.Author,
				"files_changed": pr.FilesChanged,
				"additions":     pr.Additions,
				"deletions":     pr.Deletions,
			},
			"files":   pr.Files,
			"summary": "已从 GitHub 获取 PR 基本信息和变更文件，AI 分析会在下一步接入。",
			"risks": []gin.H{
				{
					"level":       "medium",
					"file":        riskFile,
					"line":        1,
					"title":       "待 AI 分析",
					"description": "当前结果来自 GitHub API，尚未经过模型审查。",
					"suggestion":  "下一步接入 AI 后，用真实 diff 生成风险判断。",
				},
			},
			"review_comments": []gin.H{
				{
					"file":    riskFile,
					"line":    1,
					"comment": "GitHub 数据获取已完成，等待 AI 生成具体 review 建议。",
				},
			},
			"final_review": "已读取 " + ref.Owner + "/" + ref.Repo + "#" + strconv.Itoa(ref.Number) + "，共 " + strconv.Itoa(pr.FilesChanged) + " 个文件变更。",
		})
	})

	return router
}
