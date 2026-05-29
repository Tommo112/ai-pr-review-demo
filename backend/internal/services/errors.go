package services

import "fmt"

type ErrorKind string

const (
	ErrorKindGitHubNotFound     ErrorKind = "github_not_found"
	ErrorKindGitHubUnauthorized ErrorKind = "github_unauthorized"
	ErrorKindGitHubRateLimited  ErrorKind = "github_rate_limited"
	ErrorKindGitHubUnavailable  ErrorKind = "github_unavailable"
	ErrorKindAIUnavailable      ErrorKind = "ai_unavailable"
)

type ServiceError struct {
	Kind    ErrorKind
	Message string
	Err     error
}

func (err ServiceError) Error() string {
	if err.Err == nil {
		return err.Message
	}
	return fmt.Sprintf("%s: %v", err.Message, err.Err)
}

func (err ServiceError) Unwrap() error {
	return err.Err
}
