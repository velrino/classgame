package tutor

import (
	"context"
	"encoding/json"
	"fmt"

	"classgame-api/internal/modules/ai"
	"classgame-api/internal/modules/ai/providers/genai"
)

func Run(ctx context.Context, client genai.StructuredGenerator, input ai.BuildTutorTurnInput) (ai.TutorTurnBlueprint, genai.StructuredResult, error) {
	payload, _ := json.MarshalIndent(input, "", "  ")
	var blueprint ai.TutorTurnBlueprint
	result, err := client.GenerateStructured(ctx, genai.StructuredRequest{
		SystemInstruction: instruction,
		Prompt: fmt.Sprintf(`Respond to this live student turn:

%s

The API will return only student-safe fields to the learner. teacher_observation is persisted for the teacher.`, payload),
		ResponseJSONSchema: schema(),
		Temperature:        0.25,
		Labels: map[string]string{
			"app":      "classgame",
			"workflow": "student_tutor_turn",
		},
	}, &blueprint)
	return blueprint, result, err
}
