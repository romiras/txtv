package segmenter

import "testing"

func TestCountAndCut_ASCII(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxTokens int
		wantCount int
		wantCut   int
	}{
		{
			name:      "two words",
			input:     "Hello world",
			maxTokens: 0,
			wantCount: 3, // "Hello", " ", "world"
			wantCut:   11,
		},
		{
			name:      "stop after first word",
			input:     "Hello world",
			maxTokens: 1,
			wantCount: 1,
			wantCut:   5, // "Hello"
		},
		{
			name:      "stop after space",
			input:     "Hello world",
			maxTokens: 2,
			wantCount: 2,
			wantCut:   6, // "Hello "
		},
		{
			name:      "stop after second word",
			input:     "Hello world",
			maxTokens: 3,
			wantCount: 3,
			wantCut:   11, // "Hello world"
		},
		{
			name:      "empty string",
			input:     "",
			maxTokens: 0,
			wantCount: 0,
			wantCut:   0,
		},
		{
			name:      "newline only",
			input:     "\n",
			maxTokens: 0,
			wantCount: 1, // bare \n = 1 token
			wantCut:   1,
		},
		{
			name:      "punctuation",
			input:     "Hello, world!",
			maxTokens: 0,
			wantCount: 5, // "Hello", ",", " ", "world", "!"
			wantCut:   13,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			count, cut := CountAndCut([]byte(tc.input), tc.maxTokens)
			if count != tc.wantCount {
				t.Errorf("count: want %d, got %d", tc.wantCount, count)
			}
			if cut != tc.wantCut {
				t.Errorf("cutoff: want %d, got %d (segment: %q)", tc.wantCut, cut, tc.input[:cut])
			}
		})
	}
}

func TestCountAndCut_Greek(t *testing.T) {
	// "Γεια σου κόσμε" = 3 words + 2 spaces = 5 tokens
	input := "Γεια σου κόσμε"

	count, cut := CountAndCut([]byte(input), 0)
	if count != 5 {
		t.Errorf("total count: want 5, got %d", count)
	}
	if cut != len(input) {
		t.Errorf("cutoff should be full length, got %d", cut)
	}

	// Stop after 2 tokens: "Γεια", " "
	count2, cut2 := CountAndCut([]byte(input), 2)
	if count2 != 2 {
		t.Errorf("count at 2: want 2, got %d", count2)
	}
	got := string([]byte(input)[:cut2])
	if got != "Γεια " {
		t.Errorf("cut at 2 tokens: want %q, got %q", "Γεια ", got)
	}

	// Stop after 3 tokens: "Γεια", " ", "σου" → "Γεια σου"
	count3, cut3 := CountAndCut([]byte(input), 3)
	if count3 != 3 {
		t.Errorf("count at 3: want 3, got %d", count3)
	}
	got3 := string([]byte(input)[:cut3])
	if got3 != "Γεια σου" {
		t.Errorf("cut at 3 tokens: want %q, got %q", "Γεια σου", got3)
	}
}

func TestCountAndCut_CJK(t *testing.T) {
	// Each CJK ideograph is a standalone token (no grouping).
	input := "こんにちは" // 5 katakana/hiragana characters

	count, cut := CountAndCut([]byte(input), 0)
	if count != 5 {
		t.Errorf("CJK: want 5 tokens, got %d", count)
	}
	if cut != len(input) {
		t.Errorf("CJK cutoff: want %d, got %d", len(input), cut)
	}

	// Stop after 2 tokens
	count2, cut2 := CountAndCut([]byte(input), 2)
	if count2 != 2 {
		t.Errorf("CJK stop at 2: want 2, got %d", count2)
	}
	// Each hiragana rune is 3 bytes: こ(3)+ん(3) = 6 bytes
	if cut2 != 6 {
		t.Errorf("CJK cutoff at 2: want 6 bytes, got %d", cut2)
	}
}

func TestCountAndCut_EmptyLines(t *testing.T) {
	// Each \n is a token.
	input := "a\n\nb"
	// tokens: "a"(1), "\n"(2), "\n"(3), "b"(4)
	count, cut := CountAndCut([]byte(input), 0)
	if count != 4 {
		t.Errorf("want 4 tokens, got %d", count)
	}
	if cut != len(input) {
		t.Errorf("cutoff mismatch")
	}

	// Stop at 3 tokens: "a", "\n", "\n"
	_, cut3 := CountAndCut([]byte(input), 3)
	if string([]byte(input)[:cut3]) != "a\n\n" {
		t.Errorf("cut3 = %q", string([]byte(input)[:cut3]))
	}
}
