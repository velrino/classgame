package missions

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	attachmentsmodule "classgame-api/internal/modules/attachments"
	"classgame-api/internal/modules/classrooms"
	"classgame-api/internal/uploadcare"
)

func (repository *FirestoreRepository) Prepare(
	ctx context.Context,
	job attachmentsmodule.Job,
) (attachmentsmodule.TransferPlan, error) {
	if repository == nil || repository.client == nil {
		return attachmentsmodule.TransferPlan{}, errors.New("mission attachment repository is not configured")
	}

	plan := attachmentsmodule.TransferPlan{}

	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		revision, missionRef, revisionRef, exists, err := repository.findRevisionForTransfer(tx, job)
		if err != nil {
			return err
		}
		if !exists {
			return repository.classifyMissingRevision(tx, job.ClassID, job.MissionID, job.Revision)
		}
		if revision.Revision != job.Revision || revision.ClassID != job.ClassID || revision.MissionID != job.MissionID {
			return attachmentsmodule.ErrRevisionObsolete
		}

		mission, hasMission, err := readMissionForTransfer(tx, missionRef, job.ClassID)
		if err != nil {
			return err
		}

		now := time.Now()
		changed := false
		plan.Attachments = make([]attachmentsmodule.AttachmentSnapshot, 0, len(revision.Snapshot.Attachments))
		for index := range revision.Snapshot.Attachments {
			attachment := &revision.Snapshot.Attachments[index]
			if attachmentIsStored(*attachment) || attachmentFailedPermanently(*attachment) {
				continue
			}

			storage := copyAttachmentStorage(attachment.Storage)
			storage.Status = AttachmentStorageCopying
			storage.Attempts++
			storage.Retryable = true
			storage.Error = ""
			storage.UpdatedAt = now
			attachment.Storage = storage
			plan.Attachments = append(plan.Attachments, toTransferAttachment(*attachment))
			changed = true
		}

		if !changed {
			return nil
		}

		revision.AttachmentTransferStatus = AttachmentStorageCopying
		revision.AttachmentTransferAt = now
		if err := tx.Set(revisionRef, revision); err != nil {
			return fmt.Errorf("mark mission attachment revision copying: %w", err)
		}
		if hasMission {
			mirrorRevisionAttachments(&mission, job.Revision, revision.Snapshot.Attachments)
			if err := tx.Set(missionRef, mission.firestoreMap()); err != nil {
				return fmt.Errorf("mirror mission attachments copying: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return attachmentsmodule.TransferPlan{}, err
	}

	return plan, nil
}

func (repository *FirestoreRepository) MarkStored(
	ctx context.Context,
	job attachmentsmodule.Job,
	target attachmentsmodule.AttachmentSnapshot,
	stored attachmentsmodule.StoredObject,
) error {
	return repository.updateRevisionAttachment(ctx, job, target, func(attachment *MissionAttachment) {
		now := time.Now().UTC().Truncate(time.Microsecond)
		storage := copyAttachmentStorage(attachment.Storage)
		storage.Status = AttachmentStorageStored
		storage.Bucket = stored.Bucket
		storage.ObjectPath = stored.ObjectPath
		storage.GSURI = stored.GSURI
		storage.MimeType = stored.MIMEType
		storage.SizeBytes = stored.SizeBytes
		storage.Retryable = false
		storage.Error = ""
		storage.UpdatedAt = now
		storage.StoredAt = now
		attachment.Storage = storage
		if attachment.UploadcareCDNURL == "" {
			attachment.UploadcareCDNURL = uploadcareCDNURL(attachment.CDNURL)
		}
		if storageURL := AttachmentStoragePublicURL(storage); storageURL != "" {
			attachment.CDNURL = storageURL
		}
	})
}

func (repository *FirestoreRepository) MarkFailed(
	ctx context.Context,
	job attachmentsmodule.Job,
	target attachmentsmodule.AttachmentSnapshot,
	message string,
	retryable bool,
) error {
	return repository.updateRevisionAttachment(ctx, job, target, func(attachment *MissionAttachment) {
		storage := copyAttachmentStorage(attachment.Storage)
		storage.Status = AttachmentStorageFailed
		storage.Retryable = retryable
		storage.Error = message
		storage.UpdatedAt = time.Now()
		attachment.Storage = storage
	})
}

func (repository *FirestoreRepository) Complete(ctx context.Context, job attachmentsmodule.Job) error {
	if repository == nil || repository.client == nil {
		return errors.New("mission attachment repository is not configured")
	}

	return repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		revision, _, revisionRef, exists, err := repository.findRevisionForTransfer(tx, job)
		if err != nil {
			return err
		}
		if !exists {
			return attachmentsmodule.ErrRevisionObsolete
		}
		transferStatus := "complete"
		for _, attachment := range revision.Snapshot.Attachments {
			if attachment.Storage != nil && attachment.Storage.Status == AttachmentStorageFailed {
				transferStatus = "complete_with_failures"
			}
			if !attachmentIsStored(attachment) && !attachmentFailedPermanently(attachment) {
				return errors.New("mission attachment revision still has pending transfers")
			}
		}

		revision.AttachmentTransferStatus = transferStatus
		revision.AttachmentTransferAt = time.Now()
		if err := tx.Set(revisionRef, revision); err != nil {
			return fmt.Errorf("complete mission attachment revision: %w", err)
		}

		return nil
	})
}

