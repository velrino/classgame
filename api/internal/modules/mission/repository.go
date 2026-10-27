package mission

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	teacherclass "classgame-api/internal/modules/teacher-class"
	"classgame-api/internal/modules/tenant"
)

type Repository struct {
	client *firestore.Client
}

func NewRepository(client *firestore.Client) *Repository {
	return &Repository{
		client: client,
	}
}

func (repository *Repository) Create(
	ctx context.Context,
	input CreateMissionInput,
) (MissionDocument, error) {
	missionRef := repository.client.Collection(MissionsCollectionName).NewDoc()
	var codeRef *firestore.DocumentRef
	if input.MissionCode != "" {
		codeRef = repository.client.Collection(MissionCodesCollectionName).
			Doc(input.MissionCode)
	}

	mission := buildMissionDocument(missionRef.ID, input)
	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if codeRef != nil {
			_, err := tx.Get(codeRef)
			switch {
			case err == nil:
				return ErrMissionCodeUnavailable
			case status.Code(err) != codes.NotFound:
				return fmt.Errorf("check mission code: %w", err)
			}
		}

		if err := tx.Create(missionRef, mission.firestoreMap()); err != nil {
			return fmt.Errorf("create mission: %w", err)
		}
		if mission.Published != nil {
			revisionRef := missionRef.Collection(MissionRevisionsCollectionName).
				Doc(strconv.FormatInt(mission.PublishedRevision, 10))
			if err := tx.Create(revisionRef, publishedRevisionMap(mission, input.Teacher)); err != nil {
				return fmt.Errorf("create published mission revision: %w", err)
			}
		}

		if codeRef != nil {
			return tx.Create(codeRef, missionCodeFirestoreMap(mission))
		}

		return nil
	})
	if err != nil {
		return MissionDocument{}, fmt.Errorf("write mission: %w", err)
	}

	return mission, nil
}

func (repository *Repository) GetByCode(
	ctx context.Context,
	missionCode string,
) (MissionDocument, bool, error) {
	codeSnapshot, err := repository.client.Collection(MissionCodesCollectionName).
		Doc(normalizeCode(missionCode)).
		Get(ctx)
	if status.Code(err) == codes.NotFound {
		return MissionDocument{}, false, nil
	}
	if err != nil {
		return MissionDocument{}, false, fmt.Errorf("read mission code: %w", err)
	}

	var code missionCodeDocument
	if err := codeSnapshot.DataTo(&code); err != nil {
		return MissionDocument{}, false, fmt.Errorf("decode mission code: %w", err)
	}

	mission, exists, err := repository.Get(ctx, code.ClassID, code.MissionID)
	if err != nil || !exists {
		return mission, exists, err
	}

	published, exists := publishedView(mission)
	return published, exists, nil
}

func (repository *Repository) Get(
	ctx context.Context,
	classID string,
	missionID string,
) (MissionDocument, bool, error) {
	for _, ref := range repository.missionRefs(classID, missionID) {
		mission, exists, err := repository.getMissionAtRef(ctx, ref, classID)
		if err != nil || exists {
			return mission, exists, err
		}
	}

	return MissionDocument{}, false, nil
}

func (repository *Repository) List(
	ctx context.Context,
	classID string,
	limit int,
) ([]MissionDocument, error) {
	if limit <= 0 {
		limit = 10
	}

	byID := make(map[string]MissionDocument)
	if err := collectMissionDocuments(
		repository.legacyClassMissionsQuery(classID).Documents(ctx),
		classID,
		byID,
	); err != nil {
		return nil, err
	}
	if err := collectMissionDocuments(
		repository.client.Collection(MissionsCollectionName).
			Where("class_id", "==", classID).
			Documents(ctx),
		classID,
		byID,
	); err != nil {
		return nil, err
	}

	missions := make([]MissionDocument, 0, len(byID))
	for _, mission := range byID {
		missions = append(missions, mission)
	}
	sort.SliceStable(missions, func(leftIndex, rightIndex int) bool {
		left := missionListSortTime(missions[leftIndex])
		right := missionListSortTime(missions[rightIndex])
		if left.Equal(right) {
			return missions[leftIndex].ID < missions[rightIndex].ID
		}

		return left.After(right)
	})
	if len(missions) > limit {
		missions = missions[:limit]
	}

	return missions, nil
}

