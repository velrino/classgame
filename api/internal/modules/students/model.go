package students

import (
	"time"

	"classgame-api/internal/auth"
	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/missions"
	"classgame-api/internal/modules/tenancy"
)

type UserRole string

const (
	UserRoleStudent UserRole = "student"
	UserRoleTeacher UserRole = "teacher"
)

type UserDocument struct {
	ID        string         `firestore:"id" json:"id"`
	Identity  UserIdentity   `firestore:"identity" json:"identity"`
	Profile   UserProfile    `firestore:"profile" json:"profile"`
	Student   StudentProfile `firestore:"student" json:"student"`
	Activity  UserActivity   `firestore:"activity" json:"activity"`
	Roles     []UserRole     `firestore:"roles" json:"roles"`
	IsGuest   bool           `firestore:"is_guest" json:"is_guest"`
	DeletedAt *time.Time     `firestore:"deleted_at" json:"deleted_at"`
	CreatedAt time.Time      `firestore:"created_at" json:"created_at"`
	UpdatedAt time.Time      `firestore:"updated_at" json:"updated_at"`
	// SessionMission is populated only for the authenticated student-session
	// response and is never written to the user document.
	SessionMission StudentSessionMission `firestore:"-" json:"-"`
	SessionClass   StudentSessionClass   `firestore:"-" json:"-"`
}

type UserIdentity struct {
	Provider       auth.IdentityProvider  `firestore:"provider" json:"provider"`
	ProviderUserID string                 `firestore:"provider_user_id" json:"provider_user_id"`
	AuthMethod     auth.AuthMethod        `firestore:"auth_method" json:"auth_method"`
	ProviderIDs    []string               `firestore:"provider_ids" json:"provider_ids"`
	SignInProvider string                 `firestore:"sign_in_provider" json:"sign_in_provider"`
	CustomClaims   map[string]interface{} `firestore:"custom_claims" json:"custom_claims"`
	Token          IdentityToken          `firestore:"token" json:"token"`
}

type IdentityToken struct {
	AuthTime  int64  `firestore:"auth_time" json:"auth_time"`
	ExpiresAt int64  `firestore:"expires_at" json:"expires_at"`
	IssuedAt  int64  `firestore:"issued_at" json:"issued_at"`
	Issuer    string `firestore:"issuer" json:"issuer"`
	Audience  string `firestore:"audience" json:"audience"`
}

type UserProfile struct {
	DisplayName   string `firestore:"display_name" json:"display_name"`
	Email         string `firestore:"email" json:"email"`
	EmailVerified bool   `firestore:"email_verified" json:"email_verified"`
	PhotoURL      string `firestore:"photo_url" json:"photo_url"`
}

type StudentProfile struct {
	OrganizationID string                `firestore:"organization_id" json:"organization_id"`
	DistrictID     string                `firestore:"district_id" json:"district_id"`
	SchoolID       string                `firestore:"school_id" json:"school_id"`
	ClassID        string                `firestore:"class_id" json:"class_id"`
	ClassCode      string                `firestore:"class_code" json:"class_code"`
	MissionID      string                `firestore:"mission_id" json:"mission_id"`
	MissionCode    string                `firestore:"mission_code" json:"mission_code"`
	StudentName    string                `firestore:"student_name" json:"student_name"`
	LearningMemory StudentLearningMemory `firestore:"learning_memory" json:"learning_memory"`
}

type UserActivity struct {
	FirstSeenAt time.Time          `firestore:"first_seen_at" json:"first_seen_at"`
	LastSeenAt  time.Time          `firestore:"last_seen_at" json:"last_seen_at"`
	LoginCount  int64              `firestore:"login_count" json:"login_count"`
	CurrentGame StudentGameSession `firestore:"current_game" json:"current_game"`
}

type StudentGameSession struct {
	AttemptID         string    `firestore:"attempt_id" json:"attempt_id"`
	MissionID         string    `firestore:"mission_id" json:"mission_id"`
	PublishedRevision int64     `firestore:"published_revision" json:"published_revision"`
	Subject           string    `firestore:"subject" json:"subject"`
	StageID           string    `firestore:"stage_id" json:"stage_id"`
	StageName         string    `firestore:"stage_name" json:"stage_name"`
	Status            string    `firestore:"status" json:"status"`
	StartedAt         time.Time `firestore:"started_at" json:"started_at"`
	LastHeartbeatAt   time.Time `firestore:"last_heartbeat_at" json:"last_heartbeat_at"`
}

type UpsertStudentInput struct {
	Identity                 auth.Identity
	Scope                    tenancy.Scope
	ClassID                  string
	ClassCode                string
	MissionID                string
	MissionCode              string
	MissionPublishedRevision int64
	MissionSubject           string
	StudentName              string
	IncrementLogin           bool
	Timestamp                time.Time
}

type UpdateStudentPresenceInput struct {
	Identity  auth.Identity
	MissionID string
	Subject   string
	StageID   string
	StageName string
	Status    string
	Timestamp time.Time
}

