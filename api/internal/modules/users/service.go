package users

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	mission "classgame-api/internal/modules/mission"
	"classgame-api/internal/modules/orchestrator"
	teacherclass "classgame-api/internal/modules/teacher-class"
	"classgame-api/internal/modules/tenant"
	sharedauth "classgame-api/internal/shared/auth"
	sharedratelimit "classgame-api/internal/shared/ratelimit"
)

var ErrStudentRequiresAnonymousAuth = errors.New("student session requires anonymous auth")
var ErrStudentNotFound = errors.New("student user not found")
var ErrClassCodeNotFound = errors.New("active class code not found")
var ErrPublishedMissionNotFound = errors.New("published class mission not found")
var ErrMissionCodeNotFound = errors.New("published mission code not found")
var ErrStudentMissionMismatch = errors.New("student mission does not match current session")
var ErrStudentAttemptNotFound = errors.New("student learning attempt not found")
var ErrStudentQuestionNotFound = errors.New("student mission question not found")
var ErrMissionRevisionNotFound = errors.New("published student mission revision not found")
var ErrStudentTutorTurnLimit = errors.New("student tutor turn limit reached")

const maxGeneratedTutorTurnsPerAttempt = 36

type MissionReader interface {
	GetByCode(context.Context, string) (mission.MissionDocument, bool, error)
	Get(context.Context, string, string) (mission.MissionDocument, bool, error)
}

type ClassReader interface {
	Get(context.Context, string) (teacherclass.TeacherClassDocument, bool, error)
}

type MissionRevisionReader interface {
	GetPublishedRevision(
		context.Context,
		string,
		string,
		int64,
	) (mission.MissionPublishedRevisionDocument, bool, error)
}

type LearningTraceGenerator interface {
	BuildLearningTrace(context.Context, orchestrator.BuildLearningTraceInput) (orchestrator.LearningTraceBlueprint, error)
}

type TutorTurnGenerator interface {
	BuildTutorTurn(context.Context, orchestrator.BuildTutorTurnInput) (orchestrator.TutorTurnBlueprint, error)
}

type ClassLearningGenerator interface {
	BuildClassLearningSynthesis(context.Context, orchestrator.BuildClassLearningSynthesisInput) (orchestrator.ClassLearningSynthesis, error)
}

type Service struct {
	abandonmentWindow         time.Duration
	learningTaskEnqueuer      LearningTaskEnqueuer
	learningGenerationLimiter sharedratelimit.Limiter
	learningTraceGenerator    LearningTraceGenerator
	tutorTurnGenerator        TutorTurnGenerator
	classLearningGenerator    ClassLearningGenerator
	classReader               ClassReader
	missionReader             MissionReader
	repository                *Repository
}

func NewService(
	repository *Repository,
	missionReader MissionReader,
	learningTraceGenerators ...LearningTraceGenerator,
) *Service {
	service := &Service{
		abandonmentWindow: defaultStudentAbandonmentWindow,
		missionReader:     missionReader,
		repository:        repository,
	}
	if len(learningTraceGenerators) > 0 {
		service.learningTraceGenerator = learningTraceGenerators[0]
		service.tutorTurnGenerator, _ = learningTraceGenerators[0].(TutorTurnGenerator)
		service.classLearningGenerator, _ = learningTraceGenerators[0].(ClassLearningGenerator)
	}

	return service
}

func (service *Service) ConfigureLearningTasks(
	enqueuer LearningTaskEnqueuer,
	abandonmentWindow time.Duration,
) {
	service.learningTaskEnqueuer = enqueuer
	if abandonmentWindow > 0 {
		service.abandonmentWindow = abandonmentWindow
	}
}

func (service *Service) ConfigureLearningGenerationLimiter(limiter sharedratelimit.Limiter) {
	service.learningGenerationLimiter = limiter
}

func (service *Service) ConfigureClassReader(reader ClassReader) {
	service.classReader = reader
}

func (service *Service) UpsertStudent(
	ctx context.Context,
	identity sharedauth.Identity,
	classCode string,
	studentName string,
) (UserDocument, error) {
	if identity.AuthMethod != sharedauth.AuthMethodAnonymous {
		return UserDocument{}, ErrStudentRequiresAnonymousAuth
	}

	teacherClass, err := service.classByCode(ctx, classCode)
	if err != nil {
		return UserDocument{}, err
	}

	user, err := service.repository.UpsertStudent(ctx, UpsertStudentInput{
		Identity: identity,
		Scope: tenant.Scope{
			OrganizationID: teacherClass.OrganizationID,
			DistrictID:     teacherClass.DistrictID,
			SchoolID:       teacherClass.SchoolID,
		},
		ClassID:        teacherClass.ClassID,
		ClassCode:      teacherClass.ClassCode,
		StudentName:    normalizeStudentName(studentName),
		IncrementLogin: true,
		Timestamp:      time.Now(),
	})
	if err != nil {
		return UserDocument{}, err
	}
	user.SessionClass = studentSessionClassFromDocument(teacherClass)

	return user, nil
}

