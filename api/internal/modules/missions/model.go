package missions

import (
	"time"

	"classgame-api/internal/modules/classrooms"
	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/tenancy"
)

const (
	MissionsCollectionName             = "missions"
	MissionCodesCollectionName         = "mission_codes"
	MissionRevisionsCollectionName     = "published_revisions"
	StudentClassMissionsCollectionName = "student_class_missions"
	StudentClassMissionsSubcollection  = "missions"
	AudienceAll                        = "all"
	AudienceSelected                   = "selected"
	StatusDraft                        = "draft"
	// StatusPreview is kept only to decode legacy documents and requests.
	StatusPreview            = "preview"
	StatusPublished          = "published"
	StatusDeleted            = "deleted"
	AttachmentStoragePending = "pending"
	AttachmentStorageCopying = "copying"
	AttachmentStorageStored  = "stored"
	AttachmentStorageFailed  = "failed"
)

type MissionDocument struct {
	ID                       string                         `firestore:"id" json:"id"`
	MissionCode              string                         `firestore:"mission_code" json:"mission_code"`
	OrganizationID           string                         `firestore:"organization_id" json:"organization_id"`
	DistrictID               string                         `firestore:"district_id" json:"district_id"`
	SchoolID                 string                         `firestore:"school_id" json:"school_id"`
	TeacherID                string                         `firestore:"teacher_id" json:"teacher_id"`
	ClassID                  string                         `firestore:"class_id" json:"class_id"`
	ClassCode                string                         `firestore:"class_code" json:"class_code"`
	ClassName                string                         `firestore:"class_name" json:"class_name"`
	MissionName              string                         `firestore:"mission_name" json:"mission_name"`
	Title                    string                         `firestore:"title" json:"title"`
	Summary                  string                         `firestore:"summary" json:"summary"`
	Subject                  string                         `firestore:"subject" json:"subject"`
	GradeLevel               string                         `firestore:"grade_level" json:"grade_level"`
	LearningGoal             string                         `firestore:"learning_goal" json:"learning_goal"`
	TeacherPrompt            string                         `firestore:"teacher_prompt" json:"teacher_prompt"`
	AudienceType             string                         `firestore:"audience_type" json:"audience_type"`
	TargetStudentIDs         []string                       `firestore:"target_student_ids" json:"target_student_ids"`
	Attachments              []MissionAttachment            `firestore:"attachments" json:"attachments"`
	PreviewQuestions         []generation.PreviewQuestion   `firestore:"preview_questions" json:"preview_questions"`
	KnowledgeChunks          []generation.KnowledgeChunk    `firestore:"knowledge_chunks" json:"knowledge_chunks"`
	SuggestedDurationMinutes int                            `firestore:"suggested_duration_minutes" json:"suggested_duration_minutes"`
	Stages                   []generation.MissionStage      `firestore:"stages" json:"stages"`
	Suggestions              []generation.MissionSuggestion `firestore:"suggestions" json:"suggestions"`
	PositiveSignals          []string                       `firestore:"positive_signals" json:"positive_signals"`
	ImprovementSignals       []string                       `firestore:"improvement_signals" json:"improvement_signals"`
	EducationGuardrail       generation.EducationGuardrail  `firestore:"education_guardrail" json:"education_guardrail"`
	MissionFlow              generation.MissionFlow         `firestore:"mission_flow" json:"mission_flow"`
	Orchestration            MissionOrchestration           `firestore:"orchestration" json:"orchestration"`
	Draft                    *MissionSnapshot               `firestore:"draft,omitempty" json:"-"`
	Published                *MissionSnapshot               `firestore:"published,omitempty" json:"-"`
	DraftRevision            int64                          `firestore:"draft_revision" json:"draft_revision"`
	PublishedRevision        int64                          `firestore:"published_revision" json:"published_revision"`
	HasUnpublishedChanges    bool                           `firestore:"has_unpublished_changes" json:"has_unpublished_changes"`
	Status                   string                         `firestore:"status" json:"status"`
	StatusBeforeDelete       string                         `firestore:"status_before_delete,omitempty" json:"-"`
	CreatedBy                classrooms.TeacherSummary      `firestore:"created_by" json:"created_by"`
	UpdatedBy                classrooms.TeacherSummary      `firestore:"updated_by" json:"updated_by"`
	DeletedBy                classrooms.TeacherSummary      `firestore:"deleted_by,omitempty" json:"-"`
	PublishedAt              time.Time                      `firestore:"published_at" json:"published_at"`
	DeletedAt                time.Time                      `firestore:"deleted_at,omitempty" json:"deleted_at"`
	CreatedAt                time.Time                      `firestore:"created_at" json:"created_at"`
	UpdatedAt                time.Time                      `firestore:"updated_at" json:"updated_at"`
}

