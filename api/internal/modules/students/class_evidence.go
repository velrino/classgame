package students

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"classgame-api/internal/modules/classrooms"
	"classgame-api/internal/modules/tenancy"
)

const ClassEvidenceCollection = "class_learning_students"

type ClassEvidenceEntry struct {
	Status            string    `firestore:"status"`
	MissionID         string    `firestore:"mission_id"`
	PublishedRevision int64     `firestore:"published_revision"`
	Version           int       `firestore:"version"`
	AttemptID         string    `firestore:"attempt_id"`
	StudentID         string    `firestore:"student_id"`
	StudentName       string    `firestore:"student_name"`
	ClassID           string    `firestore:"class_id"`
	OrganizationID    string    `firestore:"organization_id"`
	DistrictID        string    `firestore:"district_id"`
	SchoolID          string    `firestore:"school_id"`
	EvidenceRevision  int64     `firestore:"evidence_revision"`
	SnapshotRevision  int64     `firestore:"snapshot_revision"`
	EvidenceAt        time.Time `firestore:"evidence_at"`
	ProjectedAt       time.Time `firestore:"projected_at"`
	Strengths         []string  `firestore:"strengths"`
	ImprovementAreas  []string  `firestore:"improvement_areas"`
}

var classEvidenceSourceFields = []string{
	"id", "mission_id", "published_revision", "status", "student_id", "student_name", "class_id",
	"organization_id", "district_id", "school_id", "evidence_revision", "snapshot.revision",
	"last_activity_at", "updated_at", "trace.finalized", "trace.strengths", "trace.improvement_areas",
}

func classEvidence(attempt StudentMissionAttemptDocument, now time.Time) ClassEvidenceEntry {
	return ClassEvidenceEntry{
		Status: attempt.Status, MissionID: attempt.MissionID, PublishedRevision: attempt.PublishedRevision,
		Version: 1, AttemptID: attempt.ID, StudentID: attempt.StudentID, StudentName: attempt.StudentName,
		ClassID: attempt.ClassID, OrganizationID: attempt.OrganizationID,
		DistrictID: attempt.DistrictID, SchoolID: attempt.SchoolID,
		EvidenceRevision: attempt.EvidenceRevision, SnapshotRevision: attempt.Snapshot.Revision,
		EvidenceAt: studentAttemptEvidenceTime(attempt), ProjectedAt: now,
		Strengths: attempt.Trace.Strengths, ImprovementAreas: attempt.Trace.ImprovementAreas,
	}
}
func (entry ClassEvidenceEntry) matches(attempt StudentMissionAttemptDocument) bool {
	return entry.Version == 1 &&
		attempt.Trace.Finalized &&
		(attempt.Status == "completed" || attempt.Status == "abandoned") &&
		entry.Status == attempt.Status &&
		entry.MissionID == attempt.MissionID &&
		entry.PublishedRevision == attempt.PublishedRevision &&
		entry.AttemptID == attempt.ID &&
		entry.StudentID == attempt.StudentID &&
		entry.ClassID == attempt.ClassID &&
		entry.OrganizationID == attempt.OrganizationID &&
		entry.DistrictID == attempt.DistrictID &&
		entry.SchoolID == attempt.SchoolID &&
		entry.EvidenceRevision == attempt.EvidenceRevision &&
		entry.SnapshotRevision == attempt.Snapshot.Revision &&
		entry.EvidenceAt.Equal(studentAttemptEvidenceTime(attempt))
}
func (repository *FirestoreRepository) classEvidenceWrite(ctx context.Context, tx *firestore.Transaction, attempt StudentMissionAttemptDocument, now time.Time) (func() error, error) {
	ref := repository.client.Collection(ClassEvidenceCollection).Doc(attempt.ClassID).Collection("students").Doc(attempt.StudentID)
	incoming := classEvidence(attempt, now)
	classSnapshot, err := tx.Get(repository.client.Collection(classrooms.CollectionName).Doc(attempt.ClassID))
	if status.Code(err) == codes.NotFound {
		return func() error { return nil }, nil
	}
	if err != nil {
		return nil, err
	}
	var class classrooms.TeacherClassDocument
	if err := classSnapshot.DataTo(&class); err != nil {
		return nil, err
	}
	if !sameEvidenceTenant(attempt, class) {
		return func() error { return nil }, nil
	}
	snapshot, err := tx.Get(ref)
	if err == nil {
		var existing ClassEvidenceEntry
		if err := snapshot.DataTo(&existing); err != nil {
			return nil, err
		}
		// All reads precede writes, including a source check when the older rebuild
		// candidate would otherwise replace a newer student's projection.
		newer := existing.EvidenceAt.After(incoming.EvidenceAt) || existing.EvidenceAt.Equal(incoming.EvidenceAt) && (existing.AttemptID < incoming.AttemptID || existing.AttemptID == incoming.AttemptID && existing.EvidenceRevision >= incoming.EvidenceRevision)
		if newer && existing.ClassID == incoming.ClassID && existing.StudentID == incoming.StudentID && existing.AttemptID != "" && !strings.Contains(existing.AttemptID, "/") {
			current, sourceErr := tx.Get(repository.client.Collection(studentAttemptsCollectionName).Doc(existing.AttemptID))
			if sourceErr == nil {
				var source StudentMissionAttemptDocument
				if err := current.DataTo(&source); err != nil {
					return nil, err
				}
				source.ID = current.Ref.ID
				if existing.matches(source) {
					return func() error { return nil }, nil
				}
			} else if status.Code(sourceErr) != codes.NotFound {
				return nil, sourceErr
			}
		}
	} else if status.Code(err) != codes.NotFound {
		return nil, err
	}
	writeRefresh, err := repository.classRefreshWrite(tx, attempt.ClassID, class, now)
	if err != nil {
		return nil, err
	}
	return func() error {
		if err := tx.Set(ref, incoming); err != nil {
			return err
		}
		return writeRefresh()
	}, nil
}

