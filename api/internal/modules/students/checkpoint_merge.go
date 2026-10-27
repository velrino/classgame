package students

import (
	"sort"
)

func mergeLearningSnapshots(
	existing StudentLearningSnapshot,
	incoming StudentLearningSnapshot,
) StudentLearningSnapshot {
	incoming.Notes = mergeLearningNotes(existing.Notes, incoming.Notes)
	incoming.Decisions = mergeStudentDecisions(existing.Decisions, incoming.Decisions)
	incoming.Events = mergeLearningEvents(existing.Events, incoming.Events)

	return incoming
}

func mergeLearningNotes(existing []StudentLearningNote, incoming []StudentLearningNote) []StudentLearningNote {
	byID := make(map[string]StudentLearningNote, len(existing)+len(incoming))
	order := make([]string, 0, len(existing)+len(incoming))
	for _, note := range append(append([]StudentLearningNote{}, existing...), incoming...) {
		if note.ID == "" {
			continue
		}
		if _, exists := byID[note.ID]; !exists {
			order = append(order, note.ID)
		}
		byID[note.ID] = note
	}
	result := make([]StudentLearningNote, 0, min(len(order), 40))
	start := max(0, len(order)-40)
	for _, id := range order[start:] {
		result = append(result, byID[id])
	}

	return result
}

func mergeStudentDecisions(existing []StudentDecision, incoming []StudentDecision) []StudentDecision {
	byID := make(map[string]StudentDecision, len(existing)+len(incoming))
	for _, decision := range append(append([]StudentDecision{}, existing...), incoming...) {
		if decision.ID != "" {
			byID[decision.ID] = decision
		}
	}
	result := make([]StudentDecision, 0, len(byID))
	for _, decision := range byID {
		result = append(result, decision)
	}
	sort.SliceStable(result, func(left int, right int) bool {
		return result[left].At > result[right].At
	})
	if len(result) > 40 {
		result = result[:40]
	}

	return result
}

func mergeLearningEvents(existing []StudentLearningEvent, incoming []StudentLearningEvent) []StudentLearningEvent {
	byID := make(map[string]StudentLearningEvent, len(existing)+len(incoming))
	for _, event := range append(append([]StudentLearningEvent{}, existing...), incoming...) {
		if event.ID != "" {
			byID[event.ID] = event
		}
	}
	result := make([]StudentLearningEvent, 0, len(byID))
	for _, event := range byID {
		result = append(result, event)
	}
	sort.SliceStable(result, func(left int, right int) bool {
		if result[left].Sequence == result[right].Sequence {
			return result[left].OccurredAt < result[right].OccurredAt
		}

		return result[left].Sequence < result[right].Sequence
	})
	if len(result) > 160 {
		result = result[len(result)-160:]
	}

	return result
}
