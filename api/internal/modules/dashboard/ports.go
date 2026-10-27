package dashboard

import (
	"classgame-api/internal/modules/users"
	"context"
	"time"
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
	) ([]users.StudentMissionAttemptDocument, error)
	GetClassLearningSynthesis(
		ctx context.Context,
		classID string,
	) (users.ClassLearningSynthesisDocument, bool, error)
}
