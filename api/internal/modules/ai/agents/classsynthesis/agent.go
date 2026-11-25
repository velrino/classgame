package classsynthesis

import (
	"context"
	"encoding/json"

	"classgame-api/internal/modules/ai"
	"classgame-api/internal/modules/ai/providers/genai"
)

func Run(ctx context.Context, client genai.StructuredGenerator, input ai.BuildClassLearningSynthesisInput) (ai.ClassLearningSynthesis, genai.StructuredResult, error) {
	payload, _ := json.MarshalIndent(input, "", "  ")
	var synthesis ai.ClassLearningSynthesis
	result, err := client.GenerateStructured(ctx, genai.StructuredRequest{
		SystemInstruction:  instruction,
		Prompt:             "Create the teacher-only class learning synthesis from this finalized evidence:\n\n" + string(payload),
		ResponseJSONSchema: schema(),
		Temperature:        0.15,
		Labels: map[string]string{
			"app":      "classgame",
			"workflow": "class_learning_synthesis",
		},
	}, &synthesis)
	return synthesis, result, err
}
