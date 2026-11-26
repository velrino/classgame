package learningtrace

import (
	"context"

	"classgame-api/internal/modules/ai"
	"classgame-api/internal/modules/ai/agents/structured"
	"classgame-api/internal/modules/ai/providers/genai"
)

func Run(ctx context.Context, client genai.StructuredGenerator, input ai.BuildLearningTraceInput) (ai.LearningTraceBlueprint, genai.StructuredResult, error) {
	var blueprint ai.LearningTraceBlueprint
	result, err := structured.RunJSON(ctx, client, taskPrompt, input, genai.StructuredRequest{
		SystemInstruction:  instruction,
		ResponseJSONSchema: schema(),
		Temperature:        0.2,
		Labels: map[string]string{
			"app":      "classgame",
			"workflow": "student_learning_trace",
		},
	}, &blueprint)
	return blueprint, result, err
}

const taskPrompt = `Build the student's evidence-based learning trace from this mission context and interaction log. The attempt may be completed or may have ended after inactivity:

{{.input}}

The result is shown only to the student's teacher and becomes persistent memory for future ClassGame interactions.`
