package batchreview

import (
	"fmt"
	"os"

	"github.com/voocel/litellm"
	"github.com/voocel/litellm/provider/gemini"
)

const modelReview = "gemini-3.6-flash"

type LLMProvider string

const (
	ProviderGemini LLMProvider = "gemini"
)

type LLMConfig struct {
	Provider LLMProvider
	APIKey   string
	BaseURL  string
}

func NewLLMClient(config LLMConfig) (*litellm.Client, error) {
	switch config.Provider {
	case ProviderGemini:
		if config.APIKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY not set")
		}
		client, err := gemini.NewClient(gemini.Config{APIKey: config.APIKey, BaseURL: config.BaseURL})
		if err != nil {
			return nil, fmt.Errorf("failed to build gemini client: %w", err)
		}
		return client, nil
	default:
		return nil, fmt.Errorf("unknown LLM provider %q", config.Provider)
	}
}

func ClientConfigFromEnv(provider LLMProvider) LLMConfig {
	switch provider {
	case ProviderGemini:
		return LLMConfig{
			Provider: provider,
			APIKey:   os.Getenv("GEMINI_API_KEY"),
		}
	default:
		return LLMConfig{Provider: provider}
	}
}
