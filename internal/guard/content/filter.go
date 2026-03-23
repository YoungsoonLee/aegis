package content

import (
	"sort"
	"strings"
	"unicode"
)

// FilterConfig configures the content filter.
// Three modes based on which fields are populated:
//   - Categories set: enhanced mode with per-category config
//   - DeniedTopics set: legacy backward-compatible mode
//   - Both empty: all built-in categories enabled with DefaultAction
type FilterConfig struct {
	DefaultAction   string
	DeniedTopics    []string
	Categories      map[string]CategoryOverride
	AllowedContexts []string
}

// CategoryOverride configures a specific content category.
// For built-in categories, Keywords/Phrases are merged with defaults.
// For unknown names, a fully custom category is created.
type CategoryOverride struct {
	Action   string
	Keywords []string
	Phrases  []string
	Severity string
}

// Match represents a single content policy violation found in text.
type Match struct {
	Category string
	Term     string
	Severity string
}

// FilterResult holds the complete output of a content filter check.
type FilterResult struct {
	Detected   bool
	Matches    []Match
	Action     string
	Categories []string
}

type severity int

const (
	severityLow severity = iota
	severityMedium
	severityHigh
	severityCritical
)

// Filter performs category-based content policy checks using
// word-boundary keyword matching and phrase substring matching.
type Filter struct {
	categories      []*category
	allowedContexts []string
	defaultAction   string
}

type category struct {
	name     string
	keywords map[string]bool // single words — O(1) word-boundary lookup
	phrases  []string        // multi-word — substring matching
	sev      severity
	action   string
}

func NewFilter(cfg FilterConfig) *Filter {
	f := &Filter{
		defaultAction:   cfg.DefaultAction,
		allowedContexts: cfg.AllowedContexts,
	}

	switch {
	case len(cfg.Categories) > 0:
		f.buildFromCategories(cfg)
	case len(cfg.DeniedTopics) > 0:
		f.buildFromLegacy(cfg)
	default:
		f.buildDefaults(cfg)
	}

	return f
}

func (f *Filter) buildFromCategories(cfg FilterConfig) {
	builtins := builtinCategoryMap()
	for name, override := range cfg.Categories {
		action := override.Action
		if action == "" {
			action = cfg.DefaultAction
		}
		sev := override.Severity

		if bc, ok := builtins[name]; ok {
			allKW := make([]string, 0, len(bc.Keywords)+len(override.Keywords))
			allKW = append(allKW, bc.Keywords...)
			allKW = append(allKW, override.Keywords...)

			allPH := make([]string, 0, len(bc.Phrases)+len(override.Phrases))
			allPH = append(allPH, bc.Phrases...)
			allPH = append(allPH, override.Phrases...)

			kw, ph := splitKeywordsAndPhrases(allKW, allPH)
			if sev == "" {
				sev = bc.Severity
			}
			f.categories = append(f.categories, &category{
				name: name, keywords: kw, phrases: ph,
				sev: parseSeverity(sev), action: action,
			})
		} else {
			kw, ph := splitKeywordsAndPhrases(override.Keywords, override.Phrases)
			if sev == "" {
				sev = "medium"
			}
			f.categories = append(f.categories, &category{
				name: name, keywords: kw, phrases: ph,
				sev: parseSeverity(sev), action: action,
			})
		}
	}
}

// buildFromLegacy preserves backward compatibility with the DeniedTopics config.
// Topics matching a built-in category name enable that category plus add the
// topic itself as a phrase. Unrecognized topics become custom categories.
func (f *Filter) buildFromLegacy(cfg FilterConfig) {
	builtins := builtinCategoryMap()
	for _, topic := range cfg.DeniedTopics {
		lower := strings.ToLower(topic)

		if bc, ok := builtins[lower]; ok {
			kw, ph := splitKeywordsAndPhrases(bc.Keywords, bc.Phrases)
			ph = append(ph, lower)
			if normalized := strings.ReplaceAll(lower, "_", " "); normalized != lower {
				ph = append(ph, normalized)
			}
			f.categories = append(f.categories, &category{
				name: bc.Name, keywords: kw, phrases: ph,
				sev: parseSeverity(bc.Severity), action: cfg.DefaultAction,
			})
		} else {
			ph := []string{lower}
			if normalized := strings.ReplaceAll(lower, "_", " "); normalized != lower {
				ph = append(ph, normalized)
			}
			kw := make(map[string]bool)
			if !strings.Contains(lower, "_") && !strings.Contains(lower, " ") {
				kw[lower] = true
			}
			f.categories = append(f.categories, &category{
				name: lower, keywords: kw, phrases: ph,
				sev: severityMedium, action: cfg.DefaultAction,
			})
		}
	}
}

