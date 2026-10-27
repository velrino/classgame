package users

import (
	"classgame-api/internal/modules/orchestrator"
	"context"
	"time"
)

// SessionStore is the persistence port owned by this workflow.
type SessionStore interface {
	UpsertStudent(
		ctx context.Context,
		input UpsertStudentInput,
	) (UserDocument, error)
	UpdateStudentPresence(
		ctx context.Context,
		input UpdateStudentPresenceInput,
	) (UserDocument, error)
	GetCurrentStudentLearningAttempt(
		ctx context.Context,
		identityProviderUserID string,
		missionID string,
	) (StudentMissionAttemptDocument, UserDocument, bool, error)
	GetStudentClassMembership(ctx context.Context, studentID string) (string, string, error)
}

// LearningStore is the persistence port owned by this workflow.
type LearningStore interface {
	SaveStudentLearningSnapshot(
		ctx context.Context,
		input SaveStudentLearningSnapshotInput,
	) (StudentMissionAttemptDocument, UserDocument, bool, error)
	AppendStudentTutorTurn(
		ctx context.Context,
		input SaveStudentTutorTurnInput,
		tutor orchestrator.TutorTurnBlueprint,
	) (StudentTutorTurn, error)
	GetStudentLearningAttempt(
		ctx context.Context,
		studentID string,
		attemptID string,
	) (StudentMissionAttemptDocument, UserDocument, bool, error)
	MarkStudentAttemptAbandoned(
		ctx context.Context,
		studentID string,
		attemptID string,
		expectedLastActivity time.Time,
		timestamp time.Time,
	) (StudentMissionAttemptDocument, bool, error)
	SetStudentAttemptTraceState(
		ctx context.Context,
		attemptID string,
		state string,
		timestamp time.Time,
	) error
	FinalizeStudentLearningTrace(
		ctx context.Context,
		studentID string,
		attemptID string,
		trace orchestrator.LearningTraceBlueprint,
		memory StudentLearningMemory,
		timestamp time.Time,
	) (StudentMissionAttemptDocument, error)
}

// ClassLearningStore is the persistence port owned by this workflow.
type ClassLearningStore interface {
	ListLatestFinalizedAttemptsByClassID(
		ctx context.Context,
		classID string,
		limit int,
	) ([]StudentMissionAttemptDocument, error)
	SaveClassLearningSynthesis(
		ctx context.Context,
		document ClassLearningSynthesisDocument,
	) error
}
