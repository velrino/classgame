package users

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"classgame-api/internal/modules/orchestrator"
	"classgame-api/internal/modules/tenant"
)

const (
	usersCollectionName              = "users"
	studentAttemptsCollectionName    = "student_mission_attempts"
	studentCompletionsCollectionName = "student_mission_completions"
	studentCompletionsSubcollection  = "missions"
	classLearningCollectionName      = "class_learning_summaries"
	maxTutorTurnsPerAttempt          = 80
)

type Repository struct {
	client *firestore.Client
}

func NewRepository(client *firestore.Client) *Repository {
	return &Repository{
		client: client,
	}
}

func (repository *Repository) UpsertStudent(
	ctx context.Context,
	input UpsertStudentInput,
) (UserDocument, error) {
	ref := repository.client.Collection(usersCollectionName).Doc(input.Identity.ProviderUserID)

	var user UserDocument
	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		now := input.Timestamp
		createdAt := now
		firstSeenAt := now
		loginCount := int64(0)
		if input.IncrementLogin {
			loginCount = 1
		}
		currentGame := StudentGameSession{}
		learningMemory := StudentLearningMemory{}

		snapshot, err := tx.Get(ref)
		switch {
		case err == nil:
			var existing UserDocument
			if err := snapshot.DataTo(&existing); err != nil {
				return fmt.Errorf("decode existing user: %w", err)
			}

			if !existing.CreatedAt.IsZero() {
				createdAt = existing.CreatedAt
			}
			if !existing.Activity.FirstSeenAt.IsZero() {
				firstSeenAt = existing.Activity.FirstSeenAt
			}
			loginCount = existing.Activity.LoginCount
			if input.IncrementLogin {
				loginCount++
			}
			if existing.Student.ClassID == input.ClassID {
				currentGame = existing.Activity.CurrentGame
				learningMemory = existing.Student.LearningMemory
			}
		case status.Code(err) != codes.NotFound:
			return fmt.Errorf("read user: %w", err)
		}
		if loginCount <= 0 {
			loginCount = 1
		}

		if input.MissionID == "" {
			currentGame = StudentGameSession{}
		}
		revisionChanged := input.MissionID != "" && currentGame.PublishedRevision > 0 &&
			currentGame.PublishedRevision != input.MissionPublishedRevision
		startNewRevision := input.MissionID != "" && currentGame.MissionID == input.MissionID &&
			revisionChanged && currentGame.Status == "completed"
		if input.MissionID == "" {
			// Joining a class with no previously selected mission keeps the student in the lobby.
		} else if currentGame.MissionID == input.MissionID && !startNewRevision {
			if currentGame.AttemptID == "" {
				currentGame.AttemptID = learningAttemptID(ref.ID, input.MissionID, input.MissionPublishedRevision)
			}
			if currentGame.PublishedRevision <= 0 {
				currentGame.PublishedRevision = input.MissionPublishedRevision
			}
			currentGame.LastHeartbeatAt = now
			if currentGame.Status == "" || currentGame.Status == "paused" || currentGame.Status == "abandoned" {
				currentGame.Status = "exploring"
			}
		} else {
			currentGame = StudentGameSession{
				AttemptID:         learningAttemptID(ref.ID, input.MissionID, input.MissionPublishedRevision),
				MissionID:         input.MissionID,
				PublishedRevision: input.MissionPublishedRevision,
				Subject:           input.MissionSubject,
				Status:            "exploring",
				StartedAt:         now,
				LastHeartbeatAt:   now,
			}
		}

		user = buildStudentUserDocument(buildStudentUserDocumentInput{
			ID:             ref.ID,
			Input:          input,
			CreatedAt:      createdAt,
			FirstSeenAt:    firstSeenAt,
			LoginCount:     loginCount,
			CurrentGame:    currentGame,
			LearningMemory: learningMemory,
		})

		if err := tx.Set(ref, user.firestoreMap(), firestore.MergeAll); err != nil {
			return fmt.Errorf("write user: %w", err)
		}

		return nil
	})
	if err != nil {
		return UserDocument{}, fmt.Errorf("upsert student user: %w", err)
	}

	return user, nil
}

