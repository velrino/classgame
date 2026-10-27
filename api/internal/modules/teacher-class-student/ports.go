package teacherclassstudent

import (
	"context"
)

// NotebookStore is the persistence port owned by this workflow.
type NotebookStore interface {
	UpsertMissionSummary(
		ctx context.Context,
		input UpsertStudentMissionSummaryInput,
	) (ClassStudentDocument, error)
}
