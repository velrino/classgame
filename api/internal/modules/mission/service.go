package mission

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"classgame-api/internal/modules/missionattachment"
	"classgame-api/internal/modules/orchestrator"
	teacherclass "classgame-api/internal/modules/teacher-class"
	"classgame-api/internal/modules/tenant"
	sharedauth "classgame-api/internal/shared/auth"
)

var (
	ErrInvalidMissionAttachment     = errors.New("mission attachment has no trusted source")
	ErrMissionNotFound              = errors.New("teacher class mission not found")
	ErrMissionCodeUnavailable       = errors.New("teacher class mission code is unavailable")
	ErrMissionCodeGeneration        = errors.New("could not generate a unique mission code")
	ErrMissionDeleted               = errors.New("teacher class mission is deleted")
	ErrMissionDraftNotFound         = errors.New("teacher class mission draft not found")
	ErrMissionDraftConflict         = errors.New("teacher class mission draft revision changed")
	ErrMissionDraftRevisionRequired = errors.New("teacher class mission expected draft revision is required")
	ErrMissionAttachmentQueue       = errors.New("teacher class mission attachment queue is not configured")
	ErrMissionGuardrailBlocked      = errors.New("teacher class mission was blocked by the education guardrail")
)

const (
	missionCodeAlphabet       = "0123456789"
	missionCodeLength         = 6
	missionCodeCreateAttempts = 12
	missionListLimit          = 100
)

type Orchestrator interface {
	BuildMission(context.Context, orchestrator.BuildMissionInput) (orchestrator.MissionBlueprint, error)
}

type TeacherClassService interface {
	Get(context.Context, sharedauth.Identity, string) (teacherclass.TeacherClassDocument, bool, error)
	TeacherFromIdentity(sharedauth.Identity) (teacherclass.TeacherSummary, error)
}

type MissionRepository interface {
	Create(context.Context, CreateMissionInput) (MissionDocument, error)
	Get(context.Context, string, string) (MissionDocument, bool, error)
	List(context.Context, string, int) ([]MissionDocument, error)
	Publish(
		context.Context,
		string,
		string,
		teacherclass.TeacherSummary,
		int64,
		MissionDraftEdits,
		time.Time,
	) (MissionDocument, error)
	SoftDelete(
		context.Context,
		string,
		string,
		teacherclass.TeacherSummary,
		time.Time,
	) (MissionDocument, error)
	UpdateDraft(context.Context, CreateMissionInput) (MissionDocument, error)
}

type Service struct {
	attachmentReader    AttachmentReader
	attachmentEnqueuer  missionattachment.Enqueuer
	orchestrator        Orchestrator
	repository          MissionRepository
	teacherClassService TeacherClassService
}

func NewService(
	repository MissionRepository,
	teacherClassService TeacherClassService,
	orchestrator Orchestrator,
	options ...ServiceOption,
) *Service {
	service := &Service{
		orchestrator:        orchestrator,
		repository:          repository,
		teacherClassService: teacherClassService,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}

	return service
}

type ServiceOption func(*Service)

func WithAttachmentEnqueuer(enqueuer missionattachment.Enqueuer) ServiceOption {
	return func(service *Service) {
		service.attachmentEnqueuer = enqueuer
	}
}

func (service *Service) Preview(
	ctx context.Context,
	identity sharedauth.Identity,
	input CreateMissionInput,
) (MissionDocument, error) {
	return service.createDraft(ctx, identity, input)
}

