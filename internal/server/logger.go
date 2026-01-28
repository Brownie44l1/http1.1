package server

import (
	"fmt"
	"log"
	"os"
	"time"
)

// ✅ Issue #17: Structured Logging

// Logger interface for structured logging
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
}

// Field represents a structured log field
type Field struct {
	Key   string
	Value interface{}
}

// DefaultLogger is a simple stdout logger
type DefaultLogger struct {
	logger *log.Logger
}

func (l *DefaultLogger) Debug(msg string, fields ...Field) {
	l.log("DEBUG", msg, fields...)
}

func (l *DefaultLogger) Info(msg string, fields ...Field) {
	l.log("INFO", msg, fields...)
}

func (l *DefaultLogger) Error(msg string, fields ...Field) {
	l.log("ERROR", msg, fields...)
}

func (l *DefaultLogger) Warn(msg string, fields ...Field) {
	l.log("WARN", msg, fields...)
}	

func (l *DefaultLogger) log(level, msg string, fields ...Field) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	output := fmt.Sprintf("[%s] %s: %s", timestamp, level, msg)
	
	if len(fields) > 0 {
		output += " |"
		for _, f := range fields {
			output += fmt.Sprintf(" %s=%v", f.Key, sanitizeValue(f.Value))
		}
	}
	
	if l.logger == nil {
		l.logger = log.New(os.Stdout, "", 0)
	}
	
	l.logger.Println(output)
}

// ✅ Issue #22: Sanitize sensitive values in logs
func sanitizeValue(v interface{}) interface{} {
	if s, ok := v.(string); ok {
		// Don't log full values of potentially sensitive headers
		if len(s) > 100 {
			return s[:100] + "...[truncated]"
		}
	}
	return v
}

// NullLogger discards all logs (for testing)
type NullLogger struct{}

func (n *NullLogger) Debug(msg string, fields ...Field) {}
func (n *NullLogger) Info(msg string, fields ...Field)  {}
func (n *NullLogger) Error(msg string, fields ...Field) {}
func (n *NullLogger) Warn(msg string, fields ...Field)  {}