package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/romiras/txtv/internal/engine"
)

func main() {
	var maxTokens int
	var maxLines int
	var softStop bool
	var flush bool
	var summaryMode string

	flag.IntVar(&maxTokens, "max-tokens", 0, "Maximum tokens to emit")
	flag.IntVar(&maxLines, "max-lines", 0, "Maximum lines to emit (LF separated)")
	flag.BoolVar(&softStop, "soft", false, "Continue until line ends or 1MB reached when token limit is hit")
	flag.BoolVar(&flush, "flush", false, "Enable real-time piping by flushing stdout after every token")
	flag.StringVar(&summaryMode, "summary", "kv", "Output format for stderr (kv|json|off)")

	flag.Parse()

	// Validate inputs
	if summaryMode != "kv" && summaryMode != "json" && summaryMode != "off" {
		fmt.Fprintf(os.Stderr, "Error: invalid summary mode %q\n", summaryMode)
		os.Exit(1)
	}

	e := &engine.Engine{
		MaxTokens:   maxTokens,
		MaxLines:    maxLines,
		SoftStop:    softStop,
		SummaryMode: summaryMode,
		Flush:       flush,
	}

	err := e.Process(os.Stdin, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if errors.Is(err, engine.ErrInvalidUTF8) {
			e.Report(os.Stderr)
			os.Exit(2)
		}
		e.Report(os.Stderr)
		os.Exit(1)
	}
	e.Report(os.Stderr)
}
