package attachments

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"classgame-api/internal/taskqueue"
)

const defaultScheduleDelay = 2 * time.Second

type TaskQueue interface {
	Enqueue(context.Context, taskqueue.Message) error
}

type Enqueuer interface {
	Enqueue(context.Context, Job) error
}

type TaskScheduler struct {
	queue         TaskQueue
	scheduleDelay time.Duration
	now           func() time.Time
}

func NewTaskScheduler(queue TaskQueue, scheduleDelay time.Duration) *TaskScheduler {
	if scheduleDelay <= 0 {
		scheduleDelay = defaultScheduleDelay
	}

	return &TaskScheduler{
		queue:         queue,
		scheduleDelay: scheduleDelay,
		now:           time.Now,
	}
}

func (enqueuer *TaskScheduler) Enqueue(ctx context.Context, job Job) error {
	if enqueuer == nil || enqueuer.queue == nil {
		return fmt.Errorf("mission attachment task queue is not configured")
	}
	if err := job.Validate(); err != nil {
		return err
	}

	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("encode mission attachment task: %w", err)
	}

	return enqueuer.queue.Enqueue(ctx, taskqueue.Message{
		ID:         TaskID(job),
		Payload:    body,
		ScheduleAt: enqueuer.now().Add(enqueuer.scheduleDelay),
	})
}

func TaskID(job Job) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d", job.ClassID, job.MissionID, job.Revision)))
	return "mission-attachments-" + hex.EncodeToString(digest[:])
}
