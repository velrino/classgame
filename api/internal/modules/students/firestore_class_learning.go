package students

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (repository *FirestoreRepository) ListLatestFinalizedAttemptsByClassID(
	ctx context.Context,
	classID string,
	limit int,
) ([]StudentMissionAttemptDocument, error) {
	if limit <= 0 {
		limit = 100
	}
	iter := repository.client.Collection(studentAttemptsCollectionName).
		Where("class_id", "==", classID).
		Documents(ctx)
	defer iter.Stop()

	latest := make(map[string]StudentMissionAttemptDocument)
	for {
		snapshot, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterate class learning attempts: %w", err)
		}
		var attempt StudentMissionAttemptDocument
		if err := snapshot.DataTo(&attempt); err != nil {
			return nil, fmt.Errorf("decode class learning attempt %s: %w", snapshot.Ref.ID, err)
		}
		if !attempt.Trace.Finalized {
			continue
		}
		current, exists := latest[attempt.StudentID]
		if !exists || studentAttemptEvidenceTime(attempt).After(studentAttemptEvidenceTime(current)) {
			latest[attempt.StudentID] = attempt
		}
	}
	result := make([]StudentMissionAttemptDocument, 0, len(latest))
	for _, attempt := range latest {
		result = append(result, attempt)
	}
	sort.SliceStable(result, func(left int, right int) bool {
		return studentAttemptEvidenceTime(result[left]).After(studentAttemptEvidenceTime(result[right]))
	})
	if len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

func (repository *FirestoreRepository) SaveClassLearningSynthesis(
	ctx context.Context,
	document ClassLearningSynthesisDocument,
) error {
	ref := repository.client.Collection(classLearningCollectionName).Doc(document.ClassID)
	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snapshot, err := tx.Get(ref)
		if err == nil {
			var existing ClassLearningSynthesisDocument
			if err := snapshot.DataTo(&existing); err != nil {
				return fmt.Errorf("decode class learning synthesis: %w", err)
			}
			if existing.EvidenceUpdatedAt.After(document.EvidenceUpdatedAt) ||
				existing.EvidenceUpdatedAt.Equal(document.EvidenceUpdatedAt) &&
					existing.AttemptCount > document.AttemptCount {
				return nil
			}
		} else if status.Code(err) != codes.NotFound {
			return fmt.Errorf("read class learning synthesis: %w", err)
		}

		return tx.Set(ref, map[string]any{
			"organization_id":     document.OrganizationID,
			"district_id":         document.DistrictID,
			"school_id":           document.SchoolID,
			"class_id":            document.ClassID,
			"synthesis":           document.Synthesis,
			"attempt_count":       document.AttemptCount,
			"evidence_updated_at": document.EvidenceUpdatedAt,
			"updated_at":          document.UpdatedAt,
		}, firestore.MergeAll)
	})
	if err != nil {
		return fmt.Errorf("save class learning synthesis: %w", err)
	}

	return nil
}
