package insights

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"golang.org/x/sync/errgroup"

	"classgame-api/internal/auth"
	"classgame-api/internal/modules/classrooms"
	"classgame-api/internal/modules/students"
	"classgame-api/internal/modules/tenancy"
	"classgame-api/internal/pagination"
)

var ErrPaginationRequired = errors.New("use the paged dashboard endpoint for this class")
var ErrInvalidPage = errors.New("invalid dashboard page or cursor")

type OverviewPageMetadata struct {
	Limit        int    `json:"limit"`
	Returned     int    `json:"returned"`
	NextCursor   string `json:"next_cursor"`
	ReadAt       string `json:"read_at"`
	MetricsScope string `json:"metrics_scope"`
	HistoryScope string `json:"history_scope"`
}
type StudentPage struct {
	Students      []studentUserDocument
	NextCursor    string
	ReadDocuments int
}
type AttemptPage struct {
	Attempts      []students.StudentMissionAttemptDocument
	NextCursor    string
	ReadDocuments int
}
type StudentHistoryResponse struct {
	Attempts   []StudentLearningAttemptResponse `json:"attempts"`
	NextCursor string                           `json:"next_cursor"`
	ReadAt     string                           `json:"read_at"`
}
type pageCursor struct {
	Kind       string    `json:"kind"`
	ClassID    string    `json:"class"`
	StudentID  string    `json:"student,omitempty"`
	ID         string    `json:"id"`
	ActivityAt time.Time `json:"at,omitempty"`
}

func decodePage(page pagination.Params, classID, studentID, kind string) (pageCursor, error) {
	if page.Limit < 1 || page.Limit > 100 {
		return pageCursor{}, ErrInvalidPage
	}
	if page.After == "" {
		return pageCursor{}, nil
	}
	var cursor pageCursor
	if json.Unmarshal([]byte(page.After), &cursor) != nil || cursor.Kind != kind || cursor.ClassID != classID || cursor.StudentID != studentID || cursor.ID == "" || len(cursor.ID) > 150 || strings.ContainsAny(cursor.ID, "/\x00") {
		return cursor, ErrInvalidPage
	}
	return cursor, nil
}
func encodePage(cursor pageCursor) string {
	raw, _ := json.Marshal(cursor)
	return pagination.Cursor(string(raw))
}
func (repository *FirestoreRepository) ListStudentPage(ctx context.Context, classID string, page pagination.Params) (StudentPage, error) {
	result := StudentPage{Students: []studentUserDocument{}}
	cursor, err := decodePage(page, classID, "", "students")
	if err != nil {
		return result, err
	}
	query := repository.client.Collection(usersCollectionName).Where("student.class_id", "==", classID).OrderBy(firestore.DocumentID, firestore.Asc).Limit(page.Limit+1).
		Select("id", "profile.display_name", "student.organization_id", "student.district_id", "student.school_id", "student.class_id", "student.student_name", "activity", "roles")
	if cursor.ID != "" {
		query = query.StartAfter(cursor.ID)
	}
	rows, err := query.Documents(ctx).GetAll()
	if err != nil {
		return result, fmt.Errorf("list dashboard student page: %w", err)
	}
	result.ReadDocuments = len(rows)
	if len(rows) > page.Limit {
		rows = rows[:page.Limit]
		result.NextCursor = encodePage(pageCursor{Kind: "students", ClassID: classID, ID: rows[len(rows)-1].Ref.ID})
	}
	for _, row := range rows {
		var student studentUserDocument
		if err := row.DataTo(&student); err != nil {
			return result, err
		}
		if !slices.Contains(student.Roles, "student") {
			continue
		}
		student.ID = row.Ref.ID
		result.Students = append(result.Students, student)
	}
	return result, nil
}

var attemptSummaryFields = []string{"id", "organization_id", "district_id", "school_id", "student_id", "student_name", "class_id", "mission_id", "mission_code", "published_revision", "subject", "status", "snapshot.progress", "snapshot.notes", "trace.strengths", "trace.improvement_areas", "trace.finalized", "trace_state", "started_at", "last_activity_at", "completed_at", "abandoned_at", "created_at", "updated_at"}

