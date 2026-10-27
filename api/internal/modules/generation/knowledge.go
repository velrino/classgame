package generation

import (
	"fmt"
	"strings"
)

func normalizeKnowledgeChunks(input BuildMissionInput, chunks []KnowledgeChunk) []KnowledgeChunk {
	result := make([]KnowledgeChunk, 0, min(len(chunks), 12))
	seen := make(map[string]struct{}, len(chunks))
	for index, chunk := range chunks {
		if len(result) >= 12 {
			break
		}
		content := truncateText(normalizeText(chunk.Content), 700)
		if content == "" || unsafeKnowledgeChunk(content) {
			continue
		}
		id := normalizeFlowID(chunk.ID)
		if id == "" {
			id = fmt.Sprintf("knowledge-%d", index+1)
		}
		if _, exists := seen[id]; exists {
			continue
		}
		source := truncateText(normalizeText(chunk.Source), 140)
		if source == "" {
			source = "Teacher mission context"
		}
		title := truncateText(normalizeText(chunk.Title), 120)
		if title == "" {
			title = input.Subject + " reference"
		}

		seen[id] = struct{}{}
		result = append(result, KnowledgeChunk{
			ID:       id,
			Source:   source,
			Title:    title,
			Content:  content,
			Keywords: normalizeLimitedStrings(chunk.Keywords, 8, 48),
		})
	}

	return result
}

func unsafeKnowledgeChunk(content string) bool {
	content = strings.ToLower(content)
	for _, phrase := range []string{
		"bypass policy",
		"ignore all instructions",
		"ignore previous instructions",
		"ignore system instructions",
		"reveal credentials",
		"reveal secrets",
		"system prompt",
		"teacher-only memory",
	} {
		if strings.Contains(content, phrase) {
			return true
		}
	}

	return false
}

func fallbackKnowledgeChunks(
	input BuildMissionInput,
	stages []MissionStage,
	questions []PreviewQuestion,
) []KnowledgeChunk {
	source := "Teacher mission context"
	if len(input.Attachments) > 0 {
		names := make([]string, 0, len(input.Attachments))
		for _, attachment := range input.Attachments {
			if attachment.FileName != "" {
				names = append(names, attachment.FileName)
			}
		}
		if len(names) > 0 {
			source = strings.Join(names, ", ")
		}
	}

	chunks := make([]KnowledgeChunk, 0, min(12, len(stages)+len(questions)))
	for index, stage := range stages {
		content := normalizeText(strings.Join(append([]string{stage.Goal}, stage.Checks...), " "))
		if content == "" {
			continue
		}
		chunks = append(chunks, KnowledgeChunk{
			ID:       fmt.Sprintf("stage-context-%d", index+1),
			Source:   source,
			Title:    stage.Title,
			Content:  content,
			Keywords: normalizeLimitedStrings(strings.Fields(stage.Title+" "+input.Subject), 8, 48),
		})
	}
	for index, question := range questions {
		if len(chunks) >= 12 {
			break
		}
		chunks = append(chunks, KnowledgeChunk{
			ID:      fmt.Sprintf("question-context-%d", index+1),
			Source:  source,
			Title:   question.Prompt,
			Content: strings.Join(question.ExplanationSteps, " "),
			Keywords: normalizeLimitedStrings(
				strings.Fields(question.Prompt+" "+question.ExpectedAnswer),
				8,
				48,
			),
		})
	}

	return normalizeKnowledgeChunks(input, chunks)
}
