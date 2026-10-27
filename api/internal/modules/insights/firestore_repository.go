package insights

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	studentsmodule "classgame-api/internal/modules/students"
	"classgame-api/internal/modules/tenancy"
)

type FirestoreRepository struct {
	client *firestore.Client
}

func NewFirestoreRepository(client *firestore.Client) *FirestoreRepository {
	return &FirestoreRepository{
		client: client,
	}
}

func (repository *FirestoreRepository) UpsertTeacher(
	ctx context.Context,
	identity TeacherIdentity,
	timestamp time.Time,
) (TeacherSummary, error) {
	ref := repository.client.Collection(teachersCollectionName).Doc(identity.ProviderUserID)
	summary := teacherSummaryFromIdentity(identity)
	scope := tenancy.DefaultScope()
	identityFields := map[string]any{
		"provider":         identity.Provider,
		"provider_user_id": identity.ProviderUserID,
		"auth_method":      identity.AuthMethod,
		"email":            identity.Email,
		"email_verified":   identity.EmailVerified,
		"display_name":     identity.DisplayName,
		"photo_url":        identity.PhotoURL,
		"provider_ids":     identity.ProviderIDs,
		"sign_in_provider": identity.SignInProvider,
	}

	updateExisting := func() error {
		_, err := ref.Update(ctx, []firestore.Update{
			{Path: "identity.provider", Value: identity.Provider},
			{Path: "identity.provider_user_id", Value: identity.ProviderUserID},
			{Path: "identity.auth_method", Value: identity.AuthMethod},
			{Path: "identity.email", Value: identity.Email},
			{Path: "identity.email_verified", Value: identity.EmailVerified},
			{Path: "identity.display_name", Value: identity.DisplayName},
			{Path: "identity.photo_url", Value: identity.PhotoURL},
			{Path: "identity.provider_ids", Value: identity.ProviderIDs},
			{Path: "identity.sign_in_provider", Value: identity.SignInProvider},
			{Path: "profile.provider_user_id", Value: summary.ProviderUserID},
			{Path: "profile.display_name", Value: summary.DisplayName},
			{Path: "profile.email", Value: summary.Email},
			{Path: "profile.photo_url", Value: summary.PhotoURL},
			{Path: "roles", Value: []string{"teacher"}},
			{Path: "activity.last_seen_at", Value: timestamp},
			{Path: "activity.login_count", Value: firestore.Increment(1)},
			{Path: "updated_at", Value: timestamp},
		})
		return err
	}

	err := updateExisting()
	if err == nil {
		return summary, nil
	}
	if status.Code(err) != codes.NotFound {
		return TeacherSummary{}, fmt.Errorf("update teacher: %w", err)
	}

	_, err = ref.Create(ctx, map[string]any{
		"id":              ref.ID,
		"organization_id": scope.OrganizationID,
		"district_id":     scope.DistrictID,
		"school_id":       scope.SchoolID,
		"identity":        identityFields,
		"profile":         summary.firestoreMap(),
		"roles":           []string{"teacher"},
		"activity": map[string]any{
			"first_seen_at": timestamp,
			"last_seen_at":  timestamp,
			"login_count":   int64(1),
		},
		"created_at": timestamp,
		"updated_at": timestamp,
	})
	if err == nil {
		return summary, nil
	}
	if status.Code(err) != codes.AlreadyExists {
		return TeacherSummary{}, fmt.Errorf("create teacher: %w", err)
	}
	if err := updateExisting(); err != nil {
		return TeacherSummary{}, fmt.Errorf("update teacher: %w", err)
	}

	return summary, nil
}

func (repository *FirestoreRepository) GetClassLearningSynthesis(
	ctx context.Context,
	classID string,
) (studentsmodule.ClassLearningSynthesisDocument, bool, error) {
	snapshot, err := repository.client.Collection("class_learning_summaries").Doc(classID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return studentsmodule.ClassLearningSynthesisDocument{}, false, nil
	}
	if err != nil {
		return studentsmodule.ClassLearningSynthesisDocument{}, false, fmt.Errorf("read class learning synthesis: %w", err)
	}
	var document studentsmodule.ClassLearningSynthesisDocument
	if err := snapshot.DataTo(&document); err != nil {
		return studentsmodule.ClassLearningSynthesisDocument{}, false, fmt.Errorf("decode class learning synthesis: %w", err)
	}

	return document, true, nil
}

func (teacher TeacherSummary) firestoreMap() map[string]any {
	return map[string]any{
		"provider_user_id": teacher.ProviderUserID,
		"display_name":     teacher.DisplayName,
		"email":            teacher.Email,
		"photo_url":        teacher.PhotoURL,
	}
}

func teacherSummaryFromIdentity(identity TeacherIdentity) TeacherSummary {
	displayName := identity.DisplayName
	if displayName == "" {
		displayName = identity.Email
	}

	return TeacherSummary{
		ProviderUserID: identity.ProviderUserID,
		DisplayName:    displayName,
		Email:          identity.Email,
		PhotoURL:       identity.PhotoURL,
	}
}