func (repository *Repository) UpdateStudentPresence(
	ctx context.Context,
	input UpdateStudentPresenceInput,
) (UserDocument, error) {
	ref := repository.client.Collection(usersCollectionName).Doc(input.Identity.ProviderUserID)
	var user UserDocument

	requestedStatus := input.Status
	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		input.Status = requestedStatus
		snapshot, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			return ErrStudentNotFound
		}
		if err != nil {
			return fmt.Errorf("read student user: %w", err)
		}
		if err := snapshot.DataTo(&user); err != nil {
			return fmt.Errorf("decode student user: %w", err)
		}
		if user.ID == "" {
			user.ID = ref.ID
		}
		if user.Activity.CurrentGame.MissionID != input.MissionID || user.Student.MissionID != input.MissionID {
			return ErrStudentMissionMismatch
		}

		startedAt := input.Timestamp
		publishedRevision := int64(0)
		publishedRevision = user.Activity.CurrentGame.PublishedRevision
		attemptID := user.Activity.CurrentGame.AttemptID
		if attemptID == "" {
			attemptID = learningAttemptID(user.ID, input.MissionID, publishedRevision)
		}
		if !user.Activity.CurrentGame.StartedAt.IsZero() {
			startedAt = user.Activity.CurrentGame.StartedAt
		}

		// Presence cannot award mission completion independently of verified learning.
		if input.Status == "completed" {
			recorded, readErr := tx.Get(repository.client.Collection(studentAttemptsCollectionName).Doc(attemptID))
			verified := false
			if readErr == nil {
				var attempt StudentMissionAttemptDocument
				if err := recorded.DataTo(&attempt); err != nil {
					return err
				}
				verified = attempt.StudentID == user.ID && attempt.Status == "completed"
			}
			if readErr != nil && status.Code(readErr) != codes.NotFound {
				return fmt.Errorf("read presence completion: %w", readErr)
			}
			if !verified {
				input.Status = "learning"
			}
		}

		user.Activity.LastSeenAt = input.Timestamp
		user.Activity.CurrentGame = StudentGameSession{
			AttemptID:         attemptID,
			MissionID:         input.MissionID,
			PublishedRevision: publishedRevision,
			Subject:           input.Subject,
			StageID:           input.StageID,
			StageName:         input.StageName,
			Status:            input.Status,
			StartedAt:         startedAt,
			LastHeartbeatAt:   input.Timestamp,
		}
		user.UpdatedAt = input.Timestamp

		return tx.Set(ref, map[string]any{
			"activity": map[string]any{
				"last_seen_at": user.Activity.LastSeenAt,
				"current_game": user.Activity.CurrentGame.firestoreMap(),
			},
			"updated_at": user.UpdatedAt,
		}, firestore.MergeAll)
	})
	if err != nil {
		return UserDocument{}, fmt.Errorf("update student presence: %w", err)
	}

	return user, nil
}