func (repository *FirestoreRepository) updateRevisionAttachment(
	ctx context.Context,
	job attachmentsmodule.Job,
	target attachmentsmodule.AttachmentSnapshot,
	update func(*MissionAttachment),
) error {
	if repository == nil || repository.client == nil {
		return errors.New("mission attachment repository is not configured")
	}

	return repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		revision, missionRef, revisionRef, exists, err := repository.findRevisionForTransfer(tx, job)
		if err != nil {
			return err
		}
		if !exists {
			return attachmentsmodule.ErrRevisionObsolete
		}

		attachmentIndex := findTransferAttachment(revision.Snapshot.Attachments, target)
		if attachmentIndex < 0 {
			return attachmentsmodule.ErrRevisionObsolete
		}

		mission, hasMission, err := readMissionForTransfer(tx, missionRef, job.ClassID)
		if err != nil {
			return err
		}

		update(&revision.Snapshot.Attachments[attachmentIndex])
		revision.AttachmentTransferAt = time.Now()
		if err := tx.Set(revisionRef, revision); err != nil {
			return fmt.Errorf("update mission attachment revision: %w", err)
		}
		if hasMission {
			mirrorRevisionAttachments(&mission, job.Revision, revision.Snapshot.Attachments)
			if err := tx.Set(missionRef, mission.firestoreMap()); err != nil {
				return fmt.Errorf("mirror mission attachment update: %w", err)
			}
		}

		return nil
	})
}

func (repository *FirestoreRepository) findRevisionForTransfer(
	tx *firestore.Transaction,
	job attachmentsmodule.Job,
) (MissionPublishedRevisionDocument, *firestore.DocumentRef, *firestore.DocumentRef, bool, error) {
	for _, refs := range repository.missionRevisionRefs(job.ClassID, job.MissionID, job.Revision) {
		snapshot, err := tx.Get(refs.revisionRef)
		if status.Code(err) == codes.NotFound {
			continue
		}
		if err != nil {
			return MissionPublishedRevisionDocument{}, nil, nil, false, fmt.Errorf("read mission attachment revision: %w", err)
		}

		var revision MissionPublishedRevisionDocument
		if err := snapshot.DataTo(&revision); err != nil {
			return MissionPublishedRevisionDocument{}, nil, nil, false, fmt.Errorf("decode mission attachment revision: %w", err)
		}
		hydratePublishedRevision(&revision, job.ClassID, job.MissionID)
		if revision.Revision != job.Revision || revision.ClassID != job.ClassID || revision.MissionID != job.MissionID {
			return MissionPublishedRevisionDocument{}, nil, nil, false, attachmentsmodule.ErrRevisionObsolete
		}

		return revision, refs.missionRef, refs.revisionRef, true, nil
	}

	return MissionPublishedRevisionDocument{}, nil, nil, false, nil
}

func (repository *FirestoreRepository) classifyMissingRevision(
	tx *firestore.Transaction,
	classID string,
	missionID string,
	revision int64,
) error {
	for _, missionRef := range repository.missionRefs(classID, missionID) {
		mission, exists, err := readMissionForTransfer(tx, missionRef, classID)
		if err != nil {
			return err
		}
		if !exists || mission.Status == StatusDeleted || mission.PublishedRevision >= revision || mission.DraftRevision > revision {
			continue
		}

		return attachmentsmodule.ErrRevisionPending
	}

	return attachmentsmodule.ErrRevisionObsolete
}