type classEvidenceManifest struct {
	Version        int       `firestore:"version"`
	Generation     string    `firestore:"generation"`
	ClassID        string    `firestore:"class_id"`
	OrganizationID string    `firestore:"organization_id"`
	DistrictID     string    `firestore:"district_id"`
	SchoolID       string    `firestore:"school_id"`
	NextCursor     string    `firestore:"next_cursor"`
	Complete       bool      `firestore:"complete"`
	UpdatedAt      time.Time `firestore:"updated_at"`
}

func (repository *FirestoreRepository) ListLatestFinalizedAttemptsByClassID(ctx context.Context, classID string, limit int) ([]StudentMissionAttemptDocument, error) {
	if limit < 1 || limit > 100 {
		return nil, errors.New("class evidence limit must be 1 to 100")
	}
	classSnapshot, err := repository.client.Collection(classrooms.CollectionName).Doc(classID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return []StudentMissionAttemptDocument{}, nil
	}
	if err != nil {
		return nil, err
	}
	var class classrooms.TeacherClassDocument
	if err := classSnapshot.DataTo(&class); err != nil {
		return nil, err
	}
	manifestSnapshot, err := repository.client.Collection(ClassEvidenceCollection).Doc(classID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return repository.legacyClassEvidenceWindow(ctx, classID, limit, class)
	}
	if err != nil {
		return nil, err
	}
	var manifest classEvidenceManifest
	if err := manifestSnapshot.DataTo(&manifest); err != nil {
		return nil, err
	}
	if manifest.Version != 1 || !manifest.Complete || manifest.ClassID != classID || !sameEvidenceTenant(StudentMissionAttemptDocument{OrganizationID: manifest.OrganizationID, DistrictID: manifest.DistrictID, SchoolID: manifest.SchoolID}, class) {
		return repository.legacyClassEvidenceWindow(ctx, classID, limit, class)
	}
	rows, err := repository.client.Collection(ClassEvidenceCollection).Doc(classID).Collection("students").OrderBy("evidence_at", firestore.Desc).OrderBy(firestore.DocumentID, firestore.Asc).Limit(limit).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	entries := map[string]ClassEvidenceEntry{}
	ids := []string{}
	for _, row := range rows {
		var entry ClassEvidenceEntry
		if err := row.DataTo(&entry); err != nil {
			return nil, err
		}
		if entry.AttemptID == "" || strings.Contains(entry.AttemptID, "/") || entry.ClassID != classID || entry.StudentID != row.Ref.ID {
			continue
		}
		entries[entry.AttemptID] = entry
		ids = append(ids, entry.AttemptID)
	}
	result := []StudentMissionAttemptDocument{}
	// Validate source versions so a resumed attempt never remains finalized only
	// because its derived row has not yet been rebuilt. Field masks keep reads small.
	for start := 0; start < len(ids); start += 30 {
		end := min(start+30, len(ids))
		references := make([]*firestore.DocumentRef, 0, end-start)
		for _, id := range ids[start:end] {
			references = append(references, repository.client.Collection(studentAttemptsCollectionName).Doc(id))
		}
		sources, err := repository.client.Collection(studentAttemptsCollectionName).Where(firestore.DocumentID, "in", references).Select(classEvidenceSourceFields...).Documents(ctx).GetAll()
		if err != nil {
			return nil, err
		}
		for _, source := range sources {
			var attempt StudentMissionAttemptDocument
			if err := source.DataTo(&attempt); err != nil {
				return nil, err
			}
			attempt.ID = source.Ref.ID
			if entries[attempt.ID].matches(attempt) && sameEvidenceTenant(attempt, class) {
				result = append(result, attempt)
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := studentAttemptEvidenceTime(result[i]), studentAttemptEvidenceTime(result[j])
		if a.Equal(b) {
			return result[i].StudentID < result[j].StudentID
		}
		return a.After(b)
	})
	return result, nil
}
func (repository *FirestoreRepository) legacyClassEvidenceWindow(ctx context.Context, classID string, limit int, class classrooms.TeacherClassDocument) ([]StudentMissionAttemptDocument, error) {
	rows, err := repository.client.Collection(studentAttemptsCollectionName).Where("class_id", "==", classID).Where("trace.finalized", "==", true).OrderBy("last_activity_at", firestore.Desc).OrderBy(firestore.DocumentID, firestore.Asc).Limit(limit * 4).Select(classEvidenceSourceFields...).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	result := []StudentMissionAttemptDocument{}
	seen := map[string]bool{}
	for _, row := range rows {
		var attempt StudentMissionAttemptDocument
		if err := row.DataTo(&attempt); err != nil {
			return nil, err
		}
		attempt.ID = row.Ref.ID
		if seen[attempt.StudentID] || !sameEvidenceTenant(attempt, class) || (attempt.Status != "completed" && attempt.Status != "abandoned") {
			continue
		}
		seen[attempt.StudentID] = true
		result = append(result, attempt)
		if len(result) == limit {
			break
		}
	}
	return result, nil
}

type EvidenceRebuildPage struct {
	Scanned    int    `json:"scanned"`
	Candidates int    `json:"candidates"`
	NextCursor string `json:"next_cursor"`
	Apply      bool   `json:"apply"`
}

// RebuildClassEvidencePage is bounded and read-only unless explicitly applied.
func (repository *FirestoreRepository) RebuildClassEvidencePage(ctx context.Context, classID, after string, limit int, apply bool) (EvidenceRebuildPage, error) {
	result := EvidenceRebuildPage{Apply: apply}
	if classID == "" || strings.ContainsAny(classID+after, "/\x00") || len(classID) > 100 || len(after) > 150 || limit < 1 || limit > 100 {
		return result, errors.New("class, bounded cursor and limit 1 to 100 are required")
	}
	manifestRef := repository.client.Collection(ClassEvidenceCollection).Doc(classID)
	var manifest classEvidenceManifest
	if apply {
		if after == "" {
			seed := make([]byte, 16)
			if _, err := rand.Read(seed); err != nil {
				return result, err
			}
			err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
				classSnapshot, err := tx.Get(repository.client.Collection(classrooms.CollectionName).Doc(classID))
				if err != nil {
					return err
				}
				var class classrooms.TeacherClassDocument
				if err := classSnapshot.DataTo(&class); err != nil {
					return err
				}
				manifest = classEvidenceManifest{Version: 1, Generation: hex.EncodeToString(seed), ClassID: classID, OrganizationID: class.OrganizationID, DistrictID: class.DistrictID, SchoolID: class.SchoolID, UpdatedAt: time.Now()}
				writeRefresh, err := repository.classRefreshWrite(tx, classID, class, manifest.UpdatedAt)
				if err != nil {
					return err
				}
				if err := tx.Set(manifestRef, manifest); err != nil {
					return err
				}
				return writeRefresh()
			})
			if err != nil {
				return result, err
			}
		} else {
			snapshot, err := manifestRef.Get(ctx)
			if err != nil {
				return result, err
			}
			if err := snapshot.DataTo(&manifest); err != nil {
				return result, err
			}
			if manifest.Version != 1 || manifest.ClassID != classID || manifest.Complete || manifest.NextCursor != after {
				return result, errors.New("rebuild cursor does not match the active class rebuild")
			}
		}
	}
	query := repository.client.Collection(studentAttemptsCollectionName).Where("class_id", "==", classID).OrderBy(firestore.DocumentID, firestore.Asc).Limit(limit + 1).Select(classEvidenceSourceFields...)
	if after != "" {
		query = query.StartAfter(after)
	}
	rows, err := query.Documents(ctx).GetAll()
	if err != nil {
		return result, err
	}
	if len(rows) > limit {
		rows = rows[:limit]
		result.NextCursor = rows[len(rows)-1].Ref.ID
	}
	result.Scanned = len(rows)
	for _, row := range rows {
		var candidate StudentMissionAttemptDocument
		if err := row.DataTo(&candidate); err != nil {
			return result, err
		}
		candidate.ID = row.Ref.ID
		if !candidate.Trace.Finalized || candidate.StudentID == "" || (candidate.Status != "completed" && candidate.Status != "abandoned") {
			continue
		}
		if !apply {
			result.Candidates++
			continue
		}
		err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			snapshot, err := tx.Get(row.Ref)
			if status.Code(err) == codes.NotFound {
				return nil
			}
			if err != nil {
				return err
			}
			var current StudentMissionAttemptDocument
			if err := snapshot.DataTo(&current); err != nil {
				return err
			}
			current.ID = row.Ref.ID
			if !classEvidence(candidate, time.Time{}).matches(current) {
				return nil
			}
			write, err := repository.classEvidenceWrite(ctx, tx, current, time.Now())
			if err != nil {
				return err
			}
			return write()
		})
		if err != nil {
			return result, fmt.Errorf("rebuild class evidence: %w", err)
		}
		result.Candidates++
	}
	if apply {
		err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			snapshot, err := tx.Get(manifestRef)
			if err != nil {
				return err
			}
			var current classEvidenceManifest
			if err := snapshot.DataTo(&current); err != nil {
				return err
			}
			if current.Generation != manifest.Generation || current.NextCursor != after || current.Complete {
				return errors.New("another rebuild changed this class page; inspect again")
			}
			current.NextCursor = result.NextCursor
			current.Complete = result.NextCursor == ""
			current.UpdatedAt = time.Now()
			writeRefresh := func() error { return nil }
			if current.Complete {
				classRow, err := tx.Get(repository.client.Collection(classrooms.CollectionName).Doc(classID))
				if err != nil {
					return err
				}
				var class classrooms.TeacherClassDocument
				if err := classRow.DataTo(&class); err != nil {
					return err
				}
				if !sameEvidenceTenant(StudentMissionAttemptDocument{OrganizationID: current.OrganizationID, DistrictID: current.DistrictID, SchoolID: current.SchoolID}, class) {
					return errors.New("class scope changed during evidence rebuild")
				}
				writeRefresh, err = repository.classRefreshWrite(tx, classID, class, current.UpdatedAt)
				if err != nil {
					return err
				}
			}
			if err := tx.Set(manifestRef, current); err != nil {
				return err
			}
			return writeRefresh()
		})
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func sameEvidenceTenant(attempt StudentMissionAttemptDocument, class classrooms.TeacherClassDocument) bool {
	a := tenancy.NormalizeScope(tenancy.Scope{OrganizationID: attempt.OrganizationID, DistrictID: attempt.DistrictID, SchoolID: attempt.SchoolID})
	b := tenancy.NormalizeScope(tenancy.Scope{OrganizationID: class.OrganizationID, DistrictID: class.DistrictID, SchoolID: class.SchoolID})
	return a == b
}