func (service *Service) Approve(
	ctx context.Context,
	identity sharedauth.Identity,
	classID string,
	missionID string,
	input ApproveMissionInput,
) (MissionDocument, error) {
	if service.repository == nil || service.teacherClassService == nil {
		return MissionDocument{}, errors.New("teacher class mission dependencies are not configured")
	}
	input = normalizeApproveInput(input)
	expectedDraftRevision := input.ExpectedDraftRevision
	if expectedDraftRevision <= 0 {
		return MissionDocument{}, ErrMissionDraftRevisionRequired
	}

	teacherClass, exists, err := service.teacherClassService.Get(ctx, identity, classID)
	if err != nil {
		return MissionDocument{}, err
	}
	if !exists {
		return MissionDocument{}, teacherclass.ErrClassNotFound
	}

	teacher, err := service.teacherClassService.TeacherFromIdentity(identity)
	if err != nil {
		return MissionDocument{}, err
	}

	missionID = strings.TrimSpace(missionID)
	draftMission, exists, err := service.repository.Get(ctx, teacherClass.ClassID, missionID)
	if err != nil {
		return MissionDocument{}, err
	}
	if !exists {
		return MissionDocument{}, ErrMissionNotFound
	}
	if draftMission.Status == StatusDeleted {
		return MissionDocument{}, ErrMissionDeleted
	}
	if draftMission.DraftRevision != expectedDraftRevision {
		return MissionDocument{}, ErrMissionDraftConflict
	}
	if draftMission.Draft == nil {
		return MissionDocument{}, ErrMissionDraftNotFound
	}
	if missionGuardrailBlocked(draftMission.EducationGuardrail) {
		return MissionDocument{}, ErrMissionGuardrailBlocked
	}
	alreadyPublished := draftMission.Published != nil &&
		draftMission.PublishedRevision == draftMission.DraftRevision &&
		!draftMission.HasUnpublishedChanges
	if !alreadyPublished && attachmentsNeedTransfer(draftMission.Draft.Attachments) {
		if service.attachmentEnqueuer == nil {
			slog.Warn(
				"mission attachment queue disabled; publishing without background transfer",
				"class_id", teacherClass.ClassID,
				"mission_id", missionID,
				"revision", expectedDraftRevision,
			)
		} else {
			if err := service.attachmentEnqueuer.Enqueue(ctx, missionattachment.Job{
				ClassID:   teacherClass.ClassID,
				MissionID: missionID,
				Revision:  expectedDraftRevision,
			}); err != nil {
				return MissionDocument{}, fmt.Errorf("%w: enqueue mission attachments: %w", ErrMissionAttachmentQueue, err)
			}
		}
	}

	mission, err := service.repository.Publish(
		ctx,
		teacherClass.ClassID,
		missionID,
		teacher,
		expectedDraftRevision,
		input.DraftEdits,
		time.Now(),
	)
	if err != nil {
		return MissionDocument{}, err
	}

	return mission, nil
}

func missionGuardrailBlocked(guardrail orchestrator.EducationGuardrail) bool {
	// Older stored missions predate this field and decode to the zero value.
	// A normalized blocked result always has an explanatory reason.
	return !guardrail.Allowed && strings.TrimSpace(guardrail.Reason) != ""
}

func (service *Service) Delete(
	ctx context.Context,
	identity sharedauth.Identity,
	classID string,
	missionID string,
) error {
	if service.repository == nil || service.teacherClassService == nil {
		return errors.New("teacher class mission dependencies are not configured")
	}

	teacherClass, exists, err := service.teacherClassService.Get(ctx, identity, classID)
	if err != nil {
		return err
	}
	if !exists {
		return teacherclass.ErrClassNotFound
	}

	teacher, err := service.teacherClassService.TeacherFromIdentity(identity)
	if err != nil {
		return err
	}

	_, err = service.repository.SoftDelete(
		ctx,
		teacherClass.ClassID,
		strings.TrimSpace(missionID),
		teacher,
		time.Now(),
	)
	return err
}

func (service *Service) GetActive(
	ctx context.Context,
	identity sharedauth.Identity,
	classID string,
) (MissionDocument, bool, error) {
	teacherClass, exists, err := service.teacherClassService.Get(ctx, identity, classID)
	if err != nil || !exists || teacherClass.ActiveMissionID == "" {
		return MissionDocument{}, false, err
	}

	mission, exists, err := service.repository.Get(ctx, teacherClass.ClassID, teacherClass.ActiveMissionID)
	if err != nil || !exists {
		return mission, exists, err
	}

	published, exists := publishedView(mission)
	return published, exists, nil
}

