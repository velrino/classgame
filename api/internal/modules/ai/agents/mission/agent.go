package mission

import (
	"context"

	"classgame-api/internal/modules/ai"
	"classgame-api/internal/modules/ai/agents/structured"
	"classgame-api/internal/modules/ai/providers/genai"
)

func Run(ctx context.Context, client genai.StructuredGenerator, input ai.BuildMissionInput, files ...genai.FilePart) (ai.MissionBlueprint, genai.StructuredResult, error) {
	var blueprint ai.MissionBlueprint
	result, err := structured.RunJSON(ctx, client, taskPrompt, input, genai.StructuredRequest{
		SystemInstruction:  instruction,
		ResponseJSONSchema: schema(),
		FileParts:          files,
		Temperature:        0.45,
		Labels: map[string]string{
			"app":      "classgame",
			"workflow": "teacher_mission",
		},
	}, &blueprint)
	return blueprint, result, err
}

const taskPrompt = `Create a ClassGame mission blueprint from this teacher request:

{{.input}}

The output will be saved as a teacher-class mission and shown directly in a teacher dashboard.
Make it easy for a teacher to understand what students will practice and what Mia will observe.
For the student game, create exactly 3 multiple-choice preview questions total.
Read any attached PDF directly. Use its relevant examples and rules to build the mission.
If attachments include a summary field, treat that summary as the readable attachment content.`
