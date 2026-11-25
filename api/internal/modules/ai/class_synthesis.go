package ai

type BuildClassLearningSynthesisInput struct {
	ClassID  string                         `json:"class_id"`
	Evidence []ClassLearningStudentEvidence `json:"evidence"`
}

type ClassLearningStudentEvidence struct {
	StudentID        string   `json:"student_id"`
	StudentName      string   `json:"student_name"`
	Strengths        []string `json:"strengths"`
	ImprovementAreas []string `json:"improvement_areas"`
}

type ClassLearningSynthesis struct {
	CommonStrengths []ClassLearningCluster `firestore:"common_strengths" json:"common_strengths"`
	CommonNeeds     []ClassLearningCluster `firestore:"common_needs" json:"common_needs"`
	NextAdaptation  string                 `firestore:"next_adaptation" json:"next_adaptation"`
	GeneratedByAI   bool                   `firestore:"generated_by_ai" json:"generated_by_ai"`
	Provider        string                 `firestore:"provider" json:"provider"`
	Model           string                 `firestore:"model" json:"model"`
	ModelVersion    string                 `firestore:"model_version" json:"model_version"`
	FallbackReason  string                 `firestore:"fallback_reason" json:"fallback_reason"`
	Usage           GenAIUsage             `firestore:"usage" json:"usage"`
}

type ClassLearningCluster struct {
	Label          string   `firestore:"label" json:"label"`
	StudentIDs     []string `firestore:"student_ids" json:"student_ids"`
	EvidenceLabels []string `firestore:"evidence_labels" json:"evidence_labels"`
}
