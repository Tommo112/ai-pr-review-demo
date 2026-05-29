package services

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"demo/backend/internal/models"
)

var ErrInvalidPRURL = errors.New("invalid github pull request url")

func ParsePRURL(rawURL string) (models.PRRef, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return models.PRRef{}, err
	}

	if parsed.Scheme != "https" || parsed.Hostname() != "github.com" {
		return models.PRRef{}, ErrInvalidPRURL
	}

	parts := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	if len(parts) != 4 || parts[2] != "pull" {
		return models.PRRef{}, ErrInvalidPRURL
	}

	number, err := strconv.Atoi(parts[3])
	if err != nil || number <= 0 {
		return models.PRRef{}, ErrInvalidPRURL
	}

	owner, err := url.PathUnescape(parts[0])
	if err != nil {
		return models.PRRef{}, err
	}
	repo, err := url.PathUnescape(parts[1])
	if err != nil {
		return models.PRRef{}, err
	}
	if owner == "" || repo == "" {
		return models.PRRef{}, ErrInvalidPRURL
	}

	return models.PRRef{Owner: owner, Repo: repo, Number: number}, nil
}
