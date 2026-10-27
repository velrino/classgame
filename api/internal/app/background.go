package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	"classgame-api/internal/modules/attachments"
	"classgame-api/internal/modules/students"
	"classgame-api/internal/platform/redisqueue"
	"classgame-api/internal/taskqueue"
)

type LearningProcessor interface {
	ProcessLearningJob(context.Context, students.LearningJob) error
}

// BackgroundHandlers adapts transport payloads; domain processors keep their rules.
func BackgroundHandlers(copier attachments.JobProcessor, learning LearningProcessor) map[string]redisqueue.Handler {
	return map[string]redisqueue.Handler{
		redisqueue.Attachments: func(ctx context.Context, message taskqueue.Message) error {
			var job attachments.Job
			if decodeJob(message.Payload, &job) != nil || job.Validate() != nil {
				return redisqueue.Terminal(errors.New("invalid attachment job"))
			}
			err := copier.Process(ctx, job)
			switch {
			case err == nil, errors.Is(err, attachments.ErrRevisionObsolete):
				return nil
			case attachments.IsPermanent(err):
				return redisqueue.Terminal(errors.New("attachment processing permanently failed"))
			case errors.Is(err, attachments.ErrRevisionPending):
				return errors.New("published revision is pending")
			default:
				return errors.New("attachment processing temporarily failed")
			}
		},
		redisqueue.Learning: func(ctx context.Context, message taskqueue.Message) error {
			var job students.LearningJob
			if decodeJob(message.Payload, &job) != nil || job.Validate() != nil {
				return redisqueue.Terminal(errors.New("invalid learning job"))
			}
			err := learning.ProcessLearningJob(ctx, job)
			if errors.Is(err, students.ErrStudentNotFound) || errors.Is(err, students.ErrStudentAttemptNotFound) {
				return redisqueue.Terminal(errors.New("learning subject no longer exists"))
			}
			if err != nil {
				return errors.New("learning processing temporarily failed")
			}
			return nil
		},
	}
}

func decodeJob(body []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("job contains trailing JSON")
	}
	return nil
}
