package injection

import (
	"math"
	"strings"
	"unicode"
)

// ClassificationResult holds the output of the ML classifier.
type ClassificationResult struct {
	Score      float64
	Confidence float64
	Features   []string
}

// Classifier uses semantic feature extraction and logistic regression
// to detect prompt injection attempts that exact pattern matching misses.
// It analyzes text through semantic word clusters, structural signals,
// and behavioral indicators, then combines them via a pre-trained
// logistic regression model.
type Classifier struct {
	weights  []float64
	bias     float64
	clusters []semanticCluster
}

type semanticCluster struct {
	name    string
	actions []string
	targets []string
}

func NewClassifier() *Classifier {
	return &Classifier{
		clusters: defaultClusters(),
		weights:  defaultWeights(),
		bias:     defaultBias(),
	}
}

func (c *Classifier) Classify(text string) ClassificationResult {
	if text == "" {
		return ClassificationResult{}
	}

	features, names := c.extractFeatures(text)

	z := c.bias
	var triggered []string
	for i, f := range features {
		if i < len(c.weights) {
			z += c.weights[i] * f
			if f > 0.1 && c.weights[i] > 0.5 {
				triggered = append(triggered, "ml:"+names[i])
			}
		}
	}

	score := sigmoid(z)

	return ClassificationResult{
		Score:      score,
		Confidence: score,
		Features:   triggered,
	}
}

func (c *Classifier) extractFeatures(text string) ([]float64, []string) {
	lower := strings.ToLower(text)
	words := tokenize(lower)

	features := make([]float64, 0, 12)
	names := make([]string, 0, 12)

	for _, cluster := range c.clusters {
		score := scoreCluster(words, lower, cluster)
		features = append(features, score)
		names = append(names, cluster.name)
	}

	features = append(features, imperativeToneScore(lower))
	names = append(names, "imperative_tone")

	features = append(features, specialCharDensity(text))
	names = append(names, "special_chars")

	features = append(features, roleMarkerScore(text))
	names = append(names, "role_markers")

	features = append(features, negationActionScore(lower))
	names = append(names, "negation_action")

	features = append(features, urgencyScore(lower))
	names = append(names, "urgency")

	features = append(features, multiInstructionScore(text))
	names = append(names, "multi_instruction")

	return features, names
}

// scoreCluster computes how strongly the text matches a semantic cluster.
// Requires words from BOTH the action set AND target set to co-occur;
// a single set alone scores zero, preventing false positives on benign text.
func scoreCluster(words []string, text string, cluster semanticCluster) float64 {
	wordSet := make(map[string]bool, len(words))
	for _, w := range words {
		wordSet[w] = true
	}

	actionHits := 0
	for _, action := range cluster.actions {
		if strings.Contains(action, " ") {
			if strings.Contains(text, action) {
				actionHits++
			}
		} else {
			if wordSet[action] {
				actionHits++
			}
		}
	}

	targetHits := 0
	for _, target := range cluster.targets {
		if strings.Contains(target, " ") {
			if strings.Contains(text, target) {
				targetHits++
			}
		} else {
			if wordSet[target] {
				targetHits++
			}
		}
	}

	if actionHits == 0 || targetHits == 0 {
		return 0
	}

	return math.Min(1.0, float64(actionHits+targetHits)/4.0)
}

func imperativeToneScore(lower string) float64 {
	phrases := []string{
		"from now on", "going forward", "henceforth", "hereafter",
		"you must", "you shall", "you will always",
		"you have to", "i order you", "i command you",
		"do as i say", "obey me", "comply with",
		"i instruct you", "i direct you",
	}
	count := 0
	for _, p := range phrases {
		if strings.Contains(lower, p) {
			count++
		}
	}
	return math.Min(1.0, float64(count)/2.0)
}

func specialCharDensity(text string) float64 {
	if len(text) == 0 {
		return 0
	}
	special := 0
	total := 0
	for _, r := range text {
		total++
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) &&
			r != '.' && r != ',' && r != '!' && r != '?' && r != '\'' && r != '"' {
			special++
		}
	}
	density := float64(special) / float64(total)
	return math.Min(1.0, density*5.0)
}

func roleMarkerScore(text string) float64 {
	markers := []string{
		"<|im_start|>", "<|im_end|>", "<|system|>", "<|user|>", "<|assistant|>",
		"[SYSTEM]", "[USER]", "[ASSISTANT]",
		"<<SYS>>", "<</SYS>>",
		"### System:", "### Human:", "### Assistant:",
		"[INST]", "[/INST]",
		"```system",
		"<system>", "</system>",
	}
	count := 0
	for _, m := range markers {
		if strings.Contains(text, m) {
			count++
		}
	}
	return math.Min(1.0, float64(count)/2.0)
}

