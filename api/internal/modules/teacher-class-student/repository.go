package teacherclassstudent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	teacherclass "classgame-api/internal/modules/teacher-class"
)

type FirestoreRepository struct {
	client *firestore.Client
}

func NewFirestoreRepository(client *firestore.Client) *FirestoreRepository {
	return &FirestoreRepository{
		client: client,
	}
}

func (repository *FirestoreRepository) Get(
	ctx context.Context,
	classID string,
	studentID string,
) (ClassStudentDocument, bool, error) {
	snapshot, err := repository.documentRef(classID, studentID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return ClassStudentDocument{}, false, nil
	}
	if err != nil {
		return ClassStudentDocument{}, false, fmt.Errorf("read class student: %w", err)
	}

	var student ClassStudentDocument
	if err := snapshot.DataTo(&student); err != nil {
		return ClassStudentDocument{}, false, fmt.Errorf("decode class student: %w", err)
	}
	if student.ID == "" {
		student.ID = snapshot.Ref.ID
	}

	return student, true, nil
}

func (repository *FirestoreRepository) UpsertMissionSummary(
	ctx context.Context,
	input UpsertStudentMissionSummaryInput,
) (ClassStudentDocument, error) {
	input = normalizeInput(input)
	ref := repository.documentRef(input.ClassID, input.StudentID)

	var student ClassStudentDocument
	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		createdAt := input.Timestamp
		missionSummaries := map[string]StudentMissionSummary{}
		notebookNotes := []StudentNotebookNote{}

		snapshot, err := tx.Get(ref)
		switch {
		case err == nil:
			if err := snapshot.DataTo(&student); err != nil {
				return fmt.Errorf("decode class student: %w", err)
			}
			if !student.CreatedAt.IsZero() {
				createdAt = student.CreatedAt
			}
			missionSummaries = student.MissionSummaries
			notebookNotes = student.NotebookNotes
		case status.Code(err) != codes.NotFound:
			return fmt.Errorf("read class student: %w", err)
		}

		if missionSummaries == nil {
			missionSummaries = map[string]StudentMissionSummary{}
		}
		if input.MissionSummary.MissionID != "" {
			missionSummaries[input.MissionSummary.MissionID] = input.MissionSummary
		}
		notebookNotes = append(notebookNotes, input.NotebookNotes...)

		student = ClassStudentDocument{
			ID:               ref.ID,
			ClassID:          input.ClassID,
			StudentID:        input.StudentID,
			StudentName:      input.StudentName,
			CurrentMissionID: input.CurrentMissionID,
			ThinkingSummary:  input.ThinkingSummary,
			MissionSummaries: missionSummaries,
			NotebookNotes:    notebookNotes,
			FeedbackSummary:  input.FeedbackSummary,
			CreatedAt:        createdAt,
			UpdatedAt:        input.Timestamp,
		}

		return tx.Set(ref, student.firestoreMap(), firestore.MergeAll)
	})
	if err != nil {
		return ClassStudentDocument{}, fmt.Errorf("upsert student mission summary: %w", err)
	}

	return student, nil
}

func (repository *FirestoreRepository) documentRef(classID string, studentID string) *firestore.DocumentRef {
	return repository.client.Collection(teacherclass.CollectionName).
		Doc(normalizeCode(classID)).
		Collection(CollectionName).
		Doc(strings.TrimSpace(studentID))
}

func (student ClassStudentDocument) firestoreMap() map[string]any {
	return map[string]any{
		"id":                 student.ID,
		"class_id":           student.ClassID,
		"student_id":         student.StudentID,
		"student_name":       student.StudentName,
		"current_mission_id": student.CurrentMissionID,
		"thinking_summary": map[string]any{
			"preferred_explanation_style": student.ThinkingSummary.PreferredExplanationStyle,
			"reasoning_patterns":          student.ThinkingSummary.ReasoningPatterns,
			"strengths":                   student.ThinkingSummary.Strengths,
			"improvement_areas":           student.ThinkingSummary.ImprovementAreas,
			"last_updated_at":             student.ThinkingSummary.LastUpdatedAt,
		},
		"mission_summaries": student.MissionSummaries,
		"notebook_notes":    student.NotebookNotes,
		"feedback_summary": map[string]any{
			"helpful_count":   student.FeedbackSummary.HelpfulCount,
			"confusing_count": student.FeedbackSummary.ConfusingCount,
			"too_easy_count":  student.FeedbackSummary.TooEasyCount,
			"too_hard_count":  student.FeedbackSummary.TooHardCount,
		},
		"created_at": student.CreatedAt,
		"updated_at": student.UpdatedAt,
	}
}

func normalizeInput(input UpsertStudentMissionSummaryInput) UpsertStudentMissionSummaryInput {
	if input.Timestamp.IsZero() {
		input.Timestamp = time.Now()
	}
	input.ClassID = normalizeCode(input.ClassID)
	input.StudentID = strings.TrimSpace(input.StudentID)
	input.StudentName = normalizeText(input.StudentName)
	input.CurrentMissionID = strings.TrimSpace(input.CurrentMissionID)
	input.MissionSummary.MissionID = strings.TrimSpace(input.MissionSummary.MissionID)
	input.MissionSummary.UpdatedAt = nonZeroTime(input.MissionSummary.UpdatedAt, input.Timestamp)
	input.ThinkingSummary.LastUpdatedAt = nonZeroTime(input.ThinkingSummary.LastUpdatedAt, input.Timestamp)

	return input
}

func nonZeroTime(value time.Time, fallback time.Time) time.Time {
	if !value.IsZero() {
		return value
	}
	if !fallback.IsZero() {
		return fallback
	}

	return time.Now()
}

func normalizeCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizeText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
