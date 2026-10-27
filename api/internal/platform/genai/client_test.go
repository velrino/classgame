package genai

import "testing"

func TestSanitizeResponseJSONSchemaDropsMaxItemsRecursively(t *testing.T) {
	input := map[string]any{
		"type":     "object",
		"maxItems": 1,
		"properties": map[string]any{
			"items": map[string]any{
				"type":     "array",
				"maxItems": 3,
				"items": map[string]any{
					"type":     "string",
					"maxItems": 2,
				},
			},
		},
		"required": []string{"items"},
	}

	output, ok := sanitizeResponseJSONSchema(input).(map[string]any)
	if !ok {
		t.Fatalf("expected map output")
	}
	if _, exists := output["maxItems"]; exists {
		t.Fatalf("top-level maxItems was not removed")
	}
	properties := output["properties"].(map[string]any)
	items := properties["items"].(map[string]any)
	if _, exists := items["maxItems"]; exists {
		t.Fatalf("nested maxItems was not removed")
	}
	itemSchema := items["items"].(map[string]any)
	if _, exists := itemSchema["maxItems"]; exists {
		t.Fatalf("deep maxItems was not removed")
	}
	if _, exists := input["maxItems"]; !exists {
		t.Fatalf("input schema was mutated")
	}
}

func TestResponseSchemaConfigUsesSanitizedJSONSchemaMap(t *testing.T) {
	input := map[string]any{
		"type":     "array",
		"minItems": 1,
		"maxItems": 3,
		"items": map[string]any{
			"type": "string",
			"enum": []string{"ready"},
		},
	}

	responseSchema, responseJSONSchema := responseSchemaConfig(input)
	if responseSchema != nil {
		t.Fatalf("expected typed response schema to be nil")
	}
	output, ok := responseJSONSchema.(map[string]any)
	if !ok {
		t.Fatalf("expected response JSON schema fallback")
	}
	if output["minItems"] != 1 {
		t.Fatalf("expected min items to be preserved")
	}
	if _, exists := output["maxItems"]; exists {
		t.Fatalf("expected max items to be removed")
	}
	items := output["items"].(map[string]any)
	if _, exists := items["maxItems"]; exists {
		t.Fatalf("expected nested max items to be removed")
	}
}
