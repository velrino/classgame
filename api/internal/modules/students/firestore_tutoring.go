package students

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

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
		eventRef := attemptRef.Collection("tutor_events").Doc(turnID)
		eventSnapshot, eventErr := tx.Get(eventRef)

		var reserved TutorEventDocument
		if eventErr == nil {
			if err := eventSnapshot.DataTo(&reserved); err != nil {
				return fmt.Errorf("decode tutor replay event: %w", err)
			}
			if reserved.StudentID != user.ID {
				return ErrStudentMissionMismatch
			}
			turn, complete, err := TutorReplayDecision(reserved, input, time.Now())
			if complete {
				result = turn
				return err
			}
			if err != nil && err != ErrTutorEventPending {
				return err
			}
			if input.Type != "feedback" && !MatchesTutorReservation(reserved, input, time.Now()) {
				return ErrTutorEventPending
			}
		} else if status.Code(eventErr) != codes.NotFound {
			return fmt.Errorf("read tutor replay event: %w", eventErr)
		}
		if input.Type != "feedback" && (eventErr != nil || !MatchesTutorReservation(reserved, input, time.Now())) {
			return ErrTutorEventPending
		}

		for _, turn := range attempt.TutorTurns {
			if turn.ID != turnID {
				continue
			}
			if fingerprintRecordedTurn(turn, input) != TutorEventFingerprint(input) {
				return ErrTutorEventConflict
			}
			result = turn
			if attempt.Status == "completed" {
				return nil
			}
			return tx.Create(eventRef, TutorEventDocument{State: "complete", StudentID: user.ID, EventID: input.EventID, Fingerprint: TutorEventFingerprint(input), Turn: turn, CreatedAt: input.Timestamp})
		}
		if attempt.Status == "completed" {
			return ErrStudentAttemptClosed
		}
		attempt = EffectiveTutorEvidence(attempt)
		invalidateEvidence, err := repository.invalidateClassEvidence(tx, attempt, input.Timestamp)
		if err != nil {
			return err
		}

		attempt.TutorEventSequence++
		attempt.EvidenceRevision++

		result = StudentTutorTurn{
			ID:                turnID,
			Sequence:          int(attempt.TutorEventSequence),
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
		if input.Type == "answer" && input.ExpectedAnswer != "" && normalizedStudentAnswer(input.StudentResponse) == normalizedStudentAnswer(input.ExpectedAnswer) {
			attempt.AcceptedAnswers = mergeAcceptedAnswer(attempt.AcceptedAnswers, AcceptedAnswerFact{QuestionID: input.QuestionID, StageID: input.StageID, StudentResponse: input.StudentResponse, PublishedRevision: attempt.PublishedRevision, TurnID: result.ID, AcceptedAt: input.Timestamp})
		}
		if err := invalidateEvidence(); err != nil {
			return fmt.Errorf("invalidate class learning evidence: %w", err)
		}
		if err := tx.Set(eventRef, TutorEventDocument{State: "complete", StudentID: user.ID, EventID: input.EventID, Fingerprint: TutorEventFingerprint(input), Turn: result, CreatedAt: input.Timestamp}); err != nil {
			return fmt.Errorf("write durable tutor replay: %w", err)
		}

		turns := append(attempt.TutorTurns, result)
		if len(turns) > maxTutorTurnsPerAttempt {
			turns = turns[len(turns)-maxTutorTurnsPerAttempt:]
		}

		traceState := "live"
		if attempt.Status == "abandoned" {
			traceState = "pending"
		}
		if err := tx.Set(attemptRef, map[string]any{
			"evidence_schema_version": attempt.EvidenceSchemaVersion, "accepted_answers": attempt.AcceptedAnswers, "generated_turn_count": attempt.GeneratedTurnCount, "tutor_event_sequence": attempt.TutorEventSequence, "legacy_history_incomplete": attempt.LegacyHistoryIncomplete, "evidence_revision": attempt.EvidenceRevision, "trace": map[string]any{"finalized": false}, "trace_state": traceState, "finalization_token": "", "finalization_lease_until": time.Time{},
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
