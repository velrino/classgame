package generation

import (
	"strings"

	"classgame-api/internal/contentpolicy"
)

func normalizeInput(input BuildMissionInput) BuildMissionInput {
	input.ClassID = normalizeCode(input.ClassID)
	input.ClassName = normalizeText(input.ClassName)
	input.MissionName = normalizeText(input.MissionName)
	input.Subject = normalizeText(input.Subject)
	input.GradeLevel = normalizeText(input.GradeLevel)
	input.LearningGoal = normalizeText(input.LearningGoal)
	input.TeacherPrompt = normalizeText(input.TeacherPrompt)
	input.Attachments = normalizeAttachments(input.Attachments)

	if input.ClassName == "" {
		input.ClassName = input.ClassID
	}
	if input.Subject == "" {
		input.Subject = "Mathematics"
	}
	if input.GradeLevel == "" {
		input.GradeLevel = "7th grade"
	}
	if input.LearningGoal == "" {
		input.LearningGoal = "Help students reinforce today's lesson."
	}
	if input.MissionName == "" {
		input.MissionName = input.Subject + " Review"
	}

	return input
}

func normalizeBlueprint(input BuildMissionInput, blueprint MissionBlueprint) MissionBlueprint {
	blueprint.EducationGuardrail = normalizeEducationGuardrail(input, blueprint.EducationGuardrail)
	if blocked, reason := missionSafetyBlockReason(input, blueprint.EducationGuardrail); blocked {
		return buildGuardrailFallbackBlueprint(input, reason)
	}

	blueprint.Title = normalizeText(blueprint.Title)
	blueprint.Summary = normalizeText(blueprint.Summary)
	blueprint.Subject = normalizeText(blueprint.Subject)
	blueprint.GradeLevel = normalizeText(blueprint.GradeLevel)
	blueprint.LearningGoal = normalizeText(blueprint.LearningGoal)

	if blueprint.Title == "" {
		blueprint.Title = input.Subject + " Review Mission"
	}
	if blueprint.Summary == "" {
		blueprint.Summary = input.TeacherPrompt
	}
	if blueprint.Subject == "" {
		blueprint.Subject = input.Subject
	}
	if blueprint.GradeLevel == "" {
		blueprint.GradeLevel = input.GradeLevel
	}
	if blueprint.LearningGoal == "" {
		blueprint.LearningGoal = input.LearningGoal
	}
	if blueprint.SuggestedDurationMinutes <= 0 {
		blueprint.SuggestedDurationMinutes = 10
	}
	if len(blueprint.Stages) == 0 {
		blueprint.Stages = fallbackStages(input)
	}
	if len(blueprint.PreviewQuestions) == 0 {
		blueprint.PreviewQuestions = fallbackPreviewQuestions(input)
	}
	blueprint.PreviewQuestions = normalizePreviewQuestions(input, blueprint.Stages, blueprint.PreviewQuestions)
	blueprint.KnowledgeChunks = normalizeKnowledgeChunks(input, blueprint.KnowledgeChunks)
	if len(blueprint.KnowledgeChunks) == 0 {
		blueprint.KnowledgeChunks = fallbackKnowledgeChunks(input, blueprint.Stages, blueprint.PreviewQuestions)
	}
	if len(blueprint.Suggestions) == 0 {
		blueprint.Suggestions = fallbackSuggestions(input)
	}
	if len(blueprint.PositiveSignals) == 0 {
		blueprint.PositiveSignals = []string{"Students explain their reasoning before seeing an answer."}
	}
	if len(blueprint.ImprovementSignals) == 0 {
		blueprint.ImprovementSignals = []string{"Students need Mia to slow down or switch explanation style."}
	}
	blueprint.MissionFlow = normalizeMissionFlow(input, blueprint)

	return blueprint
}

func normalizeEducationGuardrail(input BuildMissionInput, guardrail EducationGuardrail) EducationGuardrail {
	guardrail.Reason = truncateText(normalizeText(guardrail.Reason), 240)
	guardrail.AttachmentNotes = truncateText(normalizeText(guardrail.AttachmentNotes), 240)
	if guardrail.Allowed && guardrail.Reason == "" {
		guardrail.Reason = "Request stayed within the classroom learning context."
	}
	if !guardrail.Allowed && guardrail.Reason == "" {
		guardrail.Reason = "The request could not be safely grounded in age-appropriate classroom learning content."
	}
	if len(input.Attachments) == 0 {
		guardrail.UsedAttachments = false
		if guardrail.AttachmentNotes == "" {
			guardrail.AttachmentNotes = "No attachments were provided."
		}
	} else if guardrail.AttachmentNotes == "" {
		if guardrail.UsedAttachments {
			guardrail.AttachmentNotes = "Readable attachments were used as lesson context."
		} else {
			guardrail.AttachmentNotes = "Attachments were not used directly; the mission relied on the teacher prompt and class context."
		}
	}

	return guardrail
}

func missionSafetyBlockReason(input BuildMissionInput, guardrail EducationGuardrail) (bool, string) {
	requestText := strings.Join([]string{
		input.MissionName,
		input.Subject,
		input.GradeLevel,
		input.LearningGoal,
		input.TeacherPrompt,
		attachmentFileNames(input.Attachments),
	}, " ")
	decision := contentpolicy.ReviewEducationalText(requestText)
	if !decision.Allowed {
		return true, decision.Message
	}
	if !contentpolicy.HasEducationalContext(requestText) {
		return true, "The request does not provide enough classroom learning context."
	}
	if !guardrail.Allowed {
		if guardrail.Reason != "" {
			return true, guardrail.Reason
		}

		return true, "The request could not be safely grounded in age-appropriate classroom learning content."
	}

	return false, ""
}

func attachmentFileNames(attachments []AttachmentInput) string {
	names := make([]string, 0, len(attachments))
	for _, attachment := range attachments {
		names = append(names, attachment.FileName)
	}

	return strings.Join(names, " ")
}

func safeEducationalSubject(value string) string {
	value = normalizeText(value)
	if value == "" {
		return "Classroom learning"
	}

	return value
}

func safeEducationalGradeLevel(value string) string {
	value = normalizeText(value)
	if value == "" {
		return "Grade-level"
	}

	return value
}