func (repository *Repository) SaveStudentLearningSnapshot(
	ctx context.Context,
	input SaveStudentLearningSnapshotInput,
) (StudentMissionAttemptDocument, UserDocument, bool, error) {
	userRef := repository.client.Collection(usersCollectionName).Doc(input.Identity.ProviderUserID)
	var attempt StudentMissionAttemptDocument
	var user UserDocument
	completionStarted := false

	requestedStatus, submittedSnapshot := input.Status, input.Snapshot
	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		input.Status, input.Snapshot = requestedStatus, submittedSnapshot
		completionStarted = false
		userSnapshot, err := tx.Get(userRef)
		if status.Code(err) == codes.NotFound {
			return ErrStudentNotFound
		}
		if err != nil {
			return fmt.Errorf("read student user: %w", err)
		}
		if err := userSnapshot.DataTo(&user); err != nil {
			return fmt.Errorf("decode student user: %w", err)
		}
		if user.ID == "" {
			user.ID = userRef.ID
		}
		if user.Activity.CurrentGame.MissionID != input.MissionID || user.Student.MissionID != input.MissionID {
			return ErrStudentMissionMismatch
		}
		if user.Activity.CurrentGame.PublishedRevision != input.PublishedRevision {
			return ErrStudentMissionMismatch
		}

		attemptID := user.Activity.CurrentGame.AttemptID
		if attemptID == "" {
			attemptID = learningAttemptID(user.ID, input.MissionID, user.Activity.CurrentGame.PublishedRevision)
		}
		attemptRef := repository.client.Collection(studentAttemptsCollectionName).Doc(attemptID)
		createdAt := input.Timestamp
		startedAt := user.Activity.CurrentGame.StartedAt
		if startedAt.IsZero() {
			startedAt = input.Timestamp
		}
		completedAt := time.Time{}
		abandonedAt := time.Time{}
		trace := input.Trace
		traceState := "live"
		finalizeRequestedAt := time.Time{}
		tutorTurns := []StudentTutorTurn{}

		existingSnapshot, err := tx.Get(attemptRef)
		switch {
		case err == nil:
			var existing StudentMissionAttemptDocument
			if err := existingSnapshot.DataTo(&existing); err != nil {
				return fmt.Errorf("decode learning attempt: %w", err)
			}
			if existing.ID == "" {
				existing.ID = attemptID
			}
			if !existing.CreatedAt.IsZero() {
				createdAt = existing.CreatedAt
			}
			if !existing.StartedAt.IsZero() {
				startedAt = existing.StartedAt
			}
			completedAt = existing.CompletedAt
			abandonedAt = existing.AbandonedAt
			traceState = existing.TraceState
			finalizeRequestedAt = existing.FinalizeRequestedAt
			tutorTurns = existing.TutorTurns
			if existing.Status == "completed" {
				attempt = existing
				return tx.Set(
					repository.studentMissionCompletionRef(user.ID, existing.ID),
					studentMissionCompletionMap(existing),
					firestore.MergeAll,
				)
			}
			if existing.Snapshot.Revision >= input.Snapshot.Revision {
				if input.Status != "completed" {
					attempt = existing
					return nil
				}
				// A retried final checkpoint may follow a newly persisted answer turn.
				// Re-evaluate assessment without replacing a newer UI checkpoint.
				input.Snapshot = existing.Snapshot
			}
			input.Snapshot = mergeLearningSnapshots(existing.Snapshot, input.Snapshot)
			resumingAbandonedAttempt := existing.Status == "abandoned" && input.Status != "completed"
			if resumingAbandonedAttempt {
				traceState = "live"
				finalizeRequestedAt = time.Time{}
			} else if existing.Trace.Finalized && existing.Trace.EventCount == len(input.Snapshot.Events) {
				trace = existing.Trace
			}
		case status.Code(err) != codes.NotFound:
			return fmt.Errorf("read learning attempt: %w", err)
		}

		input.Snapshot, input.Status = authoritativeLearningSnapshot(input.Mission, input.Snapshot, tutorTurns, input.Status)
		liveInput := toLearningTraceInput(input.Mission, input.Snapshot, "in_progress", StudentLearningMemory{})
		trace = orchestrator.BuildObservedLearningTrace(liveInput, "")
		trace.Finalized = false
		if input.Status == "completed" && completedAt.IsZero() {
			completedAt = input.Timestamp
			completionStarted = true
		}
		if input.Status == "completed" && !trace.Finalized {
			traceState = "pending"
			finalizeRequestedAt = input.Timestamp
		}
		if input.Status != "abandoned" {
			abandonedAt = time.Time{}
		}
		scope := tenant.NormalizeScope(tenant.Scope{
			OrganizationID: user.Student.OrganizationID,
			DistrictID:     user.Student.DistrictID,
			SchoolID:       user.Student.SchoolID,
		})
		attempt = StudentMissionAttemptDocument{
			ID:                  attemptID,
			OrganizationID:      scope.OrganizationID,
			DistrictID:          scope.DistrictID,
			SchoolID:            scope.SchoolID,
			StudentID:           user.ID,
			StudentName:         user.Student.StudentName,
			ClassID:             user.Student.ClassID,
			MissionID:           input.MissionID,
			MissionCode:         user.Student.MissionCode,
			PublishedRevision:   user.Activity.CurrentGame.PublishedRevision,
			Subject:             user.Activity.CurrentGame.Subject,
			Status:              input.Status,
			Snapshot:            input.Snapshot,
			TutorTurns:          tutorTurns,
			Trace:               trace,
			TraceState:          traceState,
			FinalizeRequestedAt: finalizeRequestedAt,
			StartedAt:           startedAt,
			LastActivityAt:      input.Timestamp,
			CompletedAt:         completedAt,
			AbandonedAt:         abandonedAt,
			CreatedAt:           createdAt,
			UpdatedAt:           input.Timestamp,
		}

		user.Activity.LastSeenAt = input.Timestamp
		user.Activity.CurrentGame.AttemptID = attemptID
		user.Activity.CurrentGame.Status = input.Status
		user.Activity.CurrentGame.LastHeartbeatAt = input.Timestamp
		user.UpdatedAt = input.Timestamp

		if err := tx.Set(attemptRef, learningAttemptFirestoreMap(attempt), firestore.MergeAll); err != nil {
			return fmt.Errorf("write learning attempt: %w", err)
		}
		if attempt.Status == "completed" {
			if err := tx.Set(
				repository.studentMissionCompletionRef(user.ID, attempt.ID),
				studentMissionCompletionMap(attempt),
				firestore.MergeAll,
			); err != nil {
				return fmt.Errorf("write student mission completion: %w", err)
			}
		}
		return tx.Set(userRef, map[string]any{
			"activity": map[string]any{
				"last_seen_at": user.Activity.LastSeenAt,
				"current_game": user.Activity.CurrentGame.firestoreMap(),
			},
			"updated_at": user.UpdatedAt,
		}, firestore.MergeAll)
	})
	if err != nil {
		return StudentMissionAttemptDocument{}, UserDocument{}, false, fmt.Errorf("save student learning snapshot: %w", err)
	}

	return attempt, user, completionStarted, nil
}

