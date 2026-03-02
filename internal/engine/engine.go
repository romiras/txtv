package engine

import (
	"fmt"
	"io"

	"unicode/utf8"

	"github.com/romiras/txtv/internal/segmenter"
)

var ErrInvalidUTF8 = fmt.Errorf("invalid UTF-8 or binary data")

const (
	chunkSize      = 32 * 1024
	lookaheadSize  = 32
	softLimitBytes = 1024 * 1024
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

// calcAvailable determines how many bytes in buf are safe to process this
// iteration, respecting the lookahead window and UTF-8 rune boundaries.
func (e *Engine) calcAvailable(buf []byte, total int, isEOF bool) int {
	if isEOF {
		return total
	}
	lookahead := lookaheadSize
	if e.Flush {
		lookahead = 0
	}
	if total <= lookahead {
		return 0 // Wait for more data
	}
	available := total - lookahead
	// Backtrack to the nearest rune start to avoid splitting a multi-byte rune.
	for available > 0 && !utf8.RuneStart(buf[available]) {
		available--
	}
	return available
}

// applyLineLimit checks the line limit against the slice. If the limit is hit,
// it truncates the slice and returns true.
func (e *Engine) applyLineLimit(slice []byte) ([]byte, bool) {
	remaining := e.MaxLines - e.LinesCount
	for i, b := range slice {
		if b == '\n' {
			remaining--
			if remaining == 0 {
				return slice[:i+1], true
			}
		}
	}
	return slice, false
}

// findNewline returns the index after the first '\n' starting from offset, or -1.
func findNewline(slice []byte, offset int) int {
	for i := offset; i < len(slice); i++ {
		if slice[i] == '\n' {
			return i + 1
		}
	}
	return -1
}

// applyTokenLimit enforces the token limit, including soft-stop logic.
// Returns the (possibly truncated) slice and whether we should stop after writing.
func (e *Engine) applyTokenLimit(slice []byte) ([]byte, bool) {
	// Already past the limit — in soft-stop mode, drain until EOL or fail-safe.
	if e.TokensCount >= e.MaxTokens && e.SoftStop {
		if pos := findNewline(slice, 0); pos != -1 {
			slice = slice[:pos]
			e.StoppedBy = "max_tokens"
			tok, _ := segmenter.CountAndCut(slice, 0)
			e.TokensCount += tok
			return slice, true
		}
		// No newline found — accumulate bytes toward the 1MB fail-safe.
		e.SoftBytes += len(slice)
		tok, _ := segmenter.CountAndCut(slice, 0)
		e.TokensCount += tok
		if e.SoftBytes >= softLimitBytes {
			e.StoppedBy = "soft_limit"
			return slice, true
		}
		return slice, false
	}

	remaining := e.MaxTokens - e.TokensCount
	tokCount, cutoff := segmenter.CountAndCut(slice, remaining)
	if tokCount < remaining {
		// Limit not yet reached; count and keep going.
		e.TokensCount += tokCount
		return slice, false
	}

	// Token limit reached within this slice.
	if e.SoftStop {
		if pos := findNewline(slice, cutoff); pos != -1 {
			slice = slice[:pos]
			e.StoppedBy = "max_tokens"
			tokCount, _ = segmenter.CountAndCut(slice, 0)
			e.TokensCount += tokCount
			return slice, true
		}
		// No newline in this chunk; accumulate overhead bytes.
		e.SoftBytes += len(slice) - cutoff
		tokCount, _ = segmenter.CountAndCut(slice, 0)
		e.TokensCount += tokCount
		return slice, false
	}

	// Hard stop: truncate exactly at the cutoff.
	slice = slice[:cutoff]
	e.TokensCount += tokCount
	return slice, true
}

type flusher interface {
	Flush() error
}

// writeChunk writes slice to w, updates stats, and optionally flushes token by token.
func (e *Engine) writeChunk(w io.Writer, slice []byte) error {
	if !e.Flush {
		nw, wErr := w.Write(slice)
		e.BytesEmitted += int64(nw)
		return wErr
	}

	offset := 0
	for offset < len(slice) {
		tokCount, cutoff := segmenter.CountAndCut(slice[offset:], 1)
		if cutoff == 0 || tokCount == 0 {
			cutoff = len(slice) - offset
		}
		nw, wErr := w.Write(slice[offset : offset+cutoff])
		e.BytesEmitted += int64(nw)
		if wErr != nil {
			return wErr
		}
		if f, ok := w.(flusher); ok {
			_ = f.Flush()
		} else if s, ok := w.(interface{ Sync() error }); ok {
			_ = s.Sync()
		}
		offset += cutoff
	}
	return nil
}

// countNewlines counts '\n' bytes in slice, adding to e.LinesCount.
func (e *Engine) countNewlines(slice []byte) {
	for _, b := range slice {
		if b == '\n' {
			e.LinesCount++
		}
	}
}

// Process reads from r and writes to w, enforcing token and line limits.
func (e *Engine) Process(r io.Reader, w io.Writer) error {
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

		available := e.calcAvailable(buf, total, isEOF)

		if available > 0 {
			writeSlice := buf[:available]

			// --- UTF-8 Validation ---
			if !utf8.Valid(writeSlice) {
				e.StoppedBy = "error"
				return ErrInvalidUTF8
			}

			hitLines := false
			hitTokens := false

			// --- max-lines enforcement ---
			if e.MaxLines > 0 {
				writeSlice, hitLines = e.applyLineLimit(writeSlice)
			}

			// --- max-tokens enforcement ---
			if e.MaxTokens > 0 {
				writeSlice, hitTokens = e.applyTokenLimit(writeSlice)
			} else if e.MaxTokens == 0 {
				// No token limit: still count for reporting.
				tok, _ := segmenter.CountAndCut(writeSlice, 0)
				e.TokensCount += tok
			}

			// Count newlines in the slice we're about to emit.
			e.countNewlines(writeSlice)

			if wErr := e.writeChunk(w, writeSlice); wErr != nil {
				e.StoppedBy = "error"
				return wErr
			}

			if hitTokens {
				if e.StoppedBy == "" {
					e.StoppedBy = "max_tokens"
				}
				return nil
			}
			if hitLines {
				e.StoppedBy = "max_lines"
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
		fmt.Fprintf(w, "\n"+`{"lines": %d, "tokens": %d, "bytes": %d, "stopped": "%s"}`+"\n",
			e.LinesCount, e.TokensCount, e.BytesEmitted, e.StoppedBy)
	} else {
		// Default to kv
		fmt.Fprintf(w, "\nlines: %d, tokens: %d, bytes: %d, stopped: %s\n",
			e.LinesCount, e.TokensCount, e.BytesEmitted, e.StoppedBy)
	}
}
