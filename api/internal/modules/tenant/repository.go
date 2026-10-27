package tenant

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FirestoreRepository struct {
	client *firestore.Client
}

func NewFirestoreRepository(client *firestore.Client) *FirestoreRepository {
	return &FirestoreRepository{client: client}
}

// EnsureHierarchy creates only missing hierarchy documents. Existing tenant,
// district, and school records are preserved so this is safe on every startup
// and when several API instances start concurrently.
func (repository *FirestoreRepository) EnsureHierarchy(
	ctx context.Context,
	hierarchy Hierarchy,
	timestamp time.Time,
) (EnsureHierarchyResult, error) {
	if repository == nil || repository.client == nil {
		return EnsureHierarchyResult{}, errors.New("tenant repository is not configured")
	}

	hierarchy = NormalizeHierarchy(hierarchy)
	if err := validateHierarchy(hierarchy); err != nil {
		return EnsureHierarchyResult{}, err
	}

	tenantID := hierarchy.Scope.OrganizationID
	result := EnsureHierarchyResult{}
	var err error

	result.TenantCreated, err = createDocumentIfMissing(
		ctx,
		repository.client.Collection(TenantsCollectionName).Doc(tenantID),
		TenantDocument{
			ID:             tenantID,
			TenantID:       tenantID,
			OrganizationID: hierarchy.Scope.OrganizationID,
			Name:           hierarchy.OrganizationName,
			Status:         "active",
			CreatedAt:      timestamp,
			UpdatedAt:      timestamp,
		},
	)
	if err != nil {
		return EnsureHierarchyResult{}, fmt.Errorf("ensure tenant %q: %w", tenantID, err)
	}

	result.DistrictCreated, err = createDocumentIfMissing(
		ctx,
		repository.client.Collection(DistrictsCollectionName).Doc(hierarchy.Scope.DistrictID),
		DistrictDocument{
			ID:             hierarchy.Scope.DistrictID,
			TenantID:       tenantID,
			OrganizationID: hierarchy.Scope.OrganizationID,
			DistrictID:     hierarchy.Scope.DistrictID,
			Name:           hierarchy.DistrictName,
			Status:         "active",
			CreatedAt:      timestamp,
			UpdatedAt:      timestamp,
		},
	)
	if err != nil {
		return EnsureHierarchyResult{}, fmt.Errorf("ensure district %q: %w", hierarchy.Scope.DistrictID, err)
	}

	result.SchoolCreated, err = createDocumentIfMissing(
		ctx,
		repository.client.Collection(SchoolsCollectionName).Doc(hierarchy.Scope.SchoolID),
		SchoolDocument{
			ID:             hierarchy.Scope.SchoolID,
			TenantID:       tenantID,
			OrganizationID: hierarchy.Scope.OrganizationID,
			DistrictID:     hierarchy.Scope.DistrictID,
			SchoolID:       hierarchy.Scope.SchoolID,
			Name:           hierarchy.SchoolName,
			Status:         "active",
			CreatedAt:      timestamp,
			UpdatedAt:      timestamp,
		},
	)
	if err != nil {
		return EnsureHierarchyResult{}, fmt.Errorf("ensure school %q: %w", hierarchy.Scope.SchoolID, err)
	}

	return result, nil
}

func createDocumentIfMissing(
	ctx context.Context,
	reference *firestore.DocumentRef,
	document any,
) (bool, error) {
	if _, err := reference.Create(ctx, document); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func validateHierarchy(hierarchy Hierarchy) error {
	for label, value := range map[string]string{
		"organization_id": hierarchy.Scope.OrganizationID,
		"district_id":     hierarchy.Scope.DistrictID,
		"school_id":       hierarchy.Scope.SchoolID,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", label)
		}
		if strings.Contains(value, "/") {
			return fmt.Errorf("%s cannot contain a slash", label)
		}
	}

	return nil
}
