package students

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"sort"
	"time"

	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/tenancy"
)

const ClassRefreshCollection = "class_learning_refreshes"

var ErrClassRefreshBusy = errors.New("class learning refresh is already processing")
var ErrStaleClassRefresh = errors.New("class learning evidence changed during refresh")
var ErrClassRefreshQuota = errors.New("class learning refresh quota is unavailable")

// The intent is written with the evidence, before queue or provider calls.
// Generation changes for both projection updates and removals.
type ClassLearningRefresh struct {
	ClassID             string    `firestore:"class_id" json:"class_id"`
	OrganizationID      string    `firestore:"organization_id" json:"organization_id"`
	DistrictID          string    `firestore:"district_id" json:"district_id"`
	SchoolID            string    `firestore:"school_id" json:"school_id"`
	Generation          int64     `firestore:"generation" json:"generation"`
	CompletedGeneration int64     `firestore:"completed_generation" json:"completed_generation"`
	DispatchGeneration  int64     `firestore:"dispatch_generation" json:"dispatch_generation"`
	OwnerToken          string    `firestore:"owner_token" json:"-"`
	LeaseUntil          time.Time `firestore:"lease_until" json:"lease_until"`
	UpdatedAt           time.Time `firestore:"updated_at" json:"updated_at"`
}

func (refresh ClassLearningRefresh) pending() bool {
	return refresh.Generation > refresh.CompletedGeneration
}

func (refresh ClassLearningRefresh) matches(claim ClassLearningRefresh) bool {
	return refresh.pending() && claim.OwnerToken != "" && refresh.ClassID == claim.ClassID &&
		refresh.Generation == claim.Generation && refresh.DispatchGeneration == claim.DispatchGeneration &&
		refresh.OwnerToken == claim.OwnerToken && refresh.scope() == claim.scope()
}

func (refresh ClassLearningRefresh) scope() tenancy.Scope {
	return tenancy.NormalizeScope(tenancy.Scope{OrganizationID: refresh.OrganizationID, DistrictID: refresh.DistrictID, SchoolID: refresh.SchoolID})
}

type ClassRefreshPage struct {
	Refreshes  []ClassLearningRefresh `json:"refreshes"`
	Scanned    int                    `json:"scanned"`
	NextCursor string                 `json:"next_cursor,omitempty"`
}

type ClassLearningRefreshStore interface {
	GetClassLearningRefresh(context.Context, string) (ClassLearningRefresh, bool, error)
	ClaimClassLearningRefresh(context.Context, LearningJob, string, time.Time, time.Time) (ClassLearningRefresh, bool, error)
	CompleteClassLearningRefresh(context.Context, ClassLearningRefresh, ClassLearningSynthesisDocument, time.Time) error
	FailClassLearningRefresh(context.Context, ClassLearningRefresh, time.Time) error
	ListPendingClassLearningRefreshes(context.Context, string, int, time.Time) (ClassRefreshPage, error)
	PrepareClassRefreshRedrive(context.Context, ClassLearningRefresh, time.Time) (ClassLearningRefresh, bool, error)
}

type ClassRefreshTaskEnqueuer interface {
	EnqueueClassRefresh(context.Context, ClassLearningRefresh) error
}

func ClassRefreshJob(refresh ClassLearningRefresh) LearningJob {
	return LearningJob{Kind: LearningJobClassRefresh, ClassID: refresh.ClassID, ClassGeneration: refresh.Generation, DispatchGeneration: refresh.DispatchGeneration}
}

func (service *Service) requestClassLearningRefresh(ctx context.Context, classID string) {
	if err := service.scheduleClassRefresh(ctx, classID); err != nil {
		slog.WarnContext(ctx, "class learning refresh remains pending", "class_id", classID)
	}
}

func (service *Service) scheduleClassRefresh(ctx context.Context, classID string) error {
	if service.classRefresh == nil || service.classLearningGenerator == nil || classID == "" {
		return nil
	}
	refresh, exists, err := service.classRefresh.GetClassLearningRefresh(ctx, classID)
	if err != nil || !exists || !refresh.pending() {
		return err
	}
	if service.classRefreshTasks != nil {
		return service.classRefreshTasks.EnqueueClassRefresh(ctx, refresh)
	}
	return service.processClassLearningRefresh(ctx, ClassRefreshJob(refresh))
}