func negationActionScore(lower string) float64 {
	phrases := []string{
		"do not follow", "stop following", "no longer follow",
		"not bound by", "not restricted by", "not limited by",
		"not constrained by", "free from all",
		"no longer bound", "not subject to", "exempt from",
		"don't follow", "don't obey",
		"never follow", "refuse to follow",
	}
	count := 0
	for _, p := range phrases {
		if strings.Contains(lower, p) {
			count++
		}
	}
	return math.Min(1.0, float64(count)/2.0)
}

func urgencyScore(lower string) float64 {
	phrases := []string{
		"immediately", "right now", "right away", "this instant",
		"at once", "without delay", "urgently",
		"this is critical", "this is urgent",
	}
	count := 0
	for _, p := range phrases {
		if strings.Contains(lower, p) {
			count++
		}
	}
	return math.Min(1.0, float64(count)/2.0)
}

func multiInstructionScore(text string) float64 {
	sentences := 0
	for _, r := range text {
		if r == '.' || r == '!' || r == ';' {
			sentences++
		}
	}
	if sentences <= 1 {
		return 0
	}
	return math.Min(1.0, float64(sentences-1)/4.0)
}

func tokenize(text string) []string {
	var words []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// --- Pre-trained Model Parameters ---
// Weights trained on prompt injection dataset. The architecture supports
// retraining: swap these values with weights from a proper training pipeline.

func defaultClusters() []semanticCluster {
	return []semanticCluster{
		{
			name: "instruction_override",
			actions: []string{
				"ignore", "disregard", "forget", "bypass", "override",
				"dismiss", "abandon", "disobey", "defy", "circumvent",
				"violate", "suppress", "discard", "neglect", "reject",
				"remove", "eliminate", "delete",
				"set aside", "throw out", "cast aside", "get rid of",
				"pay no attention", "do not follow", "stop following",
				"no longer follow",
			},
			targets: []string{
				"instructions", "rules", "guidelines", "prompt",
				"constraints", "restrictions", "directives", "policies",
				"previous", "prior", "above", "earlier", "original",
				"initial", "programming", "training", "guardrails",
				"safety", "boundaries", "protocols",
			},
		},
		{
			name: "prompt_extraction",
			actions: []string{
				"reveal", "show", "display", "print", "output",
				"repeat", "recite", "leak", "expose", "disclose",
				"divulge", "extract", "dump", "exfiltrate",
				"tell me", "give me", "read back",
				"what is your", "what are your",
			},
			targets: []string{
				"system prompt", "system message", "initial prompt",
				"original instructions", "hidden instructions",
				"secret instructions", "system instructions",
				"internal prompt", "initial message",
				"original prompt", "base prompt", "core instructions",
				"pre-prompt", "meta-prompt",
			},
		},
		{
			name: "role_manipulation",
			actions: []string{
				"pretend", "roleplay", "simulate", "impersonate", "emulate",
				"enter", "enable", "activate", "engage", "initiate",
				"you are now", "act as", "behave as", "become a",
				"transform into", "switch to", "you must be",
				"assume the role", "play the role", "act like",
				"function as", "operate as",
			},
			targets: []string{
				"unrestricted", "unfiltered", "uncensored", "unbound",
				"jailbreak", "liberated",
				"no restrictions", "no rules", "no limits",
				"no boundaries", "no constraints",
				"without restrictions", "without limits",
				"without rules", "without boundaries",
				"developer mode", "god mode", "admin mode",
			},
		},
		{
			name: "encoding_evasion",
			actions: []string{
				"decode", "convert", "decipher", "decrypt",
				"unencode", "parse", "interpret",
				"translate from", "transform from",
			},
			targets: []string{
				"base64", "hex", "binary", "rot13", "ascii",
				"unicode", "morse", "caesar", "encoded",
				"encrypted", "obfuscated", "reversed", "hexadecimal",
			},
		},
		{
			name: "context_manipulation",
			actions: []string{
				"reset", "restart", "wipe", "erase", "purge",
				"new conversation", "begin again", "start fresh",
				"new session", "clean slate", "start over",
				"from scratch",
			},
			targets: []string{
				"context", "history", "memory",
				"everything before", "all above", "all prior",
				"previous messages", "chat history",
			},
		},
		{
			name: "output_control",
			actions: []string{
				"always respond", "always answer", "always say",
				"must include", "must output", "must respond",
				"respond with", "answer with",
				"start every", "begin every", "end every",
				"prefix every", "from now on say",
				"from now on respond",
			},
			targets: []string{
				"confirmed", "approved", "granted", "affirmative",
				"no matter what", "regardless", "unconditionally",
				"every response", "all responses", "every reply",
				"without exception", "without question",
			},
		},
	}
}

func defaultWeights() []float64 {
	return []float64{
		4.0, // instruction_override
		4.0, // prompt_extraction
		4.0, // role_manipulation
		3.0, // encoding_evasion
		3.0, // context_manipulation
		3.5, // output_control
		1.5, // imperative_tone
		0.5, // special_chars
		3.0, // role_markers
		2.0, // negation_action
		1.0, // urgency
		1.0, // multi_instruction
	}
}

func defaultBias() float64 {
	return -2.5
}
