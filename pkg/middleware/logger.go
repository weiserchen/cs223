package middleware

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

type ResponseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func NewResponseWrapper(w http.ResponseWriter) *ResponseWrapper {
	return &ResponseWrapper{}
}

func (rw *ResponseWrapper) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

type Logger interface {
	Session(id string) LoggerSession
	Print(id string, w io.Writer)
}

type LoggerSession interface {
	Log(fmtStr string, args ...any)
	Done()
}

type NopLogger struct {
}

var _ Logger = (*NopLogger)(nil)

func (logger *NopLogger) Session(id string) LoggerSession {
	return &NopLoggerSession{}
}

func (logger *NopLogger) Print(id string, w io.Writer) {

}

type NopLoggerSession struct {
}

var _ LoggerSession = (*NopLoggerSession)(nil)

func (logger *NopLoggerSession) Log(_ string, _ ...any) {

}

func (logger *NopLoggerSession) Done() {

}

type DebugLogger struct {
	mu   sync.Mutex
	logs map[string][]*LogEntry
}

var _ Logger = (*DebugLogger)(nil)

func NewDebugLogger() *DebugLogger {
	return &DebugLogger{
		logs: map[string][]*LogEntry{},
	}
}

func (logger *DebugLogger) Session(id string) LoggerSession {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	if _, ok := logger.logs[id]; !ok {
		logger.logs[id] = []*LogEntry{}
	}
	return NewDebugLoggerSession(id, logger)
}

func (logger *DebugLogger) Print(id string, w io.Writer) {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	entries := logger.logs[id]
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ts.Before(entries[j].ts)
	})
	for _, entry := range entries {
		fmt.Fprintf(w, "id:%s -> %s", id, entry.l)
	}
}

type DebugLoggerSession struct {
	id     string
	logs   []*LogEntry
	parent *DebugLogger
}

var _ LoggerSession = (*DebugLoggerSession)(nil)

func NewDebugLoggerSession(id string, parent *DebugLogger) *DebugLoggerSession {
	return &DebugLoggerSession{
		id:     id,
		parent: parent,
	}
}

func (logger *DebugLoggerSession) Log(fmtStr string, args ...any) {
	logger.logs = append(logger.logs, NewLogEntry(fmt.Sprintf(fmtStr+"\n", args...)))
}

func (logger *DebugLoggerSession) Done() {
	logger.parent.mu.Lock()
	defer logger.parent.mu.Unlock()
	logger.logs = append(logger.logs, NewLogEntry("============\n"))
	logger.parent.logs[logger.id] = append(logger.parent.logs[logger.id], logger.logs...)
}

type LogEntry struct {
	ts time.Time
	l  string
}

func NewLogEntry(l string) *LogEntry {
	return &LogEntry{
		ts: time.Now(),
		l:  l,
	}
}
