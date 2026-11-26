package materials

import (
	"context"
	"github.com/tmc/langchaingo/prompts"

	"classgame-api/internal/modules/ai/agents/structured"
	"classgame-api/internal/modules/ai/providers/genai"
)

func Run(ctx context.Context, client genai.StructuredGenerator, fileName, mimeType string, data []byte) (Summary, genai.StructuredResult, error) {
	var summary Summary
	result, err := structured.Run(ctx, client, materialPrompt, map[string]any{"name": fileName, "mime": mimeType}, genai.StructuredRequest{
		SystemInstruction: instruction,
		FileParts: []genai.FilePart{
			{Data: data, MIMEType: mimeType},
		},
		ResponseJSONSchema: schema(),
		Temperature:        0.2,
		MaxOutputTokens:    900,
		Labels: map[string]string{
			"app":      "classgame",
			"workflow": "teacher_mission_attachment_summary",
		},
	}, &summary)
	return summary, result, err
}

type Summary struct {
	Summary           string   `json:"summary"`
	KeyConcepts       []string `json:"key_concepts"`
	UsableForMission  bool     `json:"usable_for_mission"`
	SuggestedQuestion []string `json:"suggested_questions"`
	Cautions          []string `json:"cautions"`
}

var materialPrompt = prompts.NewPromptTemplate(`Summarize this attachment for a short ClassGame mission.

File name: {{.name}}
MIME type: {{.mime}}

Return concise JSON. Focus on facts, rules, examples, vocabulary, and likely student misconceptions that are useful for generating 3 multiple-choice questions.`, []string{"name", "mime"})
