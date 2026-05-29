package main

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
)

type prRef struct {
	Owner  string
	Repo   string
	Number int
}

func parsePRURL(rawURL string) (prRef, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return prRef{}, err
	}

	if parsed.Scheme != "https" || parsed.Hostname() != "github.com" {
		return prRef{}, errors.New("pr_url must be a GitHub pull request URL")
	}

	parts := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	if len(parts) != 4 || parts[2] != "pull" {
		return prRef{}, errors.New("pr_url must match https://github.com/{owner}/{repo}/pull/{number}")
	}

	number, err := strconv.Atoi(parts[3])
	if err != nil || number <= 0 {
		return prRef{}, errors.New("pull request number must be a positive integer")
	}

	owner, err := url.PathUnescape(parts[0])
	if err != nil {
		return prRef{}, err
	}

	repo, err := url.PathUnescape(parts[1])
	if err != nil {
		return prRef{}, err
	}

	if owner == "" || repo == "" {
		return prRef{}, errors.New("owner and repo are required")
	}

	return prRef{Owner: owner, Repo: repo, Number: number}, nil
}