func (repository *Repository) studentMissionCompletionRef(
	studentID string,
	attemptID string,
) *firestore.DocumentRef {
	return repository.client.Collection(studentCompletionsCollectionName).
		Doc(studentID).
		Collection(studentCompletionsSubcollection).
		Doc(attemptID)
}

func studentMissionCompletionMap(attempt StudentMissionAttemptDocument) map[string]any {
	return map[string]any{
		"student_id":         attempt.StudentID,
		"class_id":           attempt.ClassID,
		"mission_id":         attempt.MissionID,
		"published_revision": attempt.PublishedRevision,
		"status":             "completed",
		"completed_at":       attempt.CompletedAt,
		"updated_at":         attempt.UpdatedAt,
	}
}

func (repository *Repository) GetCurrentStudentLearningAttempt(
	ctx context.Context,
	identityProviderUserID string,
	missionID string,
) (StudentMissionAttemptDocument, UserDocument, bool, error) {
	userRef := repository.client.Collection(usersCollectionName).Doc(identityProviderUserID)
	userSnapshot, err := userRef.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return StudentMissionAttemptDocument{}, UserDocument{}, false, ErrStudentNotFound
	}
	if err != nil {
		return StudentMissionAttemptDocument{}, UserDocument{}, false, fmt.Errorf("read student user: %w", err)
	}

	var user UserDocument
	if err := userSnapshot.DataTo(&user); err != nil {
		return StudentMissionAttemptDocument{}, UserDocument{}, false, fmt.Errorf("decode student user: %w", err)
	}
	if user.ID == "" {
		user.ID = userRef.ID
	}
	if user.Activity.CurrentGame.MissionID != missionID || user.Student.MissionID != missionID {
		return StudentMissionAttemptDocument{}, user, false, ErrStudentMissionMismatch
	}

	attemptID := user.Activity.CurrentGame.AttemptID
	if attemptID == "" {
		attemptID = learningAttemptID(user.ID, missionID, user.Activity.CurrentGame.PublishedRevision)
	}
	attemptSnapshot, err := repository.client.Collection(studentAttemptsCollectionName).Doc(attemptID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return StudentMissionAttemptDocument{}, user, false, nil
	}
	if err != nil {
		return StudentMissionAttemptDocument{}, user, false, fmt.Errorf("read learning attempt: %w", err)
	}

	var attempt StudentMissionAttemptDocument
	if err := attemptSnapshot.DataTo(&attempt); err != nil {
		return StudentMissionAttemptDocument{}, user, false, fmt.Errorf("decode learning attempt: %w", err)
	}
	if attempt.ID == "" {
		attempt.ID = attemptSnapshot.Ref.ID
	}
	if attempt.StudentID != user.ID {
		return StudentMissionAttemptDocument{}, user, false, ErrStudentMissionMismatch
	}

	return attempt, user, true, nil
}

