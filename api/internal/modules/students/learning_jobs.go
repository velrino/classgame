package students

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/tenancy"
)

func (service *Service) ProcessLearningJob(ctx context.Context, job LearningJob) error {
	if err := job.Validate(); err != nil {
		return err
	}
	switch job.Kind {
	case LearningJobFinalize:
		_, err := service.finalizeStudentAttempt(ctx, job.StudentID, job.AttemptID)
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
		_, err = service.finalizeStudentAttempt(ctx, attempt.StudentID, attempt.ID)
		return err
	default:
		return errors.New("unsupported student learning job")
	}
}

func (service *Service) finalizeStudentAttempt(
	ctx context.Context,
	studentID string,
	attemptID string,
) (StudentMissionAttemptDocument, error) {
	if service.learning == nil || service.sessions == nil {
		return StudentMissionAttemptDocument{}, errors.New("student learning repository is not configured")
	}
	attempt, user, exists, err := service.learning.GetStudentLearningAttempt(ctx, studentID, attemptID)
	if err != nil {
		return StudentMissionAttemptDocument{}, err
	}
	if !exists {
		return StudentMissionAttemptDocument{}, ErrStudentAttemptNotFound
	}
	if attempt.Trace.Finalized {
		return attempt, nil
	}
	if attempt.Status != "completed" && attempt.Status != "abandoned" {
		return attempt, nil
	}
	if err := service.learning.SetStudentAttemptTraceState(ctx, attempt.ID, "processing", time.Now()); err != nil {
		return StudentMissionAttemptDocument{}, err
	}

	missionDocument, err := service.missionForAttempt(
		ctx,
		attempt,
	)
	if err != nil {
		_ = service.learning.SetStudentAttemptTraceState(ctx, attempt.ID, "failed", time.Now())
		return StudentMissionAttemptDocument{}, err
	}
	traceInput := toLearningTraceInput(
		missionDocument,
		attempt.Snapshot,
		attempt.Status,
		user.Student.LearningMemory,
	)
	trace := generation.BuildObservedLearningTrace(traceInput, "learning trace generator is not configured")
	if service.learningTraceGenerator != nil {
		if service.allowLearningGeneration(ctx, attempt.ClassID) {
			trace, err = service.learningTraceGenerator.BuildLearningTrace(ctx, traceInput)
			if err != nil {
				_ = service.learning.SetStudentAttemptTraceState(ctx, attempt.ID, "failed", time.Now())
				return StudentMissionAttemptDocument{}, err
			}
		} else {
			trace = generation.BuildObservedLearningTrace(traceInput, "learning generation quota reached")
		}
	}
	trace.Finalized = true
	trace.EventCount = len(attempt.Snapshot.Events)
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
		user.ID,
		attempt.ID,
		trace,
		memory,
		time.Now(),
	)
	if err != nil {
		_ = service.learning.SetStudentAttemptTraceState(ctx, attempt.ID, "failed", time.Now())
		return StudentMissionAttemptDocument{}, err
	}
	if err := service.refreshClassLearningSynthesis(ctx, finalized.ClassID); err != nil {
		slog.Warn("class learning synthesis refresh failed", "error", err, "class_id", finalized.ClassID)
	}

	return finalized, nil
}

func (service *Service) refreshClassLearningSynthesis(ctx context.Context, classID string) error {
	if service.classLearningGenerator == nil || strings.TrimSpace(classID) == "" {
		return nil
	}
	if !service.allowLearningGeneration(ctx, classID) {
		return nil
	}
	attempts, err := service.classLearning.ListLatestFinalizedAttemptsByClassID(ctx, classID, 100)
	if err != nil {
		return err
	}
	evidence := make([]generation.ClassLearningStudentEvidence, 0, len(attempts))
	evidenceUpdatedAt := time.Time{}
	scope := tenancy.DefaultScope()
	for _, attempt := range attempts {
		if attempt.OrganizationID != "" || attempt.DistrictID != "" || attempt.SchoolID != "" {
			scope = tenancy.NormalizeScope(tenancy.Scope{
				OrganizationID: attempt.OrganizationID,
				DistrictID:     attempt.DistrictID,
				SchoolID:       attempt.SchoolID,
			})
		}
		evidence = append(evidence, generation.ClassLearningStudentEvidence{
			StudentID:        attempt.StudentID,
			StudentName:      attempt.StudentName,
			Strengths:        attempt.Trace.Strengths,
			ImprovementAreas: attempt.Trace.ImprovementAreas,
		})
		if studentAttemptEvidenceTime(attempt).After(evidenceUpdatedAt) {
			evidenceUpdatedAt = studentAttemptEvidenceTime(attempt)
		}
	}
	synthesis, err := service.classLearningGenerator.BuildClassLearningSynthesis(
		ctx,
		generation.BuildClassLearningSynthesisInput{
			ClassID:  classID,
			Evidence: evidence,
		},
	)
	if err != nil {
		return err
	}

	return service.classLearning.SaveClassLearningSynthesis(ctx, ClassLearningSynthesisDocument{
		OrganizationID:    scope.OrganizationID,
		DistrictID:        scope.DistrictID,
		SchoolID:          scope.SchoolID,
		ClassID:           classID,
		Synthesis:         synthesis,
		AttemptCount:      len(attempts),
		EvidenceUpdatedAt: evidenceUpdatedAt,
		UpdatedAt:         time.Now(),
	})
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
