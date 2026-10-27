package students

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"

	"classgame-api/internal/modules/generation"
)

func (service *Service) ProcessLearningJob(ctx context.Context, job LearningJob) error {
	if err := job.Validate(); err != nil {
		return err
	}
	switch job.Kind {
	case LearningJobClassRefresh:
		return service.processClassLearningRefresh(ctx, job)
	case LearningJobFinalize:
		_, err := service.finalizeStudentAttempt(ctx, job)
		return err
	case LearningJobWatchdog:
		attempt, marked, err := service.learning.MarkStudentAttemptAbandoned(
			ctx,
			job.StudentID,
			job.AttemptID,
			job.ExpectedLastActivityAt,
			time.Now(),
		)
		if err != nil {
			return err
		}

		if !marked && (attempt.Status != "abandoned" || attempt.Trace.Finalized) {
			return nil
		}
		_, err = service.finalizeStudentAttempt(ctx, FinalizationJob(attempt))
		return err
	default:
		return errors.New("unsupported student learning job")
	}
}

func (service *Service) finalizeStudentAttempt(
	ctx context.Context,
	job LearningJob,
) (StudentMissionAttemptDocument, error) {
	if service.learning == nil || service.sessions == nil {
		return StudentMissionAttemptDocument{}, errors.New("student learning repository is not configured")
	}

	ownerBytes := make([]byte, 16)
	if _, err := rand.Read(ownerBytes); err != nil {
		return StudentMissionAttemptDocument{}, err
	}
	now := time.Now()
	attempt, user, claimed, err := service.learning.ClaimStudentFinalization(ctx, job, hex.EncodeToString(ownerBytes), now, now.Add(finalizationLeaseDuration))
	if err != nil {
		return attempt, err
	}
	if !claimed {
		if attempt.Trace.Finalized {
			service.requestClassLearningRefresh(ctx, attempt.ClassID)
		}
		return attempt, nil
	}
	fence := CaptureFinalizationFence(attempt)

	missionDocument, err := service.missionForAttempt(
		ctx,
		attempt,
	)
	if err != nil {
		_ = service.learning.FailStudentFinalization(ctx, fence, time.Now())
		return StudentMissionAttemptDocument{}, err
	}
	evidence := EffectiveTutorEvidence(attempt)
	safeSnapshot, _ := AssessLearningSnapshotWithEvidence(missionDocument, attempt.Snapshot, attempt.TutorTurns, evidence.AcceptedAnswers, attempt.Status)
	memoryContext := user.Student.LearningMemory
	if !sameStudentClass(user.Student, attempt) {
		memoryContext = StudentLearningMemory{}
	}
	traceInput := toLearningTraceInput(
		missionDocument,
		safeSnapshot,
		attempt.Status,
		memoryContext,
	)
	trace := generation.BuildObservedLearningTrace(traceInput, "learning trace generator is not configured")
	if service.learningTraceGenerator != nil {
		if service.allowLearningGeneration(ctx, attempt.ClassID) {
			trace, err = service.learningTraceGenerator.BuildLearningTrace(ctx, traceInput)
			if err != nil {
				_ = service.learning.FailStudentFinalization(ctx, fence, time.Now())
				return StudentMissionAttemptDocument{}, err
			}
		} else {
			trace = generation.BuildObservedLearningTrace(traceInput, "learning generation quota reached")
		}
	}
	trace.Finalized = true
	trace.EventCount = len(safeSnapshot.Events)
	memory := StudentLearningMemory{
		Strengths:            trace.Strengths,
		ImprovementAreas:     trace.ImprovementAreas,
		PreferredExplanation: attempt.Snapshot.Preferences.ExplanationStyle,
		NextAdaptation:       trace.NextAdaptation,
		LastMissionID:        missionDocument.ID,
		EvidenceAt:           studentAttemptEvidenceTime(attempt),
	}

	finalized, err := service.learning.FinalizeStudentLearningTrace(
		ctx,
		fence,
		trace,
		memory,
		time.Now(),
	)
	if errors.Is(err, ErrStaleFinalization) {
		return finalized, nil
	}
	if err != nil {
		_ = service.learning.FailStudentFinalization(ctx, fence, time.Now())
		return StudentMissionAttemptDocument{}, err
	}
	service.requestClassLearningRefresh(ctx, finalized.ClassID)

	return finalized, nil
}

func (service *Service) allowLearningGeneration(ctx context.Context, classID string) bool {
	if service.learningGenerationLimiter == nil {
		return true
	}

	decision, err := service.learningGenerationLimiter.Allow(ctx, "class:"+normalizeText(classID))
	if err != nil {
		slog.Error("student learning generation rate limiter failed", "error", err, "class_id", classID)
		return false
	}
	if !decision.Allowed {
		slog.Warn("student learning generation quota reached", "class_id", classID, "retry_after", decision.RetryAfter)
		return false
	}

	return true
}
