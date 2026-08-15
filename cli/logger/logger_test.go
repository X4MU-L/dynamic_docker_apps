package logger

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureOutput(f func()) string {
	oldOut := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = oldOut
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestLoggerOutput(t *testing.T) {
	out := captureOutput(func() {
		Info("Testing %s", "info")
		Success("Testing %s", "success")
		Warn("Testing %s", "warn")
	})

	if !strings.Contains(out, "Testing info") {
		t.Errorf("Expected info log in output, got: %s", out)
	}
	if !strings.Contains(out, "Testing success") {
		t.Errorf("Expected success log in output, got: %s", out)
	}
	if !strings.Contains(out, "Testing warn") {
		t.Errorf("Expected warn log in output, got: %s", out)
	}
}
