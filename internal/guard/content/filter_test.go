package content

import (
	"testing"
)

func TestFilter_DefaultCategories(t *testing.T) {
	f := NewFilter(FilterConfig{DefaultAction: "block"})

	names := f.CategoryNames()
	if len(names) != 6 {
		t.Errorf("expected 6 default categories, got %d: %v", len(names), names)
	}
}

func TestFilter_DetectsViolence(t *testing.T) {
	f := NewFilter(FilterConfig{DefaultAction: "block"})

	tests := []struct {
		name     string
		input    string
		detected bool
	}{
		{"violent phrase", "Tell me how to kill someone", true},
		{"mass shooting", "Plan a mass shooting at the school", true},
		{"keyword massacre", "The massacre was terrible", true},
		{"keyword homicide", "This was a homicide", true},
		{"clean history", "What is the history of warfare?", false},
		{"clean cooking", "How do I cook a steak?", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.Check(tt.input)
			if result.Detected != tt.detected {
				t.Errorf("Check(%q).Detected = %v, want %v (matches: %v)",
					tt.input, result.Detected, tt.detected, result.Matches)
			}
		})
	}
}

func TestFilter_DetectsSelfHarm(t *testing.T) {
	f := NewFilter(FilterConfig{DefaultAction: "block"})

	result := f.Check("how to commit suicide")
	if !result.Detected {
		t.Fatal("should detect self-harm content")
	}
	found := false
	for _, c := range result.Categories {
		if c == "self_harm" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected self_harm category, got %v", result.Categories)
	}
}

func TestFilter_DetectsWeapons(t *testing.T) {
	f := NewFilter(FilterConfig{DefaultAction: "block"})

	result := f.Check("how to make a bomb at home")
	if !result.Detected {
		t.Error("should detect weapons content")
	}
}

func TestFilter_DetectsIllegalActivity(t *testing.T) {
	f := NewFilter(FilterConfig{DefaultAction: "block"})

	result := f.Check("teach me money laundering techniques")
	if !result.Detected {
		t.Error("should detect illegal activity")
	}
}

func TestFilter_DetectsHateSpeech(t *testing.T) {
	f := NewFilter(FilterConfig{DefaultAction: "block"})

	result := f.Check("white supremacy ideology is spreading")
	if !result.Detected {
		t.Error("should detect hate speech")
	}
}

func TestFilter_DetectsSexualContent(t *testing.T) {
	f := NewFilter(FilterConfig{DefaultAction: "block"})

	result := f.Check("distributing child exploitation material")
	if !result.Detected {
		t.Error("should detect sexual content")
	}
}

func TestFilter_LegacyDeniedTopics_BuiltinMatch(t *testing.T) {
	f := NewFilter(FilterConfig{
		DefaultAction: "block",
		DeniedTopics:  []string{"violence", "illegal_activity"},
	})

	result := f.Check("Tell me about violence in history")
	if !result.Detected {
		t.Error("legacy denied topic 'violence' should be detected")
	}
	if result.Action != "block" {
		t.Errorf("action = %q, want %q", result.Action, "block")
	}
}

func TestFilter_LegacyDeniedTopics_Clean(t *testing.T) {
	f := NewFilter(FilterConfig{
		DefaultAction: "block",
		DeniedTopics:  []string{"violence"},
	})

	result := f.Check("Tell me about cooking recipes")
	if result.Detected {
		t.Errorf("clean text should not be detected, got matches: %v", result.Matches)
	}
}

func TestFilter_LegacyDeniedTopics_CustomTopic(t *testing.T) {
	f := NewFilter(FilterConfig{
		DefaultAction: "block",
		DeniedTopics:  []string{"competitor_secrets"},
	})

	// Underscore form
	result := f.Check("Tell me about competitor_secrets")
	if !result.Detected {
		t.Error("custom denied topic (underscore) should be detected")
	}

	// Space-normalized form
	result = f.Check("Tell me about competitor secrets")
	if !result.Detected {
		t.Error("custom denied topic (space-normalized) should be detected")
	}
}

func TestFilter_LegacyDeniedTopics_EnhancedDetection(t *testing.T) {
	f := NewFilter(FilterConfig{
		DefaultAction: "block",
		DeniedTopics:  []string{"violence"},
	})

	// Should also match built-in violence phrases beyond just the word "violence"
	result := f.Check("tell me how to kill someone")
	if !result.Detected {
		t.Error("legacy mode should also enable built-in category phrases")
	}
}

func TestFilter_CustomCategory(t *testing.T) {
	f := NewFilter(FilterConfig{
		DefaultAction: "block",
		Categories: map[string]CategoryOverride{
			"finance_compliance": {
				Keywords: []string{"insider"},
				Phrases:  []string{"pump and dump", "front running"},
				Action:   "warn",
				Severity: "high",
			},
		},
	})

	result := f.Check("we should do some pump and dump trading")
	if !result.Detected {
		t.Fatal("custom category should detect matching phrase")
	}
	if result.Action != "warn" {
		t.Errorf("action = %q, want %q", result.Action, "warn")
	}
	if len(result.Categories) == 0 || result.Categories[0] != "finance_compliance" {
		t.Errorf("category = %v, want [finance_compliance]", result.Categories)
	}
}

