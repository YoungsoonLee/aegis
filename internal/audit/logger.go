package audit

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/YoungsoonLee/aegis/internal/config"
)

type Sink interface {
	Write(event *Event) error
	Close() error
}

type Logger struct {
	sinks   []Sink
	enabled bool
	mu      sync.Mutex
}

func NewLogger(cfg config.AuditConfig) (*Logger, error) {
	l := &Logger{enabled: cfg.Enabled}

	if !cfg.Enabled {
		return l, nil
	}

	for _, sc := range cfg.Sinks {
		sink, err := newSink(sc)
		if err != nil {
			return nil, fmt.Errorf("creating audit sink %s: %w", sc.Type, err)
		}
		l.sinks = append(l.sinks, sink)
	}

	return l, nil
}

func (l *Logger) Log(event *Event) {
	if !l.enabled {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for _, s := range l.sinks {
		_ = s.Write(event)
	}
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, s := range l.sinks {
		_ = s.Close()
	}
	return nil
}

func newSink(cfg config.SinkConfig) (Sink, error) {
	switch cfg.Type {
	case "stdout":
		return &writerSink{w: os.Stdout}, nil
	case "file":
		f, err := os.OpenFile(cfg.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		return &writerSink{w: f, closer: f}, nil
	default:
		return nil, fmt.Errorf("unknown sink type: %s", cfg.Type)
	}
}

type writerSink struct {
	w      io.Writer
	closer io.Closer
}

func (s *writerSink) Write(event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(s.w, "%s\n", data)
	return err
}

func (s *writerSink) Close() error {
	if s.closer != nil {
		return s.closer.Close()
	}
	return nil
}
