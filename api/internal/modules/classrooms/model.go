package classrooms

import (
	"time"

	"classgame-api/internal/auth"
	"classgame-api/internal/modules/tenancy"
)

const CollectionName = "teacher_classes"

type TeacherSummary struct {
	ProviderUserID string `firestore:"provider_user_id" json:"provider_user_id"`
	DisplayName    string `firestore:"display_name" json:"display_name"`
	Email          string `firestore:"email" json:"email"`
	PhotoURL       string `firestore:"photo_url" json:"photo_url"`
}

type TeacherClassDocument struct {
	Enrollment      *Enrollment    `firestore:"enrollment,omitempty" json:"enrollment,omitempty"`
	ID              string         `firestore:"id" json:"id"`
	OrganizationID  string         `firestore:"organization_id" json:"organization_id"`
	DistrictID      string         `firestore:"district_id" json:"district_id"`
	SchoolID        string         `firestore:"school_id" json:"school_id"`
	ClassID         string         `firestore:"class_id" json:"class_id"`
	ClassCode       string         `firestore:"class_code" json:"class_code"`
	ClassName       string         `firestore:"class_name" json:"class_name"`
	Subject         string         `firestore:"subject" json:"subject"`
	GradeLevel      string         `firestore:"grade_level" json:"grade_level"`
	TeacherID       string         `firestore:"teacher_id" json:"teacher_id"`
	Teacher         TeacherSummary `firestore:"teacher" json:"teacher"`
	ActiveMissionID string         `firestore:"active_mission_id" json:"active_mission_id"`
	MissionCount    int64          `firestore:"mission_count" json:"mission_count"`
	Status          string         `firestore:"status" json:"status"`
	CreatedAt       time.Time      `firestore:"created_at" json:"created_at"`
	UpdatedAt       time.Time      `firestore:"updated_at" json:"updated_at"`
}

type UpsertTeacherClassInput struct {
	Enrollment *Enrollment
	Scope      tenancy.Scope
	ClassID    string
	ClassName  string
	Subject    string
	GradeLevel string
	Teacher    TeacherSummary
	Timestamp  time.Time
}

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
