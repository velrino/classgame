package students

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"classgame-api/internal/taskqueue"
)

const (
	LearningJobFinalize = "finalize"
	LearningJobWatchdog = "watchdog"

	defaultLearningFinalizationDelay = time.Second
	defaultStudentAbandonmentWindow  = 2 * time.Minute
)

type LearningJob struct {
	ExpectedEvidenceRevision int64  `json:"expected_evidence_revision,omitempty"`
	ExpectedSnapshotRevision int64  `json:"expected_snapshot_revision,omitempty"`
	ExpectedStatus           string `json:"expected_status,omitempty"`
	DispatchGeneration       int64  `json:"dispatch_generation,omitempty"`

	Kind                   string    `json:"kind"`
	StudentID              string    `json:"student_id"`
	AttemptID              string    `json:"attempt_id"`
	ExpectedLastActivityAt time.Time `json:"expected_last_activity_at,omitempty"`
}

func (job LearningJob) Validate() error {
	if job.ExpectedEvidenceRevision < 0 || job.ExpectedSnapshotRevision < 0 || job.DispatchGeneration < 0 {
		return errors.New("learning job version cannot be negative")
	}
	if job.ExpectedStatus != "" && job.ExpectedStatus != "completed" && job.ExpectedStatus != "abandoned" {
		return errors.New("learning job expected status is invalid")
	}
	if job.Kind != LearningJobFinalize && job.Kind != LearningJobWatchdog {
		return errors.New("student learning job kind is invalid")
	}
	if strings.TrimSpace(job.StudentID) == "" {
		return errors.New("student learning job student_id is required")
	}
	if strings.TrimSpace(job.AttemptID) == "" {
		return errors.New("student learning job attempt_id is required")
	}
	if job.Kind == LearningJobWatchdog && job.ExpectedLastActivityAt.IsZero() {
		return errors.New("student learning watchdog expected_last_activity_at is required")
	}

	return nil
}

type LearningTaskQueue interface {
	Enqueue(context.Context, taskqueue.Message) error
}

type LearningTaskEnqueuer interface {
	EnqueueFinalization(context.Context, StudentMissionAttemptDocument) error
	EnqueueWatchdog(context.Context, StudentMissionAttemptDocument) error
}

type LearningTaskScheduler struct {
	abandonmentWindow time.Duration
	finalizationDelay time.Duration
	now               func() time.Time
	queue             LearningTaskQueue
}

func NewLearningTaskScheduler(
	queue LearningTaskQueue,
	abandonmentWindow time.Duration,
) *LearningTaskScheduler {
	if abandonmentWindow <= 0 {
		abandonmentWindow = defaultStudentAbandonmentWindow
	}

	return &LearningTaskScheduler{
		abandonmentWindow: abandonmentWindow,
		finalizationDelay: defaultLearningFinalizationDelay,
		now:               time.Now,
		queue:             queue,
	}
}

func (enqueuer *LearningTaskScheduler) EnqueueFinalization(
	ctx context.Context,
	attempt StudentMissionAttemptDocument,
) error {
	job := FinalizationJob(attempt)

	return enqueuer.enqueue(ctx, job, enqueuer.now().Add(enqueuer.finalizationDelay))
}

func (enqueuer *LearningTaskScheduler) EnqueueWatchdog(
	ctx context.Context,
	attempt StudentMissionAttemptDocument,
) error {
	job := LearningJob{
		Kind:                   LearningJobWatchdog,
		StudentID:              attempt.StudentID,
		AttemptID:              attempt.ID,
		ExpectedLastActivityAt: attempt.LastActivityAt,
	}
	scheduleAt := attempt.LastActivityAt.Add(enqueuer.abandonmentWindow)
	if scheduleAt.Before(enqueuer.now()) {
		scheduleAt = enqueuer.now().Add(time.Second)
	}

	return enqueuer.enqueue(ctx, job, scheduleAt)
}

func (enqueuer *LearningTaskScheduler) enqueue(
	ctx context.Context,
	job LearningJob,
	scheduleAt time.Time,
) error {
	if enqueuer == nil || enqueuer.queue == nil {
		return errors.New("student learning task queue is not configured")
	}
	if err := job.Validate(); err != nil {
		return err
	}
	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("encode student learning task: %w", err)
	}

	return enqueuer.queue.Enqueue(ctx, taskqueue.Message{
		ID:         learningTaskID(job),
		Payload:    body,
		ScheduleAt: scheduleAt,
	})
}

func learningTaskID(job LearningJob) string {
	if job.Kind == LearningJobFinalize {
		digest := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%s\x00%d\x00%d\x00%s\x00%d", job.Kind, job.StudentID, job.AttemptID, job.ExpectedEvidenceRevision, job.ExpectedSnapshotRevision, job.ExpectedStatus, job.DispatchGeneration)))
		return "student-learning-" + hex.EncodeToString(digest[:])
	}
	activityVersion := int64(0)
	if job.Kind == LearningJobWatchdog {
		activityVersion = job.ExpectedLastActivityAt.UTC().UnixNano()
	}
	digest := sha256.Sum256([]byte(fmt.Sprintf(
		"%s\x00%s\x00%s\x00%d",
		job.Kind,
		job.StudentID,
		job.AttemptID,
		activityVersion,
	)))

	return "student-learning-" + job.Kind + "-" + hex.EncodeToString(digest[:])
}
