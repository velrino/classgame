package mission

import (
	"context"
	"encoding/json"
	"fmt"

	"classgame-api/internal/modules/ai"
	"classgame-api/internal/modules/ai/providers/genai"
)

func Run(ctx context.Context, client genai.StructuredGenerator, input ai.BuildMissionInput) (ai.MissionBlueprint, genai.StructuredResult, error) {
	var blueprint ai.MissionBlueprint
	result, err := client.GenerateStructured(ctx, genai.StructuredRequest{
		SystemInstruction:  instruction,
		Prompt:             buildMissionPrompt(input),
		ResponseJSONSchema: schema(),
		Temperature:        0.45,
		Labels: map[string]string{
			"app":      "classgame",
			"workflow": "teacher_mission",
		},
	}, &blueprint)
	return blueprint, result, err
}

func buildMissionPrompt(input ai.BuildMissionInput) string {
	payload, _ := json.MarshalIndent(input, "", "  ")

	return fmt.Sprintf(`Create a ClassGame mission blueprint from this teacher request:

%s

The output will be saved as a teacher-class mission and shown directly in a teacher dashboard.
Make it easy for a teacher to understand what students will practice and what Mia will observe.
For the student game, create exactly 3 multiple-choice preview questions total.
If attachments include a summary field, treat that summary as the readable attachment content.`, payload)
}
