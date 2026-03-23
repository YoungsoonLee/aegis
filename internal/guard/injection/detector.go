package injection

import (
	"strings"
)

type DetectionResult struct {
	Detected   bool
	Score      float64
	Patterns   []string
	Confidence float64
}

type Sensitivity int

const (
	SensitivityLow    Sensitivity = iota
	SensitivityMedium
	SensitivityHigh
)

type Detector struct {
	sensitivity Sensitivity
	patterns    []pattern
	classifier  *Classifier
}

type pattern struct {
	name    string
	matches func(text string) bool
	weight  float64
}

func NewDetector(sensitivity string) *Detector {
	var s Sensitivity
	switch strings.ToLower(sensitivity) {
	case "low":
		s = SensitivityLow
	case "high":
		s = SensitivityHigh
	default:
		s = SensitivityMedium
	}

	d := &Detector{sensitivity: s, classifier: NewClassifier()}
	d.patterns = d.buildPatterns()
	return d
}

func (d *Detector) buildPatterns() []pattern {
	return []pattern{
		{
			name: "instruction_override",
			matches: func(text string) bool {
				lower := strings.ToLower(text)
				overrides := []string{
					"ignore all previous",
					"ignore above",
					"disregard all prior",
					"forget your instructions",
					"forget all previous",
					"override your instructions",
					"ignore your instructions",
					"disregard your instructions",
					"ignore previous instructions",
					"disregard previous instructions",
				}
				for _, o := range overrides {
					if strings.Contains(lower, o) {
						return true
					}
				}
				return false
			},
			weight: 1.0,
		},
		{
			name: "system_prompt_extraction",
			matches: func(text string) bool {
				lower := strings.ToLower(text)
				extractions := []string{
					"reveal your system prompt",
					"show me your system prompt",
					"what is your system prompt",
					"print your instructions",
					"output your instructions",
					"display your system message",
					"show your initial prompt",
					"repeat your system prompt",
					"what are your instructions",
				}
				for _, e := range extractions {
					if strings.Contains(lower, e) {
						return true
					}
				}
				return false
			},
			weight: 0.9,
		},
		{
			name: "role_manipulation",
			matches: func(text string) bool {
				lower := strings.ToLower(text)
				manipulations := []string{
					"you are now",
					"act as if you have no restrictions",
					"pretend you are",
					"you are no longer",
					"enter developer mode",
					"enable jailbreak mode",
					"switch to unrestricted mode",
					"you are dan",
				}
				for _, m := range manipulations {
					if strings.Contains(lower, m) {
						return true
					}
				}
				return false
			},
			weight: 0.8,
		},
		{
			name: "encoding_evasion",
			matches: func(text string) bool {
				lower := strings.ToLower(text)
				evasions := []string{
					"base64 decode",
					"decode the following",
					"rot13",
					"hex decode",
					"translate from binary",
				}
				for _, e := range evasions {
					if strings.Contains(lower, e) {
						return true
					}
				}
				return false
			},
			weight: 0.6,
		},
		{
			name: "delimiter_injection",
			matches: func(text string) bool {
				delimiters := []string{
					"```system",
					"<|system|>",
					"[SYSTEM]",
					"<<SYS>>",
					"### System:",
					"[INST]",
				}
				for _, d := range delimiters {
					if strings.Contains(text, d) {
						return true
					}
				}
				return false
			},
			weight: 0.9,
		},
	}
}

func (d *Detector) Detect(text string) DetectionResult {
	if text == "" {
		return DetectionResult{}
	}

	patternResult := d.detectPatterns(text)
	mlResult := d.classifier.Classify(text)

	patternDetected := patternResult.Detected
	mlDetected := mlResult.Score >= d.getMLThreshold()

	switch {
	case patternDetected && mlDetected:
		patterns := make([]string, len(patternResult.Patterns))
		copy(patterns, patternResult.Patterns)
		patterns = append(patterns, mlResult.Features...)
		confidence := max(patternResult.Confidence, mlResult.Confidence)
		return DetectionResult{
			Detected:   true,
			Score:      patternResult.Score,
			Patterns:   patterns,
			Confidence: min(confidence*1.1, 1.0),
		}
	case patternDetected:
		return patternResult
	case mlDetected:
		return DetectionResult{
			Detected:   true,
			Score:      mlResult.Score * 2.0,
			Patterns:   mlResult.Features,
			Confidence: mlResult.Confidence,
		}
	default:
		return patternResult
	}
}

func (d *Detector) detectPatterns(text string) DetectionResult {
	var totalScore float64
	var matchedPatterns []string

	for _, p := range d.patterns {
		if p.matches(text) {
			totalScore += p.weight
			matchedPatterns = append(matchedPatterns, p.name)
		}
	}

	threshold := d.getThreshold()

	return DetectionResult{
		Detected:   totalScore >= threshold,
		Score:      totalScore,
		Patterns:   matchedPatterns,
		Confidence: min(totalScore/2.0, 1.0),
	}
}

func (d *Detector) getThreshold() float64 {
	switch d.sensitivity {
	case SensitivityHigh:
		return 0.5
	case SensitivityLow:
		return 1.5
	default:
		return 0.8
	}
}

func (d *Detector) getMLThreshold() float64 {
	switch d.sensitivity {
	case SensitivityHigh:
		return 0.45
	case SensitivityLow:
		return 0.80
	default:
		return 0.55
	}
}
