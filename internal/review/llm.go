package review

import (
	"context"
	"fmt"
	"os"

	"github.com/voocel/litellm"
	"github.com/voocel/litellm/provider/gemini"
	"github.com/voocel/litellm/provider/openai"
)

const DefaultLiteLLMProxyBaseURL = "http://127.0.0.1:9177/v1"

const (
	modelVision = "qwen3-vl"         // local vision proxy: summarise + resolve
	modelReview = "gemini-3.6-flash" // final content review
)

type LLMProvider string

const (
	ProviderGemini       LLMProvider = "gemini"
	ProviderLiteLLMProxy LLMProvider = "litellm_proxy"
)

type ChatClient interface {
	Chat(context.Context, litellm.Request) (*litellm.Response, error)
}

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
	case ProviderLiteLLMProxy:
		if config.APIKey == "" {
			return nil, fmt.Errorf("LITELLM_MASTER_KEY not set")
		}
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = DefaultLiteLLMProxyBaseURL
		}
		client, err := openai.NewClient(openai.Config{APIKey: config.APIKey, BaseURL: baseURL})
		if err != nil {
			return nil, fmt.Errorf("failed to build litellm proxy client: %w", err)
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
	case ProviderLiteLLMProxy:
		return LLMConfig{
			Provider: provider,
			APIKey:   os.Getenv("LITELLM_MASTER_KEY"),
			BaseURL:  envOrDefault("LITELLM_BASE_URL", DefaultLiteLLMProxyBaseURL),
		}
	default:
		return LLMConfig{Provider: provider}
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
