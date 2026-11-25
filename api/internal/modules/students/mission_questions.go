package students

import (
	"classgame-api/internal/modules/ai"
	"classgame-api/internal/modules/ai/orchestrator"
	"classgame-api/internal/modules/missions"
)

// playableMissionQuestions is the pinned question policy for display, tutoring,
// and accepted-answer assessment, including short and legacy publications.
func playableMissionQuestions(document missions.MissionDocument) []ai.PreviewQuestion {
	return orchestrator.PlayablePreviewQuestions(ai.BuildMissionInput{
		ClassID:       document.ClassID,
		ClassName:     document.ClassName,
		MissionName:   document.MissionName,
		Subject:       document.Subject,
		GradeLevel:    document.GradeLevel,
		LearningGoal:  document.LearningGoal,
		TeacherPrompt: document.TeacherPrompt,
	}, document.PreviewQuestions)
}
