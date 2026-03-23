package pii

import (
	"testing"
)

func TestDetector_Email(t *testing.T) {
	d := NewDetector([]string{"email"})

	tests := []struct {
		name  string
		input string
		want  int
		email string
	}{
		{"simple email", "contact me at john@example.com please", 1, "john@example.com"},
		{"email with dots", "user.name+tag@sub.domain.org", 1, "user.name+tag@sub.domain.org"},
		{"multiple emails", "from alice@a.com to bob@b.com", 2, ""},
		{"no email", "hello world no email here", 0, ""},
		{"email at start", "admin@test.io is the admin", 1, "admin@test.io"},
		{"email at end", "send to support@company.co.kr", 1, "support@company.co.kr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := d.Detect(tt.input)
			if len(results) != tt.want {
				t.Errorf("got %d detections, want %d", len(results), tt.want)
			}
			if tt.want == 1 && tt.email != "" && results[0].Match != tt.email {
				t.Errorf("got match %q, want %q", results[0].Match, tt.email)
			}
			for _, r := range results {
				if r.EntityType != "email" {
					t.Errorf("got entity type %q, want %q", r.EntityType, "email")
				}
			}
		})
	}
}

func TestDetector_Phone(t *testing.T) {
	d := NewDetector([]string{"phone"})

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"us format", "call me at 555-123-4567", 1},
		{"parens format", "phone: (555) 123-4567", 1},
		{"dots format", "555.123.4567 is my number", 1},
		{"with country code", "+1-555-123-4567", 1},
		{"no phone", "there is no phone number here", 0},
		{"too short", "123-456 is not a phone", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := d.Detect(tt.input)
			if len(results) != tt.want {
				t.Errorf("got %d detections, want %d", len(results), tt.want)
			}
		})
	}
}

func TestDetector_SSN(t *testing.T) {
	d := NewDetector([]string{"ssn"})

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"valid ssn", "my SSN is 123-45-6789", 1},
		{"in sentence", "SSN: 999-88-7777 on file", 1},
		{"no ssn", "1234-56-789 is not a valid format", 0},
		{"no dashes", "123456789 without dashes", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := d.Detect(tt.input)
			if len(results) != tt.want {
				t.Errorf("got %d detections, want %d", len(results), tt.want)
			}
		})
	}
}

func TestDetector_CreditCard(t *testing.T) {
	d := NewDetector([]string{"credit_card"})

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"with dashes", "card: 4111-1111-1111-1111", 1},
		{"with spaces", "card: 4111 1111 1111 1111", 1},
		{"no separator", "card: 4111111111111111", 1},
		{"too short", "4111-1111-1111 not enough", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := d.Detect(tt.input)
			if len(results) != tt.want {
				t.Errorf("got %d detections, want %d", len(results), tt.want)
			}
		})
	}
}

func TestDetector_APIKey(t *testing.T) {
	d := NewDetector([]string{"api_key"})

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"openai key", "use sk-abcdefghij1234567890klmn", 1},
		{"generic api key", "apikey_abcdefghijklmnopqrst12345", 1},
		{"token", "token_abcdefghijklmnopqrst12345", 1},
		{"no key", "this is just normal text with no keys", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := d.Detect(tt.input)
			if len(results) != tt.want {
				t.Errorf("got %d detections, want %d", len(results), tt.want)
			}
		})
	}
}

func TestDetector_MultipleEntities(t *testing.T) {
	d := NewDetector([]string{"email", "phone", "ssn"})

	input := "Contact john@example.com or call 555-123-4567. SSN: 123-45-6789"
	results := d.Detect(input)

	if len(results) != 3 {
		t.Fatalf("got %d detections, want 3", len(results))
	}

	types := map[string]bool{}
	for _, r := range results {
		types[r.EntityType] = true
	}
	for _, want := range []string{"email", "phone", "ssn"} {
		if !types[want] {
			t.Errorf("missing entity type %q in results", want)
		}
	}
}

func TestDetector_EmptyEntities(t *testing.T) {
	d := NewDetector([]string{})
	results := d.Detect("john@example.com 555-123-4567")
	if len(results) != 0 {
		t.Errorf("got %d detections with no entities enabled, want 0", len(results))
	}
}

func TestDetector_UnknownEntity(t *testing.T) {
	d := NewDetector([]string{"nonexistent"})
	results := d.Detect("john@example.com")
	if len(results) != 0 {
		t.Errorf("got %d detections with unknown entity, want 0", len(results))
	}
}

func TestDetector_ResultFields(t *testing.T) {
	d := NewDetector([]string{"email"})
	input := "hello john@example.com world"
	results := d.Detect(input)

	if len(results) != 1 {
		t.Fatalf("got %d detections, want 1", len(results))
	}

	r := results[0]
	if r.Start < 0 || r.End > len(input) || r.Start >= r.End {
		t.Errorf("invalid position: start=%d end=%d input_len=%d", r.Start, r.End, len(input))
	}
	if input[r.Start:r.End] != r.Match {
		t.Errorf("position mismatch: slice=%q match=%q", input[r.Start:r.End], r.Match)
	}
	if r.Confidence <= 0 || r.Confidence > 1.0 {
		t.Errorf("confidence out of range: %f", r.Confidence)
	}
}

func TestMaskString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"a", "*"},
		{"ab", "**"},
		{"abc", "a*c"},
		{"abcde", "a***e"},
		{"hello", "h***o"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := maskString(tt.input)
			if got != tt.want {
				t.Errorf("maskString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
