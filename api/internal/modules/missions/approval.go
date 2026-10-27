package missions

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
)

type missionApprovalPlan struct {
	Revision       int64
	ReusePublished bool
	ContentChanged bool
	Edits          MissionDraftEdits
	Receipt        MissionApprovalReceipt
}

// planMissionApproval is shared by queue preflight and the publication
// transaction. The transaction remains authoritative when concurrent requests
// planned against the same draft.
func planMissionApproval(mission MissionDocument, expectedRevision int64, edits MissionDraftEdits) (missionApprovalPlan, error) {
	if expectedRevision <= 0 {
		return missionApprovalPlan{}, ErrMissionDraftRevisionRequired
	}
	if mission.Draft == nil {
		return missionApprovalPlan{}, ErrMissionDraftNotFound
	}
	edits = normalizeDraftEdits(edits)
	fingerprint := missionApprovalFingerprint(edits)
	alreadyPublished := mission.Published != nil &&
		mission.PublishedRevision == mission.DraftRevision && !mission.HasUnpublishedChanges
	if mission.DraftRevision != expectedRevision {
		// A receipt permits only the exact most recent approval to replay while
		// its published content is still the current draft. Other stale intent
		// must reload instead of changing a revision-pinned publication.
		receipt := mission.ApprovalReceipt
		if alreadyPublished && receipt != nil &&
			receipt.ExpectedDraftRevision == expectedRevision &&
			receipt.PublishedRevision == mission.PublishedRevision &&
			receipt.Fingerprint == fingerprint {
			return missionApprovalPlan{Revision: mission.PublishedRevision, ReusePublished: true}, nil
		}
		return missionApprovalPlan{}, ErrMissionDraftConflict
	}

	changed := edits.HasChanges && fingerprint != missionApprovalFingerprint(MissionDraftEdits{
		Title: mission.Draft.Title, Summary: mission.Draft.Summary,
		PreviewQuestions: mission.Draft.PreviewQuestions, HasChanges: true,
	})
	revision := mission.DraftRevision
	if changed {
		if revision == math.MaxInt64 {
			return missionApprovalPlan{}, ErrMissionDraftConflict
		}
		revision++
	}
	return missionApprovalPlan{
		Revision: revision, ReusePublished: alreadyPublished && !changed,
		ContentChanged: changed, Edits: edits,
		Receipt: MissionApprovalReceipt{ExpectedDraftRevision: expectedRevision, PublishedRevision: revision, Fingerprint: fingerprint},
	}, nil
}

func missionApprovalFingerprint(edits MissionDraftEdits) string {
	// Normalized draft edits contain only strings, slices and a boolean, so
	// JSON encoding cannot fail and provides a stable field order.
	encoded, _ := json.Marshal(normalizeDraftEdits(edits))
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}
