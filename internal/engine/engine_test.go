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
}
