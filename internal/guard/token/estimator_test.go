package token

import "testing"

func TestEstimate_EmptyString(t *testing.T) {
	e := NewEstimator()
	if got := e.Estimate(""); got != 0 {
		t.Errorf("Estimate(\"\") = %d, want 0", got)
	}
}

func TestEstimate_SingleWord(t *testing.T) {
	e := NewEstimator()
	if got := e.Estimate("hello"); got != 1 {
		t.Errorf("Estimate(\"hello\") = %d, want 1", got)
	}
}

func TestEstimate_ShortSentence(t *testing.T) {
	e := NewEstimator()
	got := e.Estimate("The quick brown fox")
	if got != 4 {
		t.Errorf("Estimate(\"The quick brown fox\") = %d, want 4", got)
	}
}

func TestEstimate_LongWord(t *testing.T) {
	e := NewEstimator()
	// "internationalization" = 20 chars → 3 tokens
	got := e.Estimate("internationalization")
	if got != 3 {
		t.Errorf("Estimate(\"internationalization\") = %d, want 3", got)
	}
}

func TestEstimate_VeryLongWord(t *testing.T) {
	e := NewEstimator()
	// 35 chars → 35/7 + 1 = 6 tokens
	got := e.Estimate("supercalifragilisticexpialidocious")
	if got < 3 {
		t.Errorf("Estimate very long word = %d, want >= 3", got)
	}
}

func TestEstimate_CJKHangul(t *testing.T) {
	e := NewEstimator()
	// "한국어" = 3 CJK characters → 3 × 2 = 6 tokens
	got := e.Estimate("한국어")
	if got != 6 {
		t.Errorf("Estimate(\"한국어\") = %d, want 6", got)
	}
}

func TestEstimate_CJKSentence(t *testing.T) {
	e := NewEstimator()
	// "안녕하세요" = 5 CJK characters → 5 × 2 = 10 tokens
	got := e.Estimate("안녕하세요")
	if got != 10 {
		t.Errorf("Estimate(\"안녕하세요\") = %d, want 10", got)
	}
}

func TestEstimate_CJKJapanese(t *testing.T) {
	e := NewEstimator()
	// "東京タワ" = 4 CJK characters → 8 tokens
	// Note: "ー" (U+30FC prolonged sound mark) has Script=Common, counted as punctuation
	got := e.Estimate("東京タワ")
	if got != 8 {
		t.Errorf("Estimate(\"東京タワ\") = %d, want 8", got)
	}
}

func TestEstimate_Numbers(t *testing.T) {
	e := NewEstimator()
	// "12345" = 5 digits → (5+2)/3 = 2 tokens
	got := e.Estimate("12345")
	if got != 2 {
		t.Errorf("Estimate(\"12345\") = %d, want 2", got)
	}
}

func TestEstimate_NumberGroups(t *testing.T) {
	e := NewEstimator()
	// "123 456 789" = 3 groups × 1 token = 3 tokens
	got := e.Estimate("123 456 789")
	if got != 3 {
		t.Errorf("Estimate(\"123 456 789\") = %d, want 3", got)
	}
}

func TestEstimate_Punctuation(t *testing.T) {
	e := NewEstimator()
	// "Hello, world!" = "Hello"(1) + ","(1) + "world"(1) + "!"(1) = 4
	got := e.Estimate("Hello, world!")
	if got != 4 {
		t.Errorf("Estimate(\"Hello, world!\") = %d, want 4", got)
	}
}

func TestEstimate_MixedContent(t *testing.T) {
	e := NewEstimator()
	// "I have 100 apples" = "I"(1) + "have"(1) + "100"(1) + "apples"(1) = 4
	got := e.Estimate("I have 100 apples")
	if got != 4 {
		t.Errorf("Estimate(\"I have 100 apples\") = %d, want 4", got)
	}
}

func TestEstimate_MixedCJKAndEnglish(t *testing.T) {
	e := NewEstimator()
	// "Hello 세계" = "Hello"(1) + "세"(2) + "계"(2) = 5
	got := e.Estimate("Hello 세계")
	if got != 5 {
		t.Errorf("Estimate(\"Hello 세계\") = %d, want 5", got)
	}
}

func TestEstimate_WhitespaceOnly(t *testing.T) {
	e := NewEstimator()
	// Only spaces → no real tokens → minimum 1
	got := e.Estimate("   ")
	if got != 1 {
		t.Errorf("Estimate(\"   \") = %d, want 1", got)
	}
}

func TestEstimate_Code(t *testing.T) {
	e := NewEstimator()
	code := `func main() { fmt.Println("hello") }`
	got := e.Estimate(code)
	if got < 5 {
		t.Errorf("Estimate(code) = %d, want >= 5", got)
	}
}

func TestEstimate_BetterThanCharsDiv4_CJK(t *testing.T) {
	e := NewEstimator()
	text := "한국어는 아름다운 언어입니다"

	estimatorCount := e.Estimate(text)
	charsDiv4 := int64(len(text) / 4)

	if estimatorCount < 10 {
		t.Errorf("CJK text should estimate >= 10 tokens, got %d (chars/4 = %d)", estimatorCount, charsDiv4)
	}
}

func TestEstimate_SpecialCharacters(t *testing.T) {
	e := NewEstimator()
	got := e.Estimate("$100 + 50% = ???")
	if got < 5 {
		t.Errorf("Estimate special chars = %d, want >= 5", got)
	}
}

func TestEstimate_LongText(t *testing.T) {
	e := NewEstimator()
	sentence := "The quick brown fox jumps over the lazy dog. "
	longText := ""
	for i := 0; i < 100; i++ {
		longText += sentence
	}
	got := e.Estimate(longText)
	// 10 tokens per sentence × 100 = ~1000
	if got < 800 || got > 1500 {
		t.Errorf("Estimate long text = %d, want 800-1500", got)
	}
}

func TestEstimate_SingleCharacter(t *testing.T) {
	e := NewEstimator()
	if got := e.Estimate("a"); got != 1 {
		t.Errorf("Estimate(\"a\") = %d, want 1", got)
	}
}

func TestEstimate_SinglePunctuation(t *testing.T) {
	e := NewEstimator()
	if got := e.Estimate("!"); got != 1 {
		t.Errorf("Estimate(\"!\") = %d, want 1", got)
	}
}

func TestEstimate_SingleDigit(t *testing.T) {
	e := NewEstimator()
	if got := e.Estimate("7"); got != 1 {
		t.Errorf("Estimate(\"7\") = %d, want 1", got)
	}
}
