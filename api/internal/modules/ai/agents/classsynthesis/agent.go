package classsynthesis

import (
	"context"

	"classgame-api/internal/modules/ai"
	"classgame-api/internal/modules/ai/agents/structured"
	"classgame-api/internal/modules/ai/providers/genai"
)

func Run(ctx context.Context, client genai.StructuredGenerator, input ai.BuildClassLearningSynthesisInput) (ai.ClassLearningSynthesis, genai.StructuredResult, error) {
	var synthesis ai.ClassLearningSynthesis
	result, err := structured.RunJSON(ctx, client, taskPrompt, input, genai.StructuredRequest{
		SystemInstruction:  instruction,
		ResponseJSONSchema: schema(),
		Temperature:        0.15,
		Labels: map[string]string{
			"app":      "classgame",
			"workflow": "class_learning_synthesis",
		},
	}, &synthesis)
	return synthesis, result, err
}

const taskPrompt = `Create the teacher-only class learning synthesis from this finalized evidence:

{{.input}}`