func (service *Service) ValidateStudentClassCode(
	ctx context.Context,
	classCode string,
) error {
	_, err := service.classByCode(ctx, classCode)
	return err
}

func (service *Service) StartStudentMission(
	ctx context.Context,
	identity sharedauth.Identity,
	classCode string,
	missionID string,
	studentName string,
) (UserDocument, error) {
	if identity.AuthMethod != sharedauth.AuthMethodAnonymous {
		return UserDocument{}, ErrStudentRequiresAnonymousAuth
	}
	teacherClass, err := service.classByCode(ctx, classCode)
	if err != nil {
		return UserDocument{}, err
	}
	missionDocument, err := service.publishedMissionByID(
		ctx,
		teacherClass.ClassID,
		missionID,
	)
	if err != nil {
		return UserDocument{}, err
	}
	user, err := service.repository.UpsertStudent(ctx, UpsertStudentInput{
		Identity: identity,
		Scope: tenant.Scope{
			OrganizationID: teacherClass.OrganizationID,
			DistrictID:     teacherClass.DistrictID,
			SchoolID:       teacherClass.SchoolID,
		},
		ClassID:                  teacherClass.ClassID,
		ClassCode:                teacherClass.ClassCode,
		MissionID:                missionDocument.ID,
		MissionCode:              missionDocument.MissionCode,
		MissionPublishedRevision: missionDocument.PublishedRevision,
		MissionSubject:           missionDocument.Subject,
		StudentName:              normalizeStudentName(studentName),
		Timestamp:                time.Now(),
	})
	if err != nil {
		return UserDocument{}, err
	}
	sessionMission, err := service.missionAtPublishedRevision(
		ctx,
		missionDocument,
		user.Activity.CurrentGame.PublishedRevision,
	)
	if err != nil {
		return UserDocument{}, err
	}
	user.SessionClass = studentSessionClassFromDocument(teacherClass)
	user.SessionMission = studentSessionMissionFromDocument(sessionMission)

	return user, nil
}

func (service *Service) classByCode(
	ctx context.Context,
	classCode string,
) (teacherclass.TeacherClassDocument, error) {
	if service.classReader == nil {
		return teacherclass.TeacherClassDocument{}, errors.New("student class reader is not configured")
	}
	teacherClass, exists, err := service.classReader.Get(ctx, normalizeClassCode(classCode))
	if err != nil {
		return teacherclass.TeacherClassDocument{}, err
	}
	if !exists || teacherClass.Status != "active" {
		return teacherclass.TeacherClassDocument{}, ErrClassCodeNotFound
	}

	return teacherClass, nil
}

func (service *Service) publishedMissionByID(
	ctx context.Context,
	classID string,
	missionID string,
) (mission.MissionDocument, error) {
	if service.missionReader == nil {
		return mission.MissionDocument{}, errors.New("student mission reader is not configured")
	}
	normalizedClassID := normalizeClassCode(classID)
	normalizedMissionID := normalizeText(missionID)
	missionDocument, exists, err := service.missionReader.Get(
		ctx,
		normalizedClassID,
		normalizedMissionID,
	)
	if err != nil {
		return mission.MissionDocument{}, err
	}
	if !exists ||
		missionDocument.ID != normalizedMissionID ||
		missionDocument.ClassID != normalizedClassID {
		return mission.MissionDocument{}, ErrPublishedMissionNotFound
	}
	published, available := mission.PublishedView(missionDocument)
	if !available || published.AudienceType == mission.AudienceSelected {
		return mission.MissionDocument{}, ErrPublishedMissionNotFound
	}

	return published, nil
}

func (service *Service) publishedMissionByCode(
	ctx context.Context,
	missionCode string,
) (mission.MissionDocument, error) {
	if service.missionReader == nil {
		return mission.MissionDocument{}, errors.New("student mission reader is not configured")
	}

	missionDocument, exists, err := service.missionReader.GetByCode(ctx, normalizeMissionCode(missionCode))
	if err != nil {
		return mission.MissionDocument{}, err
	}
	if !exists || missionDocument.Status != mission.StatusPublished {
		return mission.MissionDocument{}, ErrMissionCodeNotFound
	}

	return missionDocument, nil
}

func (service *Service) missionAtPublishedRevision(
	ctx context.Context,
	current mission.MissionDocument,
	revision int64,
) (mission.MissionDocument, error) {
	if revision <= 0 || current.PublishedRevision == revision {
		return current, nil
	}

	revisionReader, ok := service.missionReader.(MissionRevisionReader)
	if !ok {
		return mission.MissionDocument{}, ErrMissionRevisionNotFound
	}
	published, exists, err := revisionReader.GetPublishedRevision(
		ctx,
		current.ClassID,
		current.ID,
		revision,
	)
	if err != nil {
		return mission.MissionDocument{}, err
	}
	if !exists {
		return mission.MissionDocument{}, ErrMissionRevisionNotFound
	}
	view, valid := mission.ViewAtPublishedRevision(current, published)
	if !valid {
		return mission.MissionDocument{}, ErrMissionRevisionNotFound
	}

	return view, nil
}

