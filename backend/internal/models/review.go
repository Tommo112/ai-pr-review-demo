package models

type ReviewRequest struct {
	PRURL string `json:"pr_url" binding:"required"`
}

type ReviewResponse struct {
	PR             PRInfo            `json:"pr"`
	Files          []PullRequestFile `json:"files"`
	Summary        string            `json:"summary"`
	Risks          []Risk            `json:"risks"`
	ReviewComments []ReviewComment   `json:"review_comments"`
	FinalReview    string            `json:"final_review"`
}

type PRInfo struct {
	Title        string `json:"title"`
	Author       string `json:"author"`
	FilesChanged int    `json:"files_changed"`
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
}

type Risk struct {
	Level       string `json:"level"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

type ReviewComment struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Comment string `json:"comment"`
}
