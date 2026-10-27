package generation

import (
	"fmt"
	"strings"
)

func StudentPreviewQuestionLimit() int {
	return maxStudentPreviewQuestions
}

func NormalizePreviewQuestionCollections(questions []PreviewQuestion) []PreviewQuestion {
	limit := len(questions)
	if limit > maxStudentPreviewQuestions {
		limit = maxStudentPreviewQuestions
	}
	normalized := make([]PreviewQuestion, 0, limit)
	for _, question := range questions {
		question.AnswerType = "multiple_choice"
		question.Options = nonNilStrings(question.Options)
		question.ExplanationSteps = nonNilStrings(question.ExplanationSteps)
		question.MisconceptionTraps = nonNilStrings(question.MisconceptionTraps)
		question.Options = NormalizeMultipleChoiceOptions(
			question.ExpectedAnswer,
			question.Options,
			question.MisconceptionTraps,
		)
		normalized = append(normalized, question)
		if len(normalized) >= maxStudentPreviewQuestions {
			break
		}
	}

	return normalized
}

func PlayablePreviewQuestions(
	input BuildMissionInput,
	questions []PreviewQuestion,
) []PreviewQuestion {
	questions = NormalizePreviewQuestionCollections(questions)
	if hasLegacyGenericFallbackQuestion(questions) {
		return NormalizePreviewQuestionCollections(fallbackPreviewQuestions(input))
	}
	if len(questions) < maxStudentPreviewQuestions {
		seen := make(map[string]struct{}, len(questions))
		for _, question := range questions {
			seen[strings.ToLower(strings.TrimSpace(question.ID))] = struct{}{}
		}
		for index, question := range NormalizePreviewQuestionCollections(fallbackPreviewQuestions(input)) {
			if len(questions) >= maxStudentPreviewQuestions {
				break
			}
			id := strings.ToLower(strings.TrimSpace(question.ID))
			if id == "" {
				id = fmt.Sprintf("fallback-question-%d", index+1)
				question.ID = id
			}
			if _, exists := seen[id]; exists {
				question.ID = fmt.Sprintf("%s-extra-%d", id, len(questions)+1)
				id = strings.ToLower(question.ID)
			}
			seen[id] = struct{}{}
			questions = append(questions, question)
		}
	}

	return questions
}

func NormalizeMultipleChoiceOptions(expected string, options []string, traps []string) []string {
	result := make([]string, 0, 4)
	seen := map[string]struct{}{}
	add := func(value string) {
		if len(result) >= 4 {
			return
		}
		value = strings.TrimSpace(value)
		key := strings.ToLower(value)
		if value == "" {
			return
		}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}

	expectedKey := strings.ToLower(strings.TrimSpace(expected))
	for _, option := range options {
		add(option)
	}
	if expectedKey != "" {
		if _, exists := seen[expectedKey]; !exists && len(result) >= 4 {
			delete(seen, strings.ToLower(result[len(result)-1]))
			result[len(result)-1] = strings.TrimSpace(expected)
			seen[expectedKey] = struct{}{}
		}
		if _, exists := seen[expectedKey]; !exists {
			add(expected)
		}
	}
	for _, trap := range traps {
		add(trap)
	}
	add("I need another clue")
	add("Not enough information")

	return result
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}

	return values
}

func hasLegacyGenericFallbackQuestion(questions []PreviewQuestion) bool {
	for _, question := range questions {
		id := strings.ToLower(strings.TrimSpace(question.ID))
		prompt := strings.ToLower(strings.TrimSpace(question.Prompt))
		expected := strings.ToLower(strings.TrimSpace(question.ExpectedAnswer))

		if id == "main-idea-recall" ||
			strings.Contains(prompt, "main idea from today's lesson") ||
			strings.Contains(expected, "student-friendly explanation of the lesson") ||
			expected == "any honest student preference." {
			return true
		}
	}

	return false
}
