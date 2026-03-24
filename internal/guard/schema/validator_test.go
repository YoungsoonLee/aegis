package schema

import (
	"encoding/json"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func TestValidator_TypeChecking(t *testing.T) {
	tests := []struct {
		name     string
		schema   *Schema
		json     string
		valid    bool
		errCount int
	}{
		{"string valid", &Schema{Type: "string"}, `"hello"`, true, 0},
		{"string invalid", &Schema{Type: "string"}, `42`, false, 1},
		{"number valid", &Schema{Type: "number"}, `3.14`, true, 0},
		{"number invalid", &Schema{Type: "number"}, `"text"`, false, 1},
		{"integer valid", &Schema{Type: "integer"}, `42`, true, 0},
		{"integer invalid float", &Schema{Type: "integer"}, `3.14`, false, 1},
		{"integer invalid string", &Schema{Type: "integer"}, `"text"`, false, 1},
		{"boolean valid true", &Schema{Type: "boolean"}, `true`, true, 0},
		{"boolean valid false", &Schema{Type: "boolean"}, `false`, true, 0},
		{"boolean invalid", &Schema{Type: "boolean"}, `1`, false, 1},
		{"object valid", &Schema{Type: "object"}, `{"a":1}`, true, 0},
		{"object invalid", &Schema{Type: "object"}, `[1,2]`, false, 1},
		{"array valid", &Schema{Type: "array"}, `[1,2,3]`, true, 0},
		{"array invalid", &Schema{Type: "array"}, `{"a":1}`, false, 1},
		{"null fails type", &Schema{Type: "string"}, `null`, false, 1},
		{"no type accepts anything", &Schema{}, `"hello"`, true, 0},
		{"no type accepts object", &Schema{}, `{"a":1}`, true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator(tt.schema)
			result := v.ValidateJSON(tt.json)
			if result.Valid != tt.valid {
				t.Errorf("valid = %v, want %v (errors: %v)", result.Valid, tt.valid, result.Errors)
			}
			if len(result.Errors) != tt.errCount {
				t.Errorf("error count = %d, want %d (errors: %v)", len(result.Errors), tt.errCount, result.Errors)
			}
		})
	}
}

func TestValidator_Required(t *testing.T) {
	s := &Schema{
		Type:     "object",
		Required: []string{"name", "age"},
	}
	v := NewValidator(s)

	t.Run("all present", func(t *testing.T) {
		result := v.ValidateJSON(`{"name":"Alice","age":30}`)
		if !result.Valid {
			t.Errorf("expected valid, got errors: %v", result.Errors)
		}
	})

	t.Run("missing one", func(t *testing.T) {
		result := v.ValidateJSON(`{"name":"Alice"}`)
		if result.Valid {
			t.Error("expected invalid")
		}
		if len(result.Errors) != 1 {
			t.Errorf("error count = %d, want 1", len(result.Errors))
		}
		if result.Errors[0].Path != "$.age" {
			t.Errorf("path = %q, want %q", result.Errors[0].Path, "$.age")
		}
	})

	t.Run("missing all", func(t *testing.T) {
		result := v.ValidateJSON(`{}`)
		if result.Valid {
			t.Error("expected invalid")
		}
		if len(result.Errors) != 2 {
			t.Errorf("error count = %d, want 2", len(result.Errors))
		}
	})
}

func TestValidator_Properties(t *testing.T) {
	s := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"name": {Type: "string"},
			"age":  {Type: "number"},
		},
	}
	v := NewValidator(s)

	t.Run("valid types", func(t *testing.T) {
		result := v.ValidateJSON(`{"name":"Alice","age":30}`)
		if !result.Valid {
			t.Errorf("expected valid, got errors: %v", result.Errors)
		}
	})

	t.Run("wrong type", func(t *testing.T) {
		result := v.ValidateJSON(`{"name":42,"age":"thirty"}`)
		if result.Valid {
			t.Error("expected invalid")
		}
		if len(result.Errors) != 2 {
			t.Errorf("error count = %d, want 2", len(result.Errors))
		}
	})

	t.Run("optional fields absent", func(t *testing.T) {
		result := v.ValidateJSON(`{"extra":"field"}`)
		if !result.Valid {
			t.Errorf("optional fields should not cause errors, got: %v", result.Errors)
		}
	})
}

