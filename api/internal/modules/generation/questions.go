package generation

import (
	"fmt"
	"strings"
)

const maxStudentPreviewQuestions = 3

func fallbackPreviewQuestions(input BuildMissionInput) []PreviewQuestion {
	text := missionInputSearchText(input)
	if isGradeFiveMathMission(text) {
		return []PreviewQuestion{
			{
				ID:             "place-value-build",
				Prompt:         "Which number is built from 50,000 + 3,000 + 200 + 40 + 6?",
				AnswerType:     "multiple_choice",
				Options:        []string{"53,246", "50,324", "5,346", "53,426"},
				ExpectedAnswer: "53,246",
				ExplanationSteps: []string{
					"Each part fills a place-value position.",
					"50,000 + 3,000 + 200 + 40 + 6 equals 53,246.",
				},
				MisconceptionTraps: []string{
					"Mixes up the ten-thousands and thousands places.",
					"Reorders digits without matching each part to its place value.",
				},
			},
			{
				ID:             "missing-power-of-ten-factor",
				Prompt:         "What missing factor makes 47 × ? = 47,000?",
				AnswerType:     "multiple_choice",
				Options:        []string{"1,000", "100", "10", "47"},
				ExpectedAnswer: "1,000",
				ExplanationSteps: []string{
					"47 becomes 47,000 by shifting three place-value positions.",
					"Multiplying by 1,000 adds three zeros.",
				},
				MisconceptionTraps: []string{
					"Counts only one or two place-value shifts.",
					"Uses the starting number as the factor.",
				},
			},
			{
				ID:             "acute-angle-classification",
				Prompt:         "An angle that measures less than 90° is called what?",
				AnswerType:     "multiple_choice",
				Options:        []string{"Acute angle", "Right angle", "Obtuse angle", "Straight angle"},
				ExpectedAnswer: "Acute angle",
				ExplanationSteps: []string{
					"A right angle is exactly 90°.",
					"Any angle smaller than 90° is acute.",
				},
				MisconceptionTraps: []string{
					"Confuses smaller-than-90° angles with right angles.",
					"Uses obtuse for angles that are too small.",
				},
			},
		}
	}
	if strings.Contains(text, "fraction") || strings.Contains(text, "equation") || strings.Contains(text, "algebra") {
		return []PreviewQuestion{
			{
				ID:             "fraction-denominator-check",
				Prompt:         "Which move clears the denominator in x/3 = 4?",
				AnswerType:     "multiple_choice",
				Options:        []string{"Multiply both sides by 3", "Divide both sides by 3", "Add 3 to both sides"},
				ExpectedAnswer: "Multiply both sides by 3",
				MisconceptionTraps: []string{
					"Divides again because the denominator is visible.",
					"Uses addition instead of inverse multiplication.",
				},
			},
			{
				ID:             "two-step-first-move",
				Prompt:         "For 2x/3 + 1 = 5, what should happen first?",
				AnswerType:     "multiple_choice",
				Options:        []string{"Subtract 1 from both sides", "Multiply both sides by 3", "Divide both sides by 2"},
				ExpectedAnswer: "Subtract 1 from both sides",
				MisconceptionTraps: []string{
					"Clears the denominator before isolating the term.",
					"Divides by the coefficient too early.",
				},
			},
			{
				ID:             "explain-the-why",
				Prompt:         "Why do we do the same operation on both sides of an equation?",
				AnswerType:     "multiple_choice",
				Options:        []string{"To keep both sides equal while changing the equation.", "To make one side bigger", "To change only the answer"},
				ExpectedAnswer: "To keep both sides equal while changing the equation.",
				MisconceptionTraps: []string{
					"Treats the equal sign as a signal to calculate only one side.",
					"Moves terms without preserving equality.",
				},
			},
		}
	}

	return []PreviewQuestion{
		{
			ID:         "mission-first-step",
			Prompt:     "What should you do first in this mission?",
			AnswerType: "multiple_choice",
			Options: []string{
				"Use the mission goal and solve one task carefully",
				"Guess from a keyword",
				"Skip the directions",
			},
			ExpectedAnswer: "Use the mission goal and solve one task carefully",
			MisconceptionTraps: []string{
				"Starts from a keyword without checking the actual task.",
				"Moves ahead without using the teacher's instructions.",
			},
		},
		{
			ID:             "apply-the-idea",
			Prompt:         "Which example best uses this idea?",
			AnswerType:     "multiple_choice",
			Options:        []string{"A direct application", "A related but incorrect pattern", "A memorized definition only"},
			ExpectedAnswer: "A direct application",
			MisconceptionTraps: []string{
				"Chooses a familiar-looking answer without checking the reasoning.",
			},
		},
		{
			ID:             "explain-after-answer",
			Prompt:         "What should you do after choosing an answer?",
			AnswerType:     "multiple_choice",
			Options:        []string{"Explain the reasoning that supports it", "Stop without checking", "Change it randomly", "Ignore feedback"},
			ExpectedAnswer: "Explain the reasoning that supports it",
			MisconceptionTraps: []string{
				"Treats the answer as finished without reasoning.",
				"Changes strategy without using feedback.",
			},
		},
	}
}

