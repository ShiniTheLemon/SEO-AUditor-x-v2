package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	AppEnv        string
	HTTPAddr      string
	DatabaseURL   string
	OpenAIAPIKey  string
	OpenAIModel   string
	OpenAIBaseURL string
	PageSpeedKey  string
	PageSpeedMode string
}

type modelConfigFile struct {
	ModelProvider        string                         `toml:"model_provider"`
	Model                string                         `toml:"model"`
	DisableResponseStore bool                           `toml:"disable_response_storage"`
	PlanReasoningEffort  string                         `toml:"plan_mode_reasoning_effort"`
	ModelProviders       map[string]modelProviderConfig `toml:"model_providers"`
}

type modelProviderConfig struct {
	Name    string `toml:"name"`
	BaseURL string `toml:"base_url"`
	WireAPI string `toml:"wire_api"`
}

func Load(path string) (Config, error) {
	if path == "" {
		path = ".env"
	}

	if err := loadDotEnv(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}

	modelConfig, err := loadModelConfig(filepath.Join(filepath.Dir(path), "config.toml"))
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppEnv:        getEnv("APP_ENV", "development"),
		HTTPAddr:      getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		OpenAIAPIKey:  os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:   firstNonEmpty(os.Getenv("OPENAI_MODEL"), modelConfig.Model, "gpt-4.1-mini"),
		OpenAIBaseURL: firstNonEmpty(os.Getenv("OPENAI_BASE_URL"), selectedProviderBaseURL(modelConfig), "https://jp.gpt.ge/v1"),
		PageSpeedKey:  os.Getenv("PAGESPEED_API_KEY"),
		PageSpeedMode: getEnv("PAGESPEED_STRATEGY", "mobile"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	if cfg.OpenAIAPIKey == "" {
		return Config{}, errors.New("OPENAI_API_KEY is required")
	}

	return cfg, nil
}

func loadModelConfig(path string) (modelConfigFile, error) {
	var config modelConfigFile
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config, nil
		}

		return config, err
	}

	if err := toml.Unmarshal(content, &config); err != nil {
		return config, err
	}

	return config, nil
}

func selectedProviderBaseURL(config modelConfigFile) string {
	provider, ok := config.ModelProviders[config.ModelProvider]
	if !ok {
		return ""
	}

	if provider.WireAPI != "" && provider.WireAPI != "responses" {
		return ""
	}

	return provider.BaseURL
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("invalid .env line %d", lineNumber)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)

		if key == "" {
			return fmt.Errorf("invalid .env line %d", lineNumber)
		}

		currentValue, exists := os.LookupEnv(key)
		if !exists || currentValue == "" {
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
	}

	return scanner.Err()
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}
