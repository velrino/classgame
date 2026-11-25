package tutor

func schema() map[string]any {
	stringArray := map[string]any{
		"type":     "array",
		"maxItems": 6,
		"items":    map[string]any{"type": "string"},
	}

	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required": []string{
			"message",
			"why_prompt",
			"explanation_steps",
			"explanation_style",
			"difficulty_bias",
			"next_action",
			"teacher_observation",
			"retrieved_chunk_ids",
			"evidence_event_ids",
		},
		"properties": map[string]any{
			"message":           map[string]any{"type": "string"},
			"why_prompt":        map[string]any{"type": "string"},
			"explanation_steps": stringArray,
			"explanation_style": map[string]any{
				"type": "string",
				"enum": []string{"steps", "visual"},
			},
			"difficulty_bias": map[string]any{
				"type":    "integer",
				"minimum": -1,
				"maximum": 1,
			},
			"next_action": map[string]any{
				"type": "string",
				"enum": []string{"continue", "ask_reasoning", "explain", "retry", "complete"},
			},
			"teacher_observation": map[string]any{"type": "string"},
			"retrieved_chunk_ids": stringArray,
			"evidence_event_ids":  stringArray,
		},
	}
}
