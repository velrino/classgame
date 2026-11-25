package ai

const PreviewQuestionCount = 3

type BuildMissionInput struct {
	ClassID       string                  `json:"class_id"`
	ClassName     string                  `json:"class_name"`
	MissionName   string                  `json:"mission_name"`
	Subject       string                  `json:"subject"`
	GradeLevel    string                  `json:"grade_level"`
	LearningGoal  string                  `json:"learning_goal"`
	TeacherPrompt string                  `json:"teacher_prompt"`
	Attachments   []AttachmentInput       `json:"attachments"`
	Progress      MissionProgressReporter `json:"-"`
}

type MissionProgressReporter func(MissionProgress)

type MissionProgress struct {
	Stage      string `json:"stage"`
	Message    string `json:"message"`
	Percent    int    `json:"percent,omitempty"`
	Current    int    `json:"current,omitempty"`
	Total      int    `json:"total,omitempty"`
	Attachment string `json:"attachment,omitempty"`
}

type MissionBlueprint struct {
	Title                    string              `json:"title"`
	Summary                  string              `json:"summary"`
	Subject                  string              `json:"subject"`
	GradeLevel               string              `json:"grade_level"`
	LearningGoal             string              `json:"learning_goal"`
	SuggestedDurationMinutes int                 `json:"suggested_duration_minutes"`
	Stages                   []MissionStage      `json:"stages"`
	PreviewQuestions         []PreviewQuestion   `json:"preview_questions"`
	KnowledgeChunks          []KnowledgeChunk    `json:"knowledge_chunks"`
	Suggestions              []MissionSuggestion `json:"suggestions"`
	PositiveSignals          []string            `json:"positive_signals"`
	ImprovementSignals       []string            `json:"improvement_signals"`
	EducationGuardrail       EducationGuardrail  `json:"education_guardrail"`
	MissionFlow              MissionFlow         `json:"mission_flow"`
	GeneratedByAI            bool                `json:"generated_by_ai"`
	GeneratedByMock          bool                `json:"generated_by_mock"`
	Provider                 string              `json:"provider"`
	Model                    string              `json:"model"`
	ModelVersion             string              `json:"model_version"`
	FallbackReason           string              `json:"fallback_reason"`
	Usage                    GenAIUsage          `json:"usage"`
}

type MissionStage struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Goal        string   `json:"goal"`
	Checks      []string `json:"checks"`
	UnlocksWhen string   `json:"unlocks_when"`
}

type MissionSuggestion struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Reason      string   `json:"reason"`
	Activity    string   `json:"activity"`
	Questions   []string `json:"questions"`
	FocusSkills []string `json:"focus_skills"`
}

type PreviewQuestion struct {
	ID                 string   `json:"id"`
	StageID            string   `json:"stage_id"`
	Prompt             string   `json:"prompt"`
	AnswerType         string   `json:"answer_type"`
	Options            []string `json:"options"`
	ExpectedAnswer     string   `json:"expected_answer"`
	ExplanationSteps   []string `json:"explanation_steps"`
	MisconceptionTraps []string `json:"misconception_traps"`
}

// KnowledgeChunk is a short, teacher-approved retrieval unit distilled from
// the mission prompt and readable attachments. Live tutor turns retrieve only
// the chunks relevant to the current question instead of resending whole PDFs.
type KnowledgeChunk struct {
	ID       string   `firestore:"id" json:"id"`
	Source   string   `firestore:"source" json:"source"`
	Title    string   `firestore:"title" json:"title"`
	Content  string   `firestore:"content" json:"content"`
	Keywords []string `firestore:"keywords" json:"keywords"`
}

type AttachmentInput struct {
	FileName   string `json:"file_name"`
	MimeType   string `json:"mime_type"`
	CDNURL     string `json:"cdn_url"`
	StorageURL string `json:"storage_url,omitempty"`
	Summary    string `json:"summary,omitempty"`
}

type GenAIUsage struct {
	PromptTokenCount     int32 `firestore:"prompt_token_count" json:"prompt_token_count"`
	CandidatesTokenCount int32 `firestore:"candidates_token_count" json:"candidates_token_count"`
	ThoughtsTokenCount   int32 `firestore:"thoughts_token_count" json:"thoughts_token_count"`
	TotalTokenCount      int32 `firestore:"total_token_count" json:"total_token_count"`
}

type EducationGuardrail struct {
	Allowed         bool   `json:"allowed"`
	Reason          string `json:"reason"`
	UsedAttachments bool   `json:"used_attachments"`
	AttachmentNotes string `json:"attachment_notes"`
}

type MissionFlow struct {
	Nodes []MissionFlowNode `firestore:"nodes" json:"nodes"`
	Edges []MissionFlowEdge `firestore:"edges" json:"edges"`
}

type MissionFlowNode struct {
	ID               string              `firestore:"id" json:"id"`
	Kind             string              `firestore:"kind" json:"kind"`
	Label            string              `firestore:"label" json:"label"`
	Description      string              `firestore:"description" json:"description"`
	EvidenceEventIDs []string            `firestore:"evidence_event_ids" json:"evidence_event_ids"`
	Position         MissionFlowPosition `firestore:"position" json:"position"`
}

type MissionFlowPosition struct {
	X float64 `firestore:"x" json:"x"`
	Y float64 `firestore:"y" json:"y"`
}

type MissionFlowEdge struct {
	ID     string `firestore:"id" json:"id"`
	Source string `firestore:"source" json:"source"`
	Target string `firestore:"target" json:"target"`
	Label  string `firestore:"label" json:"label"`
}

// BuildLearningTraceInput contains only observable learning evidence. It must
// never contain or request hidden chain-of-thought from either the student or
// the model.
