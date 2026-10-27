package students

import (
	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/missions"
)

// playableMissionQuestions is the pinned question policy for display, tutoring,
// and accepted-answer assessment, including short and legacy publications.
func playableMissionQuestions(document missions.MissionDocument) []generation.PreviewQuestion {
	return generation.PlayablePreviewQuestions(generation.BuildMissionInput{
		ClassID:       document.ClassID,
		ClassName:     document.ClassName,
		MissionName:   document.MissionName,
		Subject:       document.Subject,
		GradeLevel:    document.GradeLevel,
		LearningGoal:  document.LearningGoal,
		TeacherPrompt: document.TeacherPrompt,
	}, document.PreviewQuestions)
}
