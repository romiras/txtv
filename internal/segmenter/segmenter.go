// Package segmenter implements the language-agnostic Hybrid Heuristic tokenizer.
//
// # Strategy (UAX #29 inspired)
//
// Scan the input rune-by-rune. A token is:
//   - A run of consecutive "alphabetic word runes" (unicode.IsLetter && NOT a
//     logographic/syllabic script) → emitted as one word token.
//   - Any other rune (logographic, syllabic, punctuation, space, emoji, \n …)
//     → emitted as its own individual token.
//
// "Logographic/syllabic" scripts are those whose characters never form
// multi-character word units in UAX #29: Han (CJK), Hiragana, Katakana,
// Hangul syllables, Bopomofo, Yi, and similar. We test membership using Go's
// built-in unicode.RangeTable values — no hardcoded magic numbers.
//
// This is fully language-agnostic: Latin, Greek, Cyrillic, Arabic, Hebrew,
// Devanagari, Thai, … all form word-run tokens naturally because their
// letters are not members of the logographic range tables below.
package segmenter

import (
	"unicode"
	"unicode/utf8"
)

// logographic is the set of Unicode scripts whose characters are treated as
// standalone tokens (each character = 1 token), matching UAX #29 word-break
// behaviour for these scripts.
var logographic = []*unicode.RangeTable{
	unicode.Han,
	unicode.Hiragana,
	unicode.Katakana,
	unicode.Hangul,
	unicode.Bopomofo,
	unicode.Yi,
	unicode.Khmer,
	unicode.Lao,
	unicode.Thai,
	unicode.Myanmar,
	unicode.Tibetan,
}

// isWordRune returns true if r should be grouped with adjacent runes into a
// word token. It excludes any rune that belongs to a logographic/syllabic
// script (even though unicode.IsLetter is true for those).
func isWordRune(r rune) bool {
	if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
		return false
	}
	for _, rt := range logographic {
		if unicode.Is(rt, r) {
			return false
		}
	}
	return true
}

// CountAndCut counts the tokens in data and returns:
//   - count: total tokens found.
//   - cutoff: byte offset immediately after the maxTokens-th token
//     (or len(data) if the limit is not reached or maxTokens <= 0).
func CountAndCut(data []byte, maxTokens int) (count, cutoff int) {
	i := 0
	for i < len(data) {
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size == 1 {
			// Skip invalid UTF-8 byte — not a token.
			i++
			continue
		}

		if isWordRune(r) {
			// Consume the full run of word runes as ONE token.
			j := i + size
			for j < len(data) {
				r2, s2 := utf8.DecodeRune(data[j:])
				if r2 == utf8.RuneError && s2 == 1 {
					break
				}
				if !isWordRune(r2) {
					break
				}
				j += s2
			}
			count++
			i = j
		} else {
			// Every other rune (logographic, space, punct, emoji, \n …) is
			// its own standalone token.
			count++
			i += size
		}

		if maxTokens > 0 && count >= maxTokens {
			return count, i
		}
	}
	return count, len(data)
}
