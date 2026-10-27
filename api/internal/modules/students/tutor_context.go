package students

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/missions"
)

func (service *Service) classTutorContext(
	ctx context.Context,
	classID string,
) (generation.TutorClassContext, error) {
	attempts, err := service.classLearning.ListLatestFinalizedAttemptsByClassID(ctx, classID, 40)
	if err != nil {
		return generation.TutorClassContext{}, err
	}
	strengths := make([]string, 0)
	needs := make([]string, 0)
	for _, attempt := range attempts {
		strengths = append(strengths, attempt.Trace.Strengths...)
		needs = append(needs, attempt.Trace.ImprovementAreas...)
	}

	return generation.TutorClassContext{
		CommonStrengths: mostFrequentSignals(strengths, 4),
		CommonNeeds:     mostFrequentSignals(needs, 4),
	}, nil
}

func retrieveMissionKnowledge(
	document missions.MissionDocument,
	turn SaveStudentTutorTurnInput,
	limit int,
) []generation.KnowledgeChunk {
	chunks := append([]generation.KnowledgeChunk{}, document.KnowledgeChunks...)
	if len(chunks) == 0 {
		for index, stage := range document.Stages {
			chunks = append(chunks, generation.KnowledgeChunk{
				ID:       "legacy-stage-" + normalizeRetrievalID(stage.ID, index),
				Source:   "Published mission",
				Title:    stage.Title,
				Content:  normalizeText(strings.Join(append([]string{stage.Goal}, stage.Checks...), " ")),
				Keywords: strings.Fields(stage.Title + " " + document.Subject),
			})
		}
		for index, question := range document.PreviewQuestions {
			chunks = append(chunks, generation.KnowledgeChunk{
				ID:       "legacy-question-" + normalizeRetrievalID(question.ID, index),
				Source:   "Published mission",
				Title:    question.Prompt,
				Content:  normalizeText(strings.Join(append(question.ExplanationSteps, question.ExpectedAnswer), " ")),
				Keywords: strings.Fields(question.Prompt + " " + question.ExpectedAnswer),
			})
		}
	}
	queryTokens := retrievalTokens(strings.Join([]string{
		document.Subject,
		document.LearningGoal,
		turn.StageID,
		turn.Prompt,
		turn.StudentResponse,
		turn.ExplicitReasoning,
	}, " "))
	type scoredChunk struct {
		chunk generation.KnowledgeChunk
		score int
		index int
	}
	scored := make([]scoredChunk, 0, len(chunks))
	for index, chunk := range chunks {
		score := 0
		chunkTokens := retrievalTokens(strings.Join(append([]string{chunk.Title, chunk.Content}, chunk.Keywords...), " "))
		for token := range queryTokens {
			if _, exists := chunkTokens[token]; exists {
				score++
			}
		}
		if turn.StageID != "" && strings.Contains(strings.ToLower(chunk.ID+" "+chunk.Title), strings.ToLower(turn.StageID)) {
			score += 4
		}
		scored = append(scored, scoredChunk{chunk: chunk, score: score, index: index})
	}
	sort.SliceStable(scored, func(left int, right int) bool {
		if scored[left].score == scored[right].score {
			return scored[left].index < scored[right].index
		}

		return scored[left].score > scored[right].score
	})
	if limit <= 0 {
		limit = 5
	}
	result := make([]generation.KnowledgeChunk, 0, min(limit, len(scored)))
	for _, entry := range scored {
		if entry.score == 0 || strings.TrimSpace(entry.chunk.Content) == "" {
			continue
		}
		result = append(result, entry.chunk)
		if len(result) >= limit {
			break
		}
	}

	return result
}

func retrievalTokens(value string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, token := range strings.Fields(strings.ToLower(value)) {
		token = strings.Trim(token, ".,:;!?()[]{}\"'")
		if len(token) < 3 {
			continue
		}
		result[token] = struct{}{}
	}

	return result
}

func normalizeRetrievalID(value string, index int) string {
	value = strings.ToLower(normalizeText(value))
	value = strings.NewReplacer(" ", "-", "_", "-").Replace(value)
	value = strings.Trim(value, "-")
	if value == "" {
		return fmt.Sprintf("%d", index+1)
	}

	return value
}

func toTutorProgress(progress []StudentStageProgress) []generation.LearningTraceStageProgress {
	result := make([]generation.LearningTraceStageProgress, 0, len(progress))
	for _, stage := range progress {
		result = append(result, generation.LearningTraceStageProgress{
			StageID:  stage.StageID,
			Status:   stage.Status,
			Correct:  stage.Correct,
			Required: stage.Required,
		})
	}

	return result
}

func toTutorTurnHistory(turns []StudentTutorTurn) []generation.TutorTurnHistory {
	if len(turns) > 12 {
		turns = turns[len(turns)-12:]
	}
	result := make([]generation.TutorTurnHistory, 0, len(turns))
	for _, turn := range turns {
		result = append(result, generation.TutorTurnHistory{
			Sequence: turn.Sequence,
			Type:     turn.Type,
			StageID:  turn.StageID,
			Prompt:   turn.Prompt,
			Response: turn.StudentResponse,
			Outcome:  turn.Outcome,
			Message:  turn.Tutor.Message,
		})
	}

	return result
}

func toTutorEvents(events []StudentLearningEvent) []generation.LearningTraceEvent {
	if len(events) > 24 {
		events = events[len(events)-24:]
	}
	result := make([]generation.LearningTraceEvent, 0, len(events))
	for _, event := range events {
		result = append(result, generation.LearningTraceEvent{
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

	return result
}

func mostFrequentSignals(values []string, limit int) []string {
	type count struct {
		label string
		value int
	}
	counts := make(map[string]*count)
	for _, value := range values {
		value = normalizeText(value)
		key := strings.ToLower(value)
		if value == "" {
			continue
		}
		if counts[key] == nil {
			counts[key] = &count{label: value}
		}
		counts[key].value++
	}
	ordered := make([]count, 0, len(counts))
	for _, item := range counts {
		ordered = append(ordered, *item)
	}
	sort.SliceStable(ordered, func(left int, right int) bool {
		if ordered[left].value == ordered[right].value {
			return ordered[left].label < ordered[right].label
		}

		return ordered[left].value > ordered[right].value
	})
	result := make([]string, 0, min(limit, len(ordered)))
	for _, item := range ordered {
		result = append(result, item.label)
		if len(result) >= limit {
			break
		}
	}

	return result
}
