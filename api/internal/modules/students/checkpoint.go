package students

import (
	"strings"

	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/missions"
)

func toLearningTraceInput(
	missionDocument missions.MissionDocument,
	snapshot StudentLearningSnapshot,
	status string,
	memory StudentLearningMemory,
) generation.BuildLearningTraceInput {
	progress := make([]generation.LearningTraceStageProgress, 0, len(snapshot.Progress))
	for _, stage := range snapshot.Progress {
		progress = append(progress, generation.LearningTraceStageProgress{
			StageID:  stage.StageID,
			Status:   stage.Status,
			Correct:  stage.Correct,
			Required: stage.Required,
		})
	}
	notes := make([]generation.LearningTraceNote, 0, len(snapshot.Notes))
	for _, note := range snapshot.Notes {
		notes = append(notes, generation.LearningTraceNote{
			Kind:     note.Kind,
			Skill:    note.Skill,
			Text:     note.Text,
			Evidence: note.Evidence,
			Count:    note.Count,
		})
	}
	decisions := make([]generation.LearningTraceDecision, 0, len(snapshot.Decisions))
	for _, decision := range snapshot.Decisions {
		decisions = append(decisions, generation.LearningTraceDecision{
			Action:  decision.Action,
			Message: decision.Message,
			Why:     decision.Why,
		})
	}
	events := make([]generation.LearningTraceEvent, 0, len(snapshot.Events))
	for _, event := range snapshot.Events {
		events = append(events, generation.LearningTraceEvent{
			ID:         event.ID,
			Sequence:   event.Sequence,
			Type:       event.Type,
			StageID:    event.StageID,
			QuestionID: event.QuestionID,
			Title:      event.Title,
			Detail:     event.Detail,
			Outcome:    event.Outcome,
			OccurredAt: event.OccurredAt,
		})
	}

	return generation.BuildLearningTraceInput{
		Mission: generation.LearningTraceMission{
			Title:        missionDocument.Title,
			Subject:      missionDocument.Subject,
			GradeLevel:   missionDocument.GradeLevel,
			LearningGoal: missionDocument.LearningGoal,
		},
		Snapshot: generation.LearningTraceSnapshot{
			Status:     strings.ToLower(normalizeText(status)),
			Progress:   progress,
			Notes:      notes,
			Redirected: snapshot.Redirected,
			ReturnTo:   snapshot.ReturnTo,
			LastPod:    snapshot.LastPod,
			Preferences: generation.LearningTracePreferences{
				ExplanationStyle: snapshot.Preferences.ExplanationStyle,
				DifficultyBias:   snapshot.Preferences.DifficultyBias,
			},
			Decisions: decisions,
			Events:    events,
		},
		PreviousMemory: generation.LearningTracePreviousMemory{
			Strengths:            memory.Strengths,
			ImprovementAreas:     memory.ImprovementAreas,
			PreferredExplanation: memory.PreferredExplanation,
			NextAdaptation:       memory.NextAdaptation,
		},
	}
}

func normalizeLearningSnapshot(snapshot StudentLearningSnapshot) StudentLearningSnapshot {
	if snapshot.Revision < 0 {
		snapshot.Revision = 0
	}
	snapshot.ReturnTo = normalizeText(snapshot.ReturnTo)
	snapshot.LastPod = normalizeText(snapshot.LastPod)
	for index := range snapshot.Progress {
		progress := &snapshot.Progress[index]
		progress.StageID = normalizeText(progress.StageID)
		progress.Status = strings.ToLower(normalizeText(progress.Status))
	}
	for index := range snapshot.Notes {
		note := &snapshot.Notes[index]
		note.ID = normalizeText(note.ID)
		note.MisconceptionID = normalizeText(note.MisconceptionID)
		note.Text = normalizeText(note.Text)
		note.Kind = strings.ToLower(normalizeText(note.Kind))
		note.Skill = normalizeText(note.Skill)
	}
	for index := range snapshot.Decisions {
		decision := &snapshot.Decisions[index]
		decision.ID = normalizeText(decision.ID)
		decision.Action = strings.ToLower(normalizeText(decision.Action))
		decision.Message = normalizeText(decision.Message)
		decision.Why = normalizeText(decision.Why)
	}
	for index := range snapshot.Events {
		event := &snapshot.Events[index]
		event.ID = normalizeText(event.ID)
		event.Type = strings.ToLower(normalizeText(event.Type))
		event.StageID = normalizeText(event.StageID)
		event.QuestionID = normalizeText(event.QuestionID)
		event.Title = normalizeText(event.Title)
		event.Detail = normalizeText(event.Detail)
		event.Outcome = strings.ToLower(normalizeText(event.Outcome))
	}
	snapshot.Preferences.ExplanationStyle = strings.ToLower(normalizeText(snapshot.Preferences.ExplanationStyle))
	snapshot.Checkpoint.StageID = normalizeText(snapshot.Checkpoint.StageID)
	snapshot.Checkpoint.QuestionID = normalizeText(snapshot.Checkpoint.QuestionID)
	snapshot.Checkpoint.Phase = strings.ToLower(normalizeText(snapshot.Checkpoint.Phase))
	snapshot.Checkpoint.WrongChoice = normalizeText(snapshot.Checkpoint.WrongChoice)
	snapshot.Checkpoint.WrongMisconceptionID = normalizeText(snapshot.Checkpoint.WrongMisconceptionID)
	snapshot.Checkpoint.DecisionAction = strings.ToLower(normalizeText(snapshot.Checkpoint.DecisionAction))
	snapshot.Checkpoint.DecisionTarget = normalizeText(snapshot.Checkpoint.DecisionTarget)
	snapshot.Checkpoint.DecisionMessage = truncateStudentText(snapshot.Checkpoint.DecisionMessage, 500)
	snapshot.Checkpoint.DecisionWhy = truncateStudentText(snapshot.Checkpoint.DecisionWhy, 500)
	snapshot.Checkpoint.TutorMessage = truncateStudentText(snapshot.Checkpoint.TutorMessage, 500)
	snapshot.Checkpoint.WhyPrompt = truncateStudentText(snapshot.Checkpoint.WhyPrompt, 500)
	if len(snapshot.Checkpoint.Lines) > 12 {
		snapshot.Checkpoint.Lines = snapshot.Checkpoint.Lines[:12]
	}
	for index := range snapshot.Checkpoint.Lines {
		snapshot.Checkpoint.Lines[index] = normalizeText(snapshot.Checkpoint.Lines[index])
	}

	return snapshot
}
