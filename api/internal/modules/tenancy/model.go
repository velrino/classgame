package tenancy

import "time"

const (
	TenantsCollectionName   = "tenants"
	DistrictsCollectionName = "districts"
	SchoolsCollectionName   = "schools"
)

type TenantDocument struct {
	ID             string    `firestore:"id" json:"id"`
	TenantID       string    `firestore:"tenant_id" json:"tenant_id"`
	OrganizationID string    `firestore:"organization_id" json:"organization_id"`
	Name           string    `firestore:"name" json:"name"`
	Status         string    `firestore:"status" json:"status"`
	CreatedAt      time.Time `firestore:"created_at" json:"created_at"`
	UpdatedAt      time.Time `firestore:"updated_at" json:"updated_at"`
}

type DistrictDocument struct {
	ID             string    `firestore:"id" json:"id"`
	TenantID       string    `firestore:"tenant_id" json:"tenant_id"`
	OrganizationID string    `firestore:"organization_id" json:"organization_id"`
	DistrictID     string    `firestore:"district_id" json:"district_id"`
	Name           string    `firestore:"name" json:"name"`
	Status         string    `firestore:"status" json:"status"`
	CreatedAt      time.Time `firestore:"created_at" json:"created_at"`
	UpdatedAt      time.Time `firestore:"updated_at" json:"updated_at"`
}

type SchoolDocument struct {
	ID             string    `firestore:"id" json:"id"`
	TenantID       string    `firestore:"tenant_id" json:"tenant_id"`
	OrganizationID string    `firestore:"organization_id" json:"organization_id"`
	DistrictID     string    `firestore:"district_id" json:"district_id"`
	SchoolID       string    `firestore:"school_id" json:"school_id"`
	Name           string    `firestore:"name" json:"name"`
	Status         string    `firestore:"status" json:"status"`
	CreatedAt      time.Time `firestore:"created_at" json:"created_at"`
	UpdatedAt      time.Time `firestore:"updated_at" json:"updated_at"`
}

type EnsureHierarchyResult struct {
	TenantCreated   bool
	DistrictCreated bool
	SchoolCreated   bool
}
