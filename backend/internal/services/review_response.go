package services

import (
	"strconv"

	"demo/backend/internal/models"
)

func newPendingAIReviewResponse(ref models.PRRef, pr models.PullRequestData) models.ReviewResponse {
	response := newReviewResponseFromGitHub(pr)
	response.Summary = "已从 GitHub 获取 PR 基本信息和变更文件；配置 OPENAI_API_KEY 与 OPENAI_MODEL 后会启用 AI 分析。"
	setPendingAIDefaults(&response, ref)
	ensureReviewDefaults(&response, ref)
	return response
}

func newReviewResponseFromGitHub(pr models.PullRequestData) models.ReviewResponse {
	return models.ReviewResponse{
		PR: models.PRInfo{
			Title:        pr.Title,
			Author:       pr.Author,
			FilesChanged: pr.FilesChanged,
			Additions:    pr.Additions,
			Deletions:    pr.Deletions,
		},
		Files: pr.Files,
	}
}

func ensureReviewDefaults(response *models.ReviewResponse, ref models.PRRef) {
	riskFile := "unknown"
	if len(response.Files) > 0 {
		riskFile = response.Files[0].Filename
	}

	if response.Summary == "" {
		response.Summary = "已读取 PR 变更，暂无模型总结。"
	}
	if len(response.Risks) == 0 {
		response.Risks = []models.Risk{
			{
				Level:       "low",
				File:        riskFile,
				Line:        1,
				Title:       "暂无明确风险",
				Description: "当前未发现结构化风险项，建议仍结合 diff 人工确认关键路径。",
				Suggestion:  "重点检查异常处理、权限边界、并发路径和测试覆盖。",
			},
		}
	}
	if len(response.ReviewComments) == 0 {
		response.ReviewComments = []models.ReviewComment{
			{
				File:    riskFile,
				Line:    1,
				Comment: "暂无具体行级建议，请结合 diff 做最终确认。",
			},
		}
	}
	if response.FinalReview == "" {
		response.FinalReview = "已读取 " + ref.Owner + "/" + ref.Repo + "#" + strconv.Itoa(ref.Number) + "，共 " + strconv.Itoa(response.PR.FilesChanged) + " 个文件变更。"
	}
}

func setPendingAIDefaults(response *models.ReviewResponse, ref models.PRRef) {
	riskFile := "unknown"
	if len(response.Files) > 0 {
		riskFile = response.Files[0].Filename
	}

	response.Risks = []models.Risk{
		{
			Level:       "medium",
			File:        riskFile,
			Line:        1,
			Title:       "待 AI 分析",
			Description: "当前结果来自 GitHub API，尚未经过模型审查。",
			Suggestion:  "配置 AI 后，用真实 diff 生成风险判断。",
		},
	}
	response.ReviewComments = []models.ReviewComment{
		{
			File:    riskFile,
			Line:    1,
			Comment: "GitHub 数据获取已完成，等待 AI 生成具体 review 建议。",
		},
	}
	response.FinalReview = "已读取 " + ref.Owner + "/" + ref.Repo + "#" + strconv.Itoa(ref.Number) + "，共 " + strconv.Itoa(response.PR.FilesChanged) + " 个文件变更。"
}
