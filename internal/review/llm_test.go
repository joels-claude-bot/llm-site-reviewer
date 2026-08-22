package review

import "testing"

func TestNewLLMClientRejectsMissingKeys(t *testing.T) {
	if _, err := NewLLMClient(LLMConfig{Provider: ProviderGemini}); err == nil {
		t.Fatal("expected missing Gemini key error")
	}
	if _, err := NewLLMClient(LLMConfig{Provider: ProviderLiteLLMProxy}); err == nil {
		t.Fatal("expected missing LiteLLM proxy key error")
	}
}

func TestNewLLMClientRejectsUnknownProvider(t *testing.T) {
	_, err := NewLLMClient(LLMConfig{Provider: "nope", APIKey: "key"})
	if err == nil {
		t.Fatal("expected unknown provider error")
	}
}

func TestNewLLMClientBuildsConfiguredClients(t *testing.T) {
	for _, config := range []LLMConfig{
		{Provider: ProviderGemini, APIKey: "key"},
		{Provider: ProviderLiteLLMProxy, APIKey: "key"},
		{Provider: ProviderLiteLLMProxy, APIKey: "key", BaseURL: "http://example.test/v1"},
	} {
		client, err := NewLLMClient(config)
		if err != nil {
			t.Fatalf("NewLLMClient(%+v) returned error: %v", config, err)
		}
		if client == nil {
			t.Fatalf("NewLLMClient(%+v) returned nil client", config)
		}
	}
}

func TestClientConfigsFromEnv(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "gemini-key")
	t.Setenv("LITELLM_MASTER_KEY", "proxy-key")
	t.Setenv("LITELLM_BASE_URL", "")

	batch := ClientConfigFromEnv(ProviderGemini)
	if batch.Provider != ProviderGemini || batch.APIKey != "gemini-key" {
		t.Fatalf("batch config = %+v", batch)
	}

	dependency := ClientConfigFromEnv(ProviderLiteLLMProxy)
	if dependency.Provider != ProviderLiteLLMProxy || dependency.APIKey != "proxy-key" || dependency.BaseURL != DefaultLiteLLMProxyBaseURL {
		t.Fatalf("dependency config = %+v", dependency)
	}

	t.Setenv("LITELLM_BASE_URL", "http://custom.test/v1")
	dependency = ClientConfigFromEnv(ProviderLiteLLMProxy)
	if dependency.BaseURL != "http://custom.test/v1" {
		t.Fatalf("dependency base URL = %q", dependency.BaseURL)
	}
}
