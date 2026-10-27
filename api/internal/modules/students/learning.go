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

	attempt, _, _, err := service.learning.SaveStudentLearningSnapshot(ctx, SaveStudentLearningSnapshotInput{
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

	if FinalizationEligible(attempt) && attempt.Status == "completed" {
		// Retrying pending dispatch is deliberate: an earlier process may have died
		// after the completion transaction and before enqueue. The job ID is stable.
		if service.learningTaskEnqueuer != nil {
			if err := service.learningTaskEnqueuer.EnqueueFinalization(ctx, attempt); err == nil {
				return attempt, nil
			} else {
				slog.Warn("student learning finalization enqueue failed; using inline fallback", "error", err)
			}
		}
		finalized, err := service.finalizeStudentAttempt(ctx, FinalizationJob(attempt))
		if errors.Is(err, ErrFinalizationBusy) {
			return attempt, nil
		}
		return finalized, err
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

	if err == nil && exists && attempt.Status != "completed" {
		document, readErr := service.missionForAttempt(ctx, attempt)
		if readErr != nil {
			return StudentMissionAttemptDocument{}, false, readErr
		}
		evidence := EffectiveTutorEvidence(attempt)
		if attempt.EvidenceSchemaVersion == 0 {
			evidence.AcceptedAnswers = acceptedAnswerFacts(document, attempt.TutorTurns)
		}
		attempt.Snapshot, _ = AssessLearningSnapshotWithEvidence(document, attempt.Snapshot, attempt.TutorTurns, evidence.AcceptedAnswers, attempt.Status)
	}
	return attempt, exists, err
}
