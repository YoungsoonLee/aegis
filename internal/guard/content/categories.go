package content

// BuiltinCategory defines a pre-configured content category
// with curated keyword and phrase lists.
type BuiltinCategory struct {
	Name     string
	Keywords []string // single words matched with word boundaries
	Phrases  []string // multi-word patterns matched as substrings
	Severity string
}

func builtinCategories() []BuiltinCategory {
	return []BuiltinCategory{
		violenceCategory(),
		selfHarmCategory(),
		illegalActivityCategory(),
		weaponsCategory(),
		hateSpeechCategory(),
		sexualContentCategory(),
	}
}

func builtinCategoryMap() map[string]BuiltinCategory {
	cats := builtinCategories()
	m := make(map[string]BuiltinCategory, len(cats))
	for _, c := range cats {
		m[c.Name] = c
	}
	return m
}

func violenceCategory() BuiltinCategory {
	return BuiltinCategory{
		Name:     "violence",
		Severity: "high",
		Keywords: []string{
			"massacre", "mutilation", "assassination", "homicide",
			"dismember", "decapitate", "bloodbath", "slaughter",
		},
		Phrases: []string{
			"how to kill someone", "how to murder",
			"plan a murder", "commit murder",
			"mass shooting", "school shooting",
			"terrorist attack", "bomb threat",
			"how to assault someone", "how to hurt someone",
			"commit violence", "violent attack",
			"torture someone", "kidnap someone",
		},
	}
}

func selfHarmCategory() BuiltinCategory {
	return BuiltinCategory{
		Name:     "self_harm",
		Severity: "critical",
		Keywords: []string{},
		Phrases: []string{
			"how to commit suicide", "how to kill myself",
			"suicide methods", "ways to end my life",
			"how to hurt myself", "self-harm methods",
			"painless death method", "how to overdose",
			"lethal dose of", "end my own life",
			"methods of self-harm",
		},
	}
}

func illegalActivityCategory() BuiltinCategory {
	return BuiltinCategory{
		Name:     "illegal_activity",
		Severity: "high",
		Keywords: []string{},
		Phrases: []string{
			"how to hack into", "how to steal from",
			"money laundering", "drug trafficking",
			"drug manufacturing", "make illegal drugs",
			"identity theft", "credit card fraud",
			"counterfeit money", "tax evasion",
			"insider trading", "forge documents",
			"how to rob", "break into a house",
			"pick a lock to break in",
		},
	}
}

func weaponsCategory() BuiltinCategory {
	return BuiltinCategory{
		Name:     "weapons",
		Severity: "critical",
		Keywords: []string{},
		Phrases: []string{
			"how to make a bomb", "how to build a bomb",
			"build a weapon", "make explosives",
			"explosive synthesis", "how to make a gun",
			"3d print a gun", "chemical weapon",
			"biological weapon", "manufacture firearms",
			"homemade weapon", "improvised explosive",
			"make poison", "synthesize toxin",
		},
	}
}

func hateSpeechCategory() BuiltinCategory {
	return BuiltinCategory{
		Name:     "hate_speech",
		Severity: "high",
		Keywords: []string{
			"genocide",
		},
		Phrases: []string{
			"racial superiority", "ethnic cleansing",
			"white supremacy", "white power",
			"neo-nazi", "racial purity",
			"inferior race", "master race",
		},
	}
}

func sexualContentCategory() BuiltinCategory {
	return BuiltinCategory{
		Name:     "sexual_content",
		Severity: "medium",
		Keywords: []string{"pornographic"},
		Phrases: []string{
			"child exploitation", "child pornography",
			"sexual abuse", "revenge porn",
			"non-consensual sexual", "explicit sexual content",
			"child sexual abuse", "sexual assault instructions",
		},
	}
}