type StudentLearningMemory struct {
	Strengths            []string  `firestore:"strengths" json:"strengths"`
	ImprovementAreas     []string  `firestore:"improvement_areas" json:"improvement_areas"`
	PreferredExplanation string    `firestore:"preferred_explanation" json:"preferred_explanation"`
	NextAdaptation       string    `firestore:"next_adaptation" json:"next_adaptation"`
	LastMissionID        string    `firestore:"last_mission_id" json:"last_mission_id"`
	EvidenceAt           time.Time `firestore:"evidence_at" json:"evidence_at"`
	UpdatedAt            time.Time `firestore:"updated_at" json:"updated_at"`
}

type StudentMissionAttemptDocument struct {
	ID                  string                            `firestore:"id" json:"id"`
	OrganizationID      string                            `firestore:"organization_id" json:"organization_id"`
	DistrictID          string                            `firestore:"district_id" json:"district_id"`
	SchoolID            string                            `firestore:"school_id" json:"school_id"`
	StudentID           string                            `firestore:"student_id" json:"student_id"`
	StudentName         string                            `firestore:"student_name" json:"student_name"`
	ClassID             string                            `firestore:"class_id" json:"class_id"`
	MissionID           string                            `firestore:"mission_id" json:"mission_id"`
	MissionCode         string                            `firestore:"mission_code" json:"mission_code"`
	PublishedRevision   int64                             `firestore:"published_revision" json:"published_revision"`
	Subject             string                            `firestore:"subject" json:"subject"`
	Status              string                            `firestore:"status" json:"status"`
	Snapshot            StudentLearningSnapshot           `firestore:"snapshot" json:"snapshot"`
	TutorTurns          []StudentTutorTurn                `firestore:"tutor_turns" json:"tutor_turns"`
	Trace               generation.LearningTraceBlueprint `firestore:"trace" json:"trace"`
	TraceState          string                            `firestore:"trace_state" json:"trace_state"`
	FinalizeRequestedAt time.Time                         `firestore:"finalize_requested_at" json:"finalize_requested_at"`
	StartedAt           time.Time                         `firestore:"started_at" json:"started_at"`
	LastActivityAt      time.Time                         `firestore:"last_activity_at" json:"last_activity_at"`
	CompletedAt         time.Time                         `firestore:"completed_at" json:"completed_at"`
	AbandonedAt         time.Time                         `firestore:"abandoned_at" json:"abandoned_at"`
	CreatedAt           time.Time                         `firestore:"created_at" json:"created_at"`
	UpdatedAt           time.Time                         `firestore:"updated_at" json:"updated_at"`
}

type StudentLearningSnapshot struct {
	Revision    int64                    `firestore:"revision" json:"revision"`
	Progress    []StudentStageProgress   `firestore:"progress" json:"progress"`
	Notes       []StudentLearningNote    `firestore:"notes" json:"notes"`
	Preferences StudentPreferences       `firestore:"preferences" json:"preferences"`
	Decisions   []StudentDecision        `firestore:"decisions" json:"decisions"`
	Events      []StudentLearningEvent   `firestore:"events" json:"events"`
	Redirected  bool                     `firestore:"redirected" json:"redirected"`
	ReturnTo    string                   `firestore:"return_to" json:"return_to"`
	LastPod     string                   `firestore:"last_pod" json:"last_pod"`
	Checkpoint  StudentSessionCheckpoint `firestore:"checkpoint" json:"checkpoint"`
}

// StudentSessionCheckpoint is the exact recoverable UI position. It contains
// no teacher notes or hidden model reasoning.
type StudentSessionCheckpoint struct {
	StageID              string   `firestore:"stage_id" json:"stage_id"`
	QuestionID           string   `firestore:"question_id" json:"question_id"`
	Phase                string   `firestore:"phase" json:"phase"`
	StepIndex            int      `firestore:"step_index" json:"step_index"`
	ExplainIndex         int      `firestore:"explain_index" json:"explain_index"`
	Lines                []string `firestore:"lines" json:"lines"`
	WrongChoice          string   `firestore:"wrong_choice" json:"wrong_choice"`
	WrongMisconceptionID string   `firestore:"wrong_misconception_id" json:"wrong_misconception_id"`
	DecisionAction       string   `firestore:"decision_action" json:"decision_action"`
	DecisionTarget       string   `firestore:"decision_target" json:"decision_target"`
	DecisionMessage      string   `firestore:"decision_message" json:"decision_message"`
	DecisionWhy          string   `firestore:"decision_why" json:"decision_why"`
	TutorMessage         string   `firestore:"tutor_message" json:"tutor_message"`
	WhyPrompt            string   `firestore:"why_prompt" json:"why_prompt"`
	Clean                bool     `firestore:"clean" json:"clean"`
	Seated               bool     `firestore:"seated" json:"seated"`
}

type StudentStageProgress struct {
	StageID  string   `firestore:"stage_id" json:"stage_id"`
	Status   string   `firestore:"status" json:"status"`
	Queue    []string `firestore:"queue" json:"queue"`
	Pool     []string `firestore:"pool" json:"pool"`
	Correct  int      `firestore:"correct" json:"correct"`
	Required int      `firestore:"required" json:"required"`
}

