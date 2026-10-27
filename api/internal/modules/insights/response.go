package insights

import (
	"time"

	"classgame-api/internal/modules/classrooms"
	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/missions"
	studentsmodule "classgame-api/internal/modules/students"
)

type OverviewResponse struct {
	Page            *OverviewPageMetadata        `json:"page,omitempty"`
	OrganizationID  string                       `json:"organization_id"`
	DistrictID      string                       `json:"district_id"`
	SchoolID        string                       `json:"school_id"`
	ClassID         string                       `json:"class_id"`
	ClassCode       string                       `json:"class_code"`
	TeacherClass    *TeacherClassResponse        `json:"teacher_class"`
	ActiveMission   *MissionResponse             `json:"active_mission"`
	Metrics         DashboardMetricsResponse     `json:"metrics"`
	Students        []StudentDashboardResponse   `json:"students"`
	ClassLearning   ClassLearningSummaryResponse `json:"class_learning"`
	Suggestions     []SuggestionResponse         `json:"suggestions"`
	GeneratedByAI   bool                         `json:"generated_by_ai"`
	GeneratedByMock bool                         `json:"generated_by_mock"`
	Teacher         TeacherResponse              `json:"teacher"`
}

type TeacherClassResponse struct {
	ID              string          `json:"id"`
	OrganizationID  string          `json:"organization_id"`
	DistrictID      string          `json:"district_id"`
	SchoolID        string          `json:"school_id"`
	ClassID         string          `json:"class_id"`
	ClassCode       string          `json:"class_code"`
	ClassName       string          `json:"class_name"`
	Subject         string          `json:"subject"`
	GradeLevel      string          `json:"grade_level"`
	TeacherID       string          `json:"teacher_id"`
	Teacher         TeacherResponse `json:"teacher"`
	ActiveMissionID string          `json:"active_mission_id"`
	MissionCount    int64           `json:"mission_count"`
	Status          string          `json:"status"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
}

type MissionResponse struct {
	ID                       string                        `json:"id"`
	MissionCode              string                        `json:"mission_code"`
	OrganizationID           string                        `json:"organization_id"`
	DistrictID               string                        `json:"district_id"`
	SchoolID                 string                        `json:"school_id"`
	TeacherID                string                        `json:"teacher_id"`
	ClassID                  string                        `json:"class_id"`
	ClassCode                string                        `json:"class_code"`
	ClassName                string                        `json:"class_name"`
	MissionName              string                        `json:"mission_name"`
	Title                    string                        `json:"title"`
	Summary                  string                        `json:"summary"`
	Subject                  string                        `json:"subject"`
	GradeLevel               string                        `json:"grade_level"`
	LearningGoal             string                        `json:"learning_goal"`
	TeacherPrompt            string                        `json:"teacher_prompt"`
	Attachments              []AttachmentResponse          `json:"attachments"`
	PreviewQuestions         []PreviewQuestion             `json:"preview_questions"`
	KnowledgeChunks          []generation.KnowledgeChunk   `json:"knowledge_chunks"`
	SuggestedDurationMinutes int                           `json:"suggested_duration_minutes"`
	Stages                   []StageResponse               `json:"stages"`
	Suggestions              []SuggestionResponse          `json:"suggestions"`
	PositiveSignals          []string                      `json:"positive_signals"`
	ImprovementSignals       []string                      `json:"improvement_signals"`
	EducationGuardrail       generation.EducationGuardrail `json:"education_guardrail"`
	MissionFlow              generation.MissionFlow        `json:"mission_flow"`
	Orchestration            OrchestrationResponse         `json:"orchestration"`
	Status                   string                        `json:"status"`
	CreatedBy                TeacherResponse               `json:"created_by"`
	UpdatedBy                TeacherResponse               `json:"updated_by"`
	PublishedAt              string                        `json:"published_at"`
	CreatedAt                string                        `json:"created_at"`
	UpdatedAt                string                        `json:"updated_at"`
}

type AttachmentResponse struct {
	UploadcareUUID   string                             `json:"uploadcare_uuid"`
	CDNURL           string                             `json:"cdn_url"`
	UploadcareCDNURL string                             `json:"uploadcare_cdn_url,omitempty"`
	FileName         string                             `json:"file_name"`
	MimeType         string                             `json:"mime_type"`
	SizeBytes        int64                              `json:"size_bytes"`
	Storage          *missions.MissionAttachmentStorage `json:"storage,omitempty"`
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

type StageResponse struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Goal        string   `json:"goal"`
	Checks      []string `json:"checks"`
	UnlocksWhen string   `json:"unlocks_when"`
}

type OrchestrationResponse struct {
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

type SuggestionResponse struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Reason      string   `json:"reason"`
	Activity    string   `json:"activity"`
	Questions   []string `json:"questions"`
	FocusSkills []string `json:"focus_skills"`
}

type DashboardMetricsResponse struct {
	TotalStudents     int   `json:"total_students"`
	ActiveStudents    int   `json:"active_students"`
	CompletedStudents int   `json:"completed_students"`
	AverageProgress   int   `json:"average_progress"`
	NeedsAttention    int   `json:"needs_attention"`
	AbandonedStudents int   `json:"abandoned_students"`
	NotebookNotes     int64 `json:"notebook_notes"`
}

type StudentDashboardResponse struct {
	ID               string                           `json:"id"`
	StudentName      string                           `json:"student_name"`
	ClassID          string                           `json:"class_id"`
	Progress         StudentProgressResponse          `json:"progress"`
	Strengths        []string                         `json:"strengths"`
	ImprovementAreas []string                         `json:"improvement_areas"`
	LastActivityAt   string                           `json:"last_activity_at"`
	NotebookNotes    int64                            `json:"notebook_notes"`
	Session          StudentGameSessionResponse       `json:"session"`
	Attempts         []StudentLearningAttemptResponse `json:"attempts"`
}

type StudentLearningAttemptResponse struct {
	ID                string                                 `json:"id"`
	OrganizationID    string                                 `json:"organization_id"`
	DistrictID        string                                 `json:"district_id"`
	SchoolID          string                                 `json:"school_id"`
	StudentID         string                                 `json:"student_id"`
	StudentName       string                                 `json:"student_name"`
	ClassID           string                                 `json:"class_id"`
	MissionID         string                                 `json:"mission_id"`
	MissionCode       string                                 `json:"mission_code"`
	PublishedRevision int64                                  `json:"published_revision"`
	Subject           string                                 `json:"subject"`
	Status            string                                 `json:"status"`
	Snapshot          studentsmodule.StudentLearningSnapshot `json:"snapshot"`
	TutorTurns        []studentsmodule.StudentTutorTurn      `json:"tutor_turns"`
	Trace             generation.LearningTraceBlueprint      `json:"trace"`
	TraceState        string                                 `json:"trace_state"`
	StartedAt         string                                 `json:"started_at"`
	LastActivityAt    string                                 `json:"last_activity_at"`
	CompletedAt       string                                 `json:"completed_at"`
	AbandonedAt       string                                 `json:"abandoned_at"`
	CreatedAt         string                                 `json:"created_at"`
	UpdatedAt         string                                 `json:"updated_at"`
}

type ClassLearningSummaryResponse struct {
	TotalAttempts        int                            `json:"total_attempts"`
	CompletedAttempts    int                            `json:"completed_attempts"`
	AbandonedAttempts    int                            `json:"abandoned_attempts"`
	StudentsWithEvidence int                            `json:"students_with_evidence"`
	CommonStrengths      []ClassLearningPatternResponse `json:"common_strengths"`
	CommonNeeds          []ClassLearningPatternResponse `json:"common_needs"`
	NextAdaptation       string                         `json:"next_adaptation"`
}

type ClassLearningPatternResponse struct {
	Label        string   `json:"label"`
	Count        int      `json:"count"`
	StudentIDs   []string `json:"student_ids"`
	StudentNames []string `json:"student_names"`
}

type StudentGameSessionResponse struct {
	MissionID       string `json:"mission_id"`
	Subject         string `json:"subject"`
	StageID         string `json:"stage_id"`
	StageName       string `json:"stage_name"`
	Status          string `json:"status"`
	IsOnline        bool   `json:"is_online"`
	StartedAt       string `json:"started_at"`
	LastHeartbeatAt string `json:"last_heartbeat_at"`
}

type StudentProgressResponse struct {
	CompletedQuestions int    `json:"completed_questions"`
	TotalQuestions     int    `json:"total_questions"`
	Percentage         int    `json:"percentage"`
	Status             string `json:"status"`
}

type TeacherResponse struct {
	ProviderUserID string `json:"provider_user_id"`
	DisplayName    string `json:"display_name"`
	Email          string `json:"email"`
	PhotoURL       string `json:"photo_url"`
}

func ToOverviewResponse(overview DashboardOverview) OverviewResponse {
	return OverviewResponse{
		Page:            overview.Page,
		OrganizationID:  overview.OrganizationID,
		DistrictID:      overview.DistrictID,
		SchoolID:        overview.SchoolID,
		ClassID:         overview.ClassID,
		ClassCode:       overview.ClassCode,
		TeacherClass:    toTeacherClassResponse(overview.TeacherClass),
		ActiveMission:   toMissionResponse(overview.ActiveMission),
		Metrics:         toMetricsResponse(overview.Metrics),
		Students:        toStudentResponses(overview.Students),
		ClassLearning:   toClassLearningSummaryResponse(overview.ClassLearning),
		Suggestions:     toSuggestionResponses(overview.Suggestions),
		GeneratedByAI:   overview.GeneratedByAI,
		GeneratedByMock: overview.GeneratedByMock,
		Teacher:         toTeacherResponse(overview.Teacher),
	}
}

func toTeacherClassResponse(teacherClass *classrooms.TeacherClassDocument) *TeacherClassResponse {
	if teacherClass == nil {
		return nil
	}

	return &TeacherClassResponse{
		ID:              teacherClass.ID,
		OrganizationID:  teacherClass.OrganizationID,
		DistrictID:      teacherClass.DistrictID,
		SchoolID:        teacherClass.SchoolID,
		ClassID:         teacherClass.ClassID,
		ClassCode:       teacherClass.ClassCode,
		ClassName:       teacherClass.ClassName,
		Subject:         teacherClass.Subject,
		GradeLevel:      teacherClass.GradeLevel,
		TeacherID:       teacherClass.TeacherID,
		Teacher:         toClassTeacherResponse(teacherClass.Teacher),
		ActiveMissionID: teacherClass.ActiveMissionID,
		MissionCount:    teacherClass.MissionCount,
		Status:          teacherClass.Status,
		CreatedAt:       formatTime(teacherClass.CreatedAt),
		UpdatedAt:       formatTime(teacherClass.UpdatedAt),
	}
}

func toMissionResponse(mission *missions.MissionDocument) *MissionResponse {
	if mission == nil {
		return nil
	}

	return &MissionResponse{
		ID:                       mission.ID,
		MissionCode:              mission.MissionCode,
		OrganizationID:           mission.OrganizationID,
		DistrictID:               mission.DistrictID,
		SchoolID:                 mission.SchoolID,
		TeacherID:                mission.TeacherID,
		ClassID:                  mission.ClassID,
		ClassCode:                mission.ClassCode,
		ClassName:                mission.ClassName,
		MissionName:              mission.MissionName,
		Title:                    mission.Title,
		Summary:                  mission.Summary,
		Subject:                  mission.Subject,
		GradeLevel:               mission.GradeLevel,
		LearningGoal:             mission.LearningGoal,
		TeacherPrompt:            mission.TeacherPrompt,
		Attachments:              toAttachmentResponses(mission.Attachments),
		PreviewQuestions:         toPreviewQuestions(*mission),
		KnowledgeChunks:          mission.KnowledgeChunks,
		SuggestedDurationMinutes: mission.SuggestedDurationMinutes,
		Stages:                   toStageResponses(mission.Stages),
		Suggestions:              toSuggestionResponses(mission.Suggestions),
		PositiveSignals:          mission.PositiveSignals,
		ImprovementSignals:       mission.ImprovementSignals,
		EducationGuardrail:       mission.EducationGuardrail,
		MissionFlow:              mission.MissionFlow,
		Orchestration: OrchestrationResponse{
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
		Status:      mission.Status,
		CreatedBy:   toClassTeacherResponse(mission.CreatedBy),
		UpdatedBy:   toClassTeacherResponse(mission.UpdatedBy),
		PublishedAt: formatTime(mission.PublishedAt),
		CreatedAt:   formatTime(mission.CreatedAt),
		UpdatedAt:   formatTime(mission.UpdatedAt),
	}
}

func toMetricsResponse(metrics DashboardMetrics) DashboardMetricsResponse {
	return DashboardMetricsResponse{
		TotalStudents:     metrics.TotalStudents,
		ActiveStudents:    metrics.ActiveStudents,
		CompletedStudents: metrics.CompletedStudents,
		AverageProgress:   metrics.AverageProgress,
		NeedsAttention:    metrics.NeedsAttention,
		AbandonedStudents: metrics.AbandonedStudents,
		NotebookNotes:     metrics.NotebookNotes,
	}
}

func toStudentResponses(students []StudentDashboardItem) []StudentDashboardResponse {
	responses := make([]StudentDashboardResponse, 0, len(students))

	for _, student := range students {
		responses = append(responses, StudentDashboardResponse{
			ID:          student.ID,
			StudentName: student.StudentName,
			ClassID:     student.ClassID,
			Progress: StudentProgressResponse{
				CompletedQuestions: student.Progress.CompletedQuestions,
				TotalQuestions:     student.Progress.TotalQuestions,
				Percentage:         student.Progress.Percentage,
				Status:             student.Progress.Status,
			},
			Strengths:        nonNilSlice(student.Strengths),
			ImprovementAreas: nonNilSlice(student.ImprovementAreas),
			LastActivityAt:   formatTime(student.LastActivityAt),
			NotebookNotes:    student.NotebookNotes,
			Session: StudentGameSessionResponse{
				MissionID:       student.Session.MissionID,
				Subject:         student.Session.Subject,
				StageID:         student.Session.StageID,
				StageName:       student.Session.StageName,
				Status:          student.Session.Status,
				IsOnline:        student.Session.IsOnline,
				StartedAt:       formatTime(student.Session.StartedAt),
				LastHeartbeatAt: formatTime(student.Session.LastHeartbeatAt),
			},
			Attempts: toStudentLearningAttemptResponses(student.Attempts),
		})
	}

	return responses
}

func toStudentLearningAttemptResponses(
	attempts []studentsmodule.StudentMissionAttemptDocument,
) []StudentLearningAttemptResponse {
	responses := make([]StudentLearningAttemptResponse, 0, len(attempts))
	for _, attempt := range attempts {
		attempt = normalizeStudentLearningAttemptCollections(attempt)
		responses = append(responses, StudentLearningAttemptResponse{
			ID:                attempt.ID,
			OrganizationID:    attempt.OrganizationID,
			DistrictID:        attempt.DistrictID,
			SchoolID:          attempt.SchoolID,
			StudentID:         attempt.StudentID,
			StudentName:       attempt.StudentName,
			ClassID:           attempt.ClassID,
			MissionID:         attempt.MissionID,
			MissionCode:       attempt.MissionCode,
			PublishedRevision: attempt.PublishedRevision,
			Subject:           attempt.Subject,
			Status:            attempt.Status,
			Snapshot:          attempt.Snapshot,
			TutorTurns:        attempt.TutorTurns,
			Trace:             attempt.Trace,
			TraceState:        attempt.TraceState,
			StartedAt:         formatTime(attempt.StartedAt),
			LastActivityAt:    formatTime(attempt.LastActivityAt),
			CompletedAt:       formatTime(attempt.CompletedAt),
			AbandonedAt:       formatTime(attempt.AbandonedAt),
			CreatedAt:         formatTime(attempt.CreatedAt),
			UpdatedAt:         formatTime(attempt.UpdatedAt),
		})
	}

	return responses
}

func normalizeStudentLearningAttemptCollections(
	attempt studentsmodule.StudentMissionAttemptDocument,
) studentsmodule.StudentMissionAttemptDocument {
	attempt.Snapshot.Progress = nonNilSlice(attempt.Snapshot.Progress)
	attempt.Snapshot.Notes = nonNilSlice(attempt.Snapshot.Notes)
	attempt.Snapshot.Decisions = nonNilSlice(attempt.Snapshot.Decisions)
	attempt.Snapshot.Events = nonNilSlice(attempt.Snapshot.Events)
	attempt.Snapshot.Checkpoint.Lines = nonNilSlice(attempt.Snapshot.Checkpoint.Lines)
	for index := range attempt.Snapshot.Progress {
		attempt.Snapshot.Progress[index].Queue = nonNilSlice(attempt.Snapshot.Progress[index].Queue)
		attempt.Snapshot.Progress[index].Pool = nonNilSlice(attempt.Snapshot.Progress[index].Pool)
	}
	for index := range attempt.Snapshot.Notes {
		attempt.Snapshot.Notes[index].Evidence = nonNilSlice(attempt.Snapshot.Notes[index].Evidence)
	}

	attempt.TutorTurns = nonNilSlice(attempt.TutorTurns)
	for index := range attempt.TutorTurns {
		attempt.TutorTurns[index].Tutor.ExplanationSteps = nonNilSlice(
			attempt.TutorTurns[index].Tutor.ExplanationSteps,
		)
		attempt.TutorTurns[index].Tutor.RetrievedChunkIDs = nonNilSlice(
			attempt.TutorTurns[index].Tutor.RetrievedChunkIDs,
		)
		attempt.TutorTurns[index].Tutor.EvidenceEventIDs = nonNilSlice(
			attempt.TutorTurns[index].Tutor.EvidenceEventIDs,
		)
	}

	attempt.Trace.Strengths = nonNilSlice(attempt.Trace.Strengths)
	attempt.Trace.ImprovementAreas = nonNilSlice(attempt.Trace.ImprovementAreas)
	attempt.Trace.Flow.Nodes = nonNilSlice(attempt.Trace.Flow.Nodes)
	attempt.Trace.Flow.Edges = nonNilSlice(attempt.Trace.Flow.Edges)
	for index := range attempt.Trace.Flow.Nodes {
		attempt.Trace.Flow.Nodes[index].EvidenceEventIDs = nonNilSlice(
			attempt.Trace.Flow.Nodes[index].EvidenceEventIDs,
		)
	}

	return attempt
}

func nonNilSlice[Value any](values []Value) []Value {
	if values == nil {
		return []Value{}
	}

	return values
}

func toClassLearningSummaryResponse(summary ClassLearningSummary) ClassLearningSummaryResponse {
	return ClassLearningSummaryResponse{
		TotalAttempts:        summary.TotalAttempts,
		CompletedAttempts:    summary.CompletedAttempts,
		AbandonedAttempts:    summary.AbandonedAttempts,
		StudentsWithEvidence: summary.StudentsWithEvidence,
		CommonStrengths:      toClassLearningPatternResponses(summary.CommonStrengths),
		CommonNeeds:          toClassLearningPatternResponses(summary.CommonNeeds),
		NextAdaptation:       summary.NextAdaptation,
	}
}

func toClassLearningPatternResponses(patterns []ClassLearningPattern) []ClassLearningPatternResponse {
	responses := make([]ClassLearningPatternResponse, 0, len(patterns))
	for _, pattern := range patterns {
		responses = append(responses, ClassLearningPatternResponse{
			Label:        pattern.Label,
			Count:        pattern.Count,
			StudentIDs:   pattern.StudentIDs,
			StudentNames: pattern.StudentNames,
		})
	}

	return responses
}

func toStageResponses(stages []generation.MissionStage) []StageResponse {
	responses := make([]StageResponse, 0, len(stages))

	for _, stage := range stages {
		responses = append(responses, StageResponse{
			ID:          stage.ID,
			Title:       stage.Title,
			Goal:        stage.Goal,
			Checks:      stage.Checks,
			UnlocksWhen: stage.UnlocksWhen,
		})
	}

	return responses
}

func toAttachmentResponses(attachments []missions.MissionAttachment) []AttachmentResponse {
	responses := make([]AttachmentResponse, 0, len(attachments))

	for _, attachment := range attachments {
		attachment = missions.PrivateAttachmentResponse(attachment)
		responses = append(responses, AttachmentResponse{
			UploadcareUUID:   attachment.UploadcareUUID,
			CDNURL:           attachment.CDNURL,
			UploadcareCDNURL: attachment.UploadcareCDNURL,
			FileName:         attachment.FileName,
			MimeType:         attachment.MimeType,
			SizeBytes:        attachment.SizeBytes,
			Storage:          missions.AttachmentStorageWithPublicURL(attachment.Storage),
		})
	}

	return responses
}

func toPreviewQuestions(mission missions.MissionDocument) []PreviewQuestion {
	questions := generation.PlayablePreviewQuestions(generation.BuildMissionInput{
		ClassID:       mission.ClassID,
		ClassName:     mission.ClassName,
		MissionName:   mission.MissionName,
		Subject:       mission.Subject,
		GradeLevel:    mission.GradeLevel,
		LearningGoal:  mission.LearningGoal,
		TeacherPrompt: mission.TeacherPrompt,
	}, mission.PreviewQuestions)
	responses := make([]PreviewQuestion, 0, len(questions))

	for _, question := range questions {
		responses = append(responses, PreviewQuestion{
			ID:                 question.ID,
			StageID:            question.StageID,
			Prompt:             question.Prompt,
			AnswerType:         question.AnswerType,
			Options:            question.Options,
			ExpectedAnswer:     question.ExpectedAnswer,
			ExplanationSteps:   question.ExplanationSteps,
			MisconceptionTraps: question.MisconceptionTraps,
		})
	}

	return responses
}

func toSuggestionResponses(suggestions []generation.MissionSuggestion) []SuggestionResponse {
	responses := make([]SuggestionResponse, 0, len(suggestions))

	for _, suggestion := range suggestions {
		responses = append(responses, SuggestionResponse{
			ID:          suggestion.ID,
			Title:       suggestion.Title,
			Reason:      suggestion.Reason,
			Activity:    suggestion.Activity,
			Questions:   suggestion.Questions,
			FocusSkills: suggestion.FocusSkills,
		})
	}

	return responses
}

func toTeacherResponse(teacher TeacherSummary) TeacherResponse {
	return TeacherResponse{
		ProviderUserID: teacher.ProviderUserID,
		DisplayName:    teacher.DisplayName,
		Email:          teacher.Email,
		PhotoURL:       teacher.PhotoURL,
	}
}

func toClassTeacherResponse(teacher classrooms.TeacherSummary) TeacherResponse {
	return TeacherResponse{
		ProviderUserID: teacher.ProviderUserID,
		DisplayName:    teacher.DisplayName,
		Email:          teacher.Email,
		PhotoURL:       teacher.PhotoURL,
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.UTC().Format(time.RFC3339Nano)
}
