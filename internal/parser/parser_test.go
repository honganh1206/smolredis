package parser

import (
	"bytes"
	"net"
	"testing"

	"gitlab.com/phamhonganh12062000/redis-go/internal/logger"
)

// Mock connection for testing
type mockConn struct {
	net.Conn
	buffer *bytes.Buffer
}
type testLogger struct {
	*logger.Logger
	Buffer *bytes.Buffer // Capture log output
}

func newTestLogger(_ *testing.T) *testLogger {
	buffer := &bytes.Buffer{}

	// Configure logger for testing
	cfg := logger.LoggerConfig{
		MinLevel:   logger.LevelInfo,
		StackDepth: 0,     // Disable stack traces for tests
		ShowCaller: false, // Disable caller info for tests
	}

	l := logger.New(buffer, cfg)

	return &testLogger{
		Logger: l,
		Buffer: buffer,
	}
}
func (m *mockConn) Read(b []byte) (n int, err error) {
	return m.buffer.Read(b)
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	return m.buffer.Write(b)
}

func TestParserInlineCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
		hasError bool
	}{
		{
			name:     "Simple GET command",
			input:    "GET key\r\n",
			expected: []string{"GET", "key"},
			hasError: false,
		},
		{
			name:     "SET command with quoted string",
			input:    "SET key \"hello world\"\r\n",
			expected: []string{"SET", "key", "hello world"},
			hasError: false,
		},
		{
			name:     "Command with extra spaces",
			input:    "  DEL   key1   key2  \r\n",
			expected: []string{"DEL", "key1", "key2"},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := &mockConn{buffer: bytes.NewBuffer([]byte(tt.input))}
			tl := newTestLogger(t)
			p := NewParser(mockConn, tl.Logger)

			cmd, err := p.Command(tl.Logger)

			if tt.hasError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.hasError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(cmd.Args) != len(tt.expected) {
				t.Errorf("Expected %d arguments, got %d", len(tt.expected), len(cmd.Args))
			}

			for i, arg := range tt.expected {
				if cmd.Args[i] != arg {
					t.Errorf("Expected argument %d to be %s, got %s", i, arg, cmd.Args[i])
				}
			}
		})
	}
}

func TestParserRespArray(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
		hasError bool
	}{
		{
			name: "Simple SET command in RESP format",
			input: "*3\r\n" +
				"$3\r\n" +
				"SET\r\n" +
				"$3\r\n" +
				"key\r\n" +
				"$5\r\n" +
				"value\r\n",
			expected: []string{"SET", "key", "value"},
			hasError: false,
		},
		{
			name: "GET command in RESP format",
			input: "*2\r\n" +
				"$3\r\n" +
				"GET\r\n" +
				"$3\r\n" +
				"key\r\n",
			expected: []string{"GET", "key"},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := &mockConn{buffer: bytes.NewBuffer([]byte(tt.input))}
			tl := newTestLogger(t)
			p := NewParser(mockConn, tl.Logger)

			cmd, err := p.Command(tl.Logger)

			if tt.hasError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.hasError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(cmd.Args) != len(tt.expected) {
				t.Errorf("Expected %d arguments, got %d", len(tt.expected), len(cmd.Args))
			}

			for i, arg := range tt.expected {
				if cmd.Args[i] != arg {
					t.Errorf("Expected argument %d to be %s, got %s", i, arg, cmd.Args[i])
				}
			}
		})
	}
}

func TestConsumeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{
			name:     "Simple quoted string",
			input:    "\"hello world\"",
			expected: "hello world",
			hasError: false,
		},
		{
			name:     "String with escaped quotes",
			input:    "\"hello \\\"world\\\"\"",
			expected: "hello \"world\"",
			hasError: false,
		},
		{
			name:     "Empty string",
			input:    "\"\"",
			expected: "",
			hasError: false,
		},
		{
			name:     "Unbalanced quotes",
			input:    "\"hello world",
			expected: "",
			hasError: true,
		},
		{
			name:     "String with multiple escaped quotes",
			input:    "\"hello \\\"beautiful\\\" \\\"world\\\"\"",
			expected: "hello \"beautiful\" \"world\"",
			hasError: false,
		},
		{
			name:     "String with backslash not escaping quote",
			input:    "\"hello\\world\"",
			expected: "hello\\world",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &Parser{
				line: []byte(tt.input),
				pos:  0,
			}

			// Skip the initial quote
			parser.advance()

			result, err := parser.consumeString()

			if tt.hasError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.hasError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.hasError && string(result) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(result))
			}
		})
	}
}
