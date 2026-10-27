package students

import (
	"classgame-api/internal/modules/tenancy"
	"strings"
)

// ValidateTutorTurnScope rejects a generated turn if the current session or
// loaded attempt no longer matches the scope captured before generation.
func ValidateTutorTurnScope(input SaveStudentTutorTurnInput, current StudentGameSession, attempt StudentMissionAttemptDocument) error {
	if err := ValidateAttemptRequestScope(input.RequestScope, attempt); err != nil {
		return err
	}
	currentID := current.AttemptID
	if currentID == "" {
		currentID = learningAttemptID(input.Identity.ProviderUserID, current.MissionID, current.PublishedRevision)
	}
	if input.ExpectedAttemptID == "" || currentID != input.ExpectedAttemptID || current.MissionID != input.MissionID || current.PublishedRevision != input.ExpectedPublishedRevision ||
		attempt.ID != input.ExpectedAttemptID || attempt.MissionID != input.MissionID || attempt.StudentID != input.Identity.ProviderUserID || attempt.PublishedRevision != input.ExpectedPublishedRevision {
		return ErrStudentMissionMismatch
	}
	return nil
}

// CanUpdateLearningMemory retains historical finalization without crossing the
// student's current classroom/tenant boundary or replacing newer evidence.
func CanUpdateLearningMemory(profile StudentProfile, attempt StudentMissionAttemptDocument, memory StudentLearningMemory) bool {
	return sameStudentClass(profile, attempt) && !profile.LearningMemory.EvidenceAt.After(memory.EvidenceAt)
}
func sameStudentClass(profile StudentProfile, attempt StudentMissionAttemptDocument) bool {
	if profile.ClassID == "" || profile.ClassID != attempt.ClassID {
		return false
	}
	current := tenancy.NormalizeScope(tenancy.Scope{OrganizationID: profile.OrganizationID, DistrictID: profile.DistrictID, SchoolID: profile.SchoolID})
	historical := tenancy.NormalizeScope(tenancy.Scope{OrganizationID: attempt.OrganizationID, DistrictID: attempt.DistrictID, SchoolID: attempt.SchoolID})
	return current == historical
}

// ValidateAttemptRequestScope checks optional client provenance before any
// generation or assessment and is reused inside persistence transactions.
// Unscoped legacy requests remain compatible and require a reload on revision changes.
func ValidateAttemptRequestScope(scope AttemptScope, attempt StudentMissionAttemptDocument) error {
	id := strings.TrimSpace(scope.AttemptID)
	if scope.PublishedRevision < 0 || (scope.PublishedRevision > 0 && scope.PublishedRevision != attempt.PublishedRevision) || (id != "" && id != attempt.ID) {
		return ErrStudentMissionMismatch
	}
	return nil
}