func (repository *Repository) AppendStudentTutorTurn(
	ctx context.Context,
	input SaveStudentTutorTurnInput,
	tutor orchestrator.TutorTurnBlueprint,
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

func (repository *Repository) GetStudentLearningAttempt(
	ctx context.Context,
	studentID string,
	attemptID string,
) (StudentMissionAttemptDocument, UserDocument, bool, error) {
	attemptSnapshot, err := repository.client.Collection(studentAttemptsCollectionName).Doc(attemptID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return StudentMissionAttemptDocument{}, UserDocument{}, false, nil
	}
	if err != nil {
		return StudentMissionAttemptDocument{}, UserDocument{}, false, fmt.Errorf("read learning attempt: %w", err)
	}
	var attempt StudentMissionAttemptDocument
	if err := attemptSnapshot.DataTo(&attempt); err != nil {
		return StudentMissionAttemptDocument{}, UserDocument{}, false, fmt.Errorf("decode learning attempt: %w", err)
	}
	if attempt.ID == "" {
		attempt.ID = attemptSnapshot.Ref.ID
	}
	if attempt.StudentID != studentID {
		return StudentMissionAttemptDocument{}, UserDocument{}, false, ErrStudentMissionMismatch
	}

	userSnapshot, err := repository.client.Collection(usersCollectionName).Doc(studentID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return StudentMissionAttemptDocument{}, UserDocument{}, false, ErrStudentNotFound
	}
	if err != nil {
		return StudentMissionAttemptDocument{}, UserDocument{}, false, fmt.Errorf("read student user: %w", err)
	}
	var user UserDocument
	if err := userSnapshot.DataTo(&user); err != nil {
		return StudentMissionAttemptDocument{}, UserDocument{}, false, fmt.Errorf("decode student user: %w", err)
	}
	if user.ID == "" {
		user.ID = userSnapshot.Ref.ID
	}

	return attempt, user, true, nil
}

func (repository *Repository) MarkStudentAttemptAbandoned(
	ctx context.Context,
	studentID string,
	attemptID string,
	expectedLastActivity time.Time,
	timestamp time.Time,
) (StudentMissionAttemptDocument, bool, error) {
	userRef := repository.client.Collection(usersCollectionName).Doc(studentID)
	attemptRef := repository.client.Collection(studentAttemptsCollectionName).Doc(attemptID)
	var attempt StudentMissionAttemptDocument
	marked := false
	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snapshot, err := tx.Get(attemptRef)
		if status.Code(err) == codes.NotFound {
			return ErrStudentAttemptNotFound
		}
		if err != nil {
			return fmt.Errorf("read learning attempt: %w", err)
		}
		if err := snapshot.DataTo(&attempt); err != nil {
			return fmt.Errorf("decode learning attempt: %w", err)
		}
		if attempt.StudentID != studentID {
			return ErrStudentMissionMismatch
		}
		if attempt.Status == "completed" || attempt.Status == "abandoned" || attempt.Trace.Finalized {
			return nil
		}
		if attempt.LastActivityAt.After(expectedLastActivity) {
			return nil
		}

		attempt.Status = "abandoned"
		attempt.AbandonedAt = timestamp
		attempt.TraceState = "pending"
		attempt.FinalizeRequestedAt = timestamp
		attempt.UpdatedAt = timestamp
		marked = true
		if err := tx.Set(attemptRef, map[string]any{
			"status":                attempt.Status,
			"abandoned_at":          attempt.AbandonedAt,
			"trace_state":           attempt.TraceState,
			"finalize_requested_at": attempt.FinalizeRequestedAt,
			"updated_at":            attempt.UpdatedAt,
		}, firestore.MergeAll); err != nil {
			return fmt.Errorf("mark learning attempt abandoned: %w", err)
		}
		return tx.Set(userRef, map[string]any{
			"activity": map[string]any{
				"current_game": map[string]any{
					"status": "abandoned",
				},
			},
			"updated_at": timestamp,
		}, firestore.MergeAll)
	})
	if err != nil {
		return StudentMissionAttemptDocument{}, false, fmt.Errorf("abandon student learning attempt: %w", err)
	}

	return attempt, marked, nil
}

func (repository *Repository) SetStudentAttemptTraceState(
	ctx context.Context,
	attemptID string,
	state string,
	timestamp time.Time,
) error {
	_, err := repository.client.Collection(studentAttemptsCollectionName).Doc(attemptID).Set(ctx, map[string]any{
		"trace_state": state,
		"updated_at":  timestamp,
	}, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("set learning trace state: %w", err)
	}

	return nil
}

