package token

import "unicode"

// Estimator provides BPE-like token count estimation for LLM API requests.
// Analyzes text structure (words, CJK characters, numbers, punctuation)
// for more accurate estimates than simple character-based division (chars/4).
type Estimator struct{}

func NewEstimator() *Estimator {
	return &Estimator{}
}

// Estimate returns the estimated token count for the given text.
// Uses word-level analysis with CJK/number/punctuation awareness.
func (e *Estimator) Estimate(text string) int64 {
	if len(text) == 0 {
		return 0
	}

	var tokens int64
	runes := []rune(text)
	i := 0

	for i < len(runes) {
		r := runes[i]

		switch {
		case isCJK(r):
			tokens += 2
			i++

		case unicode.IsSpace(r):
			i++

		case unicode.IsDigit(r):
			n := scanWhile(runes[i:], unicode.IsDigit)
			tokens += (int64(n) + 2) / 3
			i += n

		case unicode.IsLetter(r):
			n := scanWhile(runes[i:], unicode.IsLetter)
			tokens += wordTokens(n)
			i += n

		default:
			tokens++
			i++
		}
	}

	if tokens == 0 {
		tokens = 1
	}

	return tokens
}

func scanWhile(runes []rune, pred func(rune) bool) int {
	n := 0
	for n < len(runes) && pred(runes[n]) {
		n++
	}
	return n
}

// wordTokens estimates BPE token count for an alphabetic word of given length.
// Common English words up to ~10 characters typically map to a single token
// in cl100k_base. Longer/uncommon words get split into subword tokens.
func wordTokens(length int) int64 {
	switch {
	case length <= 5:
		return 1
	case length <= 10:
		return 1
	case length <= 15:
		return 2
	case length <= 20:
		return 3
	default:
		return int64(length/7) + 1
	}
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hangul, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r)
}
