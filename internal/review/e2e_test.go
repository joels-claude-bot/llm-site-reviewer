package review

// Naming rule: every live-LLM test function name must end in the suffix "E2E"
// (not "Eval" or "Live"). Selective running relies on `go test -run E2E`
// matching by function name, so the suffix is what keeps these tests findable.

import (
	"os"
	"strings"
	"testing"
)

// e2eEnvVar gates every live-LLM test. These cost real tokens and need a
// reachable model, so they stay skipped unless it is set.
const e2eEnvVar = "RUN_E2E"

// e2eClient skips the test unless RUN_E2E is set, and fails loudly if it is set
// but credentials are missing: asking for the expensive lane and mis-wiring it
// should be an error, not a silent pass. Live tests build their client here.
func e2eClient(t *testing.T) ChatClient {
	t.Helper()
	if os.Getenv(e2eEnvVar) == "" {
		t.Skipf("live LLM test; set %s=1 (and LITELLM_MASTER_KEY) to run", e2eEnvVar)
	}
	config := ClientConfigFromEnv(ProviderLiteLLMProxy)
	if config.APIKey == "" {
		t.Fatalf("%s is set but LITELLM_MASTER_KEY is empty", e2eEnvVar)
	}
	client, err := NewLLMClient(config)
	if err != nil {
		t.Fatalf("failed to build e2e client: %v", err)
	}
	return client
}

// containsAtLeast fails unless at least min of the needles appear (case-folded)
// in text. Live model output is paraphrased, so eval assertions check that the
// critical facts survived, not the exact wording.
func containsAtLeast(t *testing.T, text string, needles []string, min int) {
	t.Helper()
	lowered := strings.ToLower(text)
	var found int
	var missing []string
	for _, needle := range needles {
		if strings.Contains(lowered, strings.ToLower(needle)) {
			found++
			continue
		}
		missing = append(missing, needle)
	}
	if found < min {
		t.Fatalf("found %d/%d required tokens (min %d); missing %v\nfull text:\n%s", found, len(needles), min, missing, text)
	}
}
