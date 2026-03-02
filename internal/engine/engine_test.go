package engine

import (
	"bytes"
	"io"
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
		expected := "\nlines: 10, tokens: 25, bytes: 1024, stopped: eof\n"
		if buf.String() != expected {
			t.Errorf("expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("Report JSON", func(t *testing.T) {
		e.SummaryMode = "json"
		var buf bytes.Buffer
		e.Report(&buf)
		expected := "\n" + `{"lines": 10, "tokens": 25, "bytes": 1024, "stopped": "eof"}` + "\n"
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

func TestEngineProcess_MaxTokens(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		maxTokens   int
		wantOut     string
		wantStopped string
	}{
		{
			name:        "ascii stop after 1 word",
			input:       "Hello world",
			maxTokens:   1,
			wantOut:     "Hello",
			wantStopped: "max_tokens",
		},
		{
			name:        "ascii stop after word + space",
			input:       "Hello world",
			maxTokens:   2,
			wantOut:     "Hello ",
			wantStopped: "max_tokens",
		},
		{
			name:        "ascii limit larger than input",
			input:       "Hi",
			maxTokens:   100,
			wantOut:     "Hi",
			wantStopped: "eof",
		},
		{
			name:        "greek two words",
			input:       "Γεια σου",
			maxTokens:   3,
			wantOut:     "Γεια σου",
			wantStopped: "max_tokens",
		},
		{
			name:        "cjk stop after 2",
			input:       "こんにちは",
			maxTokens:   2,
			wantOut:     "こん",
			wantStopped: "max_tokens",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := &Engine{MaxTokens: tc.maxTokens}
			r := strings.NewReader(tc.input)
			var w bytes.Buffer

			err := e.Process(r, &w)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if w.String() != tc.wantOut {
				t.Errorf("output: want %q, got %q", tc.wantOut, w.String())
			}
			if e.StoppedBy != tc.wantStopped {
				t.Errorf("StoppedBy: want %q, got %q", tc.wantStopped, e.StoppedBy)
			}
		})
	}
}

func TestEngine_UTF8Guard(t *testing.T) {
	// Invalid UTF-8 sequence: \xff
	input := []byte("valid text\xffmore text")
	e := &Engine{}

	err := e.Process(bytes.NewReader(input), io.Discard)
	if err == nil {
		t.Error("expected error for invalid UTF-8, got nil")
	} else if !strings.Contains(err.Error(), "invalid UTF-8") {
		t.Errorf("expected UTF-8 error, got: %v", err)
	}
}

func TestEngineProcess_SoftStop_FindsNewline(t *testing.T) {
	e := &Engine{MaxTokens: 1, SoftStop: true}
	input := "token1 token2 and this is the rest of the line\nsecond line"
	r := strings.NewReader(input)
	var w bytes.Buffer

	err := e.Process(r, &w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "token1 token2 and this is the rest of the line\n"
	if w.String() != expected {
		t.Errorf("output: expected %q, got %q", expected, w.String())
	}
	if e.StoppedBy != "max_tokens" {
		t.Errorf("StoppedBy: expected 'max_tokens', got %q", e.StoppedBy)
	}
}

func TestEngineProcess_SoftStop_HitsFailSafe(t *testing.T) {
	e := &Engine{MaxTokens: 1, SoftStop: true}

	size := 1024*1024 + 100
	inputData := make([]byte, size)
	for i := range size {
		inputData[i] = byte('A' + (i % 26))
	}
	inputData[0] = 'H'
	inputData[1] = 'i'
	inputData[2] = ' '

	input := string(inputData)
	r := strings.NewReader(input)
	var w bytes.Buffer

	err := e.Process(r, &w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if e.StoppedBy != "soft_limit" {
		t.Errorf("StoppedBy: expected 'soft_limit', got %q", e.StoppedBy)
	}

	if w.Len() < 1024*1024 {
		t.Errorf("expected output to be at least 1MB, got %d", w.Len())
	}
	if w.Len() == size {
		t.Errorf("expected output to be strictly less than input size, but got full size")
	}
}

type mockSyncer struct {
	bytes.Buffer
	syncs int
}

func (m *mockSyncer) Sync() error {
	m.syncs++
	return nil
}

func TestEngineProcess_Flush(t *testing.T) {
	e := &Engine{Flush: true}
	input := "hello world"
	r := strings.NewReader(input)
	w := &mockSyncer{}

	err := e.Process(r, w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.syncs == 0 {
		t.Errorf("expected Sync to be called when Flush is true")
	}
}
