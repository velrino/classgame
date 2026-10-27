package students

import "time"

type UserResponse struct {
	ID        string                 `json:"id"`
	Identity  UserIdentityResponse   `json:"identity"`
	Profile   UserProfileResponse    `json:"profile"`
	Student   StudentProfileResponse `json:"student"`
	Activity  UserActivityResponse   `json:"activity"`
	Roles     []UserRole             `json:"roles"`
	IsGuest   bool                   `json:"is_guest"`
	DeletedAt *string                `json:"deleted_at"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
}

type UserIdentityResponse struct {
	Provider       string `json:"provider"`
	ProviderUserID string `json:"provider_user_id"`
	AuthMethod     string `json:"auth_method"`
	SignInProvider string `json:"sign_in_provider"`
}

type UserProfileResponse struct {
	DisplayName   string `json:"display_name"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	PhotoURL      string `json:"photo_url"`
}

type StudentProfileResponse struct {
	OrganizationID string `json:"organization_id"`
	DistrictID     string `json:"district_id"`
	SchoolID       string `json:"school_id"`
	ClassID        string `json:"class_id"`
	ClassCode      string `json:"class_code"`
	MissionID      string `json:"mission_id"`
	MissionCode    string `json:"mission_code"`
	StudentName    string `json:"student_name"`
}

type UserActivityResponse struct {
	FirstSeenAt string                     `json:"first_seen_at"`
	LastSeenAt  string                     `json:"last_seen_at"`
	LoginCount  int64                      `json:"login_count"`
	CurrentGame StudentGameSessionResponse `json:"current_game"`
}

type StudentGameSessionResponse struct {
	AttemptID         string `json:"attempt_id"`
	MissionID         string `json:"mission_id"`
	PublishedRevision int64  `json:"published_revision"`
	Subject           string `json:"subject"`
	StageID           string `json:"stage_id"`
	StageName         string `json:"stage_name"`
	Status            string `json:"status"`
	StartedAt         string `json:"started_at"`
	LastHeartbeatAt   string `json:"last_heartbeat_at"`
}

type StudentLearningResumeResponse struct {
	ID                string                        `json:"id"`
	MissionID         string                        `json:"mission_id"`
	PublishedRevision int64                         `json:"published_revision"`
	Status            string                        `json:"status"`
	Snapshot          StudentResumeSnapshotResponse `json:"snapshot"`
	LastEventSequence int                           `json:"last_event_sequence"`
	StartedAt         string                        `json:"started_at"`
	LastActivityAt    string                        `json:"last_activity_at"`
}

type StudentResumeSnapshotResponse struct {
	Revision    int64                    `json:"revision"`
	Progress    []StudentStageProgress   `json:"progress"`
	Preferences StudentPreferences       `json:"preferences"`
	Redirected  bool                     `json:"redirected"`
	ReturnTo    string                   `json:"return_to"`
	LastPod     string                   `json:"last_pod"`
	Checkpoint  StudentSessionCheckpoint `json:"checkpoint"`
}

type StudentTutorTurnResponse struct {
	TurnID           string   `json:"turn_id"`
	Sequence         int      `json:"sequence"`
	Message          string   `json:"message"`
	WhyPrompt        string   `json:"why_prompt"`
	ExplanationSteps []string `json:"explanation_steps"`
	ExplanationStyle string   `json:"explanation_style"`
	DifficultyBias   int      `json:"difficulty_bias"`
	NextAction       string   `json:"next_action"`
}

func ToResponse(user UserDocument) UserResponse {
	return UserResponse{
		ID: user.ID,
		Identity: UserIdentityResponse{
			Provider:       string(user.Identity.Provider),
			ProviderUserID: user.Identity.ProviderUserID,
			AuthMethod:     string(user.Identity.AuthMethod),
			SignInProvider: user.Identity.SignInProvider,
		},
		Profile: UserProfileResponse{
			DisplayName:   user.Profile.DisplayName,
			Email:         user.Profile.Email,
			EmailVerified: user.Profile.EmailVerified,
			PhotoURL:      user.Profile.PhotoURL,
		},
		Student: StudentProfileResponse{
			OrganizationID: user.Student.OrganizationID,
			DistrictID:     user.Student.DistrictID,
			SchoolID:       user.Student.SchoolID,
			ClassID:        user.Student.ClassID,
			ClassCode:      user.Student.ClassCode,
			MissionID:      user.Student.MissionID,
			MissionCode:    user.Student.MissionCode,
			StudentName:    user.Student.StudentName,
		},
		Activity: UserActivityResponse{
			FirstSeenAt: formatTime(user.Activity.FirstSeenAt),
			LastSeenAt:  formatTime(user.Activity.LastSeenAt),
			LoginCount:  user.Activity.LoginCount,
			CurrentGame: StudentGameSessionResponse{
				AttemptID:         user.Activity.CurrentGame.AttemptID,
				MissionID:         user.Activity.CurrentGame.MissionID,
				PublishedRevision: user.Activity.CurrentGame.PublishedRevision,
				Subject:           user.Activity.CurrentGame.Subject,
				StageID:           user.Activity.CurrentGame.StageID,
				StageName:         user.Activity.CurrentGame.StageName,
				Status:            user.Activity.CurrentGame.Status,
				StartedAt:         formatTime(user.Activity.CurrentGame.StartedAt),
				LastHeartbeatAt:   formatTime(user.Activity.CurrentGame.LastHeartbeatAt),
			},
		},
		Roles:     user.Roles,
		IsGuest:   user.IsGuest,
		DeletedAt: optionalTime(user.DeletedAt),
		CreatedAt: formatTime(user.CreatedAt),
		UpdatedAt: formatTime(user.UpdatedAt),
	}
}

func ToStudentLearningResume(attempt StudentMissionAttemptDocument) StudentLearningResumeResponse {
	lastSequence := 0
	for _, event := range attempt.Snapshot.Events {
		if event.Sequence > lastSequence {
			lastSequence = event.Sequence
		}
	}

	return StudentLearningResumeResponse{
		ID:                attempt.ID,
		MissionID:         attempt.MissionID,
		PublishedRevision: attempt.PublishedRevision,
		Status:            attempt.Status,
		Snapshot: StudentResumeSnapshotResponse{
			Revision:    attempt.Snapshot.Revision,
			Progress:    attempt.Snapshot.Progress,
			Preferences: attempt.Snapshot.Preferences,
			Redirected:  attempt.Snapshot.Redirected,
			ReturnTo:    attempt.Snapshot.ReturnTo,
			LastPod:     attempt.Snapshot.LastPod,
			Checkpoint:  attempt.Snapshot.Checkpoint,
		},
		LastEventSequence: lastSequence,
		StartedAt:         formatTime(attempt.StartedAt),
		LastActivityAt:    formatTime(attempt.LastActivityAt),
	}
}

func ToStudentTutorTurnResponse(turn StudentTutorTurn) StudentTutorTurnResponse {
	return StudentTutorTurnResponse{
		TurnID:           turn.ID,
		Sequence:         turn.Sequence,
		Message:          turn.Tutor.Message,
		WhyPrompt:        turn.Tutor.WhyPrompt,
		ExplanationSteps: turn.Tutor.ExplanationSteps,
		ExplanationStyle: turn.Tutor.ExplanationStyle,
		DifficultyBias:   turn.Tutor.DifficultyBias,
		NextAction:       turn.Tutor.NextAction,
	}
}

func optionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}

	formatted := formatTime(*value)
	return &formatted
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.UTC().Format(time.RFC3339Nano)
}