// MissionSnapshot is one immutable content revision. MissionDocument.Draft is
// replaced whenever Mia regenerates a preview, while Published changes only
// after an explicit approval.
type MissionSnapshot struct {
	Revision                 int64                          `firestore:"revision" json:"revision"`
	OrganizationID           string                         `firestore:"organization_id" json:"organization_id"`
	DistrictID               string                         `firestore:"district_id" json:"district_id"`
	SchoolID                 string                         `firestore:"school_id" json:"school_id"`
	TeacherID                string                         `firestore:"teacher_id" json:"teacher_id"`
	MissionName              string                         `firestore:"mission_name" json:"mission_name"`
	Title                    string                         `firestore:"title" json:"title"`
	Summary                  string                         `firestore:"summary" json:"summary"`
	Subject                  string                         `firestore:"subject" json:"subject"`
	GradeLevel               string                         `firestore:"grade_level" json:"grade_level"`
	LearningGoal             string                         `firestore:"learning_goal" json:"learning_goal"`
	TeacherPrompt            string                         `firestore:"teacher_prompt" json:"teacher_prompt"`
	AudienceType             string                         `firestore:"audience_type" json:"audience_type"`
	TargetStudentIDs         []string                       `firestore:"target_student_ids" json:"target_student_ids"`
	Attachments              []MissionAttachment            `firestore:"attachments" json:"attachments"`
	PreviewQuestions         []generation.PreviewQuestion   `firestore:"preview_questions" json:"preview_questions"`
	KnowledgeChunks          []generation.KnowledgeChunk    `firestore:"knowledge_chunks" json:"knowledge_chunks"`
	SuggestedDurationMinutes int                            `firestore:"suggested_duration_minutes" json:"suggested_duration_minutes"`
	Stages                   []generation.MissionStage      `firestore:"stages" json:"stages"`
	Suggestions              []generation.MissionSuggestion `firestore:"suggestions" json:"suggestions"`
	PositiveSignals          []string                       `firestore:"positive_signals" json:"positive_signals"`
	ImprovementSignals       []string                       `firestore:"improvement_signals" json:"improvement_signals"`
	EducationGuardrail       generation.EducationGuardrail  `firestore:"education_guardrail" json:"education_guardrail"`
	MissionFlow              generation.MissionFlow         `firestore:"mission_flow" json:"mission_flow"`
	Orchestration            MissionOrchestration           `firestore:"orchestration" json:"orchestration"`
	UpdatedBy                classrooms.TeacherSummary      `firestore:"updated_by" json:"updated_by"`
	UpdatedAt                time.Time                      `firestore:"updated_at" json:"updated_at"`
}

type MissionPublishedRevisionDocument struct {
	OrganizationID           string                    `firestore:"organization_id" json:"organization_id"`
	DistrictID               string                    `firestore:"district_id" json:"district_id"`
	SchoolID                 string                    `firestore:"school_id" json:"school_id"`
	TeacherID                string                    `firestore:"teacher_id" json:"teacher_id"`
	ClassID                  string                    `firestore:"class_id" json:"class_id"`
	MissionID                string                    `firestore:"mission_id" json:"mission_id"`
	Revision                 int64                     `firestore:"revision" json:"revision"`
	Snapshot                 MissionSnapshot           `firestore:"snapshot" json:"snapshot"`
	AttachmentTransferStatus string                    `firestore:"attachment_transfer_status" json:"attachment_transfer_status"`
	AttachmentTransferAt     time.Time                 `firestore:"attachment_transfer_at,omitempty" json:"attachment_transfer_at"`
	PublishedBy              classrooms.TeacherSummary `firestore:"published_by" json:"published_by"`
	PublishedAt              time.Time                 `firestore:"published_at" json:"published_at"`
}

