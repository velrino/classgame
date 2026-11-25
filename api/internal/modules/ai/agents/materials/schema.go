package materials

func schema() map[string]any {
	stringArray := map[string]any{
		"type":  "array",
		"items": map[string]any{"type": "string"},
	}

	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required": []string{
			"summary",
			"key_concepts",
			"usable_for_mission",
			"suggested_questions",
			"cautions",
		},
		"properties": map[string]any{
			"summary":             map[string]any{"type": "string"},
			"key_concepts":        stringArray,
			"usable_for_mission":  map[string]any{"type": "boolean"},
			"suggested_questions": stringArray,
			"cautions":            stringArray,
		},
	}
}
