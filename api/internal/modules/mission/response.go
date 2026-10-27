package mission

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"classgame-api/internal/modules/orchestrator"
	teacherclass "classgame-api/internal/modules/teacher-class"
)

type MissionResponse struct {
	ID                       string                           `json:"id"`
	MissionCode              string                           `json:"mission_code"`
	OrganizationID           string                           `json:"organization_id"`
	DistrictID               string                           `json:"district_id"`
	SchoolID                 string                           `json:"school_id"`
	TeacherID                string                           `json:"teacher_id"`
	ClassID                  string                           `json:"class_id"`
	ClassCode                string                           `json:"class_code"`
	ClassName                string                           `json:"class_name"`
	MissionName              string                           `json:"mission_name"`
	Title                    string                           `json:"title"`
	Summary                  string                           `json:"summary"`
	Subject                  string                           `json:"subject"`
	GradeLevel               string                           `json:"grade_level"`
	LearningGoal             string                           `json:"learning_goal"`
	TeacherPrompt            string                           `json:"teacher_prompt"`
	AudienceType             string                           `json:"audience_type"`
	TargetStudentIDs         []string                         `json:"target_student_ids"`
	Attachments              []MissionAttachment              `json:"attachments"`
	PreviewQuestions         []orchestrator.PreviewQuestion   `json:"preview_questions"`
	KnowledgeChunks          []orchestrator.KnowledgeChunk    `json:"knowledge_chunks"`
	SuggestedDurationMinutes int                              `json:"suggested_duration_minutes"`
	Stages                   []orchestrator.MissionStage      `json:"stages"`
	Suggestions              []orchestrator.MissionSuggestion `json:"suggestions"`
	PositiveSignals          []string                         `json:"positive_signals"`
	ImprovementSignals       []string                         `json:"improvement_signals"`
	EducationGuardrail       orchestrator.EducationGuardrail  `json:"education_guardrail"`
	MissionFlow              orchestrator.MissionFlow         `json:"mission_flow"`
	Orchestration            MissionOrchestrationResponse     `json:"orchestration"`
	DraftRevision            int64                            `json:"draft_revision"`
	PublishedRevision        int64                            `json:"published_revision"`
	HasUnpublishedChanges    bool                             `json:"has_unpublished_changes"`
	Status                   string                           `json:"status"`
	CreatedBy                teacherclass.TeacherResponse     `json:"created_by"`
	UpdatedBy                teacherclass.TeacherResponse     `json:"updated_by"`
	PublishedAt              string                           `json:"published_at"`
	DeletedAt                string                           `json:"deleted_at"`
	CreatedAt                string                           `json:"created_at"`
	UpdatedAt                string                           `json:"updated_at"`
}

type MissionOrchestrationResponse struct {
	Provider             string `json:"provider"`
	Model                string `json:"model"`
	ModelVersion         string `json:"model_version"`
	GeneratedByAI        bool   `json:"generated_by_ai"`
	GeneratedByMock      bool   `json:"generated_by_mock"`
	FallbackReason       string `json:"fallback_reason"`
	PromptTokenCount     int32  `json:"prompt_token_count"`
	CandidatesTokenCount int32  `json:"candidates_token_count"`
	ThoughtsTokenCount   int32  `json:"thoughts_token_count"`
	TotalTokenCount      int32  `json:"total_token_count"`
}