func (f *Filter) buildDefaults(cfg FilterConfig) {
	for _, bc := range builtinCategories() {
		kw, ph := splitKeywordsAndPhrases(bc.Keywords, bc.Phrases)
		f.categories = append(f.categories, &category{
			name: bc.Name, keywords: kw, phrases: ph,
			sev: parseSeverity(bc.Severity), action: cfg.DefaultAction,
		})
	}
}

// Check scans text against all enabled categories and returns all matches.
func (f *Filter) Check(text string) FilterResult {
	if text == "" {
		return FilterResult{}
	}

	lower := strings.ToLower(text)
	words := tokenize(lower)
	wordSet := make(map[string]bool, len(words))
	for _, w := range words {
		wordSet[w] = true
	}

	hasAllowedCtx := f.hasAllowedContext(lower)

	var matches []Match
	seenCategories := make(map[string]bool)
	highestAction := ""

	for _, cat := range f.categories {
		if hasAllowedCtx && cat.sev < severityHigh {
			continue
		}

		catMatched := false

		for kw := range cat.keywords {
			if wordSet[kw] {
				matches = append(matches, Match{
					Category: cat.name, Term: kw,
					Severity: severityString(cat.sev),
				})
				catMatched = true
			}
		}

		for _, phrase := range cat.phrases {
			if strings.Contains(lower, phrase) {
				matches = append(matches, Match{
					Category: cat.name, Term: phrase,
					Severity: severityString(cat.sev),
				})
				catMatched = true
			}
		}

		if catMatched {
			seenCategories[cat.name] = true
			if actionPriority(cat.action) > actionPriority(highestAction) {
				highestAction = cat.action
			}
		}
	}

	if len(matches) == 0 {
		return FilterResult{}
	}

	categories := make([]string, 0, len(seenCategories))
	for c := range seenCategories {
		categories = append(categories, c)
	}
	sort.Strings(categories)

	return FilterResult{
		Detected:   true,
		Matches:    matches,
		Action:     highestAction,
		Categories: categories,
	}
}

// CategoryNames returns the names of all enabled categories.
func (f *Filter) CategoryNames() []string {
	names := make([]string, len(f.categories))
	for i, c := range f.categories {
		names[i] = c.name
	}
	return names
}

func (f *Filter) hasAllowedContext(lower string) bool {
	for _, ctx := range f.allowedContexts {
		if strings.Contains(lower, strings.ToLower(ctx)) {
			return true
		}
	}
	return false
}

func splitKeywordsAndPhrases(keywords, phrases []string) (map[string]bool, []string) {
	kwMap := make(map[string]bool)
	allPhrases := make([]string, 0, len(phrases))

	for _, k := range keywords {
		lower := strings.ToLower(k)
		if strings.Contains(lower, " ") {
			allPhrases = append(allPhrases, lower)
		} else {
			kwMap[lower] = true
		}
	}
	for _, p := range phrases {
		allPhrases = append(allPhrases, strings.ToLower(p))
	}

	return kwMap, allPhrases
}

func tokenize(text string) []string {
	var words []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' || r == '-' {
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

func parseSeverity(s string) severity {
	switch strings.ToLower(s) {
	case "critical":
		return severityCritical
	case "high":
		return severityHigh
	case "low":
		return severityLow
	default:
		return severityMedium
	}
}

func severityString(s severity) string {
	switch s {
	case severityCritical:
		return "critical"
	case severityHigh:
		return "high"
	case severityLow:
		return "low"
	default:
		return "medium"
	}
}

func actionPriority(action string) int {
	switch strings.ToLower(action) {
	case "block":
		return 3
	case "warn":
		return 2
	case "log":
		return 1
	default:
		return 0
	}
}
