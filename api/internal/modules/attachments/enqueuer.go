package attachments

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"classgame-api/internal/platform/cloudtasks"
)

const defaultScheduleDelay = 2 * time.Second

type TaskQueue interface {
	Enqueue(context.Context, cloudtasks.Task) error
}

type Enqueuer interface {
	Enqueue(context.Context, Job) error
}

type CloudTaskEnqueuer struct {
	queue         TaskQueue
	scheduleDelay time.Duration
	now           func() time.Time
}

func NewCloudTaskEnqueuer(queue TaskQueue, scheduleDelay time.Duration) *CloudTaskEnqueuer {
	if scheduleDelay <= 0 {
		scheduleDelay = defaultScheduleDelay
	}

	return &CloudTaskEnqueuer{
		queue:         queue,
		scheduleDelay: scheduleDelay,
		now:           time.Now,
	}
}

func (enqueuer *CloudTaskEnqueuer) Enqueue(ctx context.Context, job Job) error {
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

	return enqueuer.queue.Enqueue(ctx, cloudtasks.Task{
		ID:           TaskID(job),
		Body:         body,
		ScheduleTime: enqueuer.now().Add(enqueuer.scheduleDelay),
	})
}

func TaskID(job Job) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d", job.ClassID, job.MissionID, job.Revision)))
	return "mission-attachments-" + hex.EncodeToString(digest[:])
}