func ToResponse(mission MissionDocument) MissionResponse {
	return MissionResponse{
		ID:               mission.ID,
		MissionCode:      mission.MissionCode,
		OrganizationID:   mission.OrganizationID,
		DistrictID:       mission.DistrictID,
		SchoolID:         mission.SchoolID,
		TeacherID:        mission.TeacherID,
		ClassID:          mission.ClassID,
		ClassCode:        mission.ClassCode,
		ClassName:        mission.ClassName,
		MissionName:      mission.MissionName,
		Title:            mission.Title,
		Summary:          mission.Summary,
		Subject:          mission.Subject,
		GradeLevel:       mission.GradeLevel,
		LearningGoal:     mission.LearningGoal,
		TeacherPrompt:    mission.TeacherPrompt,
		AudienceType:     mission.AudienceType,
		TargetStudentIDs: mission.TargetStudentIDs,
		Attachments:      attachmentsWithPublicStorageURLs(mission.Attachments),
		PreviewQuestions: orchestrator.PlayablePreviewQuestions(orchestrator.BuildMissionInput{
			ClassID:       mission.ClassID,
			ClassName:     mission.ClassName,
			MissionName:   mission.MissionName,
			Subject:       mission.Subject,
			GradeLevel:    mission.GradeLevel,
			LearningGoal:  mission.LearningGoal,
			TeacherPrompt: mission.TeacherPrompt,
		}, mission.PreviewQuestions),
		KnowledgeChunks:          mission.KnowledgeChunks,
		SuggestedDurationMinutes: mission.SuggestedDurationMinutes,
		Stages:                   mission.Stages,
		Suggestions:              mission.Suggestions,
		PositiveSignals:          mission.PositiveSignals,
		ImprovementSignals:       mission.ImprovementSignals,
		EducationGuardrail:       mission.EducationGuardrail,
		MissionFlow:              mission.MissionFlow,
		Orchestration: MissionOrchestrationResponse{
			Provider:             mission.Orchestration.Provider,
			Model:                mission.Orchestration.Model,
			ModelVersion:         mission.Orchestration.ModelVersion,
			GeneratedByAI:        mission.Orchestration.GeneratedByAI,
			GeneratedByMock:      mission.Orchestration.GeneratedByMock,
			FallbackReason:       mission.Orchestration.FallbackReason,
			PromptTokenCount:     mission.Orchestration.PromptTokenCount,
			CandidatesTokenCount: mission.Orchestration.CandidatesTokenCount,
			ThoughtsTokenCount:   mission.Orchestration.ThoughtsTokenCount,
			TotalTokenCount:      mission.Orchestration.TotalTokenCount,
		},
		DraftRevision:         mission.DraftRevision,
		PublishedRevision:     mission.PublishedRevision,
		HasUnpublishedChanges: mission.HasUnpublishedChanges,
		Status:                mission.Status,
		CreatedBy:             toTeacherResponse(mission.CreatedBy),
		UpdatedBy:             toTeacherResponse(mission.UpdatedBy),
		PublishedAt:           formatTime(mission.PublishedAt),
		DeletedAt:             formatTime(mission.DeletedAt),
		CreatedAt:             formatTime(mission.CreatedAt),
		UpdatedAt:             formatTime(mission.UpdatedAt),
	}
}

func ToResponses(missions []MissionDocument) []MissionResponse {
	responses := make([]MissionResponse, 0, len(missions))

	for _, mission := range missions {
		responses = append(responses, ToResponse(mission))
	}

	return responses
}

func AttachmentStorageWithPublicURL(storage *MissionAttachmentStorage) *MissionAttachmentStorage {
	if storage == nil {
		return nil
	}

	output := *storage
	output.PublicURL = AttachmentStoragePublicURL(&output)

	return &output
}

// AttachmentStoragePublicURL is retained for source compatibility. Stored
// worksheets are private and must be downloaded through the authenticated API.
func AttachmentStoragePublicURL(storage *MissionAttachmentStorage) string { return "" }

func AttachmentStorageURI(storage *MissionAttachmentStorage) string {
	if storage == nil || storage.Status != AttachmentStorageStored {
		return ""
	}
	if strings.HasPrefix(strings.TrimSpace(storage.GSURI), "gs://") {
		return strings.TrimSpace(storage.GSURI)
	}
	if storage.Bucket == "" || storage.ObjectPath == "" {
		return ""
	}
	return "gs://" + storage.Bucket + "/" + strings.Trim(storage.ObjectPath, "/")
}

func attachmentsWithPublicStorageURLs(attachments []MissionAttachment) []MissionAttachment {
	output := make([]MissionAttachment, len(attachments))
	copy(output, attachments)

	for index := range output {
		output[index] = PrivateAttachmentResponse(output[index])
	}

	return output
}

func parseGSURI(value string) (string, string) {
	path := strings.TrimPrefix(strings.TrimSpace(value), "gs://")
	bucket, objectPath, _ := strings.Cut(path, "/")

	return strings.TrimSpace(bucket), strings.Trim(strings.TrimSpace(objectPath), "/")
}

func escapeObjectPath(objectPath string) string {
	segments := strings.Split(objectPath, "/")
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}

	return strings.Join(segments, "/")
}

func attachmentStoragePublicURLVersion(storage *MissionAttachmentStorage) string {
	for _, timestamp := range []time.Time{storage.UpdatedAt, storage.StoredAt} {
		if !timestamp.IsZero() {
			return strconv.FormatInt(timestamp.UTC().Truncate(time.Microsecond).UnixNano(), 10)
		}
	}

	return ""
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.UTC().Format(time.RFC3339Nano)
}

func toTeacherResponse(teacher teacherclass.TeacherSummary) teacherclass.TeacherResponse {
	return teacherclass.TeacherResponse{
		ProviderUserID: teacher.ProviderUserID,
		DisplayName:    teacher.DisplayName,
		Email:          teacher.Email,
		PhotoURL:       teacher.PhotoURL,
	}
}

// PrivateAttachmentResponse never advertises legacy public GCS object URLs.
func PrivateAttachmentResponse(attachment MissionAttachment) MissionAttachment {
	attachment.Storage = AttachmentStorageWithPublicURL(attachment.Storage)
	if attachment.Storage != nil && attachment.Storage.Status == AttachmentStorageStored {
		attachment.CDNURL = attachment.UploadcareCDNURL
	}
	return attachment
}
