package students

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (repository *FirestoreRepository) GetStudentTutorEvent(ctx context.Context, studentID, attemptID, eventID string) (TutorEventDocument, bool, error) {
	// The application verifies attempt ownership and request provenance before
	// lookup; event subcollections remain inaccessible to browser clients.
	ref := repository.client.Collection(studentAttemptsCollectionName).Doc(attemptID).Collection("tutor_events").Doc(tutorTurnID(eventID))
	snapshot, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return TutorEventDocument{}, false, nil
	}
	if err != nil {
		return TutorEventDocument{}, false, fmt.Errorf("read tutor replay event: %w", err)
	}
	var event TutorEventDocument
	if err := snapshot.DataTo(&event); err != nil {
		return TutorEventDocument{}, false, fmt.Errorf("decode tutor replay event: %w", err)
	}
	if event.StudentID != studentID {
		return TutorEventDocument{}, false, ErrStudentMissionMismatch
	}
	return event, true, nil
}