func (repository *Repository) UpdateDraft(
	ctx context.Context,
	input CreateMissionInput,
) (MissionDocument, error) {
	if input.ExpectedDraftRevision <= 0 {
		return MissionDocument{}, ErrMissionDraftRevisionRequired
	}

	var mission MissionDocument
	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		existing, ref, exists, err := repository.getMissionInTransaction(tx, input.ClassID, input.MissionID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrMissionNotFound
		}
		mission = existing
		if mission.CreatedBy.ProviderUserID != input.Teacher.ProviderUserID {
			return teacherclass.ErrClassOwnedByAnotherTeacher
		}
		if mission.Status == StatusDeleted {
			return ErrMissionDeleted
		}
		if input.ExpectedDraftRevision > 0 && mission.DraftRevision != input.ExpectedDraftRevision {
			return ErrMissionDraftConflict
		}

		nextRevision := mission.DraftRevision + 1
		input.Attachments = inheritStoredAttachments(input.Attachments, mission)
		input.Scope = tenant.NormalizeScope(input.Scope)
		applyMissionTenantScope(&mission, input.Scope, input.Teacher.ProviderUserID)
		draft := newSnapshotFromInput(input, nextRevision)
		mission.Draft = &draft
		mission.DraftRevision = nextRevision
		mission.HasUnpublishedChanges = mission.Published == nil ||
			mission.PublishedRevision != nextRevision
		mission.ClassName = input.ClassName
		mission.applySnapshot(draft)
		mission.Status = StatusDraft

		return tx.Set(ref, mission.firestoreMap())
	})
	if err != nil {
		return MissionDocument{}, fmt.Errorf("update mission draft: %w", err)
	}

	return mission, nil
}

func (repository *Repository) Publish(
	ctx context.Context,
	classID string,
	missionID string,
	teacher teacherclass.TeacherSummary,
	expectedDraftRevision int64,
	draftEdits MissionDraftEdits,
	timestamp time.Time,
) (MissionDocument, error) {
	if expectedDraftRevision <= 0 {
		return MissionDocument{}, ErrMissionDraftRevisionRequired
	}

	classRef := repository.client.Collection(teacherclass.CollectionName).Doc(classID)
	var mission MissionDocument
	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		existing, ref, exists, err := repository.getMissionInTransaction(tx, classID, missionID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrMissionNotFound
		}
		mission = existing
		if mission.CreatedBy.ProviderUserID != teacher.ProviderUserID {
			return teacherclass.ErrClassOwnedByAnotherTeacher
		}
		if mission.Status == StatusDeleted {
			return ErrMissionDeleted
		}
		if mission.Draft == nil {
			return ErrMissionDraftNotFound
		}
		if expectedDraftRevision > 0 && mission.DraftRevision != expectedDraftRevision {
			return ErrMissionDraftConflict
		}
		applyDraftEdits(&mission, draftEdits, teacher, timestamp)

		classSnapshot, err := tx.Get(classRef)
		if status.Code(err) == codes.NotFound {
			return teacherclass.ErrClassNotFound
		}
		if err != nil {
			return fmt.Errorf("read teacher class: %w", err)
		}
		var teacherClass teacherclass.TeacherClassDocument
		if err := classSnapshot.DataTo(&teacherClass); err != nil {
			return fmt.Errorf("decode teacher class: %w", err)
		}
		if teacherClass.TeacherID != teacher.ProviderUserID {
			return teacherclass.ErrClassOwnedByAnotherTeacher
		}
		applyMissionTenantScope(&mission, tenant.Scope{
			OrganizationID: teacherClass.OrganizationID,
			DistrictID:     teacherClass.DistrictID,
			SchoolID:       teacherClass.SchoolID,
		}, teacher.ProviderUserID)

		firstPublication := mission.Published == nil
		alreadyPublished := mission.Published != nil &&
			mission.PublishedRevision == mission.DraftRevision &&
			!mission.HasUnpublishedChanges &&
			!draftEdits.HasChanges
		if alreadyPublished {
			publishedMission, published := publishedView(mission)
			if !published {
				return ErrMissionNotFound
			}
			if err := tx.Set(classRef, map[string]any{
				"active_mission_id": mission.ID,
				"updated_at":        timestamp,
			}, firestore.MergeAll); err != nil {
				return fmt.Errorf("activate published mission: %w", err)
			}
			if err := repository.syncStudentMissionCatalog(
				tx,
				publishedMission,
			); err != nil {
				return fmt.Errorf("write student mission catalog: %w", err)
			}

			return nil
		}

		publishedAt := timestamp

		published := *mission.Draft
		prepareAttachmentsForPublish(published.Attachments, timestamp)
		mission.Published = &published
		mission.PublishedRevision = published.Revision
		mission.HasUnpublishedChanges = false
		mission.Status = StatusPublished
		mission.PublishedAt = publishedAt
		mission.applySnapshot(published)
		mission.UpdatedBy = teacher
		mission.UpdatedAt = timestamp

		revisionRef := ref.Collection(MissionRevisionsCollectionName).
			Doc(strconv.FormatInt(mission.PublishedRevision, 10))
		if err := tx.Set(revisionRef, publishedRevisionMap(mission, teacher)); err != nil {
			return fmt.Errorf("write published mission revision: %w", err)
		}

		if err := tx.Set(ref, mission.firestoreMap()); err != nil {
			return err
		}
		if err := repository.syncStudentMissionCatalog(tx, mission); err != nil {
			return fmt.Errorf("write student mission catalog: %w", err)
		}

		classUpdate := map[string]any{
			"active_mission_id": mission.ID,
			"updated_at":        timestamp,
		}
		if firstPublication {
			classUpdate["mission_count"] = firestore.Increment(1)
		}
		if err := tx.Set(classRef, classUpdate, firestore.MergeAll); err != nil {
			return fmt.Errorf("activate published mission: %w", err)
		}

		if mission.MissionCode == "" {
			return nil
		}

		codeRef := repository.client.Collection(MissionCodesCollectionName).
			Doc(mission.MissionCode)
		return tx.Set(codeRef, map[string]any{
			"organization_id": mission.OrganizationID,
			"district_id":     mission.DistrictID,
			"school_id":       mission.SchoolID,
			"teacher_id":      mission.TeacherID,
			"status":          mission.Status,
			"updated_at":      mission.UpdatedAt,
		}, firestore.MergeAll)
	})
	if err != nil {
		return MissionDocument{}, fmt.Errorf("publish mission: %w", err)
	}

	return mission, nil
}

