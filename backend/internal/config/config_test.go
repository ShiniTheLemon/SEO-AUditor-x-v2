package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsDotEnvFile(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_MODEL", "")
	t.Setenv("OPENAI_BASE_URL", "")

	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	content := []byte("APP_ENV=test\nHTTP_ADDR=:9090\nDATABASE_URL=postgresql://example\nOPENAI_API_KEY=secret\nPAGESPEED_API_KEY=pagespeed-secret\nPAGESPEED_STRATEGY=desktop\n")

	if err := os.WriteFile(envPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	tomlContent := []byte(`
model_provider = "sub2api"
model = "gpt-5.5"

[model_providers.sub2api]
name = "sub2api"
base_url = "https://sub2api.llmapp.org"
wire_api = "responses"
`)
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), tomlContent, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.AppEnv != "test" {
		t.Fatalf("expected AppEnv test, got %q", cfg.AppEnv)
	}

	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("expected HTTPAddr :9090, got %q", cfg.HTTPAddr)
	}

	if cfg.DatabaseURL != "postgresql://example" {
		t.Fatalf("expected DatabaseURL from .env, got %q", cfg.DatabaseURL)
	}

	if cfg.OpenAIAPIKey != "secret" {
		t.Fatalf("expected OpenAIAPIKey from .env, got %q", cfg.OpenAIAPIKey)
	}

	if cfg.OpenAIModel != "gpt-5.5" {
		t.Fatalf("expected OpenAIModel from .env, got %q", cfg.OpenAIModel)
	}

	if cfg.OpenAIBaseURL != "https://sub2api.llmapp.org" {
		t.Fatalf("expected OpenAIBaseURL from .env, got %q", cfg.OpenAIBaseURL)
	}

	if cfg.PageSpeedKey != "pagespeed-secret" {
		t.Fatalf("expected PageSpeedKey from .env, got %q", cfg.PageSpeedKey)
	}

	if cfg.PageSpeedMode != "desktop" {
		t.Fatalf("expected PageSpeedMode from .env, got %q", cfg.PageSpeedMode)
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("OPENAI_API_KEY", "secret")

	_, err := Load(filepath.Join(t.TempDir(), ".env"))
	if err == nil {
		t.Fatal("expected DATABASE_URL validation error")
	}
}

func TestLoadRequiresOpenAIAPIKey(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://example")
	t.Setenv("OPENAI_API_KEY", "")

	_, err := Load(filepath.Join(t.TempDir(), ".env"))
	if err == nil {
		t.Fatal("expected OPENAI_API_KEY validation error")
	}
}

func TestLoadAllowsEnvToOverrideModelConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://example")
	t.Setenv("OPENAI_API_KEY", "secret")
	t.Setenv("OPENAI_MODEL", "env-model")
	t.Setenv("OPENAI_BASE_URL", "https://env.example/v1")

	cfg, err := Load(filepath.Join(t.TempDir(), ".env"))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.OpenAIModel != "env-model" {
		t.Fatalf("expected env model override, got %q", cfg.OpenAIModel)
	}

	if cfg.OpenAIBaseURL != "https://env.example/v1" {
		t.Fatalf("expected env base URL override, got %q", cfg.OpenAIBaseURL)
	}
}
