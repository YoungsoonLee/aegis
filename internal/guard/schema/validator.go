package schema

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Schema struct {
	Type       string             `json:"type" yaml:"type"`
	Required   []string           `json:"required,omitempty" yaml:"required"`
	Properties map[string]*Schema `json:"properties,omitempty" yaml:"properties"`
	Items      *Schema            `json:"items,omitempty" yaml:"items"`
	Enum       []string           `json:"enum,omitempty" yaml:"enum"`
	Minimum    *float64           `json:"minimum,omitempty" yaml:"minimum"`
	Maximum    *float64           `json:"maximum,omitempty" yaml:"maximum"`
	MinLength  *int               `json:"min_length,omitempty" yaml:"min_length"`
	MaxLength  *int               `json:"max_length,omitempty" yaml:"max_length"`
}

type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func (e ValidationError) String() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

func (r ValidationResult) String() string {
	if r.Valid {
		return "valid"
	}
	msgs := make([]string, len(r.Errors))
	for i, e := range r.Errors {
		msgs[i] = e.String()
	}
	return strings.Join(msgs, "; ")
}

type Validator struct {
	schema *Schema
}

func NewValidator(s *Schema) *Validator {
	return &Validator{schema: s}
}

func (v *Validator) Validate(data any) ValidationResult {
	var errors []ValidationError
	v.schema.validate(data, "$", &errors)
	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

func (v *Validator) ValidateJSON(jsonStr string) ValidationResult {
	if jsonStr == "" {
		return ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Path:    "$",
				Message: "empty content",
			}},
		}
	}

	var data any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Path:    "$",
				Message: fmt.Sprintf("invalid JSON: %s", err.Error()),
			}},
		}
	}

	return v.Validate(data)
}

// ValidateResponse extracts the assistant's message content from an
// OpenAI-compatible chat completion response body, then validates it.
// The second return value indicates whether validation was performed.
func (v *Validator) ValidateResponse(body []byte) (ValidationResult, bool) {
	content := ExtractResponseContent(body)
	if content == "" {
		return ValidationResult{Valid: true}, false
	}
	return v.ValidateJSON(content), true
}

// ExtractResponseContent extracts the assistant's message content from an
// OpenAI-compatible chat completion response.
func ExtractResponseContent(body []byte) string {
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return ""
	}

	if len(resp.Choices) == 0 {
		return ""
	}

	return resp.Choices[0].Message.Content
}

func (s *Schema) validate(data any, path string, errors *[]ValidationError) {
	if s == nil {
		return
	}

	if s.Type != "" && !s.checkType(data, path, errors) {
		return
	}

	if len(s.Enum) > 0 {
		s.checkEnum(data, path, errors)
	}

	switch s.Type {
	case "object":
		s.validateObject(data, path, errors)
	case "array":
		s.validateArray(data, path, errors)
	case "string":
		s.validateString(data, path, errors)
	case "number", "integer":
		s.validateNumber(data, path, errors)
	}
}

func (s *Schema) checkType(data any, path string, errors *[]ValidationError) bool {
	if data == nil {
		*errors = append(*errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("expected %s, got null", s.Type),
		})
		return false
	}

	var valid bool
	switch s.Type {
	case "object":
		_, valid = data.(map[string]any)
	case "array":
		_, valid = data.([]any)
	case "string":
		_, valid = data.(string)
	case "number":
		_, valid = data.(float64)
	case "integer":
		if f, ok := data.(float64); ok {
			valid = f == float64(int64(f))
		}
	case "boolean":
		_, valid = data.(bool)
	default:
		valid = true
	}

	if !valid {
		*errors = append(*errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("expected %s, got %T", s.Type, data),
		})
	}
	return valid
}

func (s *Schema) checkEnum(data any, path string, errors *[]ValidationError) {
	dataStr := fmt.Sprintf("%v", data)
	for _, e := range s.Enum {
		if dataStr == e {
			return
		}
	}
	*errors = append(*errors, ValidationError{
		Path:    path,
		Message: fmt.Sprintf("value %q not in enum %v", dataStr, s.Enum),
	})
}

func (s *Schema) validateObject(data any, path string, errors *[]ValidationError) {
	obj, ok := data.(map[string]any)
	if !ok {
		return
	}

	for _, req := range s.Required {
		if _, exists := obj[req]; !exists {
			*errors = append(*errors, ValidationError{
				Path:    path + "." + req,
				Message: "required field missing",
			})
		}
	}

	for key, propSchema := range s.Properties {
		if val, exists := obj[key]; exists {
			propSchema.validate(val, path+"."+key, errors)
		}
	}
}

func (s *Schema) validateArray(data any, path string, errors *[]ValidationError) {
	arr, ok := data.([]any)
	if !ok {
		return
	}

	if s.Items != nil {
		for i, item := range arr {
			s.Items.validate(item, fmt.Sprintf("%s[%d]", path, i), errors)
		}
	}
}

func (s *Schema) validateString(data any, path string, errors *[]ValidationError) {
	str, ok := data.(string)
	if !ok {
		return
	}

	if s.MinLength != nil && len(str) < *s.MinLength {
		*errors = append(*errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("string length %d < minimum %d", len(str), *s.MinLength),
		})
	}
	if s.MaxLength != nil && len(str) > *s.MaxLength {
		*errors = append(*errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("string length %d > maximum %d", len(str), *s.MaxLength),
		})
	}
}

func (s *Schema) validateNumber(data any, path string, errors *[]ValidationError) {
	num, ok := data.(float64)
	if !ok {
		return
	}

	if s.Minimum != nil && num < *s.Minimum {
		*errors = append(*errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("value %g < minimum %g", num, *s.Minimum),
		})
	}
	if s.Maximum != nil && num > *s.Maximum {
		*errors = append(*errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("value %g > maximum %g", num, *s.Maximum),
		})
	}
}
