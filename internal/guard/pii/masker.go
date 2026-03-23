package pii

import "sort"

type Masker struct {
	detector *Detector
}

func NewMasker(detector *Detector) *Masker {
	return &Masker{detector: detector}
}

// Mask detects PII in the text and returns the masked version along with detection results.
func (m *Masker) Mask(text string) (string, []DetectionResult) {
	detections := m.detector.Detect(text)
	if len(detections) == 0 {
		return text, nil
	}

	// Sort by position descending so replacements don't shift indices
	sort.Slice(detections, func(i, j int) bool {
		return detections[i].Start > detections[j].Start
	})

	masked := []byte(text)
	for _, d := range detections {
		entity, ok := DefaultEntities[d.EntityType]
		if !ok {
			continue
		}
		replacement := entity.MaskFormat(d.Match)
		masked = append(masked[:d.Start], append([]byte(replacement), masked[d.End:]...)...)
	}

	return string(masked), detections
}
