package classrooms

import "time"

type TeacherClassResponse struct {
	Enrollment      *Enrollment     `json:"enrollment,omitempty"`
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

type TeacherResponse struct {
	ProviderUserID string `json:"provider_user_id"`
	DisplayName    string `json:"display_name"`
	Email          string `json:"email"`
	PhotoURL       string `json:"photo_url"`
}

func ToResponse(teacherClass TeacherClassDocument) TeacherClassResponse {
	return TeacherClassResponse{
		Enrollment:      teacherClass.Enrollment,
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
		Teacher:         toTeacherResponse(teacherClass.Teacher),
		ActiveMissionID: teacherClass.ActiveMissionID,
		MissionCount:    teacherClass.MissionCount,
		Status:          teacherClass.Status,
		CreatedAt:       formatTime(teacherClass.CreatedAt),
		UpdatedAt:       formatTime(teacherClass.UpdatedAt),
	}
}

func ToResponses(teacherClasses []TeacherClassDocument) []TeacherClassResponse {
	responses := make([]TeacherClassResponse, 0, len(teacherClasses))
	for _, teacherClass := range teacherClasses {
		responses = append(responses, ToResponse(teacherClass))
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

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.UTC().Format(time.RFC3339Nano)
}
