package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
	LevelFatal Level = "FATAL"
)

type Entry struct {
	Timestamp     string                 `json:"timestamp"`
	Level         Level                  `json:"level"`
	Message       string                 `json:"message"`
	Service       string                 `json:"service"`
	TraceID       string                 `json:"trace_id,omitempty"`
	SpanID        string                 `json:"span_id,omitempty"`
	TenantID      string                 `json:"tenant_id,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	Fields        map[string]interface{} `json:"fields,omitempty"`
	Caller        string                 `json:"caller,omitempty"`
}

type Logger struct {
	service string
	out     io.Writer
	mu      sync.Mutex
}

func NewLogger(service string) *Logger {
	return &Logger{
		service: service,
		out:     os.Stdout,
	}
}

func (l *Logger) Log(ctx context.Context, lvl Level, msg string, fields map[string]interface{}) {
	entry := Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     lvl,
		Message:   msg,
		Service:   l.service,
		Fields:    fields,
	}

	if ctx != nil {
		if reqID, ok := ctx.Value("request_id").(string); ok {
			entry.TraceID = reqID
		}
		if tenantID, ok := ctx.Value("tenant_id").(string); ok {
			entry.TenantID = tenantID
		}
		if userID, ok := ctx.Value("user_id").(string); ok {
			entry.UserID = userID
		}
	}

	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logging json error: %v\n", err)
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintln(l.out, string(data))
}

func (l *Logger) Info(ctx context.Context, msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields...)
	l.Log(ctx, LevelInfo, msg, f)
}

func (l *Logger) Error(ctx context.Context, msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields...)
	l.Log(ctx, LevelError, msg, f)
}

func (l *Logger) Warn(ctx context.Context, msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields...)
	l.Log(ctx, LevelWarn, msg, f)
}

func (l *Logger) Debug(ctx context.Context, msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields...)
	l.Log(ctx, LevelDebug, msg, f)
}

func mergeFields(fields ...map[string]interface{}) map[string]interface{} {
	if len(fields) == 0 {
		return nil
	}
	res := make(map[string]interface{})
	for _, f := range fields {
		for k, v := range f {
			res[k] = v
		}
	}
	return res
}

var DefaultLogger = NewLogger("statgate")