func TestValidator_Items(t *testing.T) {
	s := &Schema{
		Type:  "array",
		Items: &Schema{Type: "string"},
	}
	v := NewValidator(s)

	t.Run("valid items", func(t *testing.T) {
		result := v.ValidateJSON(`["a","b","c"]`)
		if !result.Valid {
			t.Errorf("expected valid, got errors: %v", result.Errors)
		}
	})

	t.Run("invalid items", func(t *testing.T) {
		result := v.ValidateJSON(`["a",42,"c"]`)
		if result.Valid {
			t.Error("expected invalid")
		}
		if len(result.Errors) != 1 {
			t.Errorf("error count = %d, want 1", len(result.Errors))
		}
		if result.Errors[0].Path != "$[1]" {
			t.Errorf("path = %q, want %q", result.Errors[0].Path, "$[1]")
		}
	})

	t.Run("empty array", func(t *testing.T) {
		result := v.ValidateJSON(`[]`)
		if !result.Valid {
			t.Errorf("empty array should be valid, got: %v", result.Errors)
		}
	})
}

func TestValidator_Enum(t *testing.T) {
	s := &Schema{
		Type: "string",
		Enum: []string{"positive", "negative", "neutral"},
	}
	v := NewValidator(s)

	t.Run("valid enum", func(t *testing.T) {
		result := v.ValidateJSON(`"positive"`)
		if !result.Valid {
			t.Errorf("expected valid, got errors: %v", result.Errors)
		}
	})

	t.Run("invalid enum", func(t *testing.T) {
		result := v.ValidateJSON(`"unknown"`)
		if result.Valid {
			t.Error("expected invalid")
		}
	})
}

func TestValidator_MinMaxNumber(t *testing.T) {
	s := &Schema{
		Type:    "number",
		Minimum: ptr(0.0),
		Maximum: ptr(1.0),
	}
	v := NewValidator(s)

	tests := []struct {
		name  string
		json  string
		valid bool
	}{
		{"in range", `0.5`, true},
		{"at minimum", `0`, true},
		{"at maximum", `1`, true},
		{"below minimum", `-0.1`, false},
		{"above maximum", `1.1`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.ValidateJSON(tt.json)
			if result.Valid != tt.valid {
				t.Errorf("valid = %v, want %v (errors: %v)", result.Valid, tt.valid, result.Errors)
			}
		})
	}
}

func TestValidator_MinMaxLength(t *testing.T) {
	s := &Schema{
		Type:      "string",
		MinLength: ptr(2),
		MaxLength: ptr(10),
	}
	v := NewValidator(s)

	tests := []struct {
		name  string
		json  string
		valid bool
	}{
		{"in range", `"hello"`, true},
		{"at min", `"hi"`, true},
		{"at max", `"helloworld"`, true},
		{"too short", `"a"`, false},
		{"too long", `"hello world!"`, false},
		{"empty string", `""`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.ValidateJSON(tt.json)
			if result.Valid != tt.valid {
				t.Errorf("valid = %v, want %v (errors: %v)", result.Valid, tt.valid, result.Errors)
			}
		})
	}
}

func TestValidator_NestedObject(t *testing.T) {
	s := &Schema{
		Type:     "object",
		Required: []string{"result"},
		Properties: map[string]*Schema{
			"result": {
				Type:     "object",
				Required: []string{"answer", "confidence"},
				Properties: map[string]*Schema{
					"answer":     {Type: "string", MinLength: ptr(1)},
					"confidence": {Type: "number", Minimum: ptr(0.0), Maximum: ptr(1.0)},
					"sources": {
						Type:  "array",
						Items: &Schema{Type: "string"},
					},
				},
			},
		},
	}
	v := NewValidator(s)

	t.Run("fully valid", func(t *testing.T) {
		j := `{"result":{"answer":"Paris","confidence":0.95,"sources":["wiki"]}}`
		result := v.ValidateJSON(j)
		if !result.Valid {
			t.Errorf("expected valid, got errors: %v", result.Errors)
		}
	})

	t.Run("missing nested required", func(t *testing.T) {
		j := `{"result":{"answer":"Paris"}}`
		result := v.ValidateJSON(j)
		if result.Valid {
			t.Error("expected invalid")
		}
		found := false
		for _, e := range result.Errors {
			if e.Path == "$.result.confidence" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected error at $.result.confidence, got: %v", result.Errors)
		}
	})

	t.Run("wrong nested type", func(t *testing.T) {
		j := `{"result":{"answer":"Paris","confidence":"high"}}`
		result := v.ValidateJSON(j)
		if result.Valid {
			t.Error("expected invalid")
		}
	})

	t.Run("confidence out of range", func(t *testing.T) {
		j := `{"result":{"answer":"Paris","confidence":1.5}}`
		result := v.ValidateJSON(j)
		if result.Valid {
			t.Error("expected invalid")
		}
	})

	t.Run("empty answer string", func(t *testing.T) {
		j := `{"result":{"answer":"","confidence":0.9}}`
		result := v.ValidateJSON(j)
		if result.Valid {
			t.Error("expected invalid for empty answer")
		}
	})
}

