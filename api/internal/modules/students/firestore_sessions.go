package students

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"classgame-api/internal/modules/classrooms"
	"classgame-api/internal/modules/tenancy"
)

func (repository *FirestoreRepository) UpsertStudent(
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
		existingMember := false

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
			existingMember = existing.Student.ClassID == input.ClassID && tenancy.NormalizeScope(tenancy.Scope{OrganizationID: existing.Student.OrganizationID, DistrictID: existing.Student.DistrictID, SchoolID: existing.Student.SchoolID}) == tenancy.NormalizeScope(input.Scope)
			if existingMember {
				currentGame = existing.Activity.CurrentGame
				learningMemory = existing.Student.LearningMemory
			}
		case status.Code(err) != codes.NotFound:
			return fmt.Errorf("read user: %w", err)
		}
		// This read participates in the membership write transaction. Closing
		// enrollment concurrently cannot admit a new student using a stale read.
		classSnapshot, err := tx.Get(repository.client.Collection(classrooms.CollectionName).Doc(input.ClassID))
		if status.Code(err) == codes.NotFound {
			return ErrClassCodeNotFound
		}
		if err != nil {
			return fmt.Errorf("read enrollment policy: %w", err)
		}
		var class classrooms.TeacherClassDocument
		if err := classSnapshot.DataTo(&class); err != nil {
			return err
		}
		if class.ClassCode != input.ClassCode || !classrooms.MayEnroll(class, existingMember, classSnapshot.ReadTime) || tenancy.NormalizeScope(tenancy.Scope{OrganizationID: class.OrganizationID, DistrictID: class.DistrictID, SchoolID: class.SchoolID}) != tenancy.NormalizeScope(input.Scope) {
			return ErrClassCodeNotFound
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

func (repository *FirestoreRepository) UpdateStudentPresence(
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
	scope := tenancy.NormalizeScope(input.Input.Scope)

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

// GetStudentClassMembership returns the minimal persisted class association.
func (repository *FirestoreRepository) GetStudentClassMembership(ctx context.Context, studentID string) (string, string, error) {
	snapshot, err := repository.client.Collection(usersCollectionName).Doc(studentID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return "", "", ErrStudentNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("read student class membership: %w", err)
	}
	var user UserDocument
	if err := snapshot.DataTo(&user); err != nil {
		return "", "", fmt.Errorf("decode student class membership: %w", err)
	}
	return user.Student.ClassID, user.Student.ClassCode, nil
}
