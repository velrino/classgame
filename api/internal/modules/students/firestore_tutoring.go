package students

import (
	"context"
	"crypto/sha256"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"classgame-api/internal/modules/generation"
)

func (repository *FirestoreRepository) AppendStudentTutorTurn(
	ctx context.Context,
	input SaveStudentTutorTurnInput,
	tutor generation.TutorTurnBlueprint,
) (StudentTutorTurn, error) {
	userRef := repository.client.Collection(usersCollectionName).Doc(input.Identity.ProviderUserID)
	var result StudentTutorTurn

	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		userSnapshot, err := tx.Get(userRef)
		if status.Code(err) == codes.NotFound {
			return ErrStudentNotFound
		}
		if err != nil {
			return fmt.Errorf("read student user: %w", err)
		}
		var user UserDocument
		if err := userSnapshot.DataTo(&user); err != nil {
			return fmt.Errorf("decode student user: %w", err)
		}
		if user.ID == "" {
			user.ID = userRef.ID
		}
		if user.Activity.CurrentGame.MissionID != input.MissionID || user.Student.MissionID != input.MissionID {
			return ErrStudentMissionMismatch
		}

		attemptID := user.Activity.CurrentGame.AttemptID
		if attemptID == "" {
			attemptID = learningAttemptID(user.ID, input.MissionID, user.Activity.CurrentGame.PublishedRevision)
		}
		if attemptID != input.ExpectedAttemptID || user.Activity.CurrentGame.PublishedRevision != input.ExpectedPublishedRevision {
			return ErrStudentMissionMismatch
		}
		attemptRef := repository.client.Collection(studentAttemptsCollectionName).Doc(attemptID)
		attemptSnapshot, err := tx.Get(attemptRef)
		if status.Code(err) == codes.NotFound {
			return ErrStudentAttemptNotFound
		}
		if err != nil {
			return fmt.Errorf("read learning attempt: %w", err)
		}
		var attempt StudentMissionAttemptDocument
		if err := attemptSnapshot.DataTo(&attempt); err != nil {
			return fmt.Errorf("decode learning attempt: %w", err)
		}
		if attempt.StudentID != user.ID {
			return ErrStudentMissionMismatch
		}

		if attempt.ID == "" {
			attempt.ID = attemptRef.ID
		}
		if !sameStudentClass(user.Student, attempt) {
			return ErrStudentMissionMismatch
		}
		if err := ValidateTutorTurnScope(input, user.Activity.CurrentGame, attempt); err != nil {
			return err
		}

		turnID := tutorTurnID(input.EventID)
		for _, turn := range attempt.TutorTurns {
			if turn.ID == turnID {
				result = turn
				return nil
			}
		}
		result = StudentTutorTurn{
			ID:                turnID,
			Sequence:          len(attempt.TutorTurns) + 1,
			Type:              input.Type,
			StageID:           input.StageID,
			QuestionID:        input.QuestionID,
			Prompt:            input.Prompt,
			StudentResponse:   input.StudentResponse,
			ExpectedAnswer:    input.ExpectedAnswer,
			Outcome:           input.Outcome,
			ExplicitReasoning: input.ExplicitReasoning,
			Feedback:          input.Feedback,
			Tutor:             tutor,
			CreatedAt:         input.Timestamp,
		}
		turns := append(attempt.TutorTurns, result)
		if len(turns) > maxTutorTurnsPerAttempt {
			turns = turns[len(turns)-maxTutorTurnsPerAttempt:]
		}

		if err := tx.Set(attemptRef, map[string]any{
			"tutor_turns":      turns,
			"last_activity_at": input.Timestamp,
			"updated_at":       input.Timestamp,
		}, firestore.MergeAll); err != nil {
			return fmt.Errorf("append tutor turn: %w", err)
		}
		return tx.Set(userRef, map[string]any{
			"activity": map[string]any{
				"last_seen_at": input.Timestamp,
				"current_game": map[string]any{
					"last_heartbeat_at": input.Timestamp,
				},
			},
			"updated_at": input.Timestamp,
		}, firestore.MergeAll)
	})
	if err != nil {
		return StudentTutorTurn{}, fmt.Errorf("save student tutor turn: %w", err)
	}

	return result, nil
}

func tutorTurnID(eventID string) string {
	digest := sha256.Sum256([]byte(eventID))

	return fmt.Sprintf("turn-%x", digest[:12])
}
