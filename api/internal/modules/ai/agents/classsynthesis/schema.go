package classsynthesis

func schema() map[string]any {
	cluster := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"label", "student_ids", "evidence_labels"},
		"properties": map[string]any{
			"label": map[string]any{"type": "string"},
			"student_ids": map[string]any{
				"type":     "array",
				"maxItems": 100,
				"items":    map[string]any{"type": "string"},
			},
			"evidence_labels": map[string]any{
				"type":     "array",
				"maxItems": 20,
				"items":    map[string]any{"type": "string"},
			},
		},
	}

	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"common_strengths", "common_needs", "next_adaptation"},
		"properties": map[string]any{
			"common_strengths": map[string]any{
				"type":     "array",
				"maxItems": 4,
				"items":    cluster,
			},
			"common_needs": map[string]any{
				"type":     "array",
				"maxItems": 4,
				"items":    cluster,
			},
			"next_adaptation": map[string]any{"type": "string"},
		},
	}
}
