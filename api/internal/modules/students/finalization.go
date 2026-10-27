package students

import (
	"classgame-api/internal/modules/tenancy"
	"time"
)

const finalizationLeaseDuration = 5 * time.Minute

type FinalizationFence struct {
	AttemptID, StudentID, ClassID, Status, Token                              string
	PublishedRevision, SnapshotRevision, EvidenceRevision, DispatchGeneration int64
	Tenant                                                                    tenancy.Scope
}

func CaptureFinalizationFence(attempt StudentMissionAttemptDocument) FinalizationFence {
	return FinalizationFence{AttemptID: attempt.ID, StudentID: attempt.StudentID, ClassID: attempt.ClassID, Status: attempt.Status, Token: attempt.FinalizationToken, PublishedRevision: attempt.PublishedRevision, SnapshotRevision: attempt.Snapshot.Revision, EvidenceRevision: attempt.EvidenceRevision, DispatchGeneration: attempt.FinalizationDispatchGeneration, Tenant: tenancy.NormalizeScope(tenancy.Scope{OrganizationID: attempt.OrganizationID, DistrictID: attempt.DistrictID, SchoolID: attempt.SchoolID})}
}

// MatchesFinalizationFence is the same compare-and-set predicate used for result
// and failure-state writes. A late owner cannot change a newer attempt result.
func MatchesFinalizationFence(attempt StudentMissionAttemptDocument, fence FinalizationFence) bool {
	if attempt.Trace.Finalized || fence.Token == "" {
		return false
	}
	return CaptureFinalizationFence(attempt) == fence
}
func FinalizationEligible(attempt StudentMissionAttemptDocument) bool {
	// Closed pre-upgrade completions remain historical; no silent regrade/backfill.
	return !attempt.Trace.Finalized && (attempt.Status == "abandoned" || (attempt.Status == "completed" && attempt.EvidenceSchemaVersion > 0))
}
func FinalizationDue(attempt StudentMissionAttemptDocument, now time.Time) bool {
	return FinalizationEligible(attempt) && (attempt.TraceState != "processing" || !attempt.FinalizationLeaseUntil.After(now))
}
func FinalizationJob(attempt StudentMissionAttemptDocument) LearningJob {
	return LearningJob{Kind: LearningJobFinalize, StudentID: attempt.StudentID, AttemptID: attempt.ID, ExpectedEvidenceRevision: attempt.EvidenceRevision, ExpectedSnapshotRevision: attempt.Snapshot.Revision, ExpectedStatus: attempt.Status, DispatchGeneration: attempt.FinalizationDispatchGeneration}
}
func FinalizationJobMatches(attempt StudentMissionAttemptDocument, job LearningJob) bool {
	if attempt.StudentID != job.StudentID || attempt.ID != job.AttemptID {
		return false
	}
	if job.ExpectedStatus == "" {
		return true
	} // already queued legacy jobs still use CAS at final write
	return attempt.Status == job.ExpectedStatus && attempt.EvidenceRevision == job.ExpectedEvidenceRevision && attempt.Snapshot.Revision == job.ExpectedSnapshotRevision && attempt.FinalizationDispatchGeneration == job.DispatchGeneration
}
