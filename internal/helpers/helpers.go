package helpers

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const MAX_BUFFER_SIZE = 64 << 10

// Cache and reuse byte slices to reduce memory allocations
var bufferPool = sync.Pool{New: func() any { return new([]byte) }}

func CleanPath(path string) string {
	parts := strings.Split(path, "/")

	if len(parts) > 3 {
		return ".../" + strings.Join(parts[len(parts)-3:], "/")
	}

	return path
}

func GetBuffer() *[]byte {
	p := bufferPool.Get().(*[]byte)

	// Reset the buffer while preserving capacity
	*p = (*p)[:0]
	return p
}

// Place the buffer back into the pool
func PutBuffer(p *[]byte) {
	// Set a hard-coded limit for buffers returning to the pool
	// If buffer size exceeds the limit, we let the garbage collector reclaim the memory
	if cap(*p) > MAX_BUFFER_SIZE {
		*p = nil
	}

	bufferPool.Put(p)
}

type CallerInfo struct {
	File     string `json:"file,omitempty"`
	Function string `json:"function,omitempty"`
	Line     int    `json:"line,omitempty"`
}

// Return the memory address pointing to the function code located in memory
func GetCaller(calldepth int) *CallerInfo {
	pc, file, line, ok := runtime.Caller(calldepth)

	if !ok {
		return nil
	}

	fn := runtime.FuncForPC(pc)

	if fn == nil {
		return nil
	}

	return &CallerInfo{
		File:     filepath.Base(file),
		Function: filepath.Base(fn.Name()),
		Line:     line,
	}
}

func FormatDebugStack(stack string) string {
	lines := strings.Split(stack, "\n")
	var formatted strings.Builder

	formatted.WriteString("\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Add indentation
		if strings.HasPrefix(line, "goroutine") {
			fmt.Fprintf(&formatted, "    %s\n", line)
		} else {
			fmt.Fprintf(&formatted, "        %s\n", line)
		}
	}

	return formatted.String()
}
