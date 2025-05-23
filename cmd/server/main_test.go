package main

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
)

func TestGetMCPServerPort(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	tests := []struct {
		name           string
		envValue       string
		want           int
		wantLogMessage string
	}{
		{
			name:           "valid port",
			envValue:       "8080",
			want:           8080,
			wantLogMessage: "",
		},
		{
			name:           "empty env var",
			envValue:       "",
			want:           8080,
			wantLogMessage: "",
		},
		{
			name:           "invalid format",
			envValue:       "not-a-number",
			want:           8080,
			wantLogMessage: "Invalid MCP_PORT value: not-a-number (must be a valid number), using default port 8080",
		},
		{
			name:           "zero port",
			envValue:       "0",
			want:           0,
			wantLogMessage: "",
		},
		{
			name:           "negative port",
			envValue:       "-1",
			want:           8080,
			wantLogMessage: "Invalid MCP_PORT value: -1 (must be between 0 and 65535), using default port 8080",
		},
		{
			name:           "port too large",
			envValue:       "65536",
			want:           8080,
			wantLogMessage: "Invalid MCP_PORT value: 65536 (must be between 0 and 65535), using default port 8080",
		},
		{
			name:           "max valid port",
			envValue:       "65535",
			want:           65535,
			wantLogMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear log buffer before each test
			buf.Reset()

			// Save current env value and restore after test
			oldValue := os.Getenv("MCP_PORT")
			defer os.Setenv("MCP_PORT", oldValue)

			// Set test environment value
			if tt.envValue != "" {
				os.Setenv("MCP_PORT", tt.envValue)
			} else {
				os.Unsetenv("MCP_PORT")
			}

			if got := getMCPServerPort(); got != tt.want {
				t.Errorf("getMCPServerPort() = %v, want %v", got, tt.want)
			}

			// Check log message if expected
			if tt.wantLogMessage != "" {
				logOutput := buf.String()
				if !strings.Contains(logOutput, tt.wantLogMessage) {
					t.Errorf("Expected log message containing %q, got %q", tt.wantLogMessage, logOutput)
				}
			} else if buf.Len() > 0 {
				t.Errorf("Unexpected log message: %q", buf.String())
			}
		})
	}
}
