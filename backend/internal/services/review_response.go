package services

import (
	"strconv"
	"strings"

	"demo/backend/internal/models"
)

func newPendingAIReviewResponse(ref models.PRRef, pr models.PullRequestData) models.ReviewResponse {
	response := newReviewResponseFromGitHub(pr)
	response.Summary = "GitHub PR metadata and changed files were fetched. Configure OPENAI_API_KEY and OPENAI_MODEL to enable AI review."
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
	riskFile := fallbackReviewFile(response.Files)

	if response.Summary == "" {
		response.Summary = "PR changes were loaded, but no model summary was returned."
	}
	if len(response.Risks) == 0 {
		response.Risks = []models.Risk{
			{
				Level:       "low",
				File:        riskFile,
				Line:        1,
				Title:       "No structured risk returned",
				Description: "No structured risk was returned by the analyzer. Review the diff manually before merging.",
				Suggestion:  "Check error handling, permission boundaries, concurrency paths, and test coverage.",
			},
		}
	}
	if len(response.ReviewComments) == 0 {
		response.ReviewComments = []models.ReviewComment{
			{
				File:    riskFile,
				Line:    1,
				Comment: "No specific inline suggestion was returned. Confirm the final decision against the diff.",
			},
		}
	}
	if response.FinalReview == "" {
		response.FinalReview = fallbackFinalReview(ref, response.PR.FilesChanged)
	}
	for i := range response.Risks {
		response.Risks[i] = normalizeRisk(response.Risks[i], riskFile)
	}
	for i := range response.ReviewComments {
		response.ReviewComments[i] = normalizeReviewComment(response.ReviewComments[i], riskFile)
	}
}

func setPendingAIDefaults(response *models.ReviewResponse, ref models.PRRef) {
	riskFile := fallbackReviewFile(response.Files)

	response.Risks = []models.Risk{
		{
			Level:       "medium",
			File:        riskFile,
			Line:        1,
			Title:       "AI review is not configured",
			Description: "This result is based on GitHub API data only and has not been reviewed by the model.",
			Suggestion:  "Configure AI settings to generate risk analysis from the real diff.",
		},
	}
	response.ReviewComments = []models.ReviewComment{
		{
			File:    riskFile,
			Line:    1,
			Comment: "GitHub data was fetched successfully. Configure AI settings to generate specific review suggestions.",
		},
	}
	response.FinalReview = fallbackFinalReview(ref, response.PR.FilesChanged)
}

func fallbackFinalReview(ref models.PRRef, filesChanged int) string {
	return "Fetched " + ref.Owner + "/" + ref.Repo + "#" + strconv.Itoa(ref.Number) + " with " + strconv.Itoa(filesChanged) + " changed files. Configure AI settings to generate a full review."
}

func fallbackReviewFile(files []models.PullRequestFile) string {
	if len(files) > 0 && strings.TrimSpace(files[0].Filename) != "" {
		return files[0].Filename
	}
	return "unknown"
}

func normalizeRisk(risk models.Risk, fallbackFile string) models.Risk {
	switch strings.ToLower(strings.TrimSpace(risk.Level)) {
	case "high":
		risk.Level = "high"
	case "medium":
		risk.Level = "medium"
	case "low":
		risk.Level = "low"
	default:
		risk.Level = "low"
	}
	if strings.TrimSpace(risk.File) == "" {
		risk.File = fallbackFile
	}
	if risk.Line <= 0 {
		risk.Line = 1
	}
	if strings.TrimSpace(risk.Title) == "" {
		risk.Title = "Review attention needed"
	}
	if strings.TrimSpace(risk.Description) == "" {
		risk.Description = "The analyzer returned an incomplete risk item. Confirm this area against the diff."
	}
	if strings.TrimSpace(risk.Suggestion) == "" {
		risk.Suggestion = "Review the related change manually and add tests or guards if needed."
	}
	return risk
}

func normalizeReviewComment(comment models.ReviewComment, fallbackFile string) models.ReviewComment {
	if strings.TrimSpace(comment.File) == "" {
		comment.File = fallbackFile
	}
	if comment.Line <= 0 {
		comment.Line = 1
	}
	if strings.TrimSpace(comment.Comment) == "" {
		comment.Comment = "Confirm this change against the diff before merging."
	}
	return comment
}
