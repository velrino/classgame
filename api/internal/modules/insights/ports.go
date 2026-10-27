package insights

import (
	"context"
	"time"

	"classgame-api/internal/modules/students"
)

// InsightStore is the persistence port owned by this workflow.
type InsightStore interface {
	UpsertTeacher(
		ctx context.Context,
		identity TeacherIdentity,
		timestamp time.Time,
	) (TeacherSummary, error)
	ListStudentsByClassID(
		ctx context.Context,
		classID string,
	) ([]studentUserDocument, error)
	ListLearningAttemptsByClassID(
		ctx context.Context,
		classID string,
	) ([]students.StudentMissionAttemptDocument, error)
	GetClassLearningSynthesis(
		ctx context.Context,
		classID string,
	) (students.ClassLearningSynthesisDocument, bool, error)
}
