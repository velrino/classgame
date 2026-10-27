package students

import (
	"classgame-api/internal/modules/classrooms"
	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/missions"
)

func studentSessionMissionFromDocument(document missions.MissionDocument) StudentSessionMission {
	return StudentSessionMission{
		ID:                document.ID,
		MissionCode:       document.MissionCode,
		PublishedRevision: document.PublishedRevision,
		MissionName:       document.MissionName,
		Title:             document.Title,
		Subject:           document.Subject,
		GradeLevel:        document.GradeLevel,
		LearningGoal:      document.LearningGoal,
		Stages:            document.Stages,
		PreviewQuestions: generation.PlayablePreviewQuestions(generation.BuildMissionInput{
			ClassID:       document.ClassID,
			ClassName:     document.ClassName,
			MissionName:   document.MissionName,
			Subject:       document.Subject,
			GradeLevel:    document.GradeLevel,
			LearningGoal:  document.LearningGoal,
			TeacherPrompt: document.TeacherPrompt,
		}, document.PreviewQuestions),
	}
}

func studentSessionClassFromDocument(document classrooms.TeacherClassDocument) StudentSessionClass {
	return StudentSessionClass{
		ID:         document.ClassID,
		ClassCode:  document.ClassCode,
		ClassName:  document.ClassName,
		Subject:    document.Subject,
		GradeLevel: document.GradeLevel,
	}
}
