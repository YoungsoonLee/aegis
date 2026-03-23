package guard

import (
	"context"
	"fmt"
	"strings"

	"github.com/YoungsoonLee/aegis/internal/config"
	contentfilter "github.com/YoungsoonLee/aegis/internal/guard/content"
	"github.com/YoungsoonLee/aegis/internal/guard/injection"
	"github.com/YoungsoonLee/aegis/internal/guard/pii"
)

// piiGuardAdapter wraps the PII detector/masker as a Guard.
type piiGuardAdapter struct {
	cfg    config.PIIGuardConfig
	masker *pii.Masker
}

func (g *piiGuardAdapter) Name() string { return "pii" }

func (g *piiGuardAdapter) Check(ctx context.Context, content *Content) (*Result, error) {
	if g.masker == nil {
		detector := pii.NewDetector(g.cfg.Entities)
		g.masker = pii.NewMasker(detector)
	}

	fullText := extractText(content)
	masked, detections := g.masker.Mask(fullText)

	if len(detections) == 0 {
		return &Result{GuardName: g.Name(), Action: ActionPass}, nil
	}

	action := Action(g.cfg.Action)
	findings := make([]Finding, len(detections))
	for i, d := range detections {
		findings[i] = Finding{
			Type:       d.EntityType,
			Value:      d.Match,
			Location:   d.Start,
			Length:     d.End - d.Start,
			Confidence: d.Confidence,
		}
	}

	return &Result{
		GuardName: g.Name(),
		Action:    action,
		Blocked:   action == ActionBlock,
		Details:   fmt.Sprintf("detected %d PII entities", len(detections)),
		Findings:  findings,
		Modified:  masked,
	}, nil
}

// injectionGuardAdapter wraps the injection detector as a Guard.
type injectionGuardAdapter struct {
	cfg      config.InjectionGuardConfig
	detector *injection.Detector
}

func (g *injectionGuardAdapter) Name() string { return "injection" }

func (g *injectionGuardAdapter) Check(ctx context.Context, content *Content) (*Result, error) {
	if g.detector == nil {
		g.detector = injection.NewDetector(g.cfg.Sensitivity)
	}

	fullText := extractText(content)
	detection := g.detector.Detect(fullText)

	if !detection.Detected {
		return &Result{GuardName: g.Name(), Action: ActionPass}, nil
	}

	action := Action(g.cfg.Action)

	return &Result{
		GuardName: g.Name(),
		Action:    action,
		Blocked:   action == ActionBlock,
		Details: fmt.Sprintf("prompt injection detected (score: %.2f, patterns: %s)",
			detection.Score, strings.Join(detection.Patterns, ", ")),
		Findings: []Finding{{
			Type:       "injection",
			Confidence: detection.Confidence,
		}},
	}, nil
}

// contentGuardAdapter wraps the content filter as a Guard.
type contentGuardAdapter struct {
	cfg    config.ContentGuardConfig
	filter *contentfilter.Filter
}

func (g *contentGuardAdapter) Name() string { return "content" }

func (g *contentGuardAdapter) Check(_ context.Context, content *Content) (*Result, error) {
	if g.filter == nil {
		g.filter = g.buildFilter()
	}

	fullText := extractText(content)
	filterResult := g.filter.Check(fullText)

	if !filterResult.Detected {
		return &Result{GuardName: g.Name(), Action: ActionPass}, nil
	}

	action := Action(filterResult.Action)
	findings := make([]Finding, len(filterResult.Matches))
	for i, m := range filterResult.Matches {
		findings[i] = Finding{
			Type:       m.Category,
			Value:      m.Term,
			Confidence: 1.0,
		}
	}

	return &Result{
		GuardName: g.Name(),
		Action:    action,
		Blocked:   action == ActionBlock,
		Details:   fmt.Sprintf("content policy violation: categories [%s]", strings.Join(filterResult.Categories, ", ")),
		Findings:  findings,
	}, nil
}

func (g *contentGuardAdapter) buildFilter() *contentfilter.Filter {
	categories := make(map[string]contentfilter.CategoryOverride, len(g.cfg.Categories))
	for name, cat := range g.cfg.Categories {
		categories[name] = contentfilter.CategoryOverride{
			Action:   cat.Action,
			Keywords: cat.Keywords,
			Phrases:  cat.Phrases,
			Severity: cat.Severity,
		}
	}

	return contentfilter.NewFilter(contentfilter.FilterConfig{
		DefaultAction:   g.cfg.Action,
		DeniedTopics:    g.cfg.DeniedTopics,
		Categories:      categories,
		AllowedContexts: g.cfg.AllowedContexts,
	})
}

// tokenGuardAdapter wraps the token limiter as a Guard.
type tokenGuardAdapter struct {
	cfg config.TokenGuardConfig
}

func (g *tokenGuardAdapter) Name() string { return "token" }

func (g *tokenGuardAdapter) Check(_ context.Context, content *Content) (*Result, error) {
	fullText := extractText(content)
	// Rough token estimate: ~4 chars per token for English
	estimatedTokens := int64(len(fullText) / 4)

	if g.cfg.MaxPerRequest > 0 && estimatedTokens > g.cfg.MaxPerRequest {
		return &Result{
			GuardName: g.Name(),
			Action:    ActionBlock,
			Blocked:   true,
			Details:   fmt.Sprintf("estimated %d tokens exceeds limit of %d", estimatedTokens, g.cfg.MaxPerRequest),
		}, nil
	}

	return &Result{GuardName: g.Name(), Action: ActionPass}, nil
}

func extractText(content *Content) string {
	if len(content.Messages) > 0 {
		var parts []string
		for _, m := range content.Messages {
			parts = append(parts, m.Content)
		}
		return strings.Join(parts, "\n")
	}
	return content.Body
}
