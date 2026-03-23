package pii

import (
	"strings"
	"testing"
)

func TestMasker_EmailMasking(t *testing.T) {
	d := NewDetector([]string{"email"})
	m := NewMasker(d)

	masked, detections := m.Mask("contact john@example.com for info")

	if len(detections) != 1 {
		t.Fatalf("got %d detections, want 1", len(detections))
	}
	if strings.Contains(masked, "john@example.com") {
		t.Error("masked text still contains original email")
	}
	if !strings.Contains(masked, "@") {
		t.Error("masked email should still contain @ symbol")
	}
	if !strings.Contains(masked, "contact ") {
		t.Error("non-PII text should be preserved")
	}
}

func TestMasker_SSNMasking(t *testing.T) {
	d := NewDetector([]string{"ssn"})
	m := NewMasker(d)

	masked, detections := m.Mask("SSN is 123-45-6789")

	if len(detections) != 1 {
		t.Fatalf("got %d detections, want 1", len(detections))
	}
	if strings.Contains(masked, "123-45") {
		t.Error("masked text still contains SSN prefix")
	}
	if !strings.Contains(masked, "6789") {
		t.Error("masked SSN should preserve last 4 digits")
	}
}

func TestMasker_CreditCardMasking(t *testing.T) {
	d := NewDetector([]string{"credit_card"})
	m := NewMasker(d)

	masked, detections := m.Mask("card: 4111-1111-1111-1111")

	if len(detections) != 1 {
		t.Fatalf("got %d detections, want 1", len(detections))
	}
	if !strings.Contains(masked, "1111") {
		t.Error("masked card should preserve last 4 digits")
	}
	if strings.Contains(masked, "4111-1111-1111-1111") {
		t.Error("masked text still contains original card number")
	}
}

func TestMasker_PhoneMasking(t *testing.T) {
	d := NewDetector([]string{"phone"})
	m := NewMasker(d)

	masked, detections := m.Mask("call 555-123-4567")

	if len(detections) != 1 {
		t.Fatalf("got %d detections, want 1", len(detections))
	}
	if !strings.Contains(masked, "4567") {
		t.Error("masked phone should preserve last 4 digits")
	}
}

func TestMasker_NoDetection(t *testing.T) {
	d := NewDetector([]string{"email", "phone"})
	m := NewMasker(d)

	original := "hello world, this is clean text"
	masked, detections := m.Mask(original)

	if len(detections) != 0 {
		t.Errorf("got %d detections, want 0", len(detections))
	}
	if masked != original {
		t.Errorf("clean text should be unchanged: got %q, want %q", masked, original)
	}
}

func TestMasker_MultiplePII(t *testing.T) {
	d := NewDetector([]string{"email", "ssn"})
	m := NewMasker(d)

	input := "email: alice@test.com, ssn: 111-22-3333"
	masked, detections := m.Mask(input)

	if len(detections) != 2 {
		t.Fatalf("got %d detections, want 2", len(detections))
	}
	if strings.Contains(masked, "alice@test.com") {
		t.Error("email should be masked")
	}
	if strings.Contains(masked, "111-22") {
		t.Error("SSN prefix should be masked")
	}
}

func TestMasker_EmptyString(t *testing.T) {
	d := NewDetector([]string{"email"})
	m := NewMasker(d)

	masked, detections := m.Mask("")

	if len(detections) != 0 {
		t.Errorf("empty string should yield 0 detections, got %d", len(detections))
	}
	if masked != "" {
		t.Errorf("empty string should remain empty, got %q", masked)
	}
}

func TestMasker_AdjacentPII(t *testing.T) {
	d := NewDetector([]string{"email"})
	m := NewMasker(d)

	input := "a@b.co c@d.co"
	masked, detections := m.Mask(input)

	if len(detections) != 2 {
		t.Fatalf("got %d detections, want 2", len(detections))
	}
	if strings.Contains(masked, "a@b.co") || strings.Contains(masked, "c@d.co") {
		t.Error("both emails should be masked")
	}
}
