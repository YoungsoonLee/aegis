package pii

import (
	"regexp"
	"strings"
)

type Entity struct {
	Type       string
	Pattern    *regexp.Regexp
	MaskChar   rune
	MaskFormat func(match string) string
}

var DefaultEntities = map[string]Entity{
	"email": {
		Type:    "email",
		Pattern: regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
		MaskFormat: func(match string) string {
			parts := strings.SplitN(match, "@", 2)
			if len(parts) != 2 {
				return "***@***"
			}
			return maskString(parts[0]) + "@" + maskString(parts[1])
		},
	},
	"phone": {
		Type:    "phone",
		Pattern: regexp.MustCompile(`(?:\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}`),
		MaskFormat: func(match string) string {
			if len(match) < 4 {
				return "****"
			}
			return strings.Repeat("*", len(match)-4) + match[len(match)-4:]
		},
	},
	"ssn": {
		Type:    "ssn",
		Pattern: regexp.MustCompile(`\b[0-9]{3}-[0-9]{2}-[0-9]{4}\b`),
		MaskFormat: func(match string) string {
			return "***-**-" + match[len(match)-4:]
		},
	},
	"credit_card": {
		Type:    "credit_card",
		Pattern: regexp.MustCompile(`\b(?:[0-9]{4}[-\s]?){3}[0-9]{4}\b`),
		MaskFormat: func(match string) string {
			cleaned := strings.ReplaceAll(strings.ReplaceAll(match, "-", ""), " ", "")
			if len(cleaned) < 4 {
				return "****"
			}
			return strings.Repeat("*", len(cleaned)-4) + cleaned[len(cleaned)-4:]
		},
	},
	"api_key": {
		Type:    "api_key",
		Pattern: regexp.MustCompile(`(?i)(?:sk|api|key|token|secret|password)[-_]?[a-zA-Z0-9]{20,}`),
		MaskFormat: func(match string) string {
			if len(match) <= 8 {
				return strings.Repeat("*", len(match))
			}
			return match[:4] + strings.Repeat("*", len(match)-8) + match[len(match)-4:]
		},
	},
}

type DetectionResult struct {
	EntityType string
	Match      string
	Start      int
	End        int
	Confidence float64
}

type Detector struct {
	entities map[string]Entity
}

func NewDetector(enabledEntities []string) *Detector {
	entities := make(map[string]Entity)
	for _, name := range enabledEntities {
		if e, ok := DefaultEntities[name]; ok {
			entities[name] = e
		}
	}
	return &Detector{entities: entities}
}

func (d *Detector) Detect(text string) []DetectionResult {
	var results []DetectionResult

	for _, entity := range d.entities {
		matches := entity.Pattern.FindAllStringIndex(text, -1)
		for _, loc := range matches {
			results = append(results, DetectionResult{
				EntityType: entity.Type,
				Match:      text[loc[0]:loc[1]],
				Start:      loc[0],
				End:        loc[1],
				Confidence: 0.9,
			})
		}
	}

	return results
}

func maskString(s string) string {
	if len(s) <= 2 {
		return strings.Repeat("*", len(s))
	}
	return string(s[0]) + strings.Repeat("*", len(s)-2) + string(s[len(s)-1])
}
