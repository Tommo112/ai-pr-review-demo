package config

import "testing"

func TestLoadDefaultConfig(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_MODEL", "")
	t.Setenv("OPENAI_BASE_URL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config, got error: %v", err)
	}

	if cfg.Port != "8080" {
		t.Fatalf("unexpected default port: %q", cfg.Port)
	}
	if cfg.OpenAIBaseURL != "https://api.openai.com/v1" {
		t.Fatalf("unexpected default OpenAI base URL: %q", cfg.OpenAIBaseURL)
	}
}

func TestLoadEnvConfig(t *testing.T) {
	t.Setenv("PORT", "9000")
	t.Setenv("GITHUB_TOKEN", "github-token")
	t.Setenv("OPENAI_API_KEY", "ai-key")
	t.Setenv("OPENAI_MODEL", "model")
	t.Setenv("OPENAI_BASE_URL", "http://example.test/v1")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config, got error: %v", err)
	}

	if cfg.Port != "9000" || cfg.GitHubToken != "github-token" || cfg.OpenAIAPIKey != "ai-key" || cfg.OpenAIModel != "model" || cfg.OpenAIBaseURL != "http://example.test/v1" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
