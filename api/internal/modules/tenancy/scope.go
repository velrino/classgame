package tenancy

import (
	"os"
	"strings"
)

const (
	DefaultOrganizationID   = "default-org"
	DefaultDistrictID       = "default-district"
	DefaultSchoolID         = "default-school"
	DefaultOrganizationName = "ClassGame"
	DefaultDistrictName     = "ClassGame District"
	DefaultSchoolName       = "ClassGame School"
)

type Scope struct {
	OrganizationID string `firestore:"organization_id" json:"organization_id"`
	DistrictID     string `firestore:"district_id" json:"district_id"`
	SchoolID       string `firestore:"school_id" json:"school_id"`
}

type Hierarchy struct {
	Scope            Scope
	OrganizationName string
	DistrictName     string
	SchoolName       string
}

func DefaultScope() Scope {
	return Scope{
		OrganizationID: envString("DEFAULT_ORGANIZATION_ID", DefaultOrganizationID),
		DistrictID:     envString("DEFAULT_DISTRICT_ID", DefaultDistrictID),
		SchoolID:       envString("DEFAULT_SCHOOL_ID", DefaultSchoolID),
	}
}

func DefaultHierarchy() Hierarchy {
	return Hierarchy{
		Scope:            DefaultScope(),
		OrganizationName: envString("DEFAULT_ORGANIZATION_NAME", DefaultOrganizationName),
		DistrictName:     envString("DEFAULT_DISTRICT_NAME", DefaultDistrictName),
		SchoolName:       envString("DEFAULT_SCHOOL_NAME", DefaultSchoolName),
	}
}

func NormalizeHierarchy(hierarchy Hierarchy) Hierarchy {
	hierarchy.Scope = NormalizeScope(hierarchy.Scope)
	hierarchy.OrganizationName = firstNonEmpty(hierarchy.OrganizationName, DefaultOrganizationName)
	hierarchy.DistrictName = firstNonEmpty(hierarchy.DistrictName, DefaultDistrictName)
	hierarchy.SchoolName = firstNonEmpty(hierarchy.SchoolName, DefaultSchoolName)

	return hierarchy
}

func NormalizeScope(scope Scope) Scope {
	defaultScope := DefaultScope()
	scope.OrganizationID = firstNonEmpty(scope.OrganizationID, defaultScope.OrganizationID)
	scope.DistrictID = firstNonEmpty(scope.DistrictID, defaultScope.DistrictID)
	scope.SchoolID = firstNonEmpty(scope.SchoolID, defaultScope.SchoolID)

	return scope
}

func envString(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}
