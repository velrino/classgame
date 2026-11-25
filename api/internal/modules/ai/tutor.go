package ai

type BuildTutorTurnInput struct {
	Mission         TutorMissionContext         `json:"mission"`
	StudentMemory   LearningTracePreviousMemory `json:"student_memory"`
	ClassContext    TutorClassContext           `json:"class_context"`
	Attempt         TutorAttemptContext         `json:"attempt"`
	Turn            TutorTurnEvidence           `json:"turn"`
	RetrievedChunks []KnowledgeChunk            `json:"retrieved_chunks"`
}

type TutorMissionContext struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Subject      string         `json:"subject"`
	GradeLevel   string         `json:"grade_level"`
	LearningGoal string         `json:"learning_goal"`
	Stages       []MissionStage `json:"stages"`
}

type TutorClassContext struct {
	CommonStrengths []string `json:"common_strengths"`
	CommonNeeds     []string `json:"common_needs"`
}

type TutorAttemptContext struct {
	Status       string                       `json:"status"`
	Progress     []LearningTraceStageProgress `json:"progress"`
	RecentTurns  []TutorTurnHistory           `json:"recent_turns"`
	RecentEvents []LearningTraceEvent         `json:"recent_events"`
}

type TutorTurnHistory struct {
	Sequence int    `json:"sequence"`
	Type     string `json:"type"`
	StageID  string `json:"stage_id"`
	Prompt   string `json:"prompt"`
	Response string `json:"response"`
	Outcome  string `json:"outcome"`
	Message  string `json:"message"`
}

type TutorTurnEvidence struct {
	EventID           string `json:"event_id"`
	Type              string `json:"type"`
	StageID           string `json:"stage_id"`
	QuestionID        string `json:"question_id"`
	Prompt            string `json:"prompt"`
	StudentResponse   string `json:"student_response"`
	ExpectedAnswer    string `json:"expected_answer"`
	Outcome           string `json:"outcome"`
	ExplicitReasoning string `json:"explicit_reasoning"`
	Feedback          string `json:"feedback"`
}

type TutorTurnBlueprint struct {
	Message            string     `firestore:"message" json:"message"`
	WhyPrompt          string     `firestore:"why_prompt" json:"why_prompt"`
	ExplanationSteps   []string   `firestore:"explanation_steps" json:"explanation_steps"`
	ExplanationStyle   string     `firestore:"explanation_style" json:"explanation_style"`
	DifficultyBias     int        `firestore:"difficulty_bias" json:"difficulty_bias"`
	NextAction         string     `firestore:"next_action" json:"next_action"`
	TeacherObservation string     `firestore:"teacher_observation" json:"teacher_observation"`
	RetrievedChunkIDs  []string   `firestore:"retrieved_chunk_ids" json:"retrieved_chunk_ids"`
	EvidenceEventIDs   []string   `firestore:"evidence_event_ids" json:"evidence_event_ids"`
	GeneratedByAI      bool       `firestore:"generated_by_ai" json:"generated_by_ai"`
	Provider           string     `firestore:"provider" json:"provider"`
	Model              string     `firestore:"model" json:"model"`
	ModelVersion       string     `firestore:"model_version" json:"model_version"`
	FallbackReason     string     `firestore:"fallback_reason" json:"fallback_reason"`
	Usage              GenAIUsage `firestore:"usage" json:"usage"`
}
