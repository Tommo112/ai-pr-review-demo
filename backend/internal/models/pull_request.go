package models

type PRRef struct {
	Owner  string
	Repo   string
	Number int
}

type PullRequestData struct {
	Title        string
	Author       string
	FilesChanged int
	Additions    int
	Deletions    int
	Files        []PullRequestFile
}

type PullRequestFile struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch"`
}
