package insights

import (
	"time"

	"classgame-api/internal/auth"
	"classgame-api/internal/modules/classrooms"
	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/missions"
	"classgame-api/internal/modules/students"
)

const (
	teachersCollectionName = "teachers"
	usersCollectionName    = "users"
)

type TeacherIdentity struct {
	Provider       auth.IdentityProvider
	ProviderUserID string
	AuthMethod     auth.AuthMethod
	Email          string
	EmailVerified  bool
	DisplayName    string
	PhotoURL       string
	ProviderIDs    []string
	SignInProvider string
}

type TeacherSummary struct {
	ProviderUserID string `firestore:"provider_user_id" json:"provider_user_id"`
	DisplayName    string `firestore:"display_name" json:"display_name"`
	Email          string `firestore:"email" json:"email"`
	PhotoURL       string `firestore:"photo_url" json:"photo_url"`
}

type TeacherDocument struct {
	ID             string          `firestore:"id" json:"id"`
	OrganizationID string          `firestore:"organization_id" json:"organization_id"`
	DistrictID     string          `firestore:"district_id" json:"district_id"`
	SchoolID       string          `firestore:"school_id" json:"school_id"`
	Identity       TeacherIdentity `firestore:"identity" json:"identity"`
	Profile        TeacherSummary  `firestore:"profile" json:"profile"`
	Roles          []string        `firestore:"roles" json:"roles"`
	Activity       TeacherActivity `firestore:"activity" json:"activity"`
	CreatedAt      time.Time       `firestore:"created_at" json:"created_at"`
	UpdatedAt      time.Time       `firestore:"updated_at" json:"updated_at"`
}

type TeacherActivity struct {
	FirstSeenAt time.Time `firestore:"first_seen_at" json:"first_seen_at"`
	LastSeenAt  time.Time `firestore:"last_seen_at" json:"last_seen_at"`
	LoginCount  int64     `firestore:"login_count" json:"login_count"`
}

type StudentDashboardItem struct {
	ID               string
	StudentName      string
	ClassID          string
	Progress         StudentProgress
	Strengths        []string
	ImprovementAreas []string
	LastActivityAt   time.Time
	NotebookNotes    int64
	Session          StudentGameSession
	Attempts         []students.StudentMissionAttemptDocument
}

type StudentGameSession struct {
	MissionID       string
	Subject         string
	StageID         string
	StageName       string
	Status          string
	IsOnline        bool
	StartedAt       time.Time
	LastHeartbeatAt time.Time
}

type StudentProgress struct {
	CompletedQuestions int
	TotalQuestions     int
	Percentage         int
	Status             string
}

type DashboardMetrics struct {
	TotalStudents     int
	ActiveStudents    int
	CompletedStudents int
	AverageProgress   int
	NeedsAttention    int
	AbandonedStudents int
	NotebookNotes     int64
}

type DashboardOverview struct {
	Page            *OverviewPageMetadata
	OrganizationID  string
	DistrictID      string
	SchoolID        string
	ClassID         string
	ClassCode       string
	TeacherClass    *classrooms.TeacherClassDocument
	ActiveMission   *missions.MissionDocument
	Metrics         DashboardMetrics
	Students        []StudentDashboardItem
	ClassLearning   ClassLearningSummary
	Suggestions     []generation.MissionSuggestion
	GeneratedByAI   bool
	GeneratedByMock bool
	Teacher         TeacherSummary
}

type ClassLearningSummary struct {
	TotalAttempts        int
	CompletedAttempts    int
	AbandonedAttempts    int
	StudentsWithEvidence int
	CommonStrengths      []ClassLearningPattern
	CommonNeeds          []ClassLearningPattern
	NextAdaptation       string
}

type ClassLearningPattern struct {
	Label        string
	Count        int
	StudentIDs   []string
	StudentNames []string
}

type studentUserDocument struct {
	ID       string             `firestore:"id"`
	Profile  studentUserProfile `firestore:"profile"`
	Student  studentProfile     `firestore:"student"`
	Activity studentActivity    `firestore:"activity"`
	Roles    []string           `firestore:"roles"`
}

type studentUserProfile struct {
	DisplayName string `firestore:"display_name"`
	PhotoURL    string `firestore:"photo_url"`
}

type studentProfile struct {
	OrganizationID string `firestore:"organization_id"`
	DistrictID     string `firestore:"district_id"`
	SchoolID       string `firestore:"school_id"`
	ClassID        string `firestore:"class_id"`
	StudentName    string `firestore:"student_name"`
}

type studentActivity struct {
	FirstSeenAt time.Time          `firestore:"first_seen_at"`
	LastSeenAt  time.Time          `firestore:"last_seen_at"`
	LoginCount  int64              `firestore:"login_count"`
	CurrentGame studentGameSession `firestore:"current_game"`
}

type studentGameSession struct {
	MissionID       string    `firestore:"mission_id"`
	Subject         string    `firestore:"subject"`
	StageID         string    `firestore:"stage_id"`
	StageName       string    `firestore:"stage_name"`
	Status          string    `firestore:"status"`
	StartedAt       time.Time `firestore:"started_at"`
	LastHeartbeatAt time.Time `firestore:"last_heartbeat_at"`
}