func (service *Service) List(
	ctx context.Context,
	identity sharedauth.Identity,
	classID string,
) ([]MissionDocument, error) {
	teacherClass, exists, err := service.teacherClassService.Get(ctx, identity, classID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []MissionDocument{}, nil
	}

	return service.repository.List(ctx, teacherClass.ClassID, missionListLimit)
}

func (service *Service) createDraft(
	ctx context.Context,
	identity sharedauth.Identity,
	input CreateMissionInput,
) (MissionDocument, error) {
	if service.orchestrator == nil || service.repository == nil || service.teacherClassService == nil {
		return MissionDocument{}, errors.New("teacher class mission dependencies are not configured")
	}

	input = normalizeInput(input)
	if input.MissionID != "" && input.ExpectedDraftRevision <= 0 {
		return MissionDocument{}, ErrMissionDraftRevisionRequired
	}
	teacherClass, exists, err := service.teacherClassService.Get(ctx, identity, input.ClassID)
	if err != nil {
		return MissionDocument{}, err
	}
	if !exists {
		return MissionDocument{}, teacherclass.ErrClassNotFound
	}

	input.ClassID = teacherClass.ClassID
	input.ClassName = teacherClass.ClassName
	input.Subject = teacherClass.Subject
	input.GradeLevel = teacherClass.GradeLevel
	input.Scope = tenant.NormalizeScope(tenant.Scope{
		OrganizationID: teacherClass.OrganizationID,
		DistrictID:     teacherClass.DistrictID,
		SchoolID:       teacherClass.SchoolID,
	})

	teacher, err := service.teacherClassService.TeacherFromIdentity(identity)
	if err != nil {
		return MissionDocument{}, err
	}

	if input.MissionID == "" {
		reusableDraft, reusableDraftExists, err := service.reusableStandaloneDraft(
			ctx,
			teacherClass.ClassID,
			teacher,
		)
		if err != nil {
			return MissionDocument{}, err
		}
		if reusableDraftExists {
			input.MissionID = reusableDraft.ID
			input.ExpectedDraftRevision = reusableDraft.DraftRevision
		}
	}

	if input.MissionID != "" {
		existing, exists, err := service.repository.Get(ctx, teacherClass.ClassID, input.MissionID)
		if err != nil {
			return MissionDocument{}, err
		}
		if !exists {
			return MissionDocument{}, ErrMissionNotFound
		}
		if existing.Status == StatusDeleted {
			return MissionDocument{}, ErrMissionDeleted
		}
		if existing.DraftRevision != input.ExpectedDraftRevision {
			return MissionDocument{}, ErrMissionDraftConflict
		}
		input.Attachments = inheritStoredAttachments(input.Attachments, existing)
	}

	for _, attachment := range input.Attachments {
		if strings.TrimSpace(attachment.CDNURL) == "" && AttachmentStorageURI(attachment.Storage) == "" {
			return MissionDocument{}, ErrInvalidMissionAttachment
		}
	}

	blueprint, err := service.orchestrator.BuildMission(ctx, orchestrator.BuildMissionInput{
		ClassID:       teacherClass.ClassID,
		ClassName:     teacherClass.ClassName,
		MissionName:   input.MissionName,
		Subject:       input.Subject,
		GradeLevel:    input.GradeLevel,
		LearningGoal:  input.LearningGoal,
		TeacherPrompt: input.TeacherPrompt,
		Attachments:   toOrchestratorAttachments(input.Attachments),
		Progress:      input.Progress,
	})
	if err != nil {
		return MissionDocument{}, err
	}
	blueprint.Subject = teacherClass.Subject
	blueprint.GradeLevel = teacherClass.GradeLevel

	timestamp := time.Now()
	input.ClassCode = teacherClass.ClassCode
	input.Teacher = teacher
	input.Blueprint = blueprint
	input.Status = StatusDraft
	input.Timestamp = timestamp

	var mission MissionDocument
	if input.MissionID == "" {
		if input.AudienceType == AudienceSelected {
			mission, err = service.createWithGeneratedCode(ctx, input)
		} else {
			mission, err = service.repository.Create(ctx, input)
		}
	} else {
		mission, err = service.repository.UpdateDraft(ctx, input)
	}
	if err != nil {
		return MissionDocument{}, err
	}

	return draftView(mission), nil
}

