package attachments

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const maximumStoredErrorLength = 500

type AttachmentSnapshot struct {
	UploadcareUUID string
	SourceURL      string
	FileName       string
	MIMEType       string
	SizeBytes      int64
}

type TransferPlan struct {
	Attachments []AttachmentSnapshot
}

// TransferRepository owns all Firestore concurrency checks. Prepare reads the
// immutable published snapshot identified by job.Revision, including historical
// revisions and revisions of soft-deleted missions, so approved files remain
// durable. Implementations return ErrRevisionPending while queue-before-publish
// is still committing and ErrRevisionObsolete when that revision cannot exist.
type TransferRepository interface {
	Prepare(context.Context, Job) (TransferPlan, error)
	MarkStored(context.Context, Job, AttachmentSnapshot, StoredObject) error
	MarkFailed(context.Context, Job, AttachmentSnapshot, string, bool) error
	Complete(context.Context, Job) error
}

type JobProcessor interface {
	Process(context.Context, Job) error
}

type Processor struct {
	repository TransferRepository
	copier     Copier
}

func NewProcessor(repository TransferRepository, copier Copier) (*Processor, error) {
	if repository == nil {
		return nil, errors.New("mission attachment transfer repository is required")
	}
	if copier == nil {
		return nil, errors.New("mission attachment storage copier is required")
	}

	return &Processor{
		repository: repository,
		copier:     copier,
	}, nil
}

func (processor *Processor) Process(ctx context.Context, job Job) error {
	if processor == nil || processor.repository == nil || processor.copier == nil {
		return errors.New("mission attachment processor is not configured")
	}
	if err := job.Validate(); err != nil {
		return MarkPermanent(err)
	}

	plan, err := processor.repository.Prepare(ctx, job)
	if errors.Is(err, ErrRevisionObsolete) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, attachment := range plan.Attachments {
		storedObject, copyErr := processor.copier.Copy(ctx, CopyInput{
			ClassID:           job.ClassID,
			MissionID:         job.MissionID,
			Revision:          job.Revision,
			UploadcareUUID:    attachment.UploadcareUUID,
			SourceURL:         attachment.SourceURL,
			FileName:          attachment.FileName,
			MIMEType:          attachment.MIMEType,
			ExpectedSizeBytes: attachment.SizeBytes,
		})
		if copyErr == nil {
			if err := processor.repository.MarkStored(ctx, job, attachment, storedObject); err != nil {
				if errors.Is(err, ErrRevisionObsolete) {
					return nil
				}
				return fmt.Errorf("mark mission attachment stored: %w", err)
			}
			continue
		}

		retryable := !IsPermanent(copyErr)
		markErr := processor.repository.MarkFailed(
			ctx,
			job,
			attachment,
			safeStoredError(copyErr),
			retryable,
		)
		if errors.Is(markErr, ErrRevisionObsolete) {
			return nil
		}
		if markErr != nil {
			// Persisting the outcome is retryable even when the copy error itself is
			// permanent. Do not wrap copyErr here, or the HTTP handler would
			// acknowledge the task before the failure state reaches Firestore.
			return fmt.Errorf(
				"mark mission attachment failure after copy error %v: %w",
				copyErr,
				markErr,
			)
		}

		if retryable {
			return fmt.Errorf("copy mission attachment: %w", copyErr)
		}
	}

	if err := processor.repository.Complete(ctx, job); err != nil {
		if errors.Is(err, ErrRevisionObsolete) {
			return nil
		}
		return fmt.Errorf("complete mission attachment transfer: %w", err)
	}

	return nil
}

func safeStoredError(err error) string {
	message := strings.Join(strings.Fields(err.Error()), " ")
	if len(message) > maximumStoredErrorLength {
		message = message[:maximumStoredErrorLength]
	}

	return message
}
