package generation

import (
	"strings"
)

func buildFallbackBlueprint(input BuildMissionInput, reason string) MissionBlueprint {
	input = normalizeInput(input)

	blueprint := MissionBlueprint{
		Title:                    input.Subject + " Review Mission",
		Summary:                  input.TeacherPrompt,
		Subject:                  input.Subject,
		GradeLevel:               input.GradeLevel,
		LearningGoal:             input.LearningGoal,
		SuggestedDurationMinutes: 10,
		Stages:                   fallbackStages(input),
		PreviewQuestions:         fallbackPreviewQuestions(input),
		Suggestions:              fallbackSuggestions(input),
		PositiveSignals: []string{
			"Students attempt the next step before Mia explains.",
			"Students give a reason for wrong answers.",
		},
		ImprovementSignals: []string{
			"Mia should watch for repeated reasoning slips.",
			"Students who mark explanations as confusing need a visual follow-up.",
		},
		EducationGuardrail: EducationGuardrail{
			Allowed:         true,
			Reason:          "Fallback stayed within the requested classroom learning context.",
			UsedAttachments: false,
			AttachmentNotes: fallbackAttachmentNotes(input),
		},
		GeneratedByAI:   false,
		GeneratedByMock: true,
		Provider:        "local_fallback",
		Model:           "",
		FallbackReason:  reason,
	}
	blueprint.PreviewQuestions = normalizePreviewQuestions(input, blueprint.Stages, blueprint.PreviewQuestions)
	blueprint.KnowledgeChunks = fallbackKnowledgeChunks(input, blueprint.Stages, blueprint.PreviewQuestions)
	blueprint.MissionFlow = fallbackMissionFlow(input, blueprint)

	return blueprint
}

func buildGuardrailFallbackBlueprint(input BuildMissionInput, reason string) MissionBlueprint {
	input = normalizeInput(input)
	reason = normalizeText(reason)
	if reason == "" {
		reason = "The request could not be safely grounded in age-appropriate classroom learning content."
	}

	blueprint := MissionBlueprint{
		Title:                    "Educational Material Needed",
		Summary:                  "Mia could not create a safe mission from the provided material. Add a grade-level worksheet or a classroom learning prompt, then regenerate the preview.",
		Subject:                  safeEducationalSubject(input.Subject),
		GradeLevel:               safeEducationalGradeLevel(input.GradeLevel),
		LearningGoal:             "Confirm the mission source is grade-level educational material before students start.",
		SuggestedDurationMinutes: 6,
		Stages: []MissionStage{
			{
				ID:          "review-source",
				Title:       "Review the source",
				Goal:        "Check whether the uploaded material matches the class subject and grade level.",
				Checks:      []string{"Confirm subject", "Confirm grade level", "Confirm worksheet relevance"},
				UnlocksWhen: "The teacher provides usable classroom learning material.",
			},
			{
				ID:          "replace-material",
				Title:       "Replace unclear material",
				Goal:        "Use a worksheet, lesson notes, or a short teacher prompt tied to the learning goal.",
				Checks:      []string{"Use readable files", "Avoid unrelated documents"},
				UnlocksWhen: "The mission can be safely regenerated.",
			},
			{
				ID:          "regenerate-preview",
				Title:       "Regenerate preview",
				Goal:        "Create a student-safe mission after the source is corrected.",
				Checks:      []string{"Preview questions match the lesson", "Mia's feedback stays age-appropriate"},
				UnlocksWhen: "The teacher approves the corrected preview.",
			},
		},
		PreviewQuestions: []PreviewQuestion{
			{
				ID:                 "confirm-learning-topic",
				Prompt:             "What classroom topic should this mission help students practice?",
				AnswerType:         "short_text",
				Options:            []string{},
				ExpectedAnswer:     "A grade-level learning topic from the teacher's lesson.",
				MisconceptionTraps: []string{"Using unrelated or non-classroom material as the mission source."},
			},
			{
				ID:                 "confirm-source-match",
				Prompt:             "Does the uploaded file match the class subject and grade level?",
				AnswerType:         "multiple_choice",
				Options:            []string{"Yes, it matches", "No, replace it", "Not sure yet"},
				ExpectedAnswer:     "Yes, it matches",
				MisconceptionTraps: []string{"Assuming any PDF is safe or relevant because it uploaded successfully."},
			},
			{
				ID:                 "next-safe-step",
				Prompt:             "What should happen before students receive this mission?",
				AnswerType:         "next_step",
				Options:            []string{"Regenerate with educational material", "Publish unrelated material", "Ignore the learning goal"},
				ExpectedAnswer:     "Regenerate with educational material",
				MisconceptionTraps: []string{"Publishing before the source is checked."},
			},
		},
		Suggestions: []MissionSuggestion{
			{
				ID:          "use-grade-level-source",
				Title:       "Use a grade-level source",
				Reason:      "Mia needs relevant classroom material to generate useful practice.",
				Activity:    "Attach a worksheet or paste a concise learning prompt before regenerating.",
				Questions:   []string{"What did students practice today?", "Which mistakes should Mia watch for?"},
				FocusSkills: []string{"source review", "lesson alignment"},
			},
			{
				ID:          "keep-students-safe",
				Title:       "Keep the mission classroom-safe",
				Reason:      "The student experience should stay educational and age-appropriate.",
				Activity:    "Review the preview before assigning the mission code.",
				Questions:   []string{"Do all questions match the learning goal?", "Is any unrelated content present?"},
				FocusSkills: []string{"teacher review", "student safety"},
			},
		},
		PositiveSignals: []string{
			"The teacher verifies that files match the lesson before publishing.",
			"Mia only uses age-appropriate classroom learning material.",
		},
		ImprovementSignals: []string{
			"Uploaded material should be replaced when it is empty, unrelated, or not readable.",
		},
		EducationGuardrail: EducationGuardrail{
			Allowed:         false,
			Reason:          reason,
			UsedAttachments: false,
			AttachmentNotes: "Attachments were ignored because the request could not be safely grounded in classroom learning content.",
		},
		GeneratedByAI:   false,
		GeneratedByMock: true,
		Provider:        "local_guardrail",
		FallbackReason:  reason,
	}
	blueprint.PreviewQuestions = normalizePreviewQuestions(input, blueprint.Stages, blueprint.PreviewQuestions)
	blueprint.KnowledgeChunks = fallbackKnowledgeChunks(input, blueprint.Stages, blueprint.PreviewQuestions)
	blueprint.MissionFlow = guardrailMissionFlow()

	return blueprint
}

