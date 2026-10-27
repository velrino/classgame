package teacherclass

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"classgame-api/internal/modules/tenant"
)

var (
	ErrClassOwnedByAnotherTeacher = errors.New("teacher class belongs to another teacher")
	ErrClassCodeUnavailable       = errors.New("teacher class code is unavailable")
	ErrClassNotFound              = errors.New("teacher class not found")
)

type FirestoreRepository struct {
	client *firestore.Client
}

func NewFirestoreRepository(client *firestore.Client) *FirestoreRepository {
	return &FirestoreRepository{
		client: client,
	}
}

func (repository *FirestoreRepository) Get(
	ctx context.Context,
	classID string,
) (TeacherClassDocument, bool, error) {
	snapshot, err := repository.client.Collection(CollectionName).Doc(classID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return TeacherClassDocument{}, false, nil
	}
	if err != nil {
		return TeacherClassDocument{}, false, fmt.Errorf("read teacher class: %w", err)
	}

	var teacherClass TeacherClassDocument
	if err := snapshot.DataTo(&teacherClass); err != nil {
		return TeacherClassDocument{}, false, fmt.Errorf("decode teacher class: %w", err)
	}
	if teacherClass.ID == "" {
		teacherClass.ID = snapshot.Ref.ID
	}
	hydrateTeacherClassTenantScope(&teacherClass)

	return teacherClass, true, nil
}

func (repository *FirestoreRepository) ListByTeacherID(
	ctx context.Context,
	teacherID string,
) ([]TeacherClassDocument, error) {
	iter := repository.client.Collection(CollectionName).
		Where("teacher_id", "==", teacherID).
		Documents(ctx)
	defer iter.Stop()

	teacherClasses := make([]TeacherClassDocument, 0)
	for {
		snapshot, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterate teacher classes: %w", err)
		}

		var teacherClass TeacherClassDocument
		if err := snapshot.DataTo(&teacherClass); err != nil {
			return nil, fmt.Errorf("decode teacher class %s: %w", snapshot.Ref.ID, err)
		}
		if teacherClass.ID == "" {
			teacherClass.ID = snapshot.Ref.ID
		}
		hydrateTeacherClassTenantScope(&teacherClass)

		teacherClasses = append(teacherClasses, teacherClass)
	}

	sort.SliceStable(teacherClasses, func(leftIndex, rightIndex int) bool {
		return teacherClasses[leftIndex].UpdatedAt.After(teacherClasses[rightIndex].UpdatedAt)
	})

	return teacherClasses, nil
}

func (repository *FirestoreRepository) Upsert(
	ctx context.Context,
	input UpsertTeacherClassInput,
) (TeacherClassDocument, error) {
	ref := repository.client.Collection(CollectionName).Doc(input.ClassID)

	var teacherClass TeacherClassDocument
	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		createdAt := input.Timestamp
		missionCount := int64(0)
		activeMissionID := ""
		scope := tenant.NormalizeScope(input.Scope)

		snapshot, err := tx.Get(ref)
		switch {
		case err == nil:
			var existing TeacherClassDocument
			if err := snapshot.DataTo(&existing); err != nil {
				return fmt.Errorf("decode teacher class: %w", err)
			}
			hydrateTeacherClassTenantScope(&existing)
			if existing.TeacherID != "" && existing.TeacherID != input.Teacher.ProviderUserID {
				return ErrClassOwnedByAnotherTeacher
			}
			if !existing.CreatedAt.IsZero() {
				createdAt = existing.CreatedAt
			}
			missionCount = existing.MissionCount
			activeMissionID = existing.ActiveMissionID
			scope = tenant.NormalizeScope(tenant.Scope{
				OrganizationID: existing.OrganizationID,
				DistrictID:     existing.DistrictID,
				SchoolID:       existing.SchoolID,
			})
		case status.Code(err) != codes.NotFound:
			return fmt.Errorf("read teacher class: %w", err)
		}

		input.Scope = scope
		teacherClass = newTeacherClassDocument(input, createdAt, activeMissionID, missionCount)

		return tx.Set(ref, teacherClass.firestoreMap(), firestore.MergeAll)
	})
	if err != nil {
		return TeacherClassDocument{}, fmt.Errorf("upsert teacher class: %w", err)
	}

	return teacherClass, nil
}