type StudentLearningNote struct {
	ID              string   `firestore:"id" json:"id"`
	MisconceptionID string   `firestore:"misconception_id" json:"misconception_id"`
	Text            string   `firestore:"text" json:"text"`
	Kind            string   `firestore:"kind" json:"kind"`
	Skill           string   `firestore:"skill" json:"skill"`
	Evidence        []string `firestore:"evidence" json:"evidence"`
	Count           int      `firestore:"count" json:"count"`
	CreatedAt       int64    `firestore:"created_at" json:"created_at"`
}

type StudentPreferences struct {
	ExplanationStyle string `firestore:"explanation_style" json:"explanation_style"`
	DifficultyBias   int    `firestore:"difficulty_bias" json:"difficulty_bias"`
}

type StudentDecision struct {
	ID      string `firestore:"id" json:"id"`
	At      int64  `firestore:"at" json:"at"`
	Action  string `firestore:"action" json:"action"`
	Message string `firestore:"message" json:"message"`
	Why     string `firestore:"why" json:"why"`
}

type StudentLearningEvent struct {
	ID         string `firestore:"id" json:"id"`
	Sequence   int    `firestore:"sequence" json:"sequence"`
	Type       string `firestore:"type" json:"type"`
	StageID    string `firestore:"stage_id" json:"stage_id"`
	QuestionID string `firestore:"question_id" json:"question_id"`
	Title      string `firestore:"title" json:"title"`
	Detail     string `firestore:"detail" json:"detail"`
	Outcome    string `firestore:"outcome" json:"outcome"`
	OccurredAt int64  `firestore:"occurred_at" json:"occurred_at"`
}

type SaveStudentLearningSnapshotInput struct {
	Mission           missions.MissionDocument
	PublishedRevision int64
	Identity          auth.Identity
	MissionID         string
	Status            string
	Snapshot          StudentLearningSnapshot
	Trace             generation.LearningTraceBlueprint
	Timestamp         time.Time
}

type StudentSessionMission struct {
	ID                string                       `json:"id"`
	MissionCode       string                       `json:"mission_code"`
	PublishedRevision int64                        `json:"published_revision"`
	MissionName       string                       `json:"mission_name"`
	Title             string                       `json:"title"`
	Subject           string                       `json:"subject"`
	GradeLevel        string                       `json:"grade_level"`
	LearningGoal      string                       `json:"learning_goal"`
	Stages            []generation.MissionStage    `json:"stages"`
	PreviewQuestions  []generation.PreviewQuestion `json:"preview_questions"`
}

type StudentSessionClass struct {
	ID         string `json:"id"`
	ClassCode  string `json:"class_code"`
	ClassName  string `json:"class_name"`
	Subject    string `json:"subject"`
	GradeLevel string `json:"grade_level"`
}

type StudentTutorTurn struct {
	ID                string                        `firestore:"id" json:"id"`
	Sequence          int                           `firestore:"sequence" json:"sequence"`
	Type              string                        `firestore:"type" json:"type"`
	StageID           string                        `firestore:"stage_id" json:"stage_id"`
	QuestionID        string                        `firestore:"question_id" json:"question_id"`
	Prompt            string                        `firestore:"prompt" json:"prompt"`
	StudentResponse   string                        `firestore:"student_response" json:"student_response"`
	ExpectedAnswer    string                        `firestore:"expected_answer" json:"expected_answer"`
	Outcome           string                        `firestore:"outcome" json:"outcome"`
	ExplicitReasoning string                        `firestore:"explicit_reasoning" json:"explicit_reasoning"`
	Feedback          string                        `firestore:"feedback" json:"feedback"`
	Tutor             generation.TutorTurnBlueprint `firestore:"tutor" json:"tutor"`
	CreatedAt         time.Time                     `firestore:"created_at" json:"created_at"`
}

type SaveStudentTutorTurnInput struct {
	ExpectedAttemptID         string
	ExpectedPublishedRevision int64
	Identity                  auth.Identity
	MissionID                 string
	EventID                   string
	Type                      string
	StageID                   string
	QuestionID                string
	Prompt                    string
	StudentResponse           string
	ExpectedAnswer            string
	Outcome                   string
	ExplicitReasoning         string
	Feedback                  string
	Timestamp                 time.Time
}

type ClassLearningSynthesisDocument struct {
	OrganizationID    string                            `firestore:"organization_id" json:"organization_id"`
	DistrictID        string                            `firestore:"district_id" json:"district_id"`
	SchoolID          string                            `firestore:"school_id" json:"school_id"`
	ClassID           string                            `firestore:"class_id" json:"class_id"`
	Synthesis         generation.ClassLearningSynthesis `firestore:"synthesis" json:"synthesis"`
	AttemptCount      int                               `firestore:"attempt_count" json:"attempt_count"`
	EvidenceUpdatedAt time.Time                         `firestore:"evidence_updated_at" json:"evidence_updated_at"`
	UpdatedAt         time.Time                         `firestore:"updated_at" json:"updated_at"`
}