func (repository *FirestoreRepository) ListAttemptPage(ctx context.Context, classID, studentID string, page pagination.Params, summary bool) (AttemptPage, error) {
	result := AttemptPage{Attempts: []students.StudentMissionAttemptDocument{}}
	cursor, err := decodePage(page, classID, studentID, "attempts")
	if err != nil {
		return result, err
	}
	query := repository.client.Collection("student_mission_attempts").Where("class_id", "==", classID)
	if studentID == "" {
		// The bounded legacy route returns every row only when the cap fits.
		// Sorting happens in the existing presenter, so it needs no new time index.
		query = query.OrderBy(firestore.DocumentID, firestore.Asc)
		if cursor.ID != "" {
			query = query.StartAfter(cursor.ID)
		}
	} else {
		query = query.Where("student_id", "==", studentID).OrderBy("last_activity_at", firestore.Desc).OrderBy(firestore.DocumentID, firestore.Asc)
		if cursor.ID != "" {
			if cursor.ActivityAt.IsZero() {
				return result, ErrInvalidPage
			}
			query = query.StartAfter(cursor.ActivityAt, cursor.ID)
		}
	}
	query = query.Limit(page.Limit + 1)
	if summary {
		query = query.Limit(page.Limit).Select(attemptSummaryFields...)
	}
	rows, err := query.Documents(ctx).GetAll()
	if err != nil {
		return result, fmt.Errorf("list student history page: %w", err)
	}
	result.ReadDocuments = len(rows)
	more := len(rows) > page.Limit
	if more {
		rows = rows[:page.Limit]
	}
	for _, row := range rows {
		var attempt students.StudentMissionAttemptDocument
		if err := row.DataTo(&attempt); err != nil {
			return result, err
		}
		attempt.ID = row.Ref.ID
		result.Attempts = append(result.Attempts, attempt)
	}
	if more {
		last := result.Attempts[len(result.Attempts)-1]
		result.NextCursor = encodePage(pageCursor{Kind: "attempts", ClassID: classID, StudentID: studentID, ID: last.ID, ActivityAt: last.LastActivityAt})
	}
	return result, nil
}
func (repository *FirestoreRepository) ListLatestAttempts(ctx context.Context, classID string, studentIDs []string) ([]students.StudentMissionAttemptDocument, error) {
	if len(studentIDs) > 100 {
		return nil, ErrInvalidPage
	}
	result := []students.StudentMissionAttemptDocument{}
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(8)
	var mutex sync.Mutex
	for _, studentID := range studentIDs {
		group.Go(func() error {
			page, err := repository.ListAttemptPage(ctx, classID, studentID, pagination.Params{Limit: 1}, true)
			if err != nil {
				return err
			}
			mutex.Lock()
			result = append(result, page.Attempts...)
			mutex.Unlock()
			return nil
		})
	}
	err := group.Wait()
	return result, err
}
func (service *Service) GetStudentHistory(ctx context.Context, identity auth.Identity, classID, studentID string, page pagination.Params) (StudentHistoryResponse, error) {
	if !auth.IsTeacherAuthMethod(identity.AuthMethod) {
		return StudentHistoryResponse{}, ErrTeacherAuthRequired
	}
	if service.teacherClassReader == nil {
		return StudentHistoryResponse{}, errors.New("class access reader is required")
	}
	classID = normalizeClassID(classID)
	class, exists, err := service.teacherClassReader.Get(ctx, identity, classID)
	if err != nil {
		return StudentHistoryResponse{}, err
	}
	if !exists {
		return StudentHistoryResponse{}, classrooms.ErrClassNotFound
	}
	if page.Limit < 1 || page.Limit > 10 || studentID == "" || len(studentID) > 128 || strings.ContainsAny(studentID, "/\x00") {
		return StudentHistoryResponse{}, ErrInvalidPage
	}
	result, err := service.repository.ListAttemptPage(ctx, classID, studentID, page, false)
	if err != nil {
		return StudentHistoryResponse{}, err
	}
	_, safeAttempts := filterTenantEvidence(nil, result.Attempts, class)
	return StudentHistoryResponse{Attempts: toStudentLearningAttemptResponses(safeAttempts), NextCursor: result.NextCursor, ReadAt: time.Now().UTC().Format(time.RFC3339Nano)}, nil
}

func filterTenantEvidence(users []studentUserDocument, attempts []students.StudentMissionAttemptDocument, class classrooms.TeacherClassDocument) ([]studentUserDocument, []students.StudentMissionAttemptDocument) {
	scope := tenancy.NormalizeScope(tenancy.Scope{OrganizationID: class.OrganizationID, DistrictID: class.DistrictID, SchoolID: class.SchoolID})
	safeUsers := make([]studentUserDocument, 0, len(users))
	for _, user := range users {
		if tenancy.NormalizeScope(tenancy.Scope{OrganizationID: user.Student.OrganizationID, DistrictID: user.Student.DistrictID, SchoolID: user.Student.SchoolID}) == scope {
			safeUsers = append(safeUsers, user)
		}
	}
	safeAttempts := make([]students.StudentMissionAttemptDocument, 0, len(attempts))
	for _, attempt := range attempts {
		if tenancy.NormalizeScope(tenancy.Scope{OrganizationID: attempt.OrganizationID, DistrictID: attempt.DistrictID, SchoolID: attempt.SchoolID}) == scope {
			safeAttempts = append(safeAttempts, attempt)
		}
	}
	return safeUsers, safeAttempts
}
