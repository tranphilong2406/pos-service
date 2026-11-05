package logger

import "testing"

func TestLogLinesEndWithPlainNewline(t *testing.T) {
	if got := newConfig().EncoderConfig.LineEnding; got != "\n" {
		t.Errorf("Expected JSON log lines to end with \"\\n\", got %q", got)
	}
}
