package main

import "strconv"

type reviewRequest struct {
	PRURL string `json:"pr_url" binding:"required"`
}

type reviewResponse struct {
	PR             prInfo            `json:"pr"`
	Files          []pullRequestFile `json:"files"`
	Summary        string            `json:"summary"`
	Risks          []risk            `json:"risks"`
	ReviewComments []reviewComment   `json:"review_comments"`
	FinalReview    string            `json:"final_review"`
}

type prInfo struct {
	Title        string `json:"title"`
	Author       string `json:"author"`
	FilesChanged int    `json:"files_changed"`
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
}

type risk struct {
	Level       string `json:"level"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

type reviewComment struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Comment string `json:"comment"`
}

func newPendingAIReviewResponse(ref prRef, pr pullRequestData) reviewResponse {
	response := newReviewResponseFromGitHub(ref, pr)
	riskFile := "unknown"
	if len(pr.Files) > 0 {
		riskFile = pr.Files[0].Filename
	}

	response.Summary = "已从 GitHub 获取 PR 基本信息和变更文件；配置 OPENAI_API_KEY 与 OPENAI_MODEL 后会启用 AI 分析。"
	response.Risks = []risk{
		{
			Level:       "medium",
			File:        riskFile,
			Line:        1,
			Title:       "待 AI 分析",
			Description: "当前结果来自 GitHub API，尚未经过模型审查。",
			Suggestion:  "配置 AI 后，用真实 diff 生成风险判断。",
		},
	}
	response.ReviewComments = []reviewComment{
		{
			File:    riskFile,
			Line:    1,
			Comment: "GitHub 数据获取已完成，等待 AI 生成具体 review 建议。",
		},
	}
	response.FinalReview = "已读取 " + ref.Owner + "/" + ref.Repo + "#" + strconv.Itoa(ref.Number) + "，共 " + strconv.Itoa(pr.FilesChanged) + " 个文件变更。"
	return response
}

func newReviewResponseFromGitHub(_ prRef, pr pullRequestData) reviewResponse {
	return reviewResponse{
		PR: prInfo{
			Title:        pr.Title,
			Author:       pr.Author,
			FilesChanged: pr.FilesChanged,
			Additions:    pr.Additions,
			Deletions:    pr.Deletions,
		},
		Files: pr.Files,
	}
}