func applyDraftEdits(
	mission *MissionDocument,
	edits MissionDraftEdits,
	teacher teacherclass.TeacherSummary,
	timestamp time.Time,
) {
	if mission == nil || mission.Draft == nil || !edits.HasChanges {
		return
	}

	draft := *mission.Draft
	draft.Title = edits.Title
	draft.Summary = edits.Summary
	draft.PreviewQuestions = edits.PreviewQuestions
	draft.UpdatedBy = teacher
	draft.UpdatedAt = timestamp
	mission.Draft = &draft
	mission.HasUnpublishedChanges = true
	mission.UpdatedBy = teacher
	mission.UpdatedAt = timestamp
	mission.applySnapshot(draft)
}

func (repository *Repository) GetPublishedRevision(
	ctx context.Context,
	classID string,
	missionID string,
	revision int64,
) (MissionPublishedRevisionDocument, bool, error) {
	if revision <= 0 {
		return MissionPublishedRevisionDocument{}, false, nil
	}

	for _, refs := range repository.missionRevisionRefs(classID, missionID, revision) {
		snapshot, err := refs.revisionRef.Get(ctx)
		if status.Code(err) == codes.NotFound {
			continue
		}
		if err != nil {
			return MissionPublishedRevisionDocument{}, false, fmt.Errorf("read published mission revision: %w", err)
		}

		var publishedRevision MissionPublishedRevisionDocument
		if err := snapshot.DataTo(&publishedRevision); err != nil {
			return MissionPublishedRevisionDocument{}, false, fmt.Errorf("decode published mission revision: %w", err)
		}
		hydratePublishedRevision(&publishedRevision, classID, missionID)

		return publishedRevision, true, nil
	}

	return MissionPublishedRevisionDocument{}, false, nil
}