func (service *Service) UpdateStudentPresence(
	ctx context.Context,
	identity sharedauth.Identity,
	missionID string,
	classCode string,
	subject string,
	stageID string,
	stageName string,
	status string,
) (UserDocument, error) {
	if identity.AuthMethod != sharedauth.AuthMethodAnonymous {
		return UserDocument{}, ErrStudentRequiresAnonymousAuth
	}
	teacherClass, err := service.classByCode(ctx, classCode)
	if err != nil {
		return UserDocument{}, err
	}
	normalizedMissionID := normalizeText(missionID)
	if _, err := service.publishedMissionByID(ctx, teacherClass.ClassID, normalizedMissionID); err != nil {
		return UserDocument{}, err
	}

	return service.repository.UpdateStudentPresence(ctx, UpdateStudentPresenceInput{
		Identity:  identity,
		MissionID: normalizedMissionID,
		Subject:   normalizeText(subject),
		StageID:   normalizeText(stageID),
		StageName: normalizeText(stageName),
		Status:    normalizeText(status),
		Timestamp: time.Now(),
	})
}

func (service *Service) SaveStudentLearningSnapshot(
	ctx context.Context,
	identity sharedauth.Identity,
	missionID string,
	status string,
	snapshot StudentLearningSnapshot,
) (StudentMissionAttemptDocument, error) {
	if identity.AuthMethod != sharedauth.AuthMethodAnonymous {
		return StudentMissionAttemptDocument{}, ErrStudentRequiresAnonymousAuth
	}
	if service.repository == nil {
		return StudentMissionAttemptDocument{}, errors.New("student learning repository is not configured")
	}

	current, user, exists, err := service.repository.GetCurrentStudentLearningAttempt(ctx, identity.ProviderUserID, normalizeText(missionID))
	if err != nil {
		return StudentMissionAttemptDocument{}, err
	}
	if !exists {
		current = StudentMissionAttemptDocument{StudentID: user.ID, ClassID: user.Student.ClassID, MissionID: normalizeText(missionID), PublishedRevision: user.Activity.CurrentGame.PublishedRevision}
	}
	document, err := service.missionForAttempt(ctx, current)
	if err != nil {
		return StudentMissionAttemptDocument{}, err
	}
	normalizedStatus := strings.ToLower(normalizeText(status))
	normalizedSnapshot := normalizeLearningSnapshot(snapshot)
	liveInput := toLearningTraceInput(mission.MissionDocument{
		ID:      normalizeText(missionID),
		Subject: "Classroom learning",
	}, normalizedSnapshot, normalizedStatus, StudentLearningMemory{})
	liveInput.Snapshot.Status = "in_progress"
	liveTrace := orchestrator.BuildObservedLearningTrace(liveInput, "")
	liveTrace.Finalized = false

	attempt, _, completionStarted, err := service.repository.SaveStudentLearningSnapshot(ctx, SaveStudentLearningSnapshotInput{
		Identity:          identity,
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
	identity sharedauth.Identity,
	missionID string,
) (StudentMissionAttemptDocument, bool, error) {
	if identity.AuthMethod != sharedauth.AuthMethodAnonymous {
		return StudentMissionAttemptDocument{}, false, ErrStudentRequiresAnonymousAuth
	}
	if service.repository == nil {
		return StudentMissionAttemptDocument{}, false, errors.New("student learning repository is not configured")
	}

	attempt, _, exists, err := service.repository.GetCurrentStudentLearningAttempt(
		ctx,
		identity.ProviderUserID,
		normalizeText(missionID),
	)
	return attempt, exists, err
}

func (service *Service) SaveStudentTutorTurn(
	ctx context.Context,
	identity sharedauth.Identity,
	input SaveStudentTutorTurnInput,
) (StudentTutorTurn, error) {
	if identity.AuthMethod != sharedauth.AuthMethodAnonymous {
		return StudentTutorTurn{}, ErrStudentRequiresAnonymousAuth
	}
	if service.repository == nil || service.tutorTurnGenerator == nil {
		return StudentTutorTurn{}, errors.New("student tutor dependencies are not configured")
	}

	input = normalizeStudentTutorTurnInput(input)
	input.Identity = identity
	input.Timestamp = time.Now()
	attempt, user, exists, err := service.repository.GetCurrentStudentLearningAttempt(
		ctx,
		identity.ProviderUserID,
		input.MissionID,
	)
	if err != nil {
		return StudentTutorTurn{}, err
	}
	if !exists {
		return StudentTutorTurn{}, ErrStudentAttemptNotFound
	}
	if existingTurn, exists := existingStudentTutorTurn(attempt.TutorTurns, input.EventID); exists {
		return existingTurn, nil
	}
	if input.Type != "feedback" && studentTutorTurnLimitReached(attempt.TutorTurns) {
		return StudentTutorTurn{}, ErrStudentTutorTurnLimit
	}
	missionDocument, err := service.missionForAttempt(
		ctx,
		attempt,
	)
	if err != nil {
		return StudentTutorTurn{}, err
	}
	input, err = groundStudentTutorTurn(missionDocument, input)
	if err != nil {
		return StudentTutorTurn{}, err
	}

	retrievedChunks := retrieveMissionKnowledge(missionDocument, input, 5)
	classContext := orchestrator.TutorClassContext{}
	if input.Type != "feedback" {
		classContext, err = service.classTutorContext(ctx, user.Student.ClassID)
		if err != nil {
			return StudentTutorTurn{}, err
		}
	}
	turnInput := orchestrator.BuildTutorTurnInput{
		Mission: orchestrator.TutorMissionContext{
			ID:           missionDocument.ID,
			Title:        missionDocument.Title,
			Subject:      missionDocument.Subject,
			GradeLevel:   missionDocument.GradeLevel,
			LearningGoal: missionDocument.LearningGoal,
			Stages:       missionDocument.Stages,
		},
		StudentMemory: orchestrator.LearningTracePreviousMemory{
			Strengths:            user.Student.LearningMemory.Strengths,
			ImprovementAreas:     user.Student.LearningMemory.ImprovementAreas,
			PreferredExplanation: user.Student.LearningMemory.PreferredExplanation,
			NextAdaptation:       user.Student.LearningMemory.NextAdaptation,
		},
		ClassContext: classContext,
		Attempt: orchestrator.TutorAttemptContext{
			Status:       attempt.Status,
			Progress:     toTutorProgress(attempt.Snapshot.Progress),
			RecentTurns:  toTutorTurnHistory(attempt.TutorTurns),
			RecentEvents: toTutorEvents(attempt.Snapshot.Events),
		},
		Turn: orchestrator.TutorTurnEvidence{
			EventID:           input.EventID,
			Type:              input.Type,
			StageID:           input.StageID,
			QuestionID:        input.QuestionID,
			Prompt:            input.Prompt,
			StudentResponse:   input.StudentResponse,
			ExpectedAnswer:    input.ExpectedAnswer,
			Outcome:           input.Outcome,
			ExplicitReasoning: input.ExplicitReasoning,
			Feedback:          input.Feedback,
		},
		RetrievedChunks: retrievedChunks,
	}

	tutor := orchestrator.BuildObservedTutorTurn(
		turnInput,
		"Feedback was recorded for the next tutor interaction without generating new AI content.",
	)
	if input.Type != "feedback" {
		tutor, err = service.tutorTurnGenerator.BuildTutorTurn(ctx, turnInput)
		if err != nil {
			return StudentTutorTurn{}, err
		}
	}

	return service.repository.AppendStudentTutorTurn(ctx, input, tutor)
}

func (service *Service) classTutorContext(
	ctx context.Context,
	classID string,
) (orchestrator.TutorClassContext, error) {
	attempts, err := service.repository.ListLatestFinalizedAttemptsByClassID(ctx, classID, 40)
	if err != nil {
		return orchestrator.TutorClassContext{}, err
	}
	strengths := make([]string, 0)
	needs := make([]string, 0)
	for _, attempt := range attempts {
		strengths = append(strengths, attempt.Trace.Strengths...)
		needs = append(needs, attempt.Trace.ImprovementAreas...)
	}

	return orchestrator.TutorClassContext{
		CommonStrengths: mostFrequentSignals(strengths, 4),
		CommonNeeds:     mostFrequentSignals(needs, 4),
	}, nil
}

func (service *Service) ProcessLearningJob(ctx context.Context, job LearningJob) error {
	if err := job.Validate(); err != nil {
		return err
	}
	switch job.Kind {
	case LearningJobFinalize:
		_, err := service.finalizeStudentAttempt(ctx, job.StudentID, job.AttemptID)
		return err
	case LearningJobWatchdog:
		attempt, marked, err := service.repository.MarkStudentAttemptAbandoned(
			ctx,
			job.StudentID,
			job.AttemptID,
			job.ExpectedLastActivityAt,
			time.Now(),
		)
		if err != nil {
			return err
		}
		// A Cloud Tasks retry sees the attempt as already abandoned. Continue the
		// unfinished finalization instead of acknowledging the retry and leaving
		// the teacher-facing trace permanently pending or failed.
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
	if service.repository == nil {
		return StudentMissionAttemptDocument{}, errors.New("student learning repository is not configured")
	}
	attempt, user, exists, err := service.repository.GetStudentLearningAttempt(ctx, studentID, attemptID)
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
	if err := service.repository.SetStudentAttemptTraceState(ctx, attempt.ID, "processing", time.Now()); err != nil {
		return StudentMissionAttemptDocument{}, err
	}

	missionDocument, err := service.missionForAttempt(
		ctx,
		attempt,
	)
	if err != nil {
		_ = service.repository.SetStudentAttemptTraceState(ctx, attempt.ID, "failed", time.Now())
		return StudentMissionAttemptDocument{}, err
	}
	traceInput := toLearningTraceInput(
		missionDocument,
		attempt.Snapshot,
		attempt.Status,
		user.Student.LearningMemory,
	)
	trace := orchestrator.BuildObservedLearningTrace(traceInput, "learning trace generator is not configured")
	if service.learningTraceGenerator != nil {
		if service.allowLearningGeneration(ctx, attempt.ClassID) {
			trace, err = service.learningTraceGenerator.BuildLearningTrace(ctx, traceInput)
			if err != nil {
				_ = service.repository.SetStudentAttemptTraceState(ctx, attempt.ID, "failed", time.Now())
				return StudentMissionAttemptDocument{}, err
			}
		} else {
			trace = orchestrator.BuildObservedLearningTrace(traceInput, "learning generation quota reached")
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

	finalized, err := service.repository.FinalizeStudentLearningTrace(
		ctx,
		user.ID,
		attempt.ID,
		trace,
		memory,
		time.Now(),
	)
	if err != nil {
		_ = service.repository.SetStudentAttemptTraceState(ctx, attempt.ID, "failed", time.Now())
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
	attempts, err := service.repository.ListLatestFinalizedAttemptsByClassID(ctx, classID, 100)
	if err != nil {
		return err
	}
	evidence := make([]orchestrator.ClassLearningStudentEvidence, 0, len(attempts))
	evidenceUpdatedAt := time.Time{}
	scope := tenant.DefaultScope()
	for _, attempt := range attempts {
		if attempt.OrganizationID != "" || attempt.DistrictID != "" || attempt.SchoolID != "" {
			scope = tenant.NormalizeScope(tenant.Scope{
				OrganizationID: attempt.OrganizationID,
				DistrictID:     attempt.DistrictID,
				SchoolID:       attempt.SchoolID,
			})
		}
		evidence = append(evidence, orchestrator.ClassLearningStudentEvidence{
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
		orchestrator.BuildClassLearningSynthesisInput{
			ClassID:  classID,
			Evidence: evidence,
		},
	)
	if err != nil {
		return err
	}

	return service.repository.SaveClassLearningSynthesis(ctx, ClassLearningSynthesisDocument{
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

func (service *Service) missionForAttempt(
	ctx context.Context,
	attempt StudentMissionAttemptDocument,
) (mission.MissionDocument, error) {
	if revisionReader, ok := service.missionReader.(MissionRevisionReader); ok && attempt.PublishedRevision > 0 {
		published, exists, err := revisionReader.GetPublishedRevision(
			ctx,
			attempt.ClassID,
			attempt.MissionID,
			attempt.PublishedRevision,
		)
		if err != nil {
			return mission.MissionDocument{}, err
		}
		if !exists {
			return mission.MissionDocument{}, ErrMissionRevisionNotFound
		}
		view, valid := mission.ViewAtPublishedRevision(mission.MissionDocument{
			ID:          attempt.MissionID,
			ClassID:     attempt.ClassID,
			MissionCode: attempt.MissionCode,
		}, published)
		if !valid {
			return mission.MissionDocument{}, ErrMissionRevisionNotFound
		}

		return view, nil
	}

	// Compatibility path for repository fakes and legacy deployments without
	// immutable revision reads.
	current, err := service.publishedMissionByID(ctx, attempt.ClassID, attempt.MissionID)
	if err != nil {
		return mission.MissionDocument{}, err
	}

	return service.missionAtPublishedRevision(ctx, current, attempt.PublishedRevision)
}

func toLearningTraceInput(
	missionDocument mission.MissionDocument,
	snapshot StudentLearningSnapshot,
	status string,
	memory StudentLearningMemory,
) orchestrator.BuildLearningTraceInput {
	progress := make([]orchestrator.LearningTraceStageProgress, 0, len(snapshot.Progress))
	for _, stage := range snapshot.Progress {
		progress = append(progress, orchestrator.LearningTraceStageProgress{
			StageID:  stage.StageID,
			Status:   stage.Status,
			Correct:  stage.Correct,
			Required: stage.Required,
		})
	}
	notes := make([]orchestrator.LearningTraceNote, 0, len(snapshot.Notes))
	for _, note := range snapshot.Notes {
		notes = append(notes, orchestrator.LearningTraceNote{
			Kind:     note.Kind,
			Skill:    note.Skill,
			Text:     note.Text,
			Evidence: note.Evidence,
			Count:    note.Count,
		})
	}
	decisions := make([]orchestrator.LearningTraceDecision, 0, len(snapshot.Decisions))
	for _, decision := range snapshot.Decisions {
		decisions = append(decisions, orchestrator.LearningTraceDecision{
			Action:  decision.Action,
			Message: decision.Message,
			Why:     decision.Why,
		})
	}
	events := make([]orchestrator.LearningTraceEvent, 0, len(snapshot.Events))
	for _, event := range snapshot.Events {
		events = append(events, orchestrator.LearningTraceEvent{
			ID:         event.ID,
			Sequence:   event.Sequence,
			Type:       event.Type,
			StageID:    event.StageID,
			QuestionID: event.QuestionID,
			Title:      event.Title,
			Detail:     event.Detail,
			Outcome:    event.Outcome,
			OccurredAt: event.OccurredAt,
		})
	}

	return orchestrator.BuildLearningTraceInput{
		Mission: orchestrator.LearningTraceMission{
			Title:        missionDocument.Title,
			Subject:      missionDocument.Subject,
			GradeLevel:   missionDocument.GradeLevel,
			LearningGoal: missionDocument.LearningGoal,
		},
		Snapshot: orchestrator.LearningTraceSnapshot{
			Status:     strings.ToLower(normalizeText(status)),
			Progress:   progress,
			Notes:      notes,
			Redirected: snapshot.Redirected,
			ReturnTo:   snapshot.ReturnTo,
			LastPod:    snapshot.LastPod,
			Preferences: orchestrator.LearningTracePreferences{
				ExplanationStyle: snapshot.Preferences.ExplanationStyle,
				DifficultyBias:   snapshot.Preferences.DifficultyBias,
			},
			Decisions: decisions,
			Events:    events,
		},
		PreviousMemory: orchestrator.LearningTracePreviousMemory{
			Strengths:            memory.Strengths,
			ImprovementAreas:     memory.ImprovementAreas,
			PreferredExplanation: memory.PreferredExplanation,
			NextAdaptation:       memory.NextAdaptation,
		},
	}
}

func normalizeLearningSnapshot(snapshot StudentLearningSnapshot) StudentLearningSnapshot {
	if snapshot.Revision < 0 {
		snapshot.Revision = 0
	}
	snapshot.ReturnTo = normalizeText(snapshot.ReturnTo)
	snapshot.LastPod = normalizeText(snapshot.LastPod)
	for index := range snapshot.Progress {
		progress := &snapshot.Progress[index]
		progress.StageID = normalizeText(progress.StageID)
		progress.Status = strings.ToLower(normalizeText(progress.Status))
	}
	for index := range snapshot.Notes {
		note := &snapshot.Notes[index]
		note.ID = normalizeText(note.ID)
		note.MisconceptionID = normalizeText(note.MisconceptionID)
		note.Text = normalizeText(note.Text)
		note.Kind = strings.ToLower(normalizeText(note.Kind))
		note.Skill = normalizeText(note.Skill)
	}
	for index := range snapshot.Decisions {
		decision := &snapshot.Decisions[index]
		decision.ID = normalizeText(decision.ID)
		decision.Action = strings.ToLower(normalizeText(decision.Action))
		decision.Message = normalizeText(decision.Message)
		decision.Why = normalizeText(decision.Why)
	}
	for index := range snapshot.Events {
		event := &snapshot.Events[index]
		event.ID = normalizeText(event.ID)
		event.Type = strings.ToLower(normalizeText(event.Type))
		event.StageID = normalizeText(event.StageID)
		event.QuestionID = normalizeText(event.QuestionID)
		event.Title = normalizeText(event.Title)
		event.Detail = normalizeText(event.Detail)
		event.Outcome = strings.ToLower(normalizeText(event.Outcome))
	}
	snapshot.Preferences.ExplanationStyle = strings.ToLower(normalizeText(snapshot.Preferences.ExplanationStyle))
	snapshot.Checkpoint.StageID = normalizeText(snapshot.Checkpoint.StageID)
	snapshot.Checkpoint.QuestionID = normalizeText(snapshot.Checkpoint.QuestionID)
	snapshot.Checkpoint.Phase = strings.ToLower(normalizeText(snapshot.Checkpoint.Phase))
	snapshot.Checkpoint.WrongChoice = normalizeText(snapshot.Checkpoint.WrongChoice)
	snapshot.Checkpoint.WrongMisconceptionID = normalizeText(snapshot.Checkpoint.WrongMisconceptionID)
	snapshot.Checkpoint.DecisionAction = strings.ToLower(normalizeText(snapshot.Checkpoint.DecisionAction))
	snapshot.Checkpoint.DecisionTarget = normalizeText(snapshot.Checkpoint.DecisionTarget)
	snapshot.Checkpoint.DecisionMessage = truncateStudentText(snapshot.Checkpoint.DecisionMessage, 500)
	snapshot.Checkpoint.DecisionWhy = truncateStudentText(snapshot.Checkpoint.DecisionWhy, 500)
	snapshot.Checkpoint.TutorMessage = truncateStudentText(snapshot.Checkpoint.TutorMessage, 500)
	snapshot.Checkpoint.WhyPrompt = truncateStudentText(snapshot.Checkpoint.WhyPrompt, 500)
	if len(snapshot.Checkpoint.Lines) > 12 {
		snapshot.Checkpoint.Lines = snapshot.Checkpoint.Lines[:12]
	}
	for index := range snapshot.Checkpoint.Lines {
		snapshot.Checkpoint.Lines[index] = normalizeText(snapshot.Checkpoint.Lines[index])
	}

	return snapshot
}

func studentSessionMissionFromDocument(document mission.MissionDocument) StudentSessionMission {
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
		PreviewQuestions: orchestrator.PlayablePreviewQuestions(orchestrator.BuildMissionInput{
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

func studentSessionClassFromDocument(document teacherclass.TeacherClassDocument) StudentSessionClass {
	return StudentSessionClass{
		ID:         document.ClassID,
		ClassCode:  document.ClassCode,
		ClassName:  document.ClassName,
		Subject:    document.Subject,
		GradeLevel: document.GradeLevel,
	}
}

func normalizeStudentTutorTurnInput(input SaveStudentTutorTurnInput) SaveStudentTutorTurnInput {
	input.MissionID = normalizeText(input.MissionID)
	input.EventID = normalizeText(input.EventID)
	input.Type = strings.ToLower(normalizeText(input.Type))
	input.StageID = normalizeText(input.StageID)
	input.QuestionID = normalizeText(input.QuestionID)
	input.Prompt = truncateStudentText(input.Prompt, 500)
	input.StudentResponse = truncateStudentText(input.StudentResponse, 800)
	input.ExpectedAnswer = truncateStudentText(input.ExpectedAnswer, 500)
	input.Outcome = strings.ToLower(normalizeText(input.Outcome))
	input.ExplicitReasoning = truncateStudentText(input.ExplicitReasoning, 800)
	input.Feedback = strings.ToLower(truncateStudentText(input.Feedback, 80))

	return input
}

func groundStudentTutorTurn(
	document mission.MissionDocument,
	input SaveStudentTutorTurnInput,
) (SaveStudentTutorTurnInput, error) {
	if input.Type == "resume" && input.QuestionID == "" {
		input.StageID = "mission-resume"
		input.Prompt = "Resume the published mission: " + document.Title
		input.ExpectedAnswer = ""
		input.Outcome = "neutral"
		return input, nil
	}

	var publishedQuestion *orchestrator.PreviewQuestion
	for index := range document.PreviewQuestions {
		question := &document.PreviewQuestions[index]
		if question.ID == input.QuestionID {
			publishedQuestion = question
			break
		}
	}
	if publishedQuestion == nil {
		return SaveStudentTutorTurnInput{}, ErrStudentQuestionNotFound
	}

	input.StageID = publishedQuestion.StageID
	input.Prompt = publishedQuestion.Prompt
	input.ExpectedAnswer = publishedQuestion.ExpectedAnswer
	switch input.Type {
	case "answer":
		if normalizedStudentAnswer(input.StudentResponse) == normalizedStudentAnswer(publishedQuestion.ExpectedAnswer) {
			input.Outcome = "correct"
		} else {
			input.Outcome = "incorrect"
		}
	case "reasoning", "feedback":
		input.Outcome = "support"
	case "resume":
		input.Outcome = "neutral"
	}

	return input, nil
}

func normalizedStudentAnswer(value string) string {
	value = strings.ToLower(normalizeText(value))
	return strings.Trim(value, " .,!?:;\"'")
}

func existingStudentTutorTurn(
	turns []StudentTutorTurn,
	eventID string,
) (StudentTutorTurn, bool) {
	turnID := tutorTurnID(eventID)
	for _, turn := range turns {
		if turn.ID == turnID {
			return turn, true
		}
	}

	return StudentTutorTurn{}, false
}

func studentTutorTurnLimitReached(turns []StudentTutorTurn) bool {
	generatedTurns := 0
	for _, turn := range turns {
		if turn.Type != "feedback" {
			generatedTurns++
		}
	}

	return generatedTurns >= maxGeneratedTutorTurnsPerAttempt
}

func retrieveMissionKnowledge(
	document mission.MissionDocument,
	turn SaveStudentTutorTurnInput,
	limit int,
) []orchestrator.KnowledgeChunk {
	chunks := append([]orchestrator.KnowledgeChunk{}, document.KnowledgeChunks...)
	if len(chunks) == 0 {
		for index, stage := range document.Stages {
			chunks = append(chunks, orchestrator.KnowledgeChunk{
				ID:       "legacy-stage-" + normalizeRetrievalID(stage.ID, index),
				Source:   "Published mission",
				Title:    stage.Title,
				Content:  normalizeText(strings.Join(append([]string{stage.Goal}, stage.Checks...), " ")),
				Keywords: strings.Fields(stage.Title + " " + document.Subject),
			})
		}
		for index, question := range document.PreviewQuestions {
			chunks = append(chunks, orchestrator.KnowledgeChunk{
				ID:       "legacy-question-" + normalizeRetrievalID(question.ID, index),
				Source:   "Published mission",
				Title:    question.Prompt,
				Content:  normalizeText(strings.Join(append(question.ExplanationSteps, question.ExpectedAnswer), " ")),
				Keywords: strings.Fields(question.Prompt + " " + question.ExpectedAnswer),
			})
		}
	}
	queryTokens := retrievalTokens(strings.Join([]string{
		document.Subject,
		document.LearningGoal,
		turn.StageID,
		turn.Prompt,
		turn.StudentResponse,
		turn.ExplicitReasoning,
	}, " "))
	type scoredChunk struct {
		chunk orchestrator.KnowledgeChunk
		score int
		index int
	}
	scored := make([]scoredChunk, 0, len(chunks))
	for index, chunk := range chunks {
		score := 0
		chunkTokens := retrievalTokens(strings.Join(append([]string{chunk.Title, chunk.Content}, chunk.Keywords...), " "))
		for token := range queryTokens {
			if _, exists := chunkTokens[token]; exists {
				score++
			}
		}
		if turn.StageID != "" && strings.Contains(strings.ToLower(chunk.ID+" "+chunk.Title), strings.ToLower(turn.StageID)) {
			score += 4
		}
		scored = append(scored, scoredChunk{chunk: chunk, score: score, index: index})
	}
	sort.SliceStable(scored, func(left int, right int) bool {
		if scored[left].score == scored[right].score {
			return scored[left].index < scored[right].index
		}

		return scored[left].score > scored[right].score
	})
	if limit <= 0 {
		limit = 5
	}
	result := make([]orchestrator.KnowledgeChunk, 0, min(limit, len(scored)))
	for _, entry := range scored {
		if entry.score == 0 || strings.TrimSpace(entry.chunk.Content) == "" {
			continue
		}
		result = append(result, entry.chunk)
		if len(result) >= limit {
			break
		}
	}

	return result
}

func retrievalTokens(value string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, token := range strings.Fields(strings.ToLower(value)) {
		token = strings.Trim(token, ".,:;!?()[]{}\"'")
		if len(token) < 3 {
			continue
		}
		result[token] = struct{}{}
	}

	return result
}

func normalizeRetrievalID(value string, index int) string {
	value = strings.ToLower(normalizeText(value))
	value = strings.NewReplacer(" ", "-", "_", "-").Replace(value)
	value = strings.Trim(value, "-")
	if value == "" {
		return fmt.Sprintf("%d", index+1)
	}

	return value
}

func toTutorProgress(progress []StudentStageProgress) []orchestrator.LearningTraceStageProgress {
	result := make([]orchestrator.LearningTraceStageProgress, 0, len(progress))
	for _, stage := range progress {
		result = append(result, orchestrator.LearningTraceStageProgress{
			StageID:  stage.StageID,
			Status:   stage.Status,
			Correct:  stage.Correct,
			Required: stage.Required,
		})
	}

	return result
}

func toTutorTurnHistory(turns []StudentTutorTurn) []orchestrator.TutorTurnHistory {
	if len(turns) > 12 {
		turns = turns[len(turns)-12:]
	}
	result := make([]orchestrator.TutorTurnHistory, 0, len(turns))
	for _, turn := range turns {
		result = append(result, orchestrator.TutorTurnHistory{
			Sequence: turn.Sequence,
			Type:     turn.Type,
			StageID:  turn.StageID,
			Prompt:   turn.Prompt,
			Response: turn.StudentResponse,
			Outcome:  turn.Outcome,
			Message:  turn.Tutor.Message,
		})
	}

	return result
}

func toTutorEvents(events []StudentLearningEvent) []orchestrator.LearningTraceEvent {
	if len(events) > 24 {
		events = events[len(events)-24:]
	}
	result := make([]orchestrator.LearningTraceEvent, 0, len(events))
	for _, event := range events {
		result = append(result, orchestrator.LearningTraceEvent{
			ID:         event.ID,
			Sequence:   event.Sequence,
			Type:       event.Type,
			StageID:    event.StageID,
			QuestionID: event.QuestionID,
			Title:      event.Title,
			Detail:     event.Detail,
			Outcome:    event.Outcome,
			OccurredAt: event.OccurredAt,
		})
	}

	return result
}

func mostFrequentSignals(values []string, limit int) []string {
	type count struct {
		label string
		value int
	}
	counts := make(map[string]*count)
	for _, value := range values {
		value = normalizeText(value)
		key := strings.ToLower(value)
		if value == "" {
			continue
		}
		if counts[key] == nil {
			counts[key] = &count{label: value}
		}
		counts[key].value++
	}
	ordered := make([]count, 0, len(counts))
	for _, item := range counts {
		ordered = append(ordered, *item)
	}
	sort.SliceStable(ordered, func(left int, right int) bool {
		if ordered[left].value == ordered[right].value {
			return ordered[left].label < ordered[right].label
		}

		return ordered[left].value > ordered[right].value
	})
	result := make([]string, 0, min(limit, len(ordered)))
	for _, item := range ordered {
		result = append(result, item.label)
		if len(result) >= limit {
			break
		}
	}

	return result
}

func truncateStudentText(value string, limit int) string {
	value = normalizeText(value)
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}

	return strings.TrimSpace(string(runes[:limit]))
}

func normalizeMissionCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizeClassCode(value string) string {
	return strings.TrimSpace(value)
}

func normalizeStudentName(value string) string {
	return truncateStudentText(value, studentNameMaxLength)
}

func normalizeText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
