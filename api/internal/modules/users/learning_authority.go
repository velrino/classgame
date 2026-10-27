package users

import (
	"classgame-api/internal/modules/mission"
)

// authoritativeLearningSnapshot derives assessment from server-recorded tutor
// turns on the pinned revision. Browser checkpoints remain presentation state.
func authoritativeLearningSnapshot(document mission.MissionDocument, snapshot StudentLearningSnapshot, turns []StudentTutorTurn, requestedStatus string) (StudentLearningSnapshot, string) {
	questions := studentSessionMissionFromDocument(document).PreviewQuestions
	byID := map[string]int{}
	stageOrder := []string{}
	byStage := map[string][]string{}
	for i, q := range questions {
		if _, exists := byID[q.ID]; exists || q.ID == "" {
			continue
		}
		byID[q.ID] = i
		if _, exists := byStage[q.StageID]; !exists {
			stageOrder = append(stageOrder, q.StageID)
		}
		byStage[q.StageID] = append(byStage[q.StageID], q.ID)
	}
	solved := map[string]bool{}
	events := make([]StudentLearningEvent, 0, len(turns))
	notes := []StudentLearningNote{}
	decisions := []StudentDecision{}
	for _, turn := range turns {
		index, known := byID[turn.QuestionID]
		if !known {
			continue
		}
		q := questions[index]
		outcome := "support"
		if turn.Type == "answer" {
			outcome = "incorrect"
			if normalizedStudentAnswer(turn.StudentResponse) == normalizedStudentAnswer(q.ExpectedAnswer) {
				outcome = "correct"
				solved[q.ID] = true
			}
		} else if turn.Type != "reasoning" && turn.Type != "feedback" {
			continue
		}
		detail := turn.StudentResponse
		if turn.Type == "reasoning" {
			detail = turn.ExplicitReasoning
		}
		if turn.Type == "feedback" {
			detail = turn.Feedback
		}
		if turn.Tutor.Message != "" {
			notes = append(notes, StudentLearningNote{ID: turn.ID, Text: turn.Tutor.Message, Kind: "support", Skill: q.StageID, Evidence: []string{turn.ID}, Count: 1, CreatedAt: turn.CreatedAt.UnixMilli()})
		}
		if turn.Tutor.NextAction != "" {
			decisions = append(decisions, StudentDecision{ID: turn.ID, At: turn.CreatedAt.UnixMilli(), Action: turn.Tutor.NextAction, Message: turn.Tutor.Message})
		}
		events = append(events, StudentLearningEvent{ID: turn.ID, Sequence: turn.Sequence, Type: turn.Type, StageID: q.StageID, QuestionID: q.ID, Title: q.Prompt, Detail: detail, Outcome: outcome, OccurredAt: turn.CreatedAt.UnixMilli()})
	}
	progress := make([]StudentStageProgress, 0, len(stageOrder))
	complete := len(stageOrder) > 0
	for _, stageID := range stageOrder {
		ids := byStage[stageID]
		correct := 0
		remaining := []string{}
		for _, id := range ids {
			if solved[id] {
				correct++
			} else {
				remaining = append(remaining, id)
			}
		}
		required := min(2, len(ids))
		status := "ready"
		if correct >= required {
			status = "complete"
		} else {
			complete = false
		}
		progress = append(progress, StudentStageProgress{StageID: stageID, Status: status, Queue: remaining, Pool: append([]string{}, ids...), Correct: correct, Required: required})
	}
	snapshot.Progress = progress
	snapshot.Events = events
	snapshot.Notes = notes[max(0, len(notes)-40):]
	snapshot.Decisions = decisions[max(0, len(decisions)-40):]
	// Notes and decisions reference persisted student-visible tutor messages only.
	if requestedStatus == "completed" && !complete {
		requestedStatus = "learning"
	}
	if !complete && snapshot.Checkpoint.DecisionAction == "mission-complete" {
		snapshot.Checkpoint.DecisionAction = "next"
		snapshot.Checkpoint.DecisionMessage = ""
		snapshot.Checkpoint.Clean = false
	}
	if complete {
		for i := range progress {
			progress[i].Queue = []string{}
		}
	}
	return snapshot, requestedStatus
}
