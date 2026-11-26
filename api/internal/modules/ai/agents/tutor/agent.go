package tutor

import (
	"context"

	"classgame-api/internal/modules/ai"
	"classgame-api/internal/modules/ai/agents/structured"
	"classgame-api/internal/modules/ai/providers/genai"
)

func Run(ctx context.Context, client genai.StructuredGenerator, input ai.BuildTutorTurnInput) (ai.TutorTurnBlueprint, genai.StructuredResult, error) {
	var blueprint ai.TutorTurnBlueprint
	result, err := structured.RunJSON(ctx, client, taskPrompt, input, genai.StructuredRequest{
		SystemInstruction:  instruction,
		ResponseJSONSchema: schema(),
		Temperature:        0.25,
		Labels: map[string]string{
			"app":      "classgame",
			"workflow": "student_tutor_turn",
		},
	}, &blueprint)
	return blueprint, result, err
}

const taskPrompt = `Respond to this live student turn:

{{.input}}

The API will return only student-safe fields to the learner. teacher_observation is persisted for the teacher.`