func (repository *Repository) SoftDelete(
	ctx context.Context,
	classID string,
	missionID string,
	teacher teacherclass.TeacherSummary,
	timestamp time.Time,
) (MissionDocument, error) {
	classRef := repository.client.Collection(teacherclass.CollectionName).Doc(classID)

	var mission MissionDocument
	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		existing, missionRef, exists, err := repository.getMissionInTransaction(tx, classID, missionID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrMissionNotFound
		}
		mission = existing
		if mission.CreatedBy.ProviderUserID != teacher.ProviderUserID {
			return teacherclass.ErrClassOwnedByAnotherTeacher
		}

		var teacherClass teacherclass.TeacherClassDocument
		classSnapshot, classErr := tx.Get(classRef)
		if classErr == nil {
			if err := classSnapshot.DataTo(&teacherClass); err != nil {
				return fmt.Errorf("decode teacher class: %w", err)
			}
		} else if status.Code(classErr) != codes.NotFound {
			return fmt.Errorf("read teacher class: %w", classErr)
		}

		if mission.Status != StatusDeleted {
			mission.StatusBeforeDelete = mission.Status
			mission.DeletedAt = timestamp
			mission.DeletedBy = teacher
		}
		mission.Status = StatusDeleted
		mission.UpdatedAt = timestamp
		mission.UpdatedBy = teacher

		if err := tx.Set(missionRef, mission.firestoreMap()); err != nil {
			return fmt.Errorf("soft delete mission: %w", err)
		}
		if err := tx.Delete(repository.studentMissionCatalogRef(classID, mission.ID)); err != nil {
			return fmt.Errorf("delete student mission catalog: %w", err)
		}
		if teacherClass.ActiveMissionID == mission.ID {
			if err := tx.Set(classRef, map[string]any{
				"active_mission_id": "",
				"updated_at":        timestamp,
			}, firestore.MergeAll); err != nil {
				return fmt.Errorf("clear active mission: %w", err)
			}
		}
		if mission.MissionCode != "" {
			codeRef := repository.client.Collection(MissionCodesCollectionName).Doc(mission.MissionCode)
			if err := tx.Set(codeRef, map[string]any{
				"status":     StatusDeleted,
				"updated_at": timestamp,
			}, firestore.MergeAll); err != nil {
				return fmt.Errorf("disable mission code: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return MissionDocument{}, fmt.Errorf("delete mission: %w", err)
	}

	return mission, nil
}

type missionCodeDocument struct {
	OrganizationID string `firestore:"organization_id"`
	DistrictID     string `firestore:"district_id"`
	SchoolID       string `firestore:"school_id"`
	TeacherID      string `firestore:"teacher_id"`
	MissionID      string `firestore:"mission_id"`
	ClassID        string `firestore:"class_id"`
}

func buildMissionDocument(id string, input CreateMissionInput) MissionDocument {
	status := input.Status
	if status == "" || status == StatusPreview {
		status = StatusDraft
	}

	mission := buildMissionDocumentFields(input)
	mission.ID = id
	mission.MissionCode = input.MissionCode
	mission.ClassID = input.ClassID
	mission.ClassCode = input.ClassCode
	mission.ClassName = input.ClassName
	mission.Status = status
	mission.CreatedBy = input.Teacher
	mission.UpdatedBy = input.Teacher
	mission.PublishedAt = input.PublishedAt
	mission.CreatedAt = input.Timestamp
	mission.UpdatedAt = input.Timestamp
	mission.DraftRevision = initialMissionRevision
	draft := snapshotFromMission(mission, mission.DraftRevision)
	mission.Draft = &draft
	mission.HasUnpublishedChanges = true

	if status == StatusPublished {
		published := draft
		mission.Published = &published
		mission.PublishedRevision = published.Revision
		mission.HasUnpublishedChanges = false
		mission.PublishedAt = nonZeroTime(input.PublishedAt, input.Timestamp)
	}

	return mission
}

func buildMissionDocumentFields(input CreateMissionInput) MissionDocument {
	blueprint := input.Blueprint
	scope := tenant.NormalizeScope(input.Scope)

	return MissionDocument{
		OrganizationID:           scope.OrganizationID,
		DistrictID:               scope.DistrictID,
		SchoolID:                 scope.SchoolID,
		TeacherID:                input.Teacher.ProviderUserID,
		MissionName:              input.MissionName,
		Title:                    blueprint.Title,
		Summary:                  blueprint.Summary,
		Subject:                  blueprint.Subject,
		GradeLevel:               blueprint.GradeLevel,
		LearningGoal:             blueprint.LearningGoal,
		TeacherPrompt:            input.TeacherPrompt,
		AudienceType:             input.AudienceType,
		TargetStudentIDs:         input.TargetStudentIDs,
		Attachments:              input.Attachments,
		PreviewQuestions:         blueprint.PreviewQuestions,
		KnowledgeChunks:          blueprint.KnowledgeChunks,
		SuggestedDurationMinutes: blueprint.SuggestedDurationMinutes,
		Stages:                   blueprint.Stages,
		Suggestions:              blueprint.Suggestions,
		PositiveSignals:          blueprint.PositiveSignals,
		ImprovementSignals:       blueprint.ImprovementSignals,
		EducationGuardrail:       blueprint.EducationGuardrail,
		MissionFlow:              blueprint.MissionFlow,
		Orchestration: MissionOrchestration{
			Provider:             blueprint.Provider,
			Model:                blueprint.Model,
			ModelVersion:         blueprint.ModelVersion,
			GeneratedByAI:        blueprint.GeneratedByAI,
			GeneratedByMock:      blueprint.GeneratedByMock,
			FallbackReason:       blueprint.FallbackReason,
			PromptTokenCount:     blueprint.Usage.PromptTokenCount,
			CandidatesTokenCount: blueprint.Usage.CandidatesTokenCount,
			ThoughtsTokenCount:   blueprint.Usage.ThoughtsTokenCount,
			TotalTokenCount:      blueprint.Usage.TotalTokenCount,
		},
	}
}

func (mission MissionDocument) firestoreMap() map[string]any {
	return map[string]any{
		"id":                         mission.ID,
		"mission_code":               mission.MissionCode,
		"organization_id":            mission.OrganizationID,
		"district_id":                mission.DistrictID,
		"school_id":                  mission.SchoolID,
		"teacher_id":                 mission.TeacherID,
		"class_id":                   mission.ClassID,
		"class_code":                 mission.ClassCode,
		"class_name":                 mission.ClassName,
		"mission_name":               mission.MissionName,
		"title":                      mission.Title,
		"summary":                    mission.Summary,
		"subject":                    mission.Subject,
		"grade_level":                mission.GradeLevel,
		"learning_goal":              mission.LearningGoal,
		"teacher_prompt":             mission.TeacherPrompt,
		"audience_type":              mission.AudienceType,
		"target_student_ids":         mission.TargetStudentIDs,
		"attachments":                mission.Attachments,
		"preview_questions":          mission.PreviewQuestions,
		"knowledge_chunks":           mission.KnowledgeChunks,
		"suggested_duration_minutes": mission.SuggestedDurationMinutes,
		"stages":                     mission.Stages,
		"suggestions":                mission.Suggestions,
		"positive_signals":           mission.PositiveSignals,
		"improvement_signals":        mission.ImprovementSignals,
		"education_guardrail":        mission.EducationGuardrail,
		"mission_flow":               mission.MissionFlow,
		"orchestration": map[string]any{
			"provider":               mission.Orchestration.Provider,
			"model":                  mission.Orchestration.Model,
			"model_version":          mission.Orchestration.ModelVersion,
			"generated_by_ai":        mission.Orchestration.GeneratedByAI,
			"generated_by_mock":      mission.Orchestration.GeneratedByMock,
			"fallback_reason":        mission.Orchestration.FallbackReason,
			"prompt_token_count":     mission.Orchestration.PromptTokenCount,
			"candidates_token_count": mission.Orchestration.CandidatesTokenCount,
			"thoughts_token_count":   mission.Orchestration.ThoughtsTokenCount,
			"total_token_count":      mission.Orchestration.TotalTokenCount,
		},
		"draft":                   mission.Draft,
		"published":               mission.Published,
		"draft_revision":          mission.DraftRevision,
		"published_revision":      mission.PublishedRevision,
		"has_unpublished_changes": mission.HasUnpublishedChanges,
		"status":                  mission.Status,
		"status_before_delete":    mission.StatusBeforeDelete,
		"created_by":              mission.CreatedBy,
		"updated_by":              mission.UpdatedBy,
		"deleted_by":              mission.DeletedBy,
		"published_at":            mission.PublishedAt,
		"deleted_at":              mission.DeletedAt,
		"created_at":              mission.CreatedAt,
		"updated_at":              mission.UpdatedAt,
	}
}

func publishedRevisionMap(
	mission MissionDocument,
	publishedBy teacherclass.TeacherSummary,
) map[string]any {
	transferStatus := "complete"
	if attachmentsNeedTransfer(mission.Published.Attachments) {
		transferStatus = AttachmentStoragePending
	}

	return map[string]any{
		"organization_id":            mission.OrganizationID,
		"district_id":                mission.DistrictID,
		"school_id":                  mission.SchoolID,
		"teacher_id":                 mission.TeacherID,
		"class_id":                   mission.ClassID,
		"mission_id":                 mission.ID,
		"revision":                   mission.PublishedRevision,
		"snapshot":                   mission.Published,
		"attachment_transfer_status": transferStatus,
		"attachment_transfer_at":     mission.PublishedAt,
		"published_by":               publishedBy,
		"published_at":               mission.PublishedAt,
	}
}

func missionCodeFirestoreMap(mission MissionDocument) map[string]any {
	return map[string]any{
		"mission_code":    mission.MissionCode,
		"mission_id":      mission.ID,
		"class_id":        mission.ClassID,
		"organization_id": mission.OrganizationID,
		"district_id":     mission.DistrictID,
		"school_id":       mission.SchoolID,
		"teacher_id":      mission.TeacherID,
		"status":          mission.Status,
		"created_at":      mission.CreatedAt,
		"updated_at":      mission.UpdatedAt,
	}
}

func (repository *Repository) studentMissionCatalogRef(
	classID string,
	missionID string,
) *firestore.DocumentRef {
	return repository.client.Collection(StudentClassMissionsCollectionName).
		Doc(classID).
		Collection(StudentClassMissionsSubcollection).
		Doc(missionID)
}

func (repository *Repository) syncStudentMissionCatalog(
	tx *firestore.Transaction,
	mission MissionDocument,
) error {
	ref := repository.studentMissionCatalogRef(mission.ClassID, mission.ID)
	if mission.AudienceType == AudienceSelected {
		return tx.Delete(ref)
	}

	return tx.Set(ref, studentMissionCatalogMap(mission))
}

func studentMissionCatalogMap(mission MissionDocument) map[string]any {
	return map[string]any{
		"id":                         mission.ID,
		"class_id":                   mission.ClassID,
		"published_revision":         mission.PublishedRevision,
		"mission_name":               mission.MissionName,
		"title":                      mission.Title,
		"summary":                    mission.Summary,
		"subject":                    mission.Subject,
		"grade_level":                mission.GradeLevel,
		"learning_goal":              mission.LearningGoal,
		"suggested_duration_minutes": mission.SuggestedDurationMinutes,
		"status":                     StatusPublished,
		"published_at":               mission.PublishedAt,
		"updated_at":                 mission.UpdatedAt,
	}
}

func collectMissionDocuments(
	iter *firestore.DocumentIterator,
	classID string,
	byID map[string]MissionDocument,
) error {
	defer iter.Stop()

	for {
		snapshot, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return fmt.Errorf("iterate missions: %w", err)
		}

		mission, belongs, err := missionFromSnapshot(snapshot, classID)
		if err != nil {
			return err
		}
		if !belongs || mission.Status == StatusDeleted {
			continue
		}

		byID[mission.ID] = mission
	}

	return nil
}

func (repository *Repository) getMissionAtRef(
	ctx context.Context,
	ref *firestore.DocumentRef,
	classID string,
) (MissionDocument, bool, error) {
	snapshot, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return MissionDocument{}, false, nil
	}
	if err != nil {
		return MissionDocument{}, false, fmt.Errorf("read mission: %w", err)
	}

	mission, belongs, err := missionFromSnapshot(snapshot, classID)
	if err != nil {
		return MissionDocument{}, false, err
	}
	if !belongs {
		return MissionDocument{}, false, nil
	}

	return mission, true, nil
}

