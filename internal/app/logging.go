package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	pcconfig "github.com/HWliao/CPA-PC/internal/config"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

const embeddedMainLogFileName = "main.log"

var embeddedLogMu sync.Mutex
var embeddedLogWriter *lumberjack.Logger

var embeddedLogFieldOrder = []string{"provider", "model", "mode", "budget", "level", "original_mode", "original_value", "min", "max", "clamped_to", "error"}

type embeddedLogFormatter struct{}

func (embeddedLogFormatter) Format(entry *log.Entry) ([]byte, error) {
	var buffer *bytes.Buffer
	if entry.Buffer != nil {
		buffer = entry.Buffer
	} else {
		buffer = &bytes.Buffer{}
	}

	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	message := strings.TrimRight(entry.Message, "\r\n")

	reqID := "--------"
	if id, ok := entry.Data["request_id"].(string); ok && id != "" {
		reqID = id
	}

	level := entry.Level.String()
	if level == "warning" {
		level = "warn"
	}
	levelStr := fmt.Sprintf("%-5s", level)

	var fieldsStr string
	if len(entry.Data) > 0 {
		fields := make([]string, 0, len(entry.Data))
		for _, key := range embeddedLogFieldOrder {
			if value, ok := entry.Data[key]; ok {
				fields = append(fields, fmt.Sprintf("%s=%v", key, value))
			}
		}
		if len(fields) > 0 {
			fieldsStr = " " + strings.Join(fields, " ")
		}
	}

	var formatted string
	if entry.Caller != nil {
		formatted = fmt.Sprintf("[%s] [%s] [%s] [%s:%d] %s%s\n", timestamp, reqID, levelStr, filepath.Base(entry.Caller.File), entry.Caller.Line, message, fieldsStr)
	} else {
		formatted = fmt.Sprintf("[%s] [%s] [%s] %s%s\n", timestamp, reqID, levelStr, message, fieldsStr)
	}
	buffer.WriteString(formatted)
	return buffer.Bytes(), nil
}

func ensureEmbeddedWritablePath(cfg *pcconfig.Config) error {
	if embeddedWritablePath() != "" || cfg == nil {
		return nil
	}

	baseDir := strings.TrimSpace(cfg.Runtime.BaseDir)
	if baseDir == "" && cfg.Runtime.ConfigPath != "" {
		baseDir = filepath.Dir(cfg.Runtime.ConfigPath)
	}
	if baseDir == "" {
		return nil
	}

	if err := os.Setenv("WRITABLE_PATH", filepath.Clean(baseDir)); err != nil {
		return fmt.Errorf("set embedded writable path: %w", err)
	}
	return nil
}

func configureEmbeddedLogOutput(cfg *pcconfig.Config) (func() error, error) {
	if cfg == nil || !cfg.LoggingToFile {
		return nil, nil
	}

	logDir := embeddedLogDirectory(cfg)
	if strings.TrimSpace(logDir) == "" {
		return nil, fmt.Errorf("embedded log directory is not configured")
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("create embedded log directory: %w", err)
	}

	writer := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, embeddedMainLogFileName),
		MaxSize:    10,
		MaxBackups: 0,
		MaxAge:     0,
		Compress:   false,
	}

	embeddedLogMu.Lock()
	previous := embeddedLogWriter
	embeddedLogWriter = writer
	log.SetReportCaller(true)
	log.SetFormatter(embeddedLogFormatter{})
	log.SetOutput(writer)
	embeddedLogMu.Unlock()

	if previous != nil {
		_ = previous.Close()
	}

	var once sync.Once
	return func() error {
		var err error
		once.Do(func() {
			embeddedLogMu.Lock()
			if embeddedLogWriter == writer {
				embeddedLogWriter = nil
				log.SetOutput(os.Stdout)
			}
			embeddedLogMu.Unlock()
			err = writer.Close()
		})
		return err
	}, nil
}

func configureEmbeddedLogLevel(debugEnabled bool) {
	if debugEnabled {
		log.SetLevel(log.DebugLevel)
		return
	}
	log.SetLevel(log.InfoLevel)
}

func embeddedLogDirectory(cfg *pcconfig.Config) string {
	if base := embeddedWritablePath(); base != "" {
		return filepath.Join(base, "logs")
	}
	if cfg != nil && strings.TrimSpace(cfg.Runtime.BaseDir) != "" {
		return filepath.Join(cfg.Runtime.BaseDir, "logs")
	}
	return "logs"
}

func embeddedWritablePath() string {
	for _, key := range []string{"WRITABLE_PATH", "writable_path"} {
		if value, ok := os.LookupEnv(key); ok {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return filepath.Clean(trimmed)
			}
		}
	}
	return ""
}

func closeCleanupFns(cleanups []func() error) {
	for i := len(cleanups) - 1; i >= 0; i-- {
		if cleanups[i] != nil {
			_ = cleanups[i]()
		}
	}
}