func (service *Service) reusableStandaloneDraft(
	ctx context.Context,
	classID string,
	teacher teacherclass.TeacherSummary,
) (MissionDocument, bool, error) {
	missions, err := service.repository.List(ctx, classID, missionListLimit)
	if err != nil {
		return MissionDocument{}, false, err
	}

	var reusableDraft MissionDocument
	reusableDraftExists := false
	deleteTimestamp := time.Now()
	for _, mission := range missions {
		if !standaloneDraftOwnedByTeacher(mission, teacher) {
			continue
		}
		if !reusableDraftExists {
			reusableDraft = mission
			reusableDraftExists = true
			continue
		}

		if _, err := service.repository.SoftDelete(ctx, classID, mission.ID, teacher, deleteTimestamp); err != nil {
			return MissionDocument{}, false, err
		}
	}

	return reusableDraft, reusableDraftExists, nil
}

func standaloneDraftOwnedByTeacher(
	mission MissionDocument,
	teacher teacherclass.TeacherSummary,
) bool {
	if mission.Status != StatusDraft && mission.Status != StatusPreview {
		return false
	}
	if mission.Published != nil {
		return false
	}
	if mission.Draft == nil {
		return false
	}

	return mission.CreatedBy.ProviderUserID == teacher.ProviderUserID
}

func (service *Service) createWithGeneratedCode(
	ctx context.Context,
	input CreateMissionInput,
) (MissionDocument, error) {
	for attempt := 0; attempt < missionCodeCreateAttempts; attempt++ {
		missionCode, err := generateMissionCode()
		if err != nil {
			return MissionDocument{}, fmt.Errorf("generate mission code: %w", err)
		}

		input.MissionCode = missionCode
		mission, err := service.repository.Create(ctx, input)
		if errors.Is(err, ErrMissionCodeUnavailable) {
			continue
		}

		return mission, err
	}

	return MissionDocument{}, ErrMissionCodeGeneration
}

func normalizeApproveInput(input ApproveMissionInput) ApproveMissionInput {
	input.DraftEdits = normalizeDraftEdits(input.DraftEdits)
	return input
}

func normalizeDraftEdits(edits MissionDraftEdits) MissionDraftEdits {
	if !edits.HasChanges {
		return MissionDraftEdits{}
	}

	edits.Title = normalizeText(edits.Title)
	edits.Summary = normalizeText(edits.Summary)
	edits.PreviewQuestions = normalizeEditedPreviewQuestions(edits.PreviewQuestions)
	if edits.Title == "" || edits.Summary == "" || len(edits.PreviewQuestions) == 0 {
		return MissionDraftEdits{}
	}

	return edits
}

func normalizeEditedPreviewQuestions(
	questions []orchestrator.PreviewQuestion,
) []orchestrator.PreviewQuestion {
	normalized := make([]orchestrator.PreviewQuestion, 0, min(len(questions), orchestrator.StudentPreviewQuestionLimit()))
	for index, question := range questions {
		if len(normalized) >= orchestrator.StudentPreviewQuestionLimit() {
			break
		}
		id := strings.TrimSpace(question.ID)
		if id == "" {
			id = fmt.Sprintf("question-%d", index+1)
		}

		prompt := normalizeText(question.Prompt)
		expectedAnswer := normalizeText(question.ExpectedAnswer)
		if prompt == "" || expectedAnswer == "" {
			continue
		}

		answerType := "multiple_choice"
		options := orchestrator.NormalizeMultipleChoiceOptions(
			expectedAnswer,
			normalizeTextList(question.Options, 4),
			normalizeTextList(question.MisconceptionTraps, 4),
		)

		normalized = append(normalized, orchestrator.PreviewQuestion{
			ID:                 id,
			StageID:            strings.TrimSpace(question.StageID),
			Prompt:             prompt,
			AnswerType:         answerType,
			Options:            options,
			ExpectedAnswer:     expectedAnswer,
			ExplanationSteps:   normalizeTextList(question.ExplanationSteps, 8),
			MisconceptionTraps: normalizeTextList(question.MisconceptionTraps, 8),
		})
	}

	return normalized
}

