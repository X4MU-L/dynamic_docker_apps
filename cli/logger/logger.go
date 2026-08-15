package logger

import (
	"fmt"
	"os"
	"strings"
)

const (
	colorReset   = "\033[0m"
	colorRed     = "\033[1;31m"
	colorGreen   = "\033[1;32m"
	colorYellow  = "\033[1;33m"
	colorCyan    = "\033[1;36m"
	colorMagenta = "\033[1;35m"
	colorDim     = "\033[2m"
)

type LiveStep struct {
	Title       string
	Lines       []string
	ActiveLines int
}

func StartStep(format string, a ...interface{}) *LiveStep {
	title := fmt.Sprintf(format, a...)
	fmt.Printf("%s[STEP]%s %s\n", colorMagenta, colorReset, title)
	return &LiveStep{Title: title}
}

func (s *LiveStep) UpdateStream(line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}
	s.Lines = append(s.Lines, trimmed)
	if len(trimmed) > 85 {
		trimmed = trimmed[:82] + "..."
	}

	s.clearActiveLines()
	fmt.Printf("  %s↳ %s%s\n", colorDim, trimmed, colorReset)
	s.ActiveLines = 1
}

func (s *LiveStep) FinishSuccess(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	s.clearActiveLines()
	fmt.Printf("\033[1A\033[2K%s[SUCCESS]%s %s\n", colorGreen, colorReset, msg)
}

func (s *LiveStep) FinishError(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	s.clearActiveLines()
	fmt.Printf("\033[1A\033[2K%s[ERROR]%s %s\n", colorRed, colorReset, msg)
	s.dumpLogs()
}

func (s *LiveStep) clearActiveLines() {
	for i := 0; i < s.ActiveLines; i++ {
		fmt.Print("\033[1A\033[2K")
	}
	s.ActiveLines = 0
}

func (s *LiveStep) dumpLogs() {
	if len(s.Lines) == 0 {
		return
	}
	fmt.Println("  \033[1;31mTraceback logs prior to failure:\033[0m")
	startIdx := 0
	if len(s.Lines) > 40 {
		startIdx = len(s.Lines) - 40
	}
	for _, l := range s.Lines[startIdx:] {
		fmt.Printf("    %s│%s %s\n", colorDim, colorReset, l)
	}
}

func Info(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s[INFO]%s %s\n", colorCyan, colorReset, msg)
}

func Success(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s[SUCCESS]%s %s\n", colorGreen, colorReset, msg)
}

func Warn(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s[WARN]%s %s\n", colorYellow, colorReset, msg)
}

func Error(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(os.Stderr, "%s[ERROR]%s %s\n", colorRed, colorReset, msg)
}

func Fatal(format string, a ...interface{}) {
	Error(format, a...)
	os.Exit(1)
}