func (repository *Repository) ListLatestFinalizedAttemptsByClassID(
	ctx context.Context,
	classID string,
	limit int,
) ([]StudentMissionAttemptDocument, error) {
	if limit <= 0 {
		limit = 100
	}
	iter := repository.client.Collection(studentAttemptsCollectionName).
		Where("class_id", "==", classID).
		Documents(ctx)
	defer iter.Stop()

	latest := make(map[string]StudentMissionAttemptDocument)
	for {
		snapshot, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterate class learning attempts: %w", err)
		}
		var attempt StudentMissionAttemptDocument
		if err := snapshot.DataTo(&attempt); err != nil {
			return nil, fmt.Errorf("decode class learning attempt %s: %w", snapshot.Ref.ID, err)
		}
		if !attempt.Trace.Finalized {
			continue
		}
		current, exists := latest[attempt.StudentID]
		if !exists || studentAttemptEvidenceTime(attempt).After(studentAttemptEvidenceTime(current)) {
			latest[attempt.StudentID] = attempt
		}
	}
	result := make([]StudentMissionAttemptDocument, 0, len(latest))
	for _, attempt := range latest {
		result = append(result, attempt)
	}
	sort.SliceStable(result, func(left int, right int) bool {
		return studentAttemptEvidenceTime(result[left]).After(studentAttemptEvidenceTime(result[right]))
	})
	if len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

func (repository *Repository) SaveClassLearningSynthesis(
	ctx context.Context,
	document ClassLearningSynthesisDocument,
) error {
	ref := repository.client.Collection(classLearningCollectionName).Doc(document.ClassID)
	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snapshot, err := tx.Get(ref)
		if err == nil {
			var existing ClassLearningSynthesisDocument
			if err := snapshot.DataTo(&existing); err != nil {
				return fmt.Errorf("decode class learning synthesis: %w", err)
			}
			if existing.EvidenceUpdatedAt.After(document.EvidenceUpdatedAt) ||
				existing.EvidenceUpdatedAt.Equal(document.EvidenceUpdatedAt) &&
					existing.AttemptCount > document.AttemptCount {
				return nil
			}
		} else if status.Code(err) != codes.NotFound {
			return fmt.Errorf("read class learning synthesis: %w", err)
		}

		return tx.Set(ref, map[string]any{
			"organization_id":     document.OrganizationID,
			"district_id":         document.DistrictID,
			"school_id":           document.SchoolID,
			"class_id":            document.ClassID,
			"synthesis":           document.Synthesis,
			"attempt_count":       document.AttemptCount,
			"evidence_updated_at": document.EvidenceUpdatedAt,
			"updated_at":          document.UpdatedAt,
		}, firestore.MergeAll)
	})
	if err != nil {
		return fmt.Errorf("save class learning synthesis: %w", err)
	}

	return nil
}

func (repository *Repository) FinalizeStudentLearningTrace(
	ctx context.Context,
	studentID string,
	attemptID string,
	trace orchestrator.LearningTraceBlueprint,
	memory StudentLearningMemory,
	timestamp time.Time,
) (StudentMissionAttemptDocument, error) {
	userRef := repository.client.Collection(usersCollectionName).Doc(studentID)
	attemptRef := repository.client.Collection(studentAttemptsCollectionName).Doc(attemptID)
	var attempt StudentMissionAttemptDocument

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

		attemptSnapshot, err := tx.Get(attemptRef)
		if status.Code(err) == codes.NotFound {
			return ErrStudentAttemptNotFound
		}
		if err != nil {
			return fmt.Errorf("read learning attempt: %w", err)
		}
		if err := attemptSnapshot.DataTo(&attempt); err != nil {
			return fmt.Errorf("decode learning attempt: %w", err)
		}
		if attempt.StudentID != studentID {
			return ErrStudentMissionMismatch
		}

		attempt.Trace = trace
		attempt.TraceState = "ready"
		attempt.UpdatedAt = timestamp
		memory.UpdatedAt = timestamp

		if err := tx.Set(attemptRef, map[string]any{
			"trace":       trace,
			"trace_state": "ready",
			"updated_at":  timestamp,
		}, firestore.MergeAll); err != nil {
			return fmt.Errorf("write learning trace: %w", err)
		}
		if user.Student.LearningMemory.EvidenceAt.After(memory.EvidenceAt) {
			return nil
		}
		return tx.Set(userRef, map[string]any{
			"student": map[string]any{
				"learning_memory": memory,
			},
			"updated_at": timestamp,
		}, firestore.MergeAll)
	})
	if err != nil {
		return StudentMissionAttemptDocument{}, fmt.Errorf("finalize student learning trace: %w", err)
	}

	return attempt, nil
}

type buildStudentUserDocumentInput struct {
	ID             string
	Input          UpsertStudentInput
	CreatedAt      time.Time
	FirstSeenAt    time.Time
	LoginCount     int64
	CurrentGame    StudentGameSession
	LearningMemory StudentLearningMemory
}

