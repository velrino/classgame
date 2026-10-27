package students

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"classgame-api/internal/auth"
	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/missions"
)

func (service *Service) SaveStudentLearningSnapshot(
	ctx context.Context,
	identity auth.Identity,
	missionID string,
	status string,
	snapshot StudentLearningSnapshot,
	requestScope AttemptScope,
) (StudentMissionAttemptDocument, error) {
	if identity.AuthMethod != auth.AuthMethodAnonymous {
		return StudentMissionAttemptDocument{}, ErrStudentRequiresAnonymousAuth
	}
	if service.learning == nil || service.sessions == nil {
		return StudentMissionAttemptDocument{}, errors.New("student learning repository is not configured")
	}

	current, user, exists, err := service.sessions.GetCurrentStudentLearningAttempt(ctx, identity.ProviderUserID, normalizeText(missionID))
	if err != nil {
		return StudentMissionAttemptDocument{}, err
	}
	if !exists {
		current = StudentMissionAttemptDocument{StudentID: user.ID, ClassID: user.Student.ClassID, MissionID: normalizeText(missionID), PublishedRevision: user.Activity.CurrentGame.PublishedRevision}
	}
	if current.ID == "" {
		current.ID = user.Activity.CurrentGame.AttemptID
	}
	if current.ID == "" {
		current.ID = learningAttemptID(user.ID, normalizeText(missionID), current.PublishedRevision)
	}
	if err := ValidateAttemptRequestScope(requestScope, current); err != nil {
		return StudentMissionAttemptDocument{}, err
	}

	document, err := service.missionForAttempt(ctx, current)
	if err != nil {
		return StudentMissionAttemptDocument{}, err
	}
	normalizedStatus := strings.ToLower(normalizeText(status))
	normalizedSnapshot := normalizeLearningSnapshot(snapshot)
	liveInput := toLearningTraceInput(missions.MissionDocument{
		ID:      normalizeText(missionID),
		Subject: "Classroom learning",
	}, normalizedSnapshot, normalizedStatus, StudentLearningMemory{})
	liveInput.Snapshot.Status = "in_progress"
	liveTrace := generation.BuildObservedLearningTrace(liveInput, "")
	liveTrace.Finalized = false

	attempt, _, completionStarted, err := service.learning.SaveStudentLearningSnapshot(ctx, SaveStudentLearningSnapshotInput{
		Identity:     identity,
		RequestScope: requestScope, ExpectedAttemptID: current.ID,
		Mission:           document,
		PublishedRevision: current.PublishedRevision,
		MissionID:         normalizeText(missionID),
		Status:            normalizedStatus,
		Snapshot:          normalizedSnapshot,
		Trace:             liveTrace,
		Timestamp:         time.Now(),
	})
	if err != nil {
		return StudentMissionAttemptDocument{}, err
	}
	if attempt.Trace.Finalized {
		return attempt, nil
	}

	if attempt.Status == "completed" {
		shouldFinalize := completionStarted || attempt.TraceState == "failed"
		if !shouldFinalize {
			return attempt, nil
		}
		if service.learningTaskEnqueuer != nil {
			if err := service.learningTaskEnqueuer.EnqueueFinalization(ctx, attempt); err == nil {
				return attempt, nil
			} else {
				slog.Warn("student learning finalization enqueue failed; using inline fallback", "error", err)
			}
		}

		return service.finalizeStudentAttempt(ctx, attempt.StudentID, attempt.ID)
	}

	if attempt.Status == "paused" && service.learningTaskEnqueuer != nil {
		if err := service.learningTaskEnqueuer.EnqueueWatchdog(ctx, attempt); err != nil {
			slog.Warn("student abandonment watchdog enqueue failed", "error", err, "attempt_id", attempt.ID)
		}
	}

	return attempt, nil
}

func (service *Service) GetCurrentStudentLearningAttempt(
	ctx context.Context,
	identity auth.Identity,
	missionID string,
) (StudentMissionAttemptDocument, bool, error) {
	if identity.AuthMethod != auth.AuthMethodAnonymous {
		return StudentMissionAttemptDocument{}, false, ErrStudentRequiresAnonymousAuth
	}
	if service.learning == nil || service.sessions == nil {
		return StudentMissionAttemptDocument{}, false, errors.New("student learning repository is not configured")
	}

	attempt, _, exists, err := service.sessions.GetCurrentStudentLearningAttempt(
		ctx,
		identity.ProviderUserID,
		normalizeText(missionID),
	)
	return attempt, exists, err
}
