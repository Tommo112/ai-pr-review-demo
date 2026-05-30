package services

import (
	"strconv"
	"strings"

	"demo/backend/internal/models"
)

func buildReviewPrompt(ref models.PRRef, pr models.PullRequestData) string {
	var builder strings.Builder
	builder.WriteString("Review this GitHub pull request and return JSON with exactly these keys: summary, risks, review_comments, final_review.\n")
	builder.WriteString("The JSON shape must be: {\"summary\":\"...\",\"risks\":[{\"level\":\"high|medium|low\",\"file\":\"path\",\"line\":1,\"title\":\"...\",\"description\":\"...\",\"suggestion\":\"...\"}],\"review_comments\":[{\"file\":\"path\",\"line\":1,\"comment\":\"...\"}],\"final_review\":\"...\"}.\n")
	builder.WriteString("Rules:\n")
	builder.WriteString("- Return only JSON. Do not wrap the output in markdown.\n")
	builder.WriteString("- Risk level must be one of high, medium, low.\n")
	builder.WriteString("- Ground every risk in the supplied diff. If a line number is unclear, use 1.\n")
	builder.WriteString("- Prefer actionable review comments over broad style advice.\n")
	builder.WriteString("- Use concise Chinese review language for all user-facing text.\n")
	builder.WriteString("- Make summary detailed enough for a reviewer who has not read the diff. Include overall overview, concrete changes, affected areas, and key points to verify.\n")
	builder.WriteString("- Make final_review suitable for posting as a PR comment. Include a brief conclusion plus the most important risks or follow-up checks.\n")
	builder.WriteString("Review priorities:\n")
	builder.WriteString("- Potential bugs or broken edge cases.\n")
	builder.WriteString("- Security or permission risks.\n")
	builder.WriteString("- Performance or resource usage regressions.\n")
	builder.WriteString("- Missing error handling or fallback behavior.\n")
	builder.WriteString("- Missing tests for changed behavior.\n")
	builder.WriteString("- Maintainability, API compatibility, and migration risks.\n\n")
	writePRContext(&builder, ref, pr)
	return builder.String()
}

func buildStreamingReviewPrompt(ref models.PRRef, pr models.PullRequestData) string {
	var builder strings.Builder
	builder.WriteString("Review this GitHub pull request and stream newline-delimited JSON events.\n")
	builder.WriteString("Each line must be one complete JSON object. Do not return markdown or prose outside JSON.\n")
	builder.WriteString("Allowed event shapes:\n")
	builder.WriteString("{\"type\":\"summary_delta\",\"text\":\"...\"}\n")
	builder.WriteString("{\"type\":\"risk\",\"risk\":{\"level\":\"high|medium|low\",\"file\":\"path\",\"line\":1,\"title\":\"...\",\"description\":\"...\",\"suggestion\":\"...\"}}\n")
	builder.WriteString("{\"type\":\"review_comment\",\"comment\":{\"file\":\"path\",\"line\":1,\"comment\":\"...\"}}\n")
	builder.WriteString("{\"type\":\"final_review_delta\",\"text\":\"...\"}\n")
	builder.WriteString("Rules:\n")
	builder.WriteString("- Use summary_delta and final_review_delta for readable text chunks.\n")
	builder.WriteString("- Emit each risk only after the whole risk object is ready.\n")
	builder.WriteString("- Emit each review_comment only after the whole comment object is ready.\n")
	builder.WriteString("- Risk level must be one of high, medium, low.\n")
	builder.WriteString("- Ground every risk in the supplied diff. If a line number is unclear, use 1.\n")
	builder.WriteString("- Use concise Chinese review language for all user-facing text.\n")
	builder.WriteString("- Use summary_delta to produce a detailed summary with overall overview, concrete changes, affected areas, and key points to verify.\n")
	builder.WriteString("- Use final_review_delta to produce a PR-comment-ready conclusion with the most important risks or follow-up checks.\n")
	builder.WriteString("Review priorities: bugs, security or permission risks, performance regressions, missing error handling, missing tests, maintainability, API compatibility, and migration risks.\n\n")
	writePRContext(&builder, ref, pr)
	return builder.String()
}

func writePRContext(builder *strings.Builder, ref models.PRRef, pr models.PullRequestData) {
	builder.WriteString("PR: ")
	builder.WriteString(ref.Owner + "/" + ref.Repo + "#" + strconv.Itoa(ref.Number) + "\n")
	builder.WriteString("Title: " + pr.Title + "\n")
	builder.WriteString("Author: " + pr.Author + "\n")
	builder.WriteString("Stats: +" + strconv.Itoa(pr.Additions) + " -" + strconv.Itoa(pr.Deletions) + ", files " + strconv.Itoa(pr.FilesChanged) + "\n\n")
	builder.WriteString("Files and patches:\n")
	builder.WriteString(TrimDiffForPrompt(pr.Files, 12000))
}

func TrimDiffForPrompt(files []models.PullRequestFile, maxChars int) string {
	var builder strings.Builder
	for _, file := range files {
		if builder.Len() >= maxChars {
			break
		}
		if isLowValueDiffFile(file.Filename) {
			continue
		}

		builder.WriteString("\n--- ")
		builder.WriteString(file.Filename)
		builder.WriteString(" (")
		builder.WriteString(file.Status)
		builder.WriteString(", +")
		builder.WriteString(strconv.Itoa(file.Additions))
		builder.WriteString(" -")
		builder.WriteString(strconv.Itoa(file.Deletions))
		builder.WriteString(") ---\n")

		patch := file.Patch
		if patch == "" {
			patch = "(no patch available)"
		}
		remaining := maxChars - builder.Len()
		if remaining <= 0 {
			break
		}
		if len(patch) > remaining {
			builder.WriteString(patch[:remaining])
			builder.WriteString("\n[diff truncated]\n")
			break
		}
		builder.WriteString(patch)
		builder.WriteString("\n")
	}

	return builder.String()
}

func isLowValueDiffFile(filename string) bool {
	lower := strings.ToLower(filename)
	if strings.HasSuffix(lower, ".lock") ||
		strings.HasSuffix(lower, ".sum") ||
		strings.HasSuffix(lower, ".min.js") ||
		strings.HasSuffix(lower, ".map") ||
		strings.HasSuffix(lower, ".png") ||
		strings.HasSuffix(lower, ".jpg") ||
		strings.HasSuffix(lower, ".jpeg") ||
		strings.HasSuffix(lower, ".gif") ||
		strings.HasSuffix(lower, ".svg") ||
		strings.HasSuffix(lower, ".webp") ||
		strings.HasSuffix(lower, ".ico") {
		return true
	}

	base := lower
	if slash := strings.LastIndexAny(base, `/\`); slash >= 0 {
		base = base[slash+1:]
	}

	switch base {
	case "package-lock.json", "pnpm-lock.yaml", "yarn.lock", "bun.lock", "go.sum":
		return true
	default:
		return false
	}
}
