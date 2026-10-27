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
	"classgame-api/internal/modules/tenancy"
)

func (repository *FirestoreRepository) SaveStudentLearningSnapshot(
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
		if attemptID != input.ExpectedAttemptID {
			return ErrStudentMissionMismatch
		}
		scopeAttempt := StudentMissionAttemptDocument{ID: attemptID, PublishedRevision: user.Activity.CurrentGame.PublishedRevision}
		if err := ValidateAttemptRequestScope(input.RequestScope, scopeAttempt); err != nil {
			return err
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

		evidence := StudentMissionAttemptDocument{EvidenceSchemaVersion: assessmentEvidenceSchemaVersion}
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
			if existing.ID != input.ExpectedAttemptID || existing.MissionID != input.MissionID || existing.PublishedRevision != input.PublishedRevision || existing.StudentID != user.ID || !sameStudentClass(user.Student, existing) {
				return ErrStudentMissionMismatch
			}
			if err := ValidateAttemptRequestScope(input.RequestScope, existing); err != nil {
				return err
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
				return nil
			}
			evidence = EffectiveTutorEvidence(existing)
			if existing.EvidenceSchemaVersion == 0 {
				evidence.AcceptedAnswers = acceptedAnswerFacts(input.Mission, existing.TutorTurns)
			}

			if existing.Snapshot.Revision >= input.Snapshot.Revision {
				if input.Status != "completed" {
					attempt = existing
					return nil
				}

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

		input.Snapshot, input.Status = AssessLearningSnapshotWithEvidence(input.Mission, input.Snapshot, tutorTurns, evidence.AcceptedAnswers, input.Status)
		liveInput := toLearningTraceInput(input.Mission, input.Snapshot, "in_progress", StudentLearningMemory{})
		trace = generation.BuildObservedLearningTrace(liveInput, "")
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
		scope := tenancy.NormalizeScope(tenancy.Scope{
			OrganizationID: user.Student.OrganizationID,
			DistrictID:     user.Student.DistrictID,
			SchoolID:       user.Student.SchoolID,
		})
		attempt = StudentMissionAttemptDocument{
			EvidenceSchemaVersion: evidence.EvidenceSchemaVersion, AcceptedAnswers: evidence.AcceptedAnswers, GeneratedTurnCount: evidence.GeneratedTurnCount, TutorEventSequence: evidence.TutorEventSequence, LegacyHistoryIncomplete: evidence.LegacyHistoryIncomplete, EvidenceRevision: evidence.EvidenceRevision + 1, FinalizationDispatchGeneration: evidence.FinalizationDispatchGeneration,

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

func (repository *FirestoreRepository) studentMissionCompletionRef(
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

func (repository *FirestoreRepository) GetCurrentStudentLearningAttempt(
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

func (repository *FirestoreRepository) GetStudentLearningAttempt(
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

func (repository *FirestoreRepository) MarkStudentAttemptAbandoned(
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
		marked = false
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

		// Read current membership before writes; historical watchdogs must not mutate
		// a newer current game or classroom.
		updateCurrent := false
		currentSnapshot, readErr := tx.Get(userRef)
		if readErr == nil {
			var user UserDocument
			if err := currentSnapshot.DataTo(&user); err != nil {
				return fmt.Errorf("decode watchdog student: %w", err)
			}
			currentID := user.Activity.CurrentGame.AttemptID
			if currentID == "" {
				currentID = learningAttemptID(studentID, user.Activity.CurrentGame.MissionID, user.Activity.CurrentGame.PublishedRevision)
			}
			updateCurrent = sameStudentClass(user.Student, attempt) && currentID == attemptID && user.Activity.CurrentGame.PublishedRevision == attempt.PublishedRevision
		} else if status.Code(readErr) != codes.NotFound {
			return fmt.Errorf("read watchdog student: %w", readErr)
		}

		attempt.EvidenceRevision++
		attempt.Status = "abandoned"
		attempt.AbandonedAt = timestamp
		attempt.TraceState = "pending"
		attempt.FinalizeRequestedAt = timestamp
		attempt.UpdatedAt = timestamp
		marked = true
		if err := tx.Set(attemptRef, map[string]any{
			"evidence_revision": attempt.EvidenceRevision, "finalization_token": "", "finalization_lease_until": time.Time{}, "trace": map[string]any{"finalized": false},
			"status":                attempt.Status,
			"abandoned_at":          attempt.AbandonedAt,
			"trace_state":           attempt.TraceState,
			"finalize_requested_at": attempt.FinalizeRequestedAt,
			"updated_at":            attempt.UpdatedAt,
		}, firestore.MergeAll); err != nil {
			return fmt.Errorf("mark learning attempt abandoned: %w", err)
		}
		if !updateCurrent {
			return nil
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

func (repository *FirestoreRepository) FinalizeStudentLearningTrace(
	ctx context.Context,
	fence FinalizationFence,
	trace generation.LearningTraceBlueprint,
	memory StudentLearningMemory,
	timestamp time.Time,
) (StudentMissionAttemptDocument, error) {
	userRef := repository.client.Collection(usersCollectionName).Doc(fence.StudentID)
	attemptRef := repository.client.Collection(studentAttemptsCollectionName).Doc(fence.AttemptID)
	var attempt StudentMissionAttemptDocument

	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		attempt = StudentMissionAttemptDocument{}
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
		if attempt.StudentID != fence.StudentID {
			return ErrStudentMissionMismatch
		}

		if attempt.ID == "" {
			attempt.ID = attemptRef.ID
		}
		if attempt.Trace.Finalized {
			return nil
		}
		if !MatchesFinalizationFence(attempt, fence) {
			return ErrStaleFinalization
		}
		trace.Finalized = true

		attempt.FinalizationToken = ""
		attempt.FinalizationLeaseUntil = time.Time{}
		attempt.Trace = trace
		attempt.TraceState = "ready"
		attempt.UpdatedAt = timestamp
		memory.UpdatedAt = timestamp

		if err := tx.Set(attemptRef, map[string]any{
			"trace":              trace,
			"trace_state":        "ready",
			"finalization_token": "", "finalization_lease_until": time.Time{},
			"updated_at": timestamp,
		}, firestore.MergeAll); err != nil {
			return fmt.Errorf("write learning trace: %w", err)
		}
		if !CanUpdateLearningMemory(user.Student, attempt, memory) {
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
		return attempt, fmt.Errorf("finalize student learning trace: %w", err)
	}

	return attempt, nil
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

func learningAttemptFirestoreMap(attempt StudentMissionAttemptDocument) map[string]any {
	return map[string]any{
		"evidence_schema_version": attempt.EvidenceSchemaVersion, "accepted_answers": attempt.AcceptedAnswers, "generated_turn_count": attempt.GeneratedTurnCount, "tutor_event_sequence": attempt.TutorEventSequence, "legacy_history_incomplete": attempt.LegacyHistoryIncomplete, "evidence_revision": attempt.EvidenceRevision, "finalization_token": attempt.FinalizationToken, "finalization_lease_until": attempt.FinalizationLeaseUntil, "finalization_dispatch_generation": attempt.FinalizationDispatchGeneration,

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