func buildStudentUserDocument(input buildStudentUserDocumentInput) UserDocument {
	identity := input.Input.Identity
	now := input.Input.Timestamp
	scope := tenant.NormalizeScope(input.Input.Scope)

	return UserDocument{
		ID: input.ID,
		Identity: UserIdentity{
			Provider:       identity.Provider,
			ProviderUserID: identity.ProviderUserID,
			AuthMethod:     identity.AuthMethod,
			ProviderIDs:    identity.ProviderIDs,
			SignInProvider: identity.SignInProvider,
			CustomClaims:   identity.CustomClaims,
			Token: IdentityToken{
				AuthTime:  identity.TokenAuthTime,
				ExpiresAt: identity.TokenExpiresAt,
				IssuedAt:  identity.TokenIssuedAt,
				Issuer:    identity.TokenIssuer,
				Audience:  identity.TokenAudience,
			},
		},
		Profile: UserProfile{
			DisplayName:   input.Input.StudentName,
			Email:         identity.Email,
			EmailVerified: identity.EmailVerified,
			PhotoURL:      identity.PhotoURL,
		},
		Student: StudentProfile{
			OrganizationID: scope.OrganizationID,
			DistrictID:     scope.DistrictID,
			SchoolID:       scope.SchoolID,
			ClassID:        input.Input.ClassID,
			ClassCode:      input.Input.ClassCode,
			MissionID:      input.Input.MissionID,
			MissionCode:    input.Input.MissionCode,
			StudentName:    input.Input.StudentName,
			LearningMemory: input.LearningMemory,
		},
		Activity: UserActivity{
			FirstSeenAt: input.FirstSeenAt,
			LastSeenAt:  now,
			LoginCount:  input.LoginCount,
			CurrentGame: input.CurrentGame,
		},
		Roles:     []UserRole{UserRoleStudent},
		IsGuest:   true,
		CreatedAt: input.CreatedAt,
		UpdatedAt: now,
	}
}

func (user UserDocument) firestoreMap() map[string]any {
	return map[string]any{
		"id": user.ID,
		"identity": map[string]any{
			"provider":         user.Identity.Provider,
			"provider_user_id": user.Identity.ProviderUserID,
			"auth_method":      user.Identity.AuthMethod,
			"provider_ids":     user.Identity.ProviderIDs,
			"sign_in_provider": user.Identity.SignInProvider,
			"custom_claims":    user.Identity.CustomClaims,
			"token": map[string]any{
				"auth_time":  user.Identity.Token.AuthTime,
				"expires_at": user.Identity.Token.ExpiresAt,
				"issued_at":  user.Identity.Token.IssuedAt,
				"issuer":     user.Identity.Token.Issuer,
				"audience":   user.Identity.Token.Audience,
			},
		},
		"profile": map[string]any{
			"display_name":   user.Profile.DisplayName,
			"email":          user.Profile.Email,
			"email_verified": user.Profile.EmailVerified,
			"photo_url":      user.Profile.PhotoURL,
		},
		"student": map[string]any{
			"organization_id": user.Student.OrganizationID,
			"district_id":     user.Student.DistrictID,
			"school_id":       user.Student.SchoolID,
			"class_id":        user.Student.ClassID,
			"class_code":      user.Student.ClassCode,
			"mission_id":      user.Student.MissionID,
			"mission_code":    user.Student.MissionCode,
			"student_name":    user.Student.StudentName,
			"learning_memory": user.Student.LearningMemory,
		},
		"activity": map[string]any{
			"first_seen_at": user.Activity.FirstSeenAt,
			"last_seen_at":  user.Activity.LastSeenAt,
			"login_count":   user.Activity.LoginCount,
			"current_game":  user.Activity.CurrentGame.firestoreMap(),
		},
		"roles":      user.Roles,
		"is_guest":   user.IsGuest,
		"deleted_at": user.DeletedAt,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	}
}

func (session StudentGameSession) firestoreMap() map[string]any {
	return map[string]any{
		"attempt_id":         session.AttemptID,
		"mission_id":         session.MissionID,
		"published_revision": session.PublishedRevision,
		"subject":            session.Subject,
		"stage_id":           session.StageID,
		"stage_name":         session.StageName,
		"status":             session.Status,
		"started_at":         session.StartedAt,
		"last_heartbeat_at":  session.LastHeartbeatAt,
	}
}

