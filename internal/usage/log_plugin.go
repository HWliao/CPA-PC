package usage

import (
	"context"
	"fmt"
	"io"

	sdkusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
)

type LogPlugin struct {
	writer io.Writer
}

func NewLogPlugin(writer io.Writer) *LogPlugin {
	return &LogPlugin{writer: writer}
}

func (p *LogPlugin) HandleUsage(_ context.Context, record sdkusage.Record) {
	if p == nil || p.writer == nil {
		return
	}

	fmt.Fprintf(
		p.writer,
		"usage provider=%q model=%q input_tokens=%d output_tokens=%d reasoning_tokens=%d cached_tokens=%d total_tokens=%d latency_ms=%d failed=%t auth_id=%q auth_type=%q source=%q\n",
		record.Provider,
		record.Model,
		record.Detail.InputTokens,
		record.Detail.OutputTokens,
		record.Detail.ReasoningTokens,
		record.Detail.CachedTokens,
		record.Detail.TotalTokens,
		record.Latency.Milliseconds(),
		record.Failed,
		record.AuthID,
		record.AuthType,
		record.Source,
	)
}

var _ sdkusage.Plugin = (*LogPlugin)(nil)
