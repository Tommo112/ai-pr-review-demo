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

type ReviewStartEvent struct {
	PR    PRInfo            `json:"pr"`
	Files []PullRequestFile `json:"files"`
}

type ReviewStatusEvent struct {
	Message string `json:"message"`
}

type ReviewTextDeltaEvent struct {
	Text string `json:"text"`
}

type ReviewRiskEvent struct {
	Risk Risk `json:"risk"`
}

type ReviewCommentEvent struct {
	Comment ReviewComment `json:"comment"`
}

type ReviewDoneEvent struct {
	Review ReviewResponse `json:"review"`
}

type ReviewErrorEvent struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

type RuntimeStatusResponse struct {
	Port   string       `json:"port"`
	GitHub GitHubStatus `json:"github"`
	AI     AIStatus     `json:"ai"`
}

type GitHubStatus struct {
	TokenConfigured bool `json:"token_configured"`
}

type AIStatus struct {
	Enabled          bool   `json:"enabled"`
	APIKeyConfigured bool   `json:"api_key_configured"`
	ModelConfigured  bool   `json:"model_configured"`
	BaseURL          string `json:"base_url"`
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