func normalizeTextList(values []string, limit int) []string {
	if limit <= 0 {
		return []string{}
	}

	normalized := make([]string, 0, min(len(values), limit))
	for _, value := range values {
		if len(normalized) >= limit {
			break
		}
		text := normalizeText(value)
		if text == "" {
			continue
		}
		normalized = append(normalized, text)
	}

	return normalized
}

func normalizeInput(input CreateMissionInput) CreateMissionInput {
	input.MissionID = strings.TrimSpace(input.MissionID)
	input.ClassID = normalizeCode(input.ClassID)
	input.ClassName = normalizeText(input.ClassName)
	input.MissionName = normalizeText(input.MissionName)
	input.Subject = normalizeText(input.Subject)
	input.GradeLevel = normalizeText(input.GradeLevel)
	input.LearningGoal = normalizeText(input.LearningGoal)
	input.TeacherPrompt = normalizeText(input.TeacherPrompt)
	input.AudienceType = strings.ToLower(strings.TrimSpace(input.AudienceType))
	input.TargetStudentIDs = normalizeStudentIDs(input.TargetStudentIDs)
	input.Attachments = normalizeAttachments(input.Attachments)

	if input.ClassName == "" {
		input.ClassName = input.ClassID
	}
	if input.MissionName == "" {
		input.MissionName = "Class review"
	}
	if input.Subject == "" {
		input.Subject = "Mathematics"
	}
	if input.GradeLevel == "" {
		input.GradeLevel = "7th grade"
	}
	if input.AudienceType != AudienceSelected || len(input.TargetStudentIDs) == 0 {
		input.AudienceType = AudienceAll
		input.TargetStudentIDs = []string{}
	}

	return input
}

func normalizeStudentIDs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		studentID := strings.TrimSpace(value)
		if studentID == "" {
			continue
		}
		if _, exists := seen[studentID]; exists {
			continue
		}

		seen[studentID] = struct{}{}
		normalized = append(normalized, studentID)
	}

	return normalized
}

func normalizeAttachments(attachments []MissionAttachment) []MissionAttachment {
	normalized := make([]MissionAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		fileName := normalizeText(attachment.FileName)
		cdnURL := strings.TrimSpace(attachment.CDNURL)
		if attachment.UploadcareUUID == "" && fileName == "" && cdnURL == "" {
			continue
		}

		normalized = append(normalized, MissionAttachment{
			UploadcareUUID:   strings.TrimSpace(attachment.UploadcareUUID),
			CDNURL:           cdnURL,
			UploadcareCDNURL: strings.TrimSpace(attachment.UploadcareCDNURL),
			FileName:         fileName,
			MimeType:         normalizeText(attachment.MimeType),
			SizeBytes:        attachment.SizeBytes,
		})
	}

	return normalized
}

func toOrchestratorAttachments(attachments []MissionAttachment) []orchestrator.AttachmentInput {
	output := make([]orchestrator.AttachmentInput, 0, len(attachments))
	for _, attachment := range attachments {
		output = append(output, orchestrator.AttachmentInput{
			FileName:   attachment.FileName,
			MimeType:   attachment.MimeType,
			CDNURL:     attachment.CDNURL,
			StorageURL: AttachmentStorageURI(attachment.Storage),
		})
	}

	return output
}

func normalizeText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func normalizeCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func generateMissionCode() (string, error) {
	code := make([]byte, missionCodeLength)
	alphabetLength := big.NewInt(int64(len(missionCodeAlphabet)))

	for index := range code {
		characterIndex, err := rand.Int(rand.Reader, alphabetLength)
		if err != nil {
			return "", err
		}
		code[index] = missionCodeAlphabet[characterIndex.Int64()]
	}

	return string(code), nil
}