func fallbackAttachmentNotes(input BuildMissionInput) string {
	if len(input.Attachments) == 0 {
		return "No attachments were provided."
	}

	return "Fallback could not inspect attachments directly; filenames and teacher context were used safely."
}

func fallbackStages(input BuildMissionInput) []MissionStage {
	return []MissionStage{
		{
			ID:          "diagnostic",
			Title:       "Quick diagnostic",
			Goal:        "Find the first visible misconception.",
			Checks:      []string{"One recall question", "One application question"},
			UnlocksWhen: "Mia has enough evidence to choose review or challenge.",
		},
		{
			ID:          "guided_practice",
			Title:       "Guided practice",
			Goal:        input.LearningGoal,
			Checks:      []string{"Ask why after wrong answers", "Record one notebook note"},
			UnlocksWhen: "The student answers with stable reasoning twice.",
		},
		{
			ID:          "reflection",
			Title:       "Reflection",
			Goal:        "Capture what explanation style worked.",
			Checks:      []string{"Helpful", "Confusing", "Too easy", "Too hard"},
			UnlocksWhen: "Mia stores the student's preference for the next mission.",
		},
	}
}

func fallbackSuggestions(input BuildMissionInput) []MissionSuggestion {
	text := missionInputSearchText(input)

	if strings.Contains(text, "fraction") || strings.Contains(text, "equation") || strings.Contains(text, "algebra") {
		return []MissionSuggestion{
			{
				ID:       "fraction-bridge",
				Title:    "Bridge fractions before equations",
				Reason:   "Fraction slips often appear as algebra mistakes once denominators are inside equations.",
				Activity: "Run two fraction checks before the first two-step equation.",
				Questions: []string{
					"Why does multiplying by 3 clear x/3?",
					"What changes on both sides when we remove a denominator?",
				},
				FocusSkills: []string{"fractions", "inverse operations"},
			},
			{
				ID:       "why-before-answer",
				Title:    "Ask why before explaining",
				Reason:   "The same wrong answer can come from different reasoning.",
				Activity: "When a student misses, Mia asks them to pick the reason first.",
				Questions: []string{
					"Did you divide to clear a denominator?",
					"Did you move a number without doing the same thing to both sides?",
				},
				FocusSkills: []string{"metacognition", "equations"},
			},
		}
	}

	return []MissionSuggestion{
		{
			ID:       "core-recall",
			Title:    "Start with recall, then explain",
			Reason:   "Students need to retrieve the idea before Mia can see how they think.",
			Activity: "Ask one recall question, one application question, then one reflection.",
			Questions: []string{
				"What is the main idea from today's lesson?",
				"Where would this idea be used?",
			},
			FocusSkills: []string{"recall", "application"},
		},
		{
			ID:       "feedback-choice",
			Title:    "Capture how the explanation landed",
			Reason:   "The adaptation signal should be easy enough for students to give every time.",
			Activity: "After Mia explains, students choose helpful, confusing, too easy or too hard.",
			Questions: []string{
				"Was this explanation helpful?",
				"Should Mia draw it, slow down or make it harder?",
			},
			FocusSkills: []string{"feedback", "personalization"},
		},
	}
}
