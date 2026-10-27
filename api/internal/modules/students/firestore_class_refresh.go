package students

import (
	"classgame-api/internal/modules/classrooms"
	"classgame-api/internal/modules/tenancy"
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"time"
)

func (repository *FirestoreRepository) GetClassLearningRefresh(ctx context.Context, classID string) (ClassLearningRefresh, bool, error) {
	var refresh ClassLearningRefresh
	row, err := repository.client.Collection(ClassRefreshCollection).Doc(classID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return refresh, false, nil
	}
	if err != nil {
		return refresh, false, err
	}
	err = row.DataTo(&refresh)
	return refresh, err == nil, err
}

// Explicit repair/bootstrap for legacy evidence that predates durable intents.
// Normal sweeps never create new generations or silently regrade history.
func (repository *FirestoreRepository) RequestClassLearningRefresh(ctx context.Context, classID string, now time.Time) (ClassLearningRefresh, error) {
	if classID == "" || len(classID) > 120 || strings.TrimSpace(classID) != classID || strings.ContainsAny(classID, "/\x00") || classID == "." || classID == ".." {
		return ClassLearningRefresh{}, errors.New("a bounded class ID is required")
	}
	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		row, err := tx.Get(repository.client.Collection(classrooms.CollectionName).Doc(classID))
		if err != nil {
			return err
		}
		var class classrooms.TeacherClassDocument
		if err := row.DataTo(&class); err != nil {
			return err
		}
		write, err := repository.classRefreshWrite(tx, classID, class, now)
		if err != nil {
			return err
		}
		return write()
	})
	if err != nil {
		return ClassLearningRefresh{}, err
	}
	refresh, _, err := repository.GetClassLearningRefresh(ctx, classID)
	return refresh, err
}

// Prepare all reads before the caller writes source evidence in its transaction.
func (repository *FirestoreRepository) classRefreshWrite(tx *firestore.Transaction, classID string, class classrooms.TeacherClassDocument, now time.Time) (func() error, error) {
	ref := repository.client.Collection(ClassRefreshCollection).Doc(classID)
	refresh := ClassLearningRefresh{}
	row, err := tx.Get(ref)
	if err == nil {
		if err := row.DataTo(&refresh); err != nil {
			return nil, err
		}
	} else if status.Code(err) != codes.NotFound {
		return nil, err
	}
	scope := tenancy.NormalizeScope(tenancy.Scope{OrganizationID: class.OrganizationID, DistrictID: class.DistrictID, SchoolID: class.SchoolID})
	refresh.ClassID = classID
	refresh.OrganizationID, refresh.DistrictID, refresh.SchoolID = scope.OrganizationID, scope.DistrictID, scope.SchoolID
	refresh.Generation++
	refresh.OwnerToken, refresh.LeaseUntil, refresh.UpdatedAt = "", time.Time{}, now
	return func() error { return tx.Set(ref, refresh) }, nil
}

// Remove a resumed finalized attempt and schedule its class refresh atomically.
func (repository *FirestoreRepository) invalidateClassEvidence(tx *firestore.Transaction, attempt StudentMissionAttemptDocument, now time.Time) (func() error, error) {
	noop := func() error { return nil }
	if !attempt.Trace.Finalized || attempt.ClassID == "" || attempt.StudentID == "" {
		return noop, nil
	}
	ref := repository.client.Collection(ClassEvidenceCollection).Doc(attempt.ClassID).Collection("students").Doc(attempt.StudentID)
	row, err := tx.Get(ref)
	if status.Code(err) == codes.NotFound {
		row = nil
	} else if err != nil {
		return nil, err
	}
	if row != nil {
		var existing ClassEvidenceEntry
		if err := row.DataTo(&existing); err != nil {
			return nil, err
		}
		if existing.AttemptID != attempt.ID {
			return noop, nil
		}
	}
	classRow, err := tx.Get(repository.client.Collection(classrooms.CollectionName).Doc(attempt.ClassID))
	if status.Code(err) == codes.NotFound {
		return func() error { return tx.Delete(ref) }, nil
	}
	if err != nil {
		return nil, err
	}
	var class classrooms.TeacherClassDocument
	if err := classRow.DataTo(&class); err != nil {
		return nil, err
	}
	writeRefresh, err := repository.classRefreshWrite(tx, attempt.ClassID, class, now)
	if err != nil {
		return nil, err
	}
	return func() error {
		if err := tx.Delete(ref); err != nil {
			return err
		}
		return writeRefresh()
	}, nil
}

