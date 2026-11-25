package learningtrace

import (
	"context"
	"encoding/json"
	"fmt"

	"classgame-api/internal/modules/ai"
	"classgame-api/internal/modules/ai/providers/genai"
)

func Run(ctx context.Context, client genai.StructuredGenerator, input ai.BuildLearningTraceInput) (ai.LearningTraceBlueprint, genai.StructuredResult, error) {
	var blueprint ai.LearningTraceBlueprint
	payload, _ := json.MarshalIndent(input, "", "  ")
	result, err := client.GenerateStructured(ctx, genai.StructuredRequest{
		SystemInstruction: instruction,
		Prompt: fmt.Sprintf(`Build the student's evidence-based learning trace from this mission context and interaction log. The attempt may be completed or may have ended after inactivity:

%s

The result is shown only to the student's teacher and becomes persistent memory for future ClassGame interactions.`, payload),
		ResponseJSONSchema: schema(),
		Temperature:        0.2,
		Labels: map[string]string{
			"app":      "classgame",
			"workflow": "student_learning_trace",
		},
	}, &blueprint)
	return blueprint, result, err
}