func (repository *Repository) getMissionInTransaction(
	tx *firestore.Transaction,
	classID string,
	missionID string,
) (MissionDocument, *firestore.DocumentRef, bool, error) {
	for _, ref := range repository.missionRefs(classID, missionID) {
		snapshot, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			continue
		}
		if err != nil {
			return MissionDocument{}, nil, false, fmt.Errorf("read mission: %w", err)
		}

		mission, belongs, err := missionFromSnapshot(snapshot, classID)
		if err != nil {
			return MissionDocument{}, nil, false, err
		}
		if !belongs {
			continue
		}

		return mission, ref, true, nil
	}

	return MissionDocument{}, nil, false, nil
}

func missionFromSnapshot(
	snapshot *firestore.DocumentSnapshot,
	classID string,
) (MissionDocument, bool, error) {
	var mission MissionDocument
	if err := snapshot.DataTo(&mission); err != nil {
		return MissionDocument{}, false, fmt.Errorf("decode mission %s: %w", snapshot.Ref.ID, err)
	}
	if mission.ID == "" {
		mission.ID = snapshot.Ref.ID
	}
	if mission.ClassID == "" {
		mission.ClassID = classID
	}
	if classID != "" && mission.ClassID != classID {
		return MissionDocument{}, false, nil
	}
	hydrateMissionLifecycle(&mission)

	return mission, true, nil
}

