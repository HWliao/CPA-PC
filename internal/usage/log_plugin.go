package usage

import (
	"context"
	"fmt"
	"io"
	"time"

	sdkusage "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/usage"
	log "github.com/sirupsen/logrus"
)

const persistTimeout = 5 * time.Second

type LogPlugin struct {
	writer io.Writer
}

type EventStore interface {
	InsertEvents(context.Context, []Event) (InsertResult, error)
}

type PersistPlugin struct {
	store  EventStore
	writer io.Writer
}

func NewLogPlugin(writer io.Writer) *LogPlugin {
	return &LogPlugin{writer: writer}
}

func NewPersistPlugin(store EventStore, writer io.Writer) *PersistPlugin {
	return &PersistPlugin{store: store, writer: writer}
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

func (p *PersistPlugin) HandleUsage(ctx context.Context, record sdkusage.Record) {
	if p == nil || p.store == nil {
		return
	}
	event := EventFromSDKRecord(record)
	persistCtx := context.Background()
	if ctx != nil {
		persistCtx = context.WithoutCancel(ctx)
	}
	persistCtx, cancel := context.WithTimeout(persistCtx, persistTimeout)
	defer cancel()

	result, err := p.store.InsertEvents(persistCtx, []Event{event})
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"provider": event.Provider,
			"model":    event.Model,
		}).Warn("usage persist failed")
		return
	}
	if p.writer != nil {
		fmt.Fprintf(p.writer, "usage persisted inserted=%d skipped=%d event_hash=%q\n", result.Inserted, result.Skipped, event.EventHash)
	}
}

var _ sdkusage.Plugin = (*LogPlugin)(nil)
var _ sdkusage.Plugin = (*PersistPlugin)(nil)
