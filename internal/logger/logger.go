package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"gitlab.com/phamhonganh12062000/smolredis/internal/helpers"
)

type Level int8

const CALL_DEPTH = 3

const (
	LevelInfo Level = iota
	LevelError
	LevelFatal
	LevelOff
)

// TODO: Upgrade logger with MultiWriter for writing logs to files?
// Check history Level String Methid Implementation
type Logger struct {
	out    io.Writer
	config LoggerConfig
	mu     sync.Mutex // coordinate the writes so log entries dont get mixed up
}

type LoggerConfig struct {
	MinLevel   Level
	StackDepth int
	ShowCaller bool // Optional to show caller info
}

func New(out io.Writer, cfg LoggerConfig) *Logger {
	return &Logger{
		out:    out,
		config: cfg,
	}
}

// Implicitly implement the Stringer interface here
func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return ""
	}
}

// Have to be Write to satisfy the io.Writer interface
func (l *Logger) Write(msg []byte) (n int, err error) {
	return l.output(LevelError, string(msg), nil)
}

func (l *Logger) Info(msg string, props map[string]string) {
	l.output(LevelInfo, msg, props)
}

func (l *Logger) Error(err error, props map[string]string) {
	l.output(LevelError, err.Error(), props)
}

func (l *Logger) Fatal(err error, props map[string]string) {
	l.output(LevelFatal, err.Error(), props)
	os.Exit(1) // Terminate the app
}

// TODO: Rewrite this without nested struct and prioritize early returns
func (l *Logger) output(level Level, msg string, props map[string]string) (int, error) {
	// No need to display level below error
	if level < l.config.MinLevel {
		return 0, nil
	}

	buf := helpers.GetBuffer()
	defer helpers.PutBuffer(buf)

	aux := struct {
		Level      string            `json:"level"`
		Time       string            `json:"time"`
		Message    string            `json:"message"`
		Properties map[string]string `json:"properties"`
		Trace      string            `json:"trace,omitempty"`
		// Pointer type is optional here, but it would be useful when caller info is not available or there is an error
		Caller *helpers.CallerInfo `json:"caller,omitempty"`
	}{
		Level:      level.String(),
		Time:       time.Now().UTC().Format(time.RFC3339),
		Message:    msg,
		Properties: props,
	}
	// Immediate caller info - Where the log was called from
	if l.config.ShowCaller {
		// Skip runtime.Caller + print + PrintInfo/PrintError
		aux.Caller = helpers.GetCaller(CALL_DEPTH)
	}

	// Detailed stack trace
	if level >= LevelError {
		if l.config.StackDepth > 0 {
			stack := make([]uintptr, l.config.StackDepth)
			// CALL_DEPTH and l.config.StackDepth serve different purposes
			length := runtime.Callers(CALL_DEPTH, stack[:]) // Createa a slice from an array
			if length > 0 {
				// Get PC values from Callers and return function/file/line information
				frames := runtime.CallersFrames(stack[:length])

				var trace strings.Builder
				frameCount := 1
				trace.WriteString("\n") // Start with a newline
				for {
					frame, more := frames.Next()

					// Clean up file path
					file := helpers.CleanPath(frame.File)

					// FIXME: No need for pretty JSON, but at least get rid of the \n character
					// Might put this as a struct
					fmt.Fprintf(&trace, "    Frame %d:\n", frameCount)
					fmt.Fprintf(&trace, "        Function: %s\n", frame.Function)
					fmt.Fprintf(&trace, "        File:     %s\n", file)
					fmt.Fprintf(&trace, "        Line:     %d\n", frame.Line)

					if more {
						trace.WriteString("\n") // Spacing between frames
					}

					frameCount++

					if !more {
						break
					}
				}

				aux.Trace = helpers.FormatDebugStack(trace.String())
			}

		} else {
			// If no depth is specified
			// Format the stack output
			stackTrace := string(debug.Stack())
			aux.Trace = helpers.FormatDebugStack(stackTrace)
		}
	}

	jsonData, err := json.Marshal(aux)

	*buf = append(*buf, jsonData...)
	*buf = append(*buf, '\n') // Ensure newline

	// Single atomic write
	l.mu.Lock()
	n, err := l.out.Write(*buf)
	l.mu.Unlock()

	return n, err
}
