package main

import "testing"

func TestParsePRURL(t *testing.T) {
	ref, err := parsePRURL("https://github.com/openai/codex/pull/123")
	if err != nil {
		t.Fatalf("expected valid PR URL, got error: %v", err)
	}

	if ref.Owner != "openai" || ref.Repo != "codex" || ref.Number != 123 {
		t.Fatalf("unexpected PR ref: %+v", ref)
	}
}

func TestParsePRURLWithQueryAndWhitespace(t *testing.T) {
	ref, err := parsePRURL(" https://github.com/owner/repo/pull/42?tab=files ")
	if err != nil {
		t.Fatalf("expected valid PR URL, got error: %v", err)
	}

	if ref.Owner != "owner" || ref.Repo != "repo" || ref.Number != 42 {
		t.Fatalf("unexpected PR ref: %+v", ref)
	}
}

func TestParsePRURLRejectsInvalidURL(t *testing.T) {
	invalidURLs := []string{
		"",
		"http://github.com/owner/repo/pull/1",
		"https://example.com/owner/repo/pull/1",
		"https://github.com/owner/repo/issues/1",
		"https://github.com/owner/repo/pull/0",
		"https://github.com/owner/repo/pull/not-a-number",
	}

	for _, rawURL := range invalidURLs {
		if _, err := parsePRURL(rawURL); err == nil {
			t.Fatalf("expected %q to be rejected", rawURL)
		}
	}
}
