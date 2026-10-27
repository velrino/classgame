package students

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