func (repository *FirestoreRepository) Create(
	ctx context.Context,
	input UpsertTeacherClassInput,
) (TeacherClassDocument, error) {
	ref := repository.client.Collection(CollectionName).Doc(input.ClassID)
	teacherClass := newTeacherClassDocument(input, input.Timestamp, "", 0)

	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		_, err := tx.Get(ref)
		switch {
		case err == nil:
			return ErrClassCodeUnavailable
		case status.Code(err) != codes.NotFound:
			return fmt.Errorf("check teacher class code: %w", err)
		}

		return tx.Create(ref, teacherClass.firestoreMap())
	})
	if err != nil {
		return TeacherClassDocument{}, fmt.Errorf("create teacher class: %w", err)
	}

	return teacherClass, nil
}

func (repository *FirestoreRepository) SetActiveMission(
	ctx context.Context,
	classID string,
	teacherID string,
	missionID string,
	timestamp time.Time,
) error {
	ref := repository.client.Collection(CollectionName).Doc(classID)

	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snapshot, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			return ErrClassNotFound
		}
		if err != nil {
			return fmt.Errorf("read teacher class: %w", err)
		}

		var teacherClass TeacherClassDocument
		if err := snapshot.DataTo(&teacherClass); err != nil {
			return fmt.Errorf("decode teacher class: %w", err)
		}
		if teacherClass.TeacherID != teacherID {
			return ErrClassOwnedByAnotherTeacher
		}

		update := map[string]any{
			"active_mission_id": missionID,
			"updated_at":        timestamp,
		}

		return tx.Set(ref, update, firestore.MergeAll)
	})
	if err != nil {
		return fmt.Errorf("set active mission: %w", err)
	}

	return nil
}

func (teacherClass TeacherClassDocument) firestoreMap() map[string]any {
	return map[string]any{
		"id":                teacherClass.ID,
		"organization_id":   teacherClass.OrganizationID,
		"district_id":       teacherClass.DistrictID,
		"school_id":         teacherClass.SchoolID,
		"class_id":          teacherClass.ClassID,
		"class_code":        teacherClass.ClassCode,
		"class_name":        teacherClass.ClassName,
		"subject":           teacherClass.Subject,
		"grade_level":       teacherClass.GradeLevel,
		"teacher_id":        teacherClass.TeacherID,
		"teacher":           teacherClass.Teacher.firestoreMap(),
		"active_mission_id": teacherClass.ActiveMissionID,
		"mission_count":     teacherClass.MissionCount,
		"status":            teacherClass.Status,
		"created_at":        teacherClass.CreatedAt,
		"updated_at":        teacherClass.UpdatedAt,
	}
}

func newTeacherClassDocument(
	input UpsertTeacherClassInput,
	createdAt time.Time,
	activeMissionID string,
	missionCount int64,
) TeacherClassDocument {
	scope := tenant.NormalizeScope(input.Scope)

	return TeacherClassDocument{
		ID:              input.ClassID,
		OrganizationID:  scope.OrganizationID,
		DistrictID:      scope.DistrictID,
		SchoolID:        scope.SchoolID,
		ClassID:         input.ClassID,
		ClassCode:       input.ClassID,
		ClassName:       input.ClassName,
		Subject:         input.Subject,
		GradeLevel:      input.GradeLevel,
		TeacherID:       input.Teacher.ProviderUserID,
		Teacher:         input.Teacher,
		ActiveMissionID: activeMissionID,
		MissionCount:    missionCount,
		Status:          "active",
		CreatedAt:       createdAt,
		UpdatedAt:       input.Timestamp,
	}
}

func hydrateTeacherClassTenantScope(teacherClass *TeacherClassDocument) {
	if teacherClass == nil {
		return
	}

	scope := tenant.NormalizeScope(tenant.Scope{
		OrganizationID: teacherClass.OrganizationID,
		DistrictID:     teacherClass.DistrictID,
		SchoolID:       teacherClass.SchoolID,
	})
	teacherClass.OrganizationID = scope.OrganizationID
	teacherClass.DistrictID = scope.DistrictID
	teacherClass.SchoolID = scope.SchoolID
}

func (teacher TeacherSummary) firestoreMap() map[string]any {
	return map[string]any{
		"provider_user_id": teacher.ProviderUserID,
		"display_name":     teacher.DisplayName,
		"email":            teacher.Email,
		"photo_url":        teacher.PhotoURL,
	}
}
