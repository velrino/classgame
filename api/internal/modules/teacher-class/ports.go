package teacherclass

import (
	"classgame-api/internal/pagination"
	"context"
	"time"
)

// ClassroomStore is the persistence port owned by this workflow.
type ClassroomStore interface {
	Get(
		ctx context.Context,
		classID string,
	) (TeacherClassDocument, bool, error)
	ListByTeacherID(
		ctx context.Context,
		teacherID string,
	) ([]TeacherClassDocument, error)
	ListPageByTeacherID(ctx context.Context, teacherID string, page pagination.Params) ([]TeacherClassDocument, string, error)
	Upsert(
		ctx context.Context,
		input UpsertTeacherClassInput,
	) (TeacherClassDocument, error)
	Create(
		ctx context.Context,
		input UpsertTeacherClassInput,
	) (TeacherClassDocument, error)
	SetActiveMission(
		ctx context.Context,
		classID string,
		teacherID string,
		missionID string,
		timestamp time.Time,
	) error
}