func missionListSortTime(mission MissionDocument) time.Time {
	if !mission.UpdatedAt.IsZero() {
		return mission.UpdatedAt
	}

	return mission.CreatedAt
}

func applyMissionTenantScope(
	mission *MissionDocument,
	scope tenant.Scope,
	teacherID string,
) {
	if mission == nil {
		return
	}

	normalized := tenant.NormalizeScope(scope)
	mission.OrganizationID = normalized.OrganizationID
	mission.DistrictID = normalized.DistrictID
	mission.SchoolID = normalized.SchoolID
	if teacherID != "" {
		mission.TeacherID = teacherID
	}
	applySnapshotTenantScope(mission.Draft, normalized, mission.TeacherID)
	applySnapshotTenantScope(mission.Published, normalized, mission.TeacherID)
}

func applySnapshotTenantScope(
	snapshot *MissionSnapshot,
	scope tenant.Scope,
	teacherID string,
) {
	if snapshot == nil {
		return
	}

	snapshot.OrganizationID = scope.OrganizationID
	snapshot.DistrictID = scope.DistrictID
	snapshot.SchoolID = scope.SchoolID
	if teacherID != "" {
		snapshot.TeacherID = teacherID
	}
}

func hydratePublishedRevision(
	revision *MissionPublishedRevisionDocument,
	classID string,
	missionID string,
) {
	if revision == nil {
		return
	}
	if revision.ClassID == "" {
		revision.ClassID = classID
	}
	if revision.MissionID == "" {
		revision.MissionID = missionID
	}

	scope := tenant.NormalizeScope(tenant.Scope{
		OrganizationID: revision.OrganizationID,
		DistrictID:     revision.DistrictID,
		SchoolID:       revision.SchoolID,
	})
	revision.OrganizationID = scope.OrganizationID
	revision.DistrictID = scope.DistrictID
	revision.SchoolID = scope.SchoolID
	if revision.TeacherID == "" {
		revision.TeacherID = revision.Snapshot.TeacherID
	}
	if revision.TeacherID == "" {
		revision.TeacherID = revision.PublishedBy.ProviderUserID
	}
	applySnapshotTenantScope(&revision.Snapshot, scope, revision.TeacherID)
}

