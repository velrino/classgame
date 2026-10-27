package teacherclassstudent

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrInvalidStudentMissionSummary = errors.New("invalid student mission summary")

type Service struct {
	repository NotebookStore
}

func NewService(repository NotebookStore) *Service {
	return &Service{
		repository: repository,
	}
}

func (service *Service) UpsertMissionSummary(
	ctx context.Context,
	input UpsertStudentMissionSummaryInput,
) (ClassStudentDocument, error) {
	if service.repository == nil {
		return ClassStudentDocument{}, errors.New("teacher class student repository is not configured")
	}

	input = normalizeInput(input)
	if input.ClassID == "" || input.StudentID == "" || input.MissionSummary.MissionID == "" {
		return ClassStudentDocument{}, ErrInvalidStudentMissionSummary
	}
	if input.CurrentMissionID == "" {
		input.CurrentMissionID = input.MissionSummary.MissionID
	}
	if input.StudentName == "" {
		input.StudentName = "Unnamed student"
	}
	if input.Timestamp.IsZero() {
		input.Timestamp = time.Now()
	}

	input.ThinkingSummary = normalizeThinkingSummary(input.ThinkingSummary, input.Timestamp)
	input.NotebookNotes = normalizeNotebookNotes(input.NotebookNotes, input.MissionSummary.MissionID, input.Timestamp)

	return service.repository.UpsertMissionSummary(ctx, input)
}

func normalizeThinkingSummary(
	summary StudentThinkingSummary,
	timestamp time.Time,
) StudentThinkingSummary {
	summary.PreferredExplanationStyle = normalizeText(summary.PreferredExplanationStyle)
	summary.ReasoningPatterns = normalizeStrings(summary.ReasoningPatterns)
	summary.Strengths = normalizeStrings(summary.Strengths)
	summary.ImprovementAreas = normalizeStrings(summary.ImprovementAreas)
	summary.LastUpdatedAt = nonZeroTime(summary.LastUpdatedAt, timestamp)

	return summary
}

func normalizeNotebookNotes(
	notes []StudentNotebookNote,
	defaultMissionID string,
	timestamp time.Time,
) []StudentNotebookNote {
	normalized := make([]StudentNotebookNote, 0, len(notes))
	for _, note := range notes {
		note.ID = strings.TrimSpace(note.ID)
		note.MissionID = strings.TrimSpace(note.MissionID)
		note.Skill = normalizeText(note.Skill)
		note.Kind = normalizeText(note.Kind)
		note.Text = normalizeText(note.Text)
		note.Evidence = normalizeStrings(note.Evidence)
		if note.MissionID == "" {
			note.MissionID = defaultMissionID
		}
		if note.Text == "" {
			continue
		}
		if note.ID == "" {
			note.ID = note.MissionID + ":" + note.Kind + ":" + note.Skill
		}
		if note.Count <= 0 {
			note.Count = 1
		}
		note.CreatedAt = nonZeroTime(note.CreatedAt, timestamp)
		note.UpdatedAt = nonZeroTime(note.UpdatedAt, timestamp)

		normalized = append(normalized, note)
	}

	return normalized
}

func normalizeStrings(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		text := normalizeText(value)
		if text != "" {
			normalized = append(normalized, text)
		}
	}

	return normalized
}