func (repository *FirestoreRepository) ClaimClassLearningRefresh(ctx context.Context, job LearningJob, owner string, now, until time.Time) (ClassLearningRefresh, bool, error) {
	var claim ClassLearningRefresh
	claimed := false
	if err := job.Validate(); err != nil || job.Kind != LearningJobClassRefresh || owner == "" || !until.After(now) {
		return claim, false, errors.New("invalid class refresh claim")
	}
	ref := repository.client.Collection(ClassRefreshCollection).Doc(job.ClassID)
	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		claim, claimed = ClassLearningRefresh{}, false
		row, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			return nil
		}
		if err != nil {
			return err
		}
		if err := row.DataTo(&claim); err != nil {
			return err
		}
		if !claim.pending() || claim.ClassID != job.ClassID || claim.Generation != job.ClassGeneration || claim.DispatchGeneration != job.DispatchGeneration {
			return nil
		}
		if claim.OwnerToken != "" && claim.LeaseUntil.After(now) {
			return ErrClassRefreshBusy
		}
		classRow, err := tx.Get(repository.client.Collection(classrooms.CollectionName).Doc(job.ClassID))
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
		valid := false
		if err == nil {
			var class classrooms.TeacherClassDocument
			if err := classRow.DataTo(&class); err != nil {
				return err
			}
			valid = sameEvidenceTenant(StudentMissionAttemptDocument{OrganizationID: claim.OrganizationID, DistrictID: claim.DistrictID, SchoolID: claim.SchoolID}, class)
		}
		if !valid {
			return tx.Set(ref, map[string]any{"completed_generation": claim.Generation, "owner_token": "", "lease_until": time.Time{}, "updated_at": now}, firestore.MergeAll)
		}
		claim.OwnerToken, claim.LeaseUntil, claim.UpdatedAt = owner, until, now
		if err := tx.Set(ref, map[string]any{"owner_token": owner, "lease_until": until, "updated_at": now}, firestore.MergeAll); err != nil {
			return err
		}
		claimed = true
		return nil
	})
	return claim, claimed, err
}

// Compare owner and source generation at the final write, after generation.
// Equal timestamps/counts never authorize a stale evidence set.
func (repository *FirestoreRepository) CompleteClassLearningRefresh(ctx context.Context, claim ClassLearningRefresh, document ClassLearningSynthesisDocument, now time.Time) error {
	ref := repository.client.Collection(ClassRefreshCollection).Doc(claim.ClassID)
	return repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		row, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			return ErrStaleClassRefresh
		}
		if err != nil {
			return err
		}
		var current ClassLearningRefresh
		if err := row.DataTo(&current); err != nil {
			return err
		}
		if !current.matches(claim) || document.ClassID != claim.ClassID || document.EvidenceGeneration != claim.Generation || document.EvidenceFingerprint == "" {
			return ErrStaleClassRefresh
		}
		if !current.LeaseUntil.After(now) {
			return ErrClassRefreshBusy // Retry this generation; it is not obsolete.
		}
		classRow, err := tx.Get(repository.client.Collection(classrooms.CollectionName).Doc(claim.ClassID))
		if status.Code(err) == codes.NotFound {
			return ErrStaleClassRefresh
		}
		if err != nil {
			return err
		}
		var class classrooms.TeacherClassDocument
		if err := classRow.DataTo(&class); err != nil {
			return err
		}
		if !sameEvidenceTenant(StudentMissionAttemptDocument{OrganizationID: claim.OrganizationID, DistrictID: claim.DistrictID, SchoolID: claim.SchoolID}, class) || claim.scope() != tenancy.NormalizeScope(tenancy.Scope{OrganizationID: document.OrganizationID, DistrictID: document.DistrictID, SchoolID: document.SchoolID}) {
			return ErrStaleClassRefresh
		}
		if err := tx.Set(repository.client.Collection(classLearningCollectionName).Doc(claim.ClassID), document); err != nil {
			return fmt.Errorf("save fenced class learning synthesis: %w", err)
		}
		return tx.Set(ref, map[string]any{"completed_generation": claim.Generation, "owner_token": "", "lease_until": time.Time{}, "updated_at": now}, firestore.MergeAll)
	})
}

