package usage

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	sdkusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
)

func TestLogPluginWritesUsageRecord(t *testing.T) {
	var output bytes.Buffer
	plugin := NewLogPlugin(&output)

	plugin.HandleUsage(context.Background(), sdkusage.Record{
		Provider: "gemini",
		Model:    "gemini-test",
		AuthID:   "auth-1",
		AuthType: "oauth",
		Source:   "openai",
		Latency:  150 * time.Millisecond,
		Failed:   true,
		Detail: sdkusage.Detail{
			InputTokens:     10,
			OutputTokens:    20,
			ReasoningTokens: 3,
			CachedTokens:    4,
			TotalTokens:     37,
		},
	})

	line := output.String()
	for _, want := range []string{
		`provider="gemini"`,
		`model="gemini-test"`,
		`input_tokens=10`,
		`output_tokens=20`,
		`reasoning_tokens=3`,
		`cached_tokens=4`,
		`total_tokens=37`,
		`latency_ms=150`,
		`failed=true`,
		`auth_id="auth-1"`,
		`auth_type="oauth"`,
		`source="openai"`,
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("log line = %q, want %s", line, want)
		}
	}
}
