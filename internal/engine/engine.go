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
			} else {
				available = 0 // Wait for more data
			}
		}

		if available > 0 {
			// Determine the actual slice to write, respecting --max-lines.
			writeSlice := buf[:available]
			hitLimit := false

			if e.MaxLines > 0 {
				// Scan for newlines and stop atomically after the Nth line.
				remaining := e.MaxLines - e.LinesCount
				pos := 0
				for i, b := range writeSlice {
					if b == '\n' {
						remaining--
						if remaining == 0 {
							// Include this newline then stop.
							pos = i + 1
							hitLimit = true
							break
						}
					}
				}
				if hitLimit {
					writeSlice = writeSlice[:pos]
				}
			}

			// Count newlines in the slice we are about to emit.
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

			if hitLimit {
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

		// --max-tokens requires Task 1.3 Segmenter; counter stays 0 until then.
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