func (repository *FirestoreRepository) FailClassLearningRefresh(ctx context.Context, claim ClassLearningRefresh, now time.Time) error {
	ref := repository.client.Collection(ClassRefreshCollection).Doc(claim.ClassID)
	return repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		row, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			return nil
		}
		if err != nil {
			return err
		}
		var current ClassLearningRefresh
		if err := row.DataTo(&current); err != nil {
			return err
		}
		if !current.matches(claim) {
			return nil
		}
		return tx.Set(ref, map[string]any{"owner_token": "", "lease_until": time.Time{}, "updated_at": now}, firestore.MergeAll)
	})
}

func (repository *FirestoreRepository) ListPendingClassLearningRefreshes(ctx context.Context, after string, limit int, now time.Time) (ClassRefreshPage, error) {
	page := ClassRefreshPage{}
	if limit < 1 || limit > 100 || len(after) > 120 || strings.ContainsAny(after, "/\x00") {
		return page, errors.New("class recovery requires limit 1..100 and a bounded document cursor")
	}
	query := repository.client.Collection(ClassRefreshCollection).OrderBy(firestore.DocumentID, firestore.Asc).Limit(limit + 1)
	if after != "" {
		query = query.StartAfter(after)
	}
	rows, err := query.Documents(ctx).GetAll()
	if err != nil {
		return page, err
	}
	if len(rows) > limit {
		rows = rows[:limit]
		page.NextCursor = rows[len(rows)-1].Ref.ID
	}
	page.Scanned = len(rows)
	for _, row := range rows {
		var refresh ClassLearningRefresh
		if err := row.DataTo(&refresh); err != nil {
			return page, err
		}
		if refresh.ClassID == row.Ref.ID && refresh.pending() && (refresh.OwnerToken == "" || !refresh.LeaseUntil.After(now)) {
			page.Refreshes = append(page.Refreshes, refresh)
		}
	}
	return page, nil
}

func (repository *FirestoreRepository) PrepareClassRefreshRedrive(ctx context.Context, expected ClassLearningRefresh, now time.Time) (ClassLearningRefresh, bool, error) {
	var current ClassLearningRefresh
	changed := false
	ref := repository.client.Collection(ClassRefreshCollection).Doc(expected.ClassID)
	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		current, changed = ClassLearningRefresh{}, false
		row, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			return nil
		}
		if err != nil {
			return err
		}
		if err := row.DataTo(&current); err != nil {
			return err
		}
		if !current.pending() || current.Generation != expected.Generation || current.DispatchGeneration != expected.DispatchGeneration || current.ClassID != expected.ClassID || current.OwnerToken != "" && current.LeaseUntil.After(now) {
			return nil
		}
		current.DispatchGeneration++
		current.OwnerToken, current.LeaseUntil, current.UpdatedAt = "", time.Time{}, now
		if err := tx.Set(ref, current); err != nil {
			return err
		}
		changed = true
		return nil
	})
	return current, changed, err
}
