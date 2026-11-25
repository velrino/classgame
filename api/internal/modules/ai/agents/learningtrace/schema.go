package learningtrace

func schema() map[string]any {
	stringArray := map[string]any{
		"type":     "array",
		"minItems": 1,
		"maxItems": 4,
		"items":    map[string]any{"type": "string"},
	}

	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required": []string{
			"summary",
			"strengths",
			"improvement_areas",
			"next_adaptation",
			"flow",
		},
		"properties": map[string]any{
			"summary": map[string]any{
				"type":        "string",
				"description": "A concise evidence-based summary of the student's observable learning path.",
			},
			"strengths":         stringArray,
			"improvement_areas": stringArray,
			"next_adaptation": map[string]any{
				"type":        "string",
				"description": "One concrete adaptation Mia should retrieve and apply in the next interaction.",
			},
			"flow": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"nodes", "edges"},
				"properties": map[string]any{
					"nodes": map[string]any{
						"type":     "array",
						"minItems": 3,
						"maxItems": 12,
						"items": map[string]any{
							"type":                 "object",
							"additionalProperties": false,
							"required":             []string{"id", "kind", "label", "description", "evidence_event_ids", "position"},
							"properties": map[string]any{
								"id": map[string]any{"type": "string"},
								"kind": map[string]any{
									"type": "string",
									"enum": []string{
										"start",
										"question",
										"answer",
										"misconception",
										"feedback",
										"adaptation",
										"mastery",
										"complete",
									},
								},
								"label":       map[string]any{"type": "string"},
								"description": map[string]any{"type": "string"},
								"evidence_event_ids": map[string]any{
									"type":     "array",
									"maxItems": 12,
									"items":    map[string]any{"type": "string"},
								},
								"position": map[string]any{
									"type":                 "object",
									"additionalProperties": false,
									"required":             []string{"x", "y"},
									"properties": map[string]any{
										"x": map[string]any{"type": "number"},
										"y": map[string]any{"type": "number"},
									},
								},
							},
						},
					},
					"edges": map[string]any{
						"type":     "array",
						"minItems": 2,
						"maxItems": 16,
						"items": map[string]any{
							"type":                 "object",
							"additionalProperties": false,
							"required":             []string{"id", "source", "target", "label"},
							"properties": map[string]any{
								"id":     map[string]any{"type": "string"},
								"source": map[string]any{"type": "string"},
								"target": map[string]any{"type": "string"},
								"label":  map[string]any{"type": "string"},
							},
						},
					},
				},
			},
		},
	}
}
