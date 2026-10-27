package attachments

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

var (
	// ErrRevisionPending tells the task handler that approval has not committed yet.
	// Cloud Tasks should retry this error.
	ErrRevisionPending = errors.New("mission published revision is not available yet")
	// ErrRevisionObsolete tells the task handler that the addressed immutable
	// revision cannot exist or does not match the job. Cloud Tasks should
	// acknowledge this error.
	ErrRevisionObsolete = errors.New("mission published revision is obsolete")
)

// Job contains only stable identifiers. Attachment URLs are always loaded from
// the published Firestore snapshot by the processor.
type Job struct {
	ClassID   string `json:"class_id"`
	MissionID string `json:"mission_id"`
	Revision  int64  `json:"revision"`
}

func (job Job) Validate() error {
	if err := validatePathID("class_id", job.ClassID, 40); err != nil {
		return err
	}
	if err := validatePathID("mission_id", job.MissionID, 120); err != nil {
		return err
	}
	if job.Revision <= 0 {
		return errors.New("revision must be greater than zero")
	}

	return nil
}

func validatePathID(field string, value string, maxLength int) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	if len(value) > maxLength {
		return fmt.Errorf("%s cannot be longer than %d characters", field, maxLength)
	}
	if value == "." || value == ".." || strings.ContainsAny(value, `/\\`) {
		return fmt.Errorf("%s contains an invalid path segment", field)
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return fmt.Errorf("%s contains an invalid control character", field)
		}
	}

	return nil
}

// PermanentError marks work that will not succeed on a later Cloud Tasks
// attempt, such as an invalid source URL or an attachment above the size limit.
type PermanentError struct {
	err error
}

func MarkPermanent(err error) error {
	if err == nil || IsPermanent(err) {
		return err
	}

	return &PermanentError{err: err}
}

func IsPermanent(err error) bool {
	var permanentError *PermanentError
	return errors.As(err, &permanentError)
}

func (err *PermanentError) Error() string {
	return err.err.Error()
}

func (err *PermanentError) Unwrap() error {
	return err.err
}
