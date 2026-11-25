package mission

import "classgame-api/internal/modules/ai"

func schema() map[string]any {
	stringArray := map[string]any{
		"type":  "array",
		"items": map[string]any{"type": "string"},
	}
	choiceOptions := map[string]any{
		"type":     "array",
		"minItems": 3,
		"maxItems": 4,
		"items":    map[string]any{"type": "string"},
	}

	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required": []string{
			"title",
			"summary",
			"subject",
			"grade_level",
			"learning_goal",
			"suggested_duration_minutes",
			"stages",
			"preview_questions",
			"knowledge_chunks",
			"suggestions",
			"positive_signals",
			"improvement_signals",
			"education_guardrail",
			"mission_flow",
		},
		"properties": map[string]any{
			"title": map[string]any{
				"type":        "string",
				"description": "Short title shown to the teacher.",
			},
			"summary": map[string]any{
				"type":        "string",
				"description": "One or two sentences explaining the mission.",
			},
			"subject": map[string]any{"type": "string"},
			"grade_level": map[string]any{
				"type":        "string",
				"description": "Student grade level in teacher-friendly wording.",
			},
			"learning_goal": map[string]any{
				"type":        "string",
				"description": "What students should understand after the mission.",
			},
			"suggested_duration_minutes": map[string]any{
				"type":    "integer",
				"minimum": 6,
				"maximum": 20,
			},
			"stages": map[string]any{
				"type":     "array",
				"minItems": 3,
				"maxItems": 4,
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"id", "title", "goal", "checks", "unlocks_when"},
					"properties": map[string]any{
						"id":           map[string]any{"type": "string"},
						"title":        map[string]any{"type": "string"},
						"goal":         map[string]any{"type": "string"},
						"checks":       stringArray,
						"unlocks_when": map[string]any{"type": "string"},
					},
				},
			},
			"preview_questions": map[string]any{
				"type":     "array",
				"minItems": ai.PreviewQuestionCount,
				"maxItems": ai.PreviewQuestionCount,
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required": []string{
						"id",
						"stage_id",
						"prompt",
						"answer_type",
						"options",
						"expected_answer",
						"explanation_steps",
						"misconception_traps",
					},
					"properties": map[string]any{
						"id":                  map[string]any{"type": "string"},
						"stage_id":            map[string]any{"type": "string"},
						"prompt":              map[string]any{"type": "string"},
						"answer_type":         map[string]any{"type": "string", "enum": []string{"multiple_choice"}},
						"options":             choiceOptions,
						"expected_answer":     map[string]any{"type": "string"},
						"explanation_steps":   stringArray,
						"misconception_traps": stringArray,
					},
				},
			},
			"knowledge_chunks": map[string]any{
				"type":     "array",
				"minItems": 3,
				"maxItems": 12,
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"id", "source", "title", "content", "keywords"},
					"properties": map[string]any{
						"id":       map[string]any{"type": "string"},
						"source":   map[string]any{"type": "string"},
						"title":    map[string]any{"type": "string"},
						"content":  map[string]any{"type": "string"},
						"keywords": stringArray,
					},
				},
			},
			"suggestions": map[string]any{
				"type":     "array",
				"minItems": 2,
				"maxItems": 4,
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"id", "title", "reason", "activity", "questions", "focus_skills"},
					"properties": map[string]any{
						"id":           map[string]any{"type": "string"},
						"title":        map[string]any{"type": "string"},
						"reason":       map[string]any{"type": "string"},
						"activity":     map[string]any{"type": "string"},
						"questions":    stringArray,
						"focus_skills": stringArray,
					},
				},
			},
			"positive_signals":    stringArray,
			"improvement_signals": stringArray,
			"education_guardrail": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"allowed", "reason", "used_attachments", "attachment_notes"},
				"properties": map[string]any{
					"allowed": map[string]any{
						"type":        "boolean",
						"description": "False only when the teacher request cannot safely be turned into an age-appropriate classroom learning mission.",
					},
					"reason": map[string]any{
						"type":        "string",
						"description": "Short explanation of the education/safety decision.",
					},
					"used_attachments": map[string]any{
						"type":        "boolean",
						"description": "True only when the generated mission directly uses readable, relevant learning material from the attached files.",
					},
					"attachment_notes": map[string]any{
						"type":        "string",
						"description": "Short note about how attachments were used or why they were ignored.",
					},
				},
			},
			"mission_flow": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"nodes", "edges"},
				"properties": map[string]any{
					"nodes": map[string]any{
						"type":     "array",
						"minItems": 4,
						"maxItems": 8,
						"items": map[string]any{
							"type":                 "object",
							"additionalProperties": false,
							"required":             []string{"id", "kind", "label", "description", "position"},
							"properties": map[string]any{
								"id": map[string]any{
									"type":        "string",
									"description": "Stable lowercase id such as start, diagnose, guided-practice, branch, publish.",
								},
								"kind": map[string]any{
									"type": "string",
									"enum": []string{
										"start",
										"setup",
										"activity",
										"question",
										"decision",
										"feedback",
										"complete",
										"guardrail",
									},
								},
								"label": map[string]any{
									"type":        "string",
									"description": "Short node title for a React Flow card.",
								},
								"description": map[string]any{
									"type":        "string",
									"description": "One short sentence explaining this mission step.",
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
						"minItems": 3,
						"maxItems": 10,
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