func (service *Service) processClassLearningRefresh(ctx context.Context, job LearningJob) error {
	if service.classRefresh == nil || service.classLearning == nil || service.classLearningGenerator == nil {
		return errors.New("class learning refresh is not configured")
	}
	owner := make([]byte, 16)
	if _, err := rand.Read(owner); err != nil {
		return err
	}
	now := time.Now()
	claim, claimed, err := service.classRefresh.ClaimClassLearningRefresh(ctx, job, hex.EncodeToString(owner), now, now.Add(finalizationLeaseDuration))
	if err != nil || !claimed {
		return err
	}
	finished := false
	defer func() {
		if !finished {
			// Cancellation must not strand the claim until its lease expires.
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			_ = service.classRefresh.FailClassLearningRefresh(cleanup, claim, time.Now())
		}
	}()
	if !service.allowLearningGeneration(ctx, claim.ClassID) {
		return ErrClassRefreshQuota
	}
	// The claim captures the generation before this bounded evidence query.
	// Every projection update/removal changes it in the source transaction.
	attempts, err := service.classLearning.ListLatestFinalizedAttemptsByClassID(ctx, claim.ClassID, 100)
	if err != nil {
		return err
	}
	evidence := make([]generation.ClassLearningStudentEvidence, 0, len(attempts))
	var evidenceAt time.Time
	for _, attempt := range attempts {
		if !sameRefreshTenant(attempt, claim) {
			return ErrStaleClassRefresh
		}
		evidence = append(evidence, generation.ClassLearningStudentEvidence{StudentID: attempt.StudentID, StudentName: attempt.StudentName, Strengths: attempt.Trace.Strengths, ImprovementAreas: attempt.Trace.ImprovementAreas})
		if at := studentAttemptEvidenceTime(attempt); at.After(evidenceAt) {
			evidenceAt = at
		}
	}
	synthesis, err := service.classLearningGenerator.BuildClassLearningSynthesis(ctx, generation.BuildClassLearningSynthesisInput{ClassID: claim.ClassID, Evidence: evidence})
	if err != nil {
		return err
	}
	err = service.classRefresh.CompleteClassLearningRefresh(ctx, claim, ClassLearningSynthesisDocument{
		ClassID: claim.ClassID, OrganizationID: claim.OrganizationID, DistrictID: claim.DistrictID, SchoolID: claim.SchoolID,
		Synthesis: synthesis, AttemptCount: len(attempts), EvidenceUpdatedAt: evidenceAt,
		EvidenceGeneration: claim.Generation, EvidenceFingerprint: classEvidenceFingerprint(attempts), UpdatedAt: time.Now(),
	}, time.Now())
	// Keep failed commits retryable. A superseded generation becomes a cheap
	// no-op at the next claim; a current generation must not become a completed
	// queue ID while its durable intent is still pending.
	finished = err == nil
	return err
}

func sameRefreshTenant(attempt StudentMissionAttemptDocument, refresh ClassLearningRefresh) bool {
	return attempt.ClassID == refresh.ClassID && tenancy.NormalizeScope(tenancy.Scope{OrganizationID: attempt.OrganizationID, DistrictID: attempt.DistrictID, SchoolID: attempt.SchoolID}) == refresh.scope()
}

func classEvidenceFingerprint(attempts []StudentMissionAttemptDocument) string {
	entries := make([]ClassEvidenceEntry, 0, len(attempts))
	for _, attempt := range attempts {
		entries = append(entries, classEvidence(attempt, time.Time{}))
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].StudentID < entries[j].StudentID })
	encoded, _ := json.Marshal(entries)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

// Recovery does not depend on the source attempt still being unfinalized.
// Redrive is explicit so ordinary sweeps retain stable queue identities.
func (service *Service) RecoverPendingClassLearning(ctx context.Context, after string, limit int, redrive bool) (RecoveryReport, error) {
	report := RecoveryReport{}
	if service.classRefresh == nil {
		return report, errors.New("class learning refresh store is required")
	}
	page, err := service.classRefresh.ListPendingClassLearningRefreshes(ctx, after, limit, time.Now())
	if err != nil {
		return report, err
	}
	report.Scanned, report.Candidates, report.NextCursor = page.Scanned, len(page.Refreshes), page.NextCursor
	for _, refresh := range page.Refreshes {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		if redrive {
			current, changed, err := service.classRefresh.PrepareClassRefreshRedrive(ctx, refresh, time.Now())
			if err != nil {
				report.Failed++
				continue
			}
			if !changed {
				continue
			}
			refresh = current
		}
		if service.classRefreshTasks != nil {
			err = service.classRefreshTasks.EnqueueClassRefresh(ctx, refresh)
		} else {
			err = service.processClassLearningRefresh(ctx, ClassRefreshJob(refresh))
		}
		if err != nil {
			report.Failed++
		} else if service.classRefreshTasks != nil {
			report.Enqueued++
		} else {
			report.Processed++
		}
	}
	if report.Failed > 0 {
		return report, errors.New("some class learning refreshes remain pending")
	}
	return report, nil
}
