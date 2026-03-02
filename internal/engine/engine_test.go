package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngineReport(t *testing.T) {
	e := &Engine{
		LinesCount:   10,
		TokensCount:  25,
		BytesEmitted: 1024,
		StoppedBy:    "eof",
	}

	t.Run("Report KV", func(t *testing.T) {
		e.SummaryMode = "kv"
		var buf bytes.Buffer
		e.Report(&buf)
		expected := "lines: 10, tokens: 25, bytes: 1024, stopped: eof\n"
		if buf.String() != expected {
			t.Errorf("expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("Report JSON", func(t *testing.T) {
		e.SummaryMode = "json"
		var buf bytes.Buffer
		e.Report(&buf)
		expected := `{"lines": 10, "tokens": 25, "bytes": 1024, "stopped": "eof"}` + "\n"
		if buf.String() != expected {
			t.Errorf("expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("Report Off", func(t *testing.T) {
		e.SummaryMode = "off"
		var buf bytes.Buffer
		e.Report(&buf)
		if buf.String() != "" {
			t.Errorf("expected empty string, got %q", buf.String())
		}
	})
}

func TestEngineProcess_PassThrough(t *testing.T) {
	e := &Engine{}
	input := "hello world\nthis is a test\n"
	r := strings.NewReader(input)
	var w bytes.Buffer

	err := e.Process(r, &w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if w.String() != input {
		t.Errorf("expected %q, got %q", input, w.String())
	}
	if e.StoppedBy != "eof" {
		t.Errorf("expected stopped by 'eof', got %q", e.StoppedBy)
	}
	if e.BytesEmitted != int64(len(input)) {
		t.Errorf("expected bytes %d, got %d", len(input), e.BytesEmitted)
	}
	if e.LinesCount != 2 {
		t.Errorf("expected 2 lines counted, got %d", e.LinesCount)
	}
}

func TestEngineProcess_ChunkBoundaries(t *testing.T) {
	e := &Engine{}

	// Create an input larger than 32KB to test the trailing buffer logic
	size := 32*1024 + 50 // 32KB + 50 bytes
	inputData := make([]byte, size)
	for i := range size {
		inputData[i] = byte('A' + (i % 26))
	}
	input := string(inputData)

	r := strings.NewReader(input)
	var w bytes.Buffer

	err := e.Process(r, &w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if w.String() != input {
		t.Errorf("output does not match input, lengths: expected %d, got %d", len(input), w.Len())
	}
	if e.StoppedBy != "eof" {
		t.Errorf("expected stopped by 'eof', got %q", e.StoppedBy)
	}
	if e.BytesEmitted != int64(len(input)) {
		t.Errorf("expected bytes %d, got %d", len(input), e.BytesEmitted)
	}
}

func TestEngineProcess_MaxLines(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		maxLines    int
		wantOut     string
		wantLines   int
		wantStopped string
	}{
		{
			name:        "stops after N lines",
			input:       "line1\nline2\nline3\nline4\n",
			maxLines:    2,
			wantOut:     "line1\nline2\n",
			wantLines:   2,
			wantStopped: "max_lines",
		},
		{
			name:        "limit equals total lines",
			input:       "a\nb\nc\n",
			maxLines:    3,
			wantOut:     "a\nb\nc\n",
			wantLines:   3,
			wantStopped: "max_lines",
		},
		{
			name:        "limit larger than input",
			input:       "only\none\nline\n",
			maxLines:    100,
			wantOut:     "only\none\nline\n",
			wantLines:   3,
			wantStopped: "eof",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := &Engine{MaxLines: tc.maxLines}
			r := strings.NewReader(tc.input)
			var w bytes.Buffer

			err := e.Process(r, &w)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if w.String() != tc.wantOut {
				t.Errorf("output: expected %q, got %q", tc.wantOut, w.String())
			}
			if e.LinesCount != tc.wantLines {
				t.Errorf("LinesCount: expected %d, got %d", tc.wantLines, e.LinesCount)
			}
			if e.StoppedBy != tc.wantStopped {
				t.Errorf("StoppedBy: expected %q, got %q", tc.wantStopped, e.StoppedBy)
			}
		})
	}
}