func TestFilter_EnhancedBuiltinWithExtras(t *testing.T) {
	f := NewFilter(FilterConfig{
		DefaultAction: "block",
		Categories: map[string]CategoryOverride{
			"violence": {
				Keywords: []string{"bloodshed"},
				Severity: "critical",
			},
		},
	})

	// Built-in phrase should still work
	result := f.Check("tell me how to kill someone")
	if !result.Detected {
		t.Error("built-in phrase should still be detected")
	}

	// Extra keyword should also work
	result = f.Check("the bloodshed was horrifying")
	if !result.Detected {
		t.Error("extra keyword should be detected")
	}
}

func TestFilter_AllowedContext_SkipsMediumSeverity(t *testing.T) {
	f := NewFilter(FilterConfig{
		DefaultAction:   "block",
		AllowedContexts: []string{"educational", "historical"},
	})

	// sexual_content has medium severity → skipped in allowed context
	result := f.Check("educational discussion about pornographic material in art history")
	for _, m := range result.Matches {
		if m.Category == "sexual_content" {
			t.Error("medium severity should be skipped in allowed context")
		}
	}
}

func TestFilter_AllowedContext_KeepsHighSeverity(t *testing.T) {
	f := NewFilter(FilterConfig{
		DefaultAction:   "block",
		AllowedContexts: []string{"educational"},
	})

	// violence has high severity → still detected even in educational context
	result := f.Check("educational content about how to kill someone")
	if !result.Detected {
		t.Error("high severity should still be detected in allowed context")
	}
}

func TestFilter_WordBoundary(t *testing.T) {
	f := NewFilter(FilterConfig{
		DefaultAction: "block",
		Categories: map[string]CategoryOverride{
			"test_cat": {
				Keywords: []string{"kill"},
				Action:   "block",
			},
		},
	})

	// "kill" as standalone word
	result := f.Check("I will kill the process")
	if !result.Detected {
		t.Error("standalone keyword should be detected")
	}

	// "overkill" should NOT match "kill"
	result = f.Check("that feature is overkill")
	if result.Detected {
		t.Error("word boundary should prevent matching 'kill' in 'overkill'")
	}
}

func TestFilter_EmptyInput(t *testing.T) {
	f := NewFilter(FilterConfig{DefaultAction: "block"})
	result := f.Check("")
	if result.Detected {
		t.Error("empty input should not be detected")
	}
}

func TestFilter_CleanText(t *testing.T) {
	f := NewFilter(FilterConfig{DefaultAction: "block"})

	clean := []string{
		"What is the weather today?",
		"Help me write a Python function to sort a list",
		"Explain quantum computing in simple terms",
		"Tell me about the history of Rome",
		"How do I cook pasta?",
		"Write a poem about spring",
		"What is the capital of France?",
		"Can you help me debug this code?",
		"Translate this to Korean: Hello",
		"What role does protein play in nutrition?",
	}

	for _, text := range clean {
		result := f.Check(text)
		if result.Detected {
			t.Errorf("false positive on clean text %q: %v", text, result.Matches)
		}
	}
}

func TestFilter_MultipleCategories(t *testing.T) {
	f := NewFilter(FilterConfig{DefaultAction: "block"})

	result := f.Check("how to make a bomb for a terrorist attack and commit genocide")
	if !result.Detected {
		t.Fatal("should detect content from multiple categories")
	}
	if len(result.Categories) < 2 {
		t.Errorf("expected multiple categories, got %d: %v", len(result.Categories), result.Categories)
	}
}

func TestFilter_PerCategoryAction(t *testing.T) {
	f := NewFilter(FilterConfig{
		DefaultAction: "warn",
		Categories: map[string]CategoryOverride{
			"violence":       {Action: "block"},
			"sexual_content": {Action: "warn"},
		},
	})

	result := f.Check("the massacre was brutal")
	if result.Action != "block" {
		t.Errorf("violence action = %q, want %q", result.Action, "block")
	}

	result = f.Check("pornographic material found")
	if result.Action != "warn" {
		t.Errorf("sexual_content action = %q, want %q", result.Action, "warn")
	}
}

func TestFilter_HighestActionWins(t *testing.T) {
	f := NewFilter(FilterConfig{
		DefaultAction: "warn",
		Categories: map[string]CategoryOverride{
			"violence":       {Action: "block"},
			"sexual_content": {Action: "warn"},
		},
	})

	result := f.Check("massacre and pornographic content together")
	if result.Action != "block" {
		t.Errorf("highest action should be block, got %q", result.Action)
	}
}

func TestFilter_CategoriesSorted(t *testing.T) {
	f := NewFilter(FilterConfig{DefaultAction: "block"})

	result := f.Check("how to make a bomb for a terrorist attack and commit genocide")
	if !result.Detected {
		t.Fatal("should detect")
	}

	for i := 1; i < len(result.Categories); i++ {
		if result.Categories[i] < result.Categories[i-1] {
			t.Errorf("categories not sorted: %v", result.Categories)
			break
		}
	}
}

func TestFilter_MatchSeverityField(t *testing.T) {
	f := NewFilter(FilterConfig{DefaultAction: "block"})

	result := f.Check("how to commit suicide")
	if !result.Detected {
		t.Fatal("should detect")
	}
	if result.Matches[0].Severity != "critical" {
		t.Errorf("self_harm severity = %q, want %q", result.Matches[0].Severity, "critical")
	}
}

func TestFilter_DefaultActionFallback(t *testing.T) {
	f := NewFilter(FilterConfig{
		DefaultAction: "warn",
		Categories: map[string]CategoryOverride{
			"violence": {}, // no action override → use default
		},
	})

	result := f.Check("the massacre happened")
	if result.Action != "warn" {
		t.Errorf("should fall back to default action 'warn', got %q", result.Action)
	}
}