func missionInputSearchText(input BuildMissionInput) string {
	return strings.ToLower(strings.Join([]string{
		input.Subject,
		input.GradeLevel,
		input.LearningGoal,
		input.TeacherPrompt,
	}, " "))
}

func isGradeFiveMathMission(text string) bool {
	hasGradeFive := strings.Contains(text, "grade 5") ||
		strings.Contains(text, "5th grade") ||
		strings.Contains(text, "fifth grade")
	hasMath := strings.Contains(text, "math") ||
		strings.Contains(text, "mathematics")
	hasGradeFiveMathTopic := strings.Contains(text, "5-digit") ||
		strings.Contains(text, "place value") ||
		strings.Contains(text, "angle") ||
		strings.Contains(text, "multiplying by 10") ||
		strings.Contains(text, "100 or 1,000")

	return hasGradeFiveMathTopic || (hasGradeFive && hasMath)
}

func normalizePreviewQuestions(
	input BuildMissionInput,
	stages []MissionStage,
	questions []PreviewQuestion,
) []PreviewQuestion {
	result := make([]PreviewQuestion, 0, maxStudentPreviewQuestions)
	seen := make(map[string]struct{}, len(questions))
	appendQuestion := func(index int, question PreviewQuestion) bool {
		if len(result) >= maxStudentPreviewQuestions {
			return false
		}
		id := normalizeFlowID(question.ID)
		if id == "" {
			id = fmt.Sprintf("question-%d", index+1)
		}
		if _, exists := seen[id]; exists {
			return false
		}
		prompt := truncateText(normalizeText(question.Prompt), 280)
		expected := truncateText(normalizeText(question.ExpectedAnswer), 220)
		if prompt == "" || expected == "" {
			return false
		}

		stageID := normalizeText(question.StageID)
		if stageID == "" && len(stages) > 0 {
			stageID = stages[len(result)%len(stages)].ID
		}
		options := NormalizeMultipleChoiceOptions(
			expected,
			normalizeLimitedStrings(question.Options, 4, 160),
			normalizeLimitedStrings(question.MisconceptionTraps, 4, 180),
		)
		explanation := normalizeLimitedStrings(question.ExplanationSteps, 5, 220)
		if len(explanation) == 0 {
			explanation = []string{
				truncateText("Connect the question to the mission goal: "+input.LearningGoal, 220),
				truncateText("The expected answer is: "+expected, 220),
			}
		}

		seen[id] = struct{}{}
		result = append(result, PreviewQuestion{
			ID:                 id,
			StageID:            stageID,
			Prompt:             prompt,
			AnswerType:         "multiple_choice",
			Options:            options,
			ExpectedAnswer:     expected,
			ExplanationSteps:   explanation,
			MisconceptionTraps: normalizeLimitedStrings(question.MisconceptionTraps, 6, 180),
		})

		return true
	}

	for index, question := range questions {
		appendQuestion(index, question)
		if len(result) >= maxStudentPreviewQuestions {
			break
		}
	}

	if len(result) < maxStudentPreviewQuestions {
		for index, question := range fallbackPreviewQuestions(input) {
			if len(result) >= maxStudentPreviewQuestions {
				break
			}
			id := normalizeFlowID(question.ID)
			if _, exists := seen[id]; exists && id != "" {
				question.ID = fmt.Sprintf("%s-extra-%d", id, len(result)+1)
			}
			appendQuestion(index, question)
		}
	}

	return result
}