func learningAttemptID(studentID string, missionID string, revision int64) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", studentID, missionID, revision)))

	return fmt.Sprintf("attempt-%x", digest[:12])
}

func studentAttemptEvidenceTime(attempt StudentMissionAttemptDocument) time.Time {
	if !attempt.LastActivityAt.IsZero() {
		return attempt.LastActivityAt
	}

	return attempt.UpdatedAt
}

func tutorTurnID(eventID string) string {
	digest := sha256.Sum256([]byte(eventID))

	return fmt.Sprintf("turn-%x", digest[:12])
}

func mergeLearningSnapshots(
	existing StudentLearningSnapshot,
	incoming StudentLearningSnapshot,
) StudentLearningSnapshot {
	incoming.Notes = mergeLearningNotes(existing.Notes, incoming.Notes)
	incoming.Decisions = mergeStudentDecisions(existing.Decisions, incoming.Decisions)
	incoming.Events = mergeLearningEvents(existing.Events, incoming.Events)

	return incoming
}

func mergeLearningNotes(existing []StudentLearningNote, incoming []StudentLearningNote) []StudentLearningNote {
	byID := make(map[string]StudentLearningNote, len(existing)+len(incoming))
	order := make([]string, 0, len(existing)+len(incoming))
	for _, note := range append(append([]StudentLearningNote{}, existing...), incoming...) {
		if note.ID == "" {
			continue
		}
		if _, exists := byID[note.ID]; !exists {
			order = append(order, note.ID)
		}
		byID[note.ID] = note
	}
	result := make([]StudentLearningNote, 0, min(len(order), 40))
	start := max(0, len(order)-40)
	for _, id := range order[start:] {
		result = append(result, byID[id])
	}

	return result
}

func mergeStudentDecisions(existing []StudentDecision, incoming []StudentDecision) []StudentDecision {
	byID := make(map[string]StudentDecision, len(existing)+len(incoming))
	for _, decision := range append(append([]StudentDecision{}, existing...), incoming...) {
		if decision.ID != "" {
			byID[decision.ID] = decision
		}
	}
	result := make([]StudentDecision, 0, len(byID))
	for _, decision := range byID {
		result = append(result, decision)
	}
	sort.SliceStable(result, func(left int, right int) bool {
		return result[left].At > result[right].At
	})
	if len(result) > 40 {
		result = result[:40]
	}

	return result
}

func mergeLearningEvents(existing []StudentLearningEvent, incoming []StudentLearningEvent) []StudentLearningEvent {
	byID := make(map[string]StudentLearningEvent, len(existing)+len(incoming))
	for _, event := range append(append([]StudentLearningEvent{}, existing...), incoming...) {
		if event.ID != "" {
			byID[event.ID] = event
		}
	}
	result := make([]StudentLearningEvent, 0, len(byID))
	for _, event := range byID {
		result = append(result, event)
	}
	sort.SliceStable(result, func(left int, right int) bool {
		if result[left].Sequence == result[right].Sequence {
			return result[left].OccurredAt < result[right].OccurredAt
		}

		return result[left].Sequence < result[right].Sequence
	})
	if len(result) > 160 {
		result = result[len(result)-160:]
	}

	return result
}

func learningAttemptFirestoreMap(attempt StudentMissionAttemptDocument) map[string]any {
	return map[string]any{
		"id":                    attempt.ID,
		"organization_id":       attempt.OrganizationID,
		"district_id":           attempt.DistrictID,
		"school_id":             attempt.SchoolID,
		"student_id":            attempt.StudentID,
		"student_name":          attempt.StudentName,
		"class_id":              attempt.ClassID,
		"mission_id":            attempt.MissionID,
		"mission_code":          attempt.MissionCode,
		"published_revision":    attempt.PublishedRevision,
		"subject":               attempt.Subject,
		"status":                attempt.Status,
		"snapshot":              attempt.Snapshot,
		"tutor_turns":           attempt.TutorTurns,
		"trace":                 attempt.Trace,
		"trace_state":           attempt.TraceState,
		"finalize_requested_at": attempt.FinalizeRequestedAt,
		"started_at":            attempt.StartedAt,
		"last_activity_at":      attempt.LastActivityAt,
		"completed_at":          attempt.CompletedAt,
		"abandoned_at":          attempt.AbandonedAt,
		"created_at":            attempt.CreatedAt,
		"updated_at":            attempt.UpdatedAt,
	}
}