type MissionAttachment struct {
	UploadcareUUID   string                    `firestore:"uploadcare_uuid" json:"uploadcare_uuid"`
	CDNURL           string                    `firestore:"cdn_url" json:"cdn_url"`
	UploadcareCDNURL string                    `firestore:"uploadcare_cdn_url,omitempty" json:"uploadcare_cdn_url,omitempty"`
	FileName         string                    `firestore:"file_name" json:"file_name"`
	MimeType         string                    `firestore:"mime_type" json:"mime_type"`
	SizeBytes        int64                     `firestore:"size_bytes" json:"size_bytes"`
	Storage          *MissionAttachmentStorage `firestore:"storage,omitempty" json:"storage,omitempty"`
}

type MissionAttachmentStorage struct {
	Status     string    `firestore:"status" json:"status"`
	Bucket     string    `firestore:"bucket,omitempty" json:"bucket,omitempty"`
	ObjectPath string    `firestore:"object_path,omitempty" json:"object_path,omitempty"`
	GSURI      string    `firestore:"gs_uri,omitempty" json:"gs_uri,omitempty"`
	PublicURL  string    `firestore:"-" json:"public_url,omitempty"`
	MimeType   string    `firestore:"mime_type,omitempty" json:"mime_type,omitempty"`
	SizeBytes  int64     `firestore:"size_bytes,omitempty" json:"size_bytes,omitempty"`
	Attempts   int       `firestore:"attempts" json:"attempts"`
	Retryable  bool      `firestore:"retryable" json:"retryable"`
	Error      string    `firestore:"error,omitempty" json:"error,omitempty"`
	UpdatedAt  time.Time `firestore:"updated_at" json:"updated_at"`
	StoredAt   time.Time `firestore:"stored_at,omitempty" json:"stored_at"`
}

type MissionOrchestration struct {
	Provider             string `firestore:"provider" json:"provider"`
	Model                string `firestore:"model" json:"model"`
	ModelVersion         string `firestore:"model_version" json:"model_version"`
	GeneratedByAI        bool   `firestore:"generated_by_ai" json:"generated_by_ai"`
	GeneratedByMock      bool   `firestore:"generated_by_mock" json:"generated_by_mock"`
	FallbackReason       string `firestore:"fallback_reason" json:"fallback_reason"`
	PromptTokenCount     int32  `firestore:"prompt_token_count" json:"prompt_token_count"`
	CandidatesTokenCount int32  `firestore:"candidates_token_count" json:"candidates_token_count"`
	ThoughtsTokenCount   int32  `firestore:"thoughts_token_count" json:"thoughts_token_count"`
	TotalTokenCount      int32  `firestore:"total_token_count" json:"total_token_count"`
}

type CreateMissionInput struct {
	MissionID             string
	ExpectedDraftRevision int64
	MissionCode           string
	Scope                 tenancy.Scope
	ClassID               string
	ClassCode             string
	ClassName             string
	MissionName           string
	Subject               string
	GradeLevel            string
	LearningGoal          string
	TeacherPrompt         string
	AudienceType          string
	TargetStudentIDs      []string
	Attachments           []MissionAttachment
	Teacher               classrooms.TeacherSummary
	Blueprint             generation.MissionBlueprint
	Progress              generation.MissionProgressReporter
	Status                string
	PublishedAt           time.Time
	Timestamp             time.Time
}

type ApproveMissionInput struct {
	ExpectedDraftRevision int64
	DraftEdits            MissionDraftEdits
}

type MissionDraftEdits struct {
	Title            string
	Summary          string
	PreviewQuestions []generation.PreviewQuestion
	HasChanges       bool
}
