package students

import "classgame-api/internal/modules/tenancy"

// ValidateTutorTurnScope rejects a generated turn if the current session or
// loaded attempt no longer matches the scope captured before generation.
func ValidateTutorTurnScope(input SaveStudentTutorTurnInput, current StudentGameSession, attempt StudentMissionAttemptDocument) error {
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