func TestValidator_ValidateJSON_Edge(t *testing.T) {
	v := NewValidator(&Schema{Type: "object"})

	t.Run("empty string", func(t *testing.T) {
		result := v.ValidateJSON("")
		if result.Valid {
			t.Error("empty string should be invalid")
		}
		if result.Errors[0].Message != "empty content" {
			t.Errorf("message = %q", result.Errors[0].Message)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		result := v.ValidateJSON("not json at all")
		if result.Valid {
			t.Error("non-JSON should be invalid")
		}
	})

	t.Run("json number when object expected", func(t *testing.T) {
		result := v.ValidateJSON("42")
		if result.Valid {
			t.Error("number should not match object schema")
		}
	})
}

func TestValidator_ValidateResponse(t *testing.T) {
	s := &Schema{
		Type:     "object",
		Required: []string{"answer"},
		Properties: map[string]*Schema{
			"answer": {Type: "string"},
		},
	}
	v := NewValidator(s)

	t.Run("valid chat completion", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": `{"answer":"Paris"}`}},
			},
		})
		result, validated := v.ValidateResponse(body)
		if !validated {
			t.Error("should have validated")
		}
		if !result.Valid {
			t.Errorf("expected valid, got errors: %v", result.Errors)
		}
	})

	t.Run("invalid content", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": `{"wrong":"field"}`}},
			},
		})
		result, validated := v.ValidateResponse(body)
		if !validated {
			t.Error("should have validated")
		}
		if result.Valid {
			t.Error("expected invalid")
		}
	})

	t.Run("no choices", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"choices": []any{}})
		_, validated := v.ValidateResponse(body)
		if validated {
			t.Error("should skip validation when no choices")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": ""}},
			},
		})
		_, validated := v.ValidateResponse(body)
		if validated {
			t.Error("should skip validation when content is empty")
		}
	})

	t.Run("non-json body", func(t *testing.T) {
		_, validated := v.ValidateResponse([]byte("not json"))
		if validated {
			t.Error("should skip validation for non-JSON body")
		}
	})

	t.Run("non-json content", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "plain text answer"}},
			},
		})
		result, validated := v.ValidateResponse(body)
		if !validated {
			t.Error("should attempt validation")
		}
		if result.Valid {
			t.Error("plain text should fail object schema")
		}
	})
}

func TestExtractResponseContent(t *testing.T) {
	tests := []struct {
		name    string
		body    any
		want    string
	}{
		{
			"standard response",
			map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{"role": "assistant", "content": "Hello!"}},
				},
			},
			"Hello!",
		},
		{
			"json content",
			map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{"content": `{"answer":"42"}`}},
				},
			},
			`{"answer":"42"}`,
		},
		{
			"empty choices",
			map[string]any{"choices": []any{}},
			"",
		},
		{
			"no choices key",
			map[string]any{"id": "test"},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			got := ExtractResponseContent(body)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("invalid json", func(t *testing.T) {
		got := ExtractResponseContent([]byte("not json"))
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

func TestValidator_MultipleErrors(t *testing.T) {
	s := &Schema{
		Type:     "object",
		Required: []string{"a", "b", "c"},
		Properties: map[string]*Schema{
			"a": {Type: "string"},
			"b": {Type: "number"},
			"c": {Type: "boolean"},
		},
	}
	v := NewValidator(s)

	result := v.ValidateJSON(`{"a": 42, "b": "text"}`)
	if result.Valid {
		t.Error("expected invalid")
	}
	// Missing "c" (required) + wrong type for "a" + wrong type for "b"
	if len(result.Errors) < 3 {
		t.Errorf("expected at least 3 errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

func TestValidationResult_String(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := ValidationResult{Valid: true}
		if r.String() != "valid" {
			t.Errorf("got %q", r.String())
		}
	})

	t.Run("invalid", func(t *testing.T) {
		r := ValidationResult{
			Valid: false,
			Errors: []ValidationError{
				{Path: "$.a", Message: "required field missing"},
			},
		}
		s := r.String()
		if s != "$.a: required field missing" {
			t.Errorf("got %q", s)
		}
	})
}
