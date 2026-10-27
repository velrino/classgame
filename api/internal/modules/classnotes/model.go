package classnotes

import "time"

const CollectionName = "students"

type ClassStudentDocument struct {
	ID               string                           `firestore:"id" json:"id"`
	ClassID          string                           `firestore:"class_id" json:"class_id"`
	StudentID        string                           `firestore:"student_id" json:"student_id"`
	StudentName      string                           `firestore:"student_name" json:"student_name"`
	CurrentMissionID string                           `firestore:"current_mission_id" json:"current_mission_id"`
	ThinkingSummary  StudentThinkingSummary           `firestore:"thinking_summary" json:"thinking_summary"`
	MissionSummaries map[string]StudentMissionSummary `firestore:"mission_summaries" json:"mission_summaries"`
	NotebookNotes    []StudentNotebookNote            `firestore:"notebook_notes" json:"notebook_notes"`
	FeedbackSummary  StudentFeedbackSummary           `firestore:"feedback_summary" json:"feedback_summary"`
	CreatedAt        time.Time                        `firestore:"created_at" json:"created_at"`
	UpdatedAt        time.Time                        `firestore:"updated_at" json:"updated_at"`
}

type StudentThinkingSummary struct {
	PreferredExplanationStyle string    `firestore:"preferred_explanation_style" json:"preferred_explanation_style"`
	ReasoningPatterns         []string  `firestore:"reasoning_patterns" json:"reasoning_patterns"`
	Strengths                 []string  `firestore:"strengths" json:"strengths"`
	ImprovementAreas          []string  `firestore:"improvement_areas" json:"improvement_areas"`
	LastUpdatedAt             time.Time `firestore:"last_updated_at" json:"last_updated_at"`
}

type StudentMissionSummary struct {
	MissionID          string    `firestore:"mission_id" json:"mission_id"`
	Status             string    `firestore:"status" json:"status"`
	CompletedQuestions int       `firestore:"completed_questions" json:"completed_questions"`
	TotalQuestions     int       `firestore:"total_questions" json:"total_questions"`
	CorrectAnswers     int       `firestore:"correct_answers" json:"correct_answers"`
	LastDecision       string    `firestore:"last_decision" json:"last_decision"`
	LastFeedback       string    `firestore:"last_feedback" json:"last_feedback"`
	UpdatedAt          time.Time `firestore:"updated_at" json:"updated_at"`
}

type StudentNotebookNote struct {
	ID        string    `firestore:"id" json:"id"`
	MissionID string    `firestore:"mission_id" json:"mission_id"`
	Skill     string    `firestore:"skill" json:"skill"`
	Kind      string    `firestore:"kind" json:"kind"`
	Text      string    `firestore:"text" json:"text"`
	Evidence  []string  `firestore:"evidence" json:"evidence"`
	Count     int       `firestore:"count" json:"count"`
	CreatedAt time.Time `firestore:"created_at" json:"created_at"`
	UpdatedAt time.Time `firestore:"updated_at" json:"updated_at"`
}

type StudentFeedbackSummary struct {
	HelpfulCount   int `firestore:"helpful_count" json:"helpful_count"`
	ConfusingCount int `firestore:"confusing_count" json:"confusing_count"`
	TooEasyCount   int `firestore:"too_easy_count" json:"too_easy_count"`
	TooHardCount   int `firestore:"too_hard_count" json:"too_hard_count"`
}

type UpsertStudentMissionSummaryInput struct {
	ClassID          string
	StudentID        string
	StudentName      string
	CurrentMissionID string
	ThinkingSummary  StudentThinkingSummary
	MissionSummary   StudentMissionSummary
	NotebookNotes    []StudentNotebookNote
	FeedbackSummary  StudentFeedbackSummary
	Timestamp        time.Time
}
