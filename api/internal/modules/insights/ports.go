package insights

import (
	"context"
	"time"

	"classgame-api/internal/pagination"

	"classgame-api/internal/modules/students"
)

// InsightStore is the persistence port owned by this workflow.
type InsightStore interface {
	UpsertTeacher(
		ctx context.Context,
		identity TeacherIdentity,
		timestamp time.Time,
	) (TeacherSummary, error)
	ListStudentPage(context.Context, string, pagination.Params) (StudentPage, error)
	ListAttemptPage(context.Context, string, string, pagination.Params, bool) (AttemptPage, error)
	ListLatestAttempts(context.Context, string, []string) ([]students.StudentMissionAttemptDocument, error)
	GetClassLearningSynthesis(
		ctx context.Context,
		classID string,
	) (students.ClassLearningSynthesisDocument, bool, error)
}
