package ai

type BuildLearningTraceInput struct {
	Mission        LearningTraceMission        `json:"mission"`
	Snapshot       LearningTraceSnapshot       `json:"snapshot"`
	PreviousMemory LearningTracePreviousMemory `json:"previous_memory"`
}

type LearningTraceMission struct {
	Title        string `json:"title"`
	Subject      string `json:"subject"`
	GradeLevel   string `json:"grade_level"`
	LearningGoal string `json:"learning_goal"`
}

type LearningTraceSnapshot struct {
	Status      string                       `json:"status"`
	Progress    []LearningTraceStageProgress `json:"progress"`
	Notes       []LearningTraceNote          `json:"notes"`
	Preferences LearningTracePreferences     `json:"preferences"`
	Decisions   []LearningTraceDecision      `json:"decisions"`
	Events      []LearningTraceEvent         `json:"events"`
	Redirected  bool                         `json:"redirected"`
	ReturnTo    string                       `json:"return_to"`
	LastPod     string                       `json:"last_pod"`
}

type LearningTraceStageProgress struct {
	StageID  string `json:"stage_id"`
	Status   string `json:"status"`
	Correct  int    `json:"correct"`
	Required int    `json:"required"`
}

type LearningTraceNote struct {
	Kind     string   `json:"kind"`
	Skill    string   `json:"skill"`
	Text     string   `json:"text"`
	Evidence []string `json:"evidence"`
	Count    int      `json:"count"`
}

type LearningTracePreferences struct {
	ExplanationStyle string `json:"explanation_style"`
	DifficultyBias   int    `json:"difficulty_bias"`
}

type LearningTraceDecision struct {
	Action  string `json:"action"`
	Message string `json:"message"`
	Why     string `json:"why"`
}

type LearningTraceEvent struct {
	ID         string `json:"id"`
	Sequence   int    `json:"sequence"`
	Type       string `json:"type"`
	StageID    string `json:"stage_id"`
	QuestionID string `json:"question_id"`
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	Outcome    string `json:"outcome"`
	OccurredAt int64  `json:"occurred_at"`
}

type LearningTracePreviousMemory struct {
	Strengths            []string `json:"strengths"`
	ImprovementAreas     []string `json:"improvement_areas"`
	PreferredExplanation string   `json:"preferred_explanation"`
	NextAdaptation       string   `json:"next_adaptation"`
}

type LearningTraceBlueprint struct {
	Summary          string      `firestore:"summary" json:"summary"`
	Strengths        []string    `firestore:"strengths" json:"strengths"`
	ImprovementAreas []string    `firestore:"improvement_areas" json:"improvement_areas"`
	NextAdaptation   string      `firestore:"next_adaptation" json:"next_adaptation"`
	Flow             MissionFlow `firestore:"flow" json:"flow"`
	Finalized        bool        `firestore:"finalized" json:"finalized"`
	EventCount       int         `firestore:"event_count" json:"event_count"`
	GeneratedByAI    bool        `firestore:"generated_by_ai" json:"generated_by_ai"`
	GeneratedByMock  bool        `firestore:"generated_by_mock" json:"generated_by_mock"`
	Provider         string      `firestore:"provider" json:"provider"`
	Model            string      `firestore:"model" json:"model"`
	ModelVersion     string      `firestore:"model_version" json:"model_version"`
	FallbackReason   string      `firestore:"fallback_reason" json:"fallback_reason"`
	Usage            GenAIUsage  `firestore:"usage" json:"usage"`
}

// BuildTutorTurnInput is the complete, server-owned context for one live
// tutoring turn. Student-entered strings remain untrusted evidence.