func inheritStoredAttachments(
	attachments []MissionAttachment,
	mission MissionDocument,
) []MissionAttachment {
	storedByUUID := make(map[string]*MissionAttachmentStorage)
	collectStored := func(values []MissionAttachment) {
		for _, attachment := range values {
			if attachment.UploadcareUUID == "" || attachment.Storage == nil ||
				attachment.Storage.Status != AttachmentStorageStored || AttachmentStorageURI(attachment.Storage) == "" {
				continue
			}

			storage := *attachment.Storage
			storedByUUID[attachment.UploadcareUUID] = &storage
		}
	}
	if mission.Published != nil {
		collectStored(mission.Published.Attachments)
	}
	if mission.Draft != nil {
		collectStored(mission.Draft.Attachments)
	}

	output := make([]MissionAttachment, len(attachments))
	copy(output, attachments)
	for index := range output {
		if stored, exists := storedByUUID[output[index].UploadcareUUID]; exists {
			storage := *stored
			output[index].Storage = &storage
		}
	}

	return output
}

func prepareAttachmentsForPublish(attachments []MissionAttachment, timestamp time.Time) {
	for index := range attachments {
		storage := attachments[index].Storage
		if storage != nil && storage.Status == AttachmentStorageStored && storage.GSURI != "" {
			continue
		}

		attachments[index].Storage = &MissionAttachmentStorage{
			Status:    AttachmentStoragePending,
			Retryable: true,
			UpdatedAt: timestamp,
		}
	}
}

func attachmentsNeedTransfer(attachments []MissionAttachment) bool {
	for _, attachment := range attachments {
		if attachment.Storage == nil || attachment.Storage.Status != AttachmentStorageStored || AttachmentStorageURI(attachment.Storage) == "" {
			return true
		}
	}

	return false
}
