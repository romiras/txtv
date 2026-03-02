package engine

import (
	"fmt"
	"io"
)

// Engine manages the streaming logic and limit tracking.
type Engine struct {
	MaxTokens    int
	MaxLines     int
	SoftStop     bool
	SummaryMode  string // "kv", "json", "off"
	TokensCount  int
	LinesCount   int
	BytesEmitted int64  // Total bytes successfully written to stdout
	SoftBytes    int    // Byte counter for the 1MB fail-safe
	StoppedBy    string // "max_tokens", "max_lines", "eof", "soft_limit"
	Flush        bool   // Flush after every token
}

// Token represents a chunk of data.
type Token struct {
	Data  []byte
	IsEOL bool // True if contains '\n'
}

// Process reads from r and writes to w.
func (e *Engine) Process(r io.Reader, w io.Writer) error {
	// A placeholder loop covering basic pass-through for 1.1 Scaffold
	// 1.2 and 1.3 will implement actual reading/segmentation logic.

	buf := make([]byte, 32*1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			// Write directly to stdout for pass-through (Alpha goal: Pass-through with UTF-8 validation)
			// (UTF-8 validation will be added in 1.4)
			nw, wErr := w.Write(buf[:n])
			e.BytesEmitted += int64(nw)
			if e.Flush {
				// Attempt to flush if the writer supports Sync()
				if f, ok := w.(interface{ Sync() error }); ok {
					_ = f.Sync()
				}
			}
			if wErr != nil {
				e.StoppedBy = "error"
				return wErr
			}
		}

		if err == io.EOF {
			e.StoppedBy = "eof"
			break
		}
		if err != nil {
			e.StoppedBy = "error"
			return err
		}

		// Break condition placeholders
		if e.MaxLines > 0 && e.LinesCount >= e.MaxLines {
			e.StoppedBy = "max_lines"
			break
		}
		if e.MaxTokens > 0 && e.TokensCount >= e.MaxTokens {
			e.StoppedBy = "max_tokens"
			break
		}
	}
	return nil
}

// Report prints the final metrics to stderr.
func (e *Engine) Report(w io.Writer) {
	if e.SummaryMode == "off" {
		return
	}
	if e.SummaryMode == "json" {
		fmt.Fprintf(w, `{"lines": %d, "tokens": %d, "bytes": %d, "stopped": "%s"}`+"\n",
			e.LinesCount, e.TokensCount, e.BytesEmitted, e.StoppedBy)
	} else {
		// Default to kv
		fmt.Fprintf(w, "lines: %d, tokens: %d, bytes: %d, stopped: %s\n",
			e.LinesCount, e.TokensCount, e.BytesEmitted, e.StoppedBy)
	}
}