func readMissionForTransfer(
	tx *firestore.Transaction,
	missionRef *firestore.DocumentRef,
	classID string,
) (MissionDocument, bool, error) {
	snapshot, err := tx.Get(missionRef)
	if status.Code(err) == codes.NotFound {
		return MissionDocument{}, false, nil
	}
	if err != nil {
		return MissionDocument{}, false, fmt.Errorf("read mission for attachment transfer: %w", err)
	}

	mission, belongs, err := missionFromSnapshot(snapshot, classID)
	if err != nil {
		return MissionDocument{}, false, fmt.Errorf("decode mission for attachment transfer: %w", err)
	}
	if !belongs {
		return MissionDocument{}, false, nil
	}

	return mission, true, nil
}

func (repository *FirestoreRepository) missionRef(classID string, missionID string) *firestore.DocumentRef {
	return repository.client.Collection(MissionsCollectionName).Doc(missionID)
}

func (repository *FirestoreRepository) legacyMissionRef(classID string, missionID string) *firestore.DocumentRef {
	return repository.client.Collection(classrooms.CollectionName).
		Doc(classID).
		Collection(MissionsCollectionName).
		Doc(missionID)
}

func (repository *FirestoreRepository) missionRefs(classID string, missionID string) []*firestore.DocumentRef {
	return []*firestore.DocumentRef{
		repository.missionRef(classID, missionID),
		repository.legacyMissionRef(classID, missionID),
	}
}

type missionRevisionRefs struct {
	missionRef  *firestore.DocumentRef
	revisionRef *firestore.DocumentRef
}

func (repository *FirestoreRepository) missionRevisionRefs(
	classID string,
	missionID string,
	revision int64,
) []missionRevisionRefs {
	revisionID := strconv.FormatInt(revision, 10)
	refs := repository.missionRefs(classID, missionID)
	output := make([]missionRevisionRefs, 0, len(refs))
	for _, missionRef := range refs {
		output = append(output, missionRevisionRefs{
			missionRef:  missionRef,
			revisionRef: missionRef.Collection(MissionRevisionsCollectionName).Doc(revisionID),
		})
	}

	return output
}

func (repository *FirestoreRepository) legacyClassMissionsQuery(classID string) *firestore.CollectionRef {
	return repository.client.Collection(classrooms.CollectionName).
		Doc(classID).
		Collection(MissionsCollectionName)
}

func mirrorRevisionAttachments(
	mission *MissionDocument,
	revision int64,
	attachments []MissionAttachment,
) {
	if mission.Published != nil && mission.PublishedRevision == revision {
		mission.Published.Attachments = attachments
	}
	if mission.Draft != nil && mission.DraftRevision == revision {
		mission.Draft.Attachments = attachments
		mission.Attachments = attachments
	}
}

func toTransferAttachment(attachment MissionAttachment) attachmentsmodule.AttachmentSnapshot {
	return attachmentsmodule.AttachmentSnapshot{
		UploadcareUUID: attachment.UploadcareUUID,
		SourceURL:      attachment.CDNURL,
		FileName:       attachment.FileName,
		MIMEType:       attachment.MimeType,
		SizeBytes:      attachment.SizeBytes,
	}
}

func findTransferAttachment(
	attachments []MissionAttachment,
	target attachmentsmodule.AttachmentSnapshot,
) int {
	for index, attachment := range attachments {
		if target.UploadcareUUID != "" && attachment.UploadcareUUID == target.UploadcareUUID {
			return index
		}
		if target.UploadcareUUID == "" && attachment.CDNURL == target.SourceURL && attachment.FileName == target.FileName {
			return index
		}
	}

	return -1
}

func copyAttachmentStorage(storage *MissionAttachmentStorage) *MissionAttachmentStorage {
	if storage == nil {
		return &MissionAttachmentStorage{}
	}

	copy := *storage
	return &copy
}

func uploadcareCDNURL(rawURL string) string {
	trimmedURL := strings.TrimSpace(rawURL)
	if trimmedURL == "" {
		return ""
	}

	parsedURL, err := url.Parse(trimmedURL)
	if err != nil || uploadcare.ValidateCDNURL(parsedURL) != nil {
		return ""
	}

	return trimmedURL
}

func attachmentIsStored(attachment MissionAttachment) bool {
	return attachment.Storage != nil &&
		attachment.Storage.Status == AttachmentStorageStored &&
		attachment.Storage.GSURI != ""
}

func attachmentFailedPermanently(attachment MissionAttachment) bool {
	return attachment.Storage != nil &&
		attachment.Storage.Status == AttachmentStorageFailed &&
		!attachment.Storage.Retryable
}
