package engine

import (
	"fmt"
	"io"

	"unicode/utf8"

	"github.com/romiras/txtv/internal/segmenter"
)

var ErrInvalidUTF8 = fmt.Errorf("invalid UTF-8 or binary data")

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

// Process reads from r and writes to w, enforcing token and line limits.
func (e *Engine) Process(r io.Reader, w io.Writer) error {
	const chunkSize = 32 * 1024
	const lookaheadSize = 32

	buf := make([]byte, chunkSize+lookaheadSize)
	pending := 0

	for {
		n, err := r.Read(buf[pending : chunkSize+pending])
		total := pending + n

		isEOF := false
		if err == io.EOF {
			isEOF = true
		} else if err != nil {
			e.StoppedBy = "error"
			return err
		}

		if total == 0 {
			if isEOF {
				e.StoppedBy = "eof"
				break
			}
			continue
		}

		// Calculate how many bytes are available to process this iteration.
		// Keep up to 32 bytes in the lookahead carry-over unless we're at EOF.
		var available int
		if isEOF {
			available = total
		} else {
			if total > lookaheadSize {
				available = total - lookaheadSize
				// Backtrack to the nearest rune start to avoid splitting a multi-byte rune.
				// utf8.RuneStart returns true if the byte is not a continuation byte.
				for available > 0 && !utf8.RuneStart(buf[available]) {
					available--
				}
			} else {
				available = 0 // Wait for more data
			}
		}

		if available > 0 {
			writeSlice := buf[:available]

			// --- UTF-8 Validation ---
			// Since we aligned available to a rune boundary, if it's invalid UTF-8,
			// it's truly invalid, not just a partial rune.
			if !utf8.Valid(writeSlice) {
				e.StoppedBy = "error"
				return ErrInvalidUTF8
			}

			hitLines := false
			hitTokens := false

			// --- max-lines enforcement ---
			if e.MaxLines > 0 {
				remaining := e.MaxLines - e.LinesCount
				pos := 0
				for i, b := range writeSlice {
					if b == '\n' {
						remaining--
						if remaining == 0 {
							pos = i + 1
							hitLines = true
							break
						}
					}
				}
				if hitLines {
					writeSlice = writeSlice[:pos]
				}
			}

			// --- max-tokens enforcement ---
			if e.MaxTokens > 0 && !hitLines {
				remaining := e.MaxTokens - e.TokensCount
				tokCount, cutoff := segmenter.CountAndCut(writeSlice, remaining)
				if tokCount >= remaining {
					// Reached or exceeded the limit within this slice.
					writeSlice = writeSlice[:cutoff]
					hitTokens = true
				}
				// Accumulate counted tokens (only those within the slice we'll emit).
				if hitTokens {
					e.TokensCount += tokCount
				} else {
					e.TokensCount += tokCount
				}
			} else if e.MaxTokens == 0 {
				// No token limit: still count for reporting.
				tokCount, _ := segmenter.CountAndCut(writeSlice, 0)
				e.TokensCount += tokCount
			}

			// Count newlines in the slice we're about to emit.
			for _, b := range writeSlice {
				if b == '\n' {
					e.LinesCount++
				}
			}

			nw, wErr := w.Write(writeSlice)
			e.BytesEmitted += int64(nw)
			if e.Flush {
				if f, ok := w.(interface{ Sync() error }); ok {
					_ = f.Sync()
				}
			}
			if wErr != nil {
				e.StoppedBy = "error"
				return wErr
			}

			if hitLines {
				e.StoppedBy = "max_lines"
				return nil
			}
			if hitTokens {
				e.StoppedBy = "max_tokens"
				return nil
			}

			// Shift remaining carry-over bytes to the front of the buffer.
			if available < total {
				copy(buf[:total-available], buf[available:total])
			}
		}
		pending = total - available

		if isEOF {
			e.StoppedBy = "eof"
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
