package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	firebaseauth "firebase.google.com/go/v4/auth"

	"classgame-api/internal/modules/tenancy"
	"classgame-api/internal/platform/firebase"
)

type Options struct {
	TeacherEmail       string
	TeacherPassword    string
	TeacherDisplayName string
	Hierarchy          tenancy.Hierarchy
}

type Result struct {
	TeacherUID      string
	TeacherCreated  bool
	TenantCreated   bool
	DistrictCreated bool
	SchoolCreated   bool
}

// Ensure provisions the minimum single-tenant demo data without overwriting
// existing records. The IDs and hierarchy remain configurable for a future
// multi-tenant resolver.
func Ensure(
	ctx context.Context,
	authClient *firebaseauth.Client,
	firestoreClient *firestore.Client,
	options Options,
) (Result, error) {
	if authClient == nil || firestoreClient == nil {
		return Result{}, errors.New("firebase bootstrap clients are not configured")
	}

	hierarchyResult, err := tenancy.NewFirestoreRepository(firestoreClient).EnsureHierarchy(
		ctx,
		options.Hierarchy,
		time.Now().UTC(),
	)
	if err != nil {
		return Result{}, fmt.Errorf("ensure tenant hierarchy: %w", err)
	}

	teacherResult, err := firebase.EnsureEmailPasswordUser(
		ctx,
		authClient,
		firebase.EmailPasswordUserInput{
			Email:         options.TeacherEmail,
			Password:      options.TeacherPassword,
			DisplayName:   options.TeacherDisplayName,
			EmailVerified: true,
		},
	)
	if err != nil {
		return Result{}, fmt.Errorf("ensure demo teacher: %w", err)
	}

	return Result{
		TeacherUID:      teacherResult.User.UID,
		TeacherCreated:  teacherResult.Created,
		TenantCreated:   hierarchyResult.TenantCreated,
		DistrictCreated: hierarchyResult.DistrictCreated,
		SchoolCreated:   hierarchyResult.SchoolCreated,
	}, nil
}
