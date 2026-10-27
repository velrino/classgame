package main

import (
	"classgame-api/internal/auth"
	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/students"
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"time"
)

type suite struct {
	ctx      context.Context
	client   *firestore.Client
	repo     *students.FirestoreRepository
	prefix   string
	fixtures []fixture
	extra    []*firestore.DocumentRef
}
type fixture struct {
	user    students.UserDocument
	attempt students.StudentMissionAttemptDocument
}

func (s *suite) seed(name, status string) (fixture, error) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	id := s.prefix + "-" + name
	a := students.StudentMissionAttemptDocument{ID: id, StudentID: id + "-student", ClassID: id + "-class", MissionID: id + "-mission", OrganizationID: "synthetic-org", DistrictID: "synthetic-district", SchoolID: "synthetic-school", PublishedRevision: 1, Status: status, EvidenceSchemaVersion: 1, Snapshot: students.StudentLearningSnapshot{Revision: 3}, EvidenceRevision: 4, TraceState: "pending", StartedAt: now, LastActivityAt: now, CreatedAt: now, UpdatedAt: now}
	if status == "learning" {
		a.TraceState = "live"
	}
	u := students.UserDocument{ID: a.StudentID, Student: students.StudentProfile{ClassID: a.ClassID, MissionID: a.MissionID, OrganizationID: a.OrganizationID, DistrictID: a.DistrictID, SchoolID: a.SchoolID}, Activity: students.UserActivity{CurrentGame: students.StudentGameSession{AttemptID: a.ID, MissionID: a.MissionID, PublishedRevision: 1, Status: status, StartedAt: now}}}
	f := fixture{u, a}
	// Cleanup only documents this run created successfully; never clear a collection.
	userRef := s.client.Collection("users").Doc(u.ID)
	if _, err := userRef.Create(s.ctx, u); err != nil {
		return f, err
	}
	s.extra = append(s.extra, userRef)
	if _, err := s.attemptRef(f).Create(s.ctx, a); err != nil {
		return f, err
	}
	s.fixtures = append(s.fixtures, f)
	return f, nil
}

func (s *suite) attemptRef(f fixture) *firestore.DocumentRef {
	return s.client.Collection("student_mission_attempts").Doc(f.attempt.ID)
}

func (s *suite) load(f fixture) (students.StudentMissionAttemptDocument, error) {
	snap, err := s.attemptRef(f).Get(s.ctx)
	if err != nil {
		return students.StudentMissionAttemptDocument{}, err
	}
	var a students.StudentMissionAttemptDocument
	err = snap.DataTo(&a)
	return a, err
}

func (s *suite) input(f fixture, event string) students.SaveStudentTutorTurnInput {
	return students.SaveStudentTutorTurnInput{Identity: auth.Identity{ProviderUserID: f.user.ID}, ExpectedAttemptID: f.attempt.ID, ExpectedPublishedRevision: 1, RequestScope: students.AttemptScope{AttemptID: f.attempt.ID, PublishedRevision: 1}, MissionID: f.attempt.MissionID, EventID: event, Type: "answer", StageID: "synthetic-stage", QuestionID: "synthetic-question", Prompt: "What is one half?", StudentResponse: "1/2", ExpectedAnswer: "1/2", Outcome: "correct", Timestamp: time.Now().UTC()}
}

func (s *suite) reserveAndAppend(input students.SaveStudentTutorTurnInput, owner string) (students.StudentTutorTurn, error) {
	now := time.Now()
	r, err := s.repo.ReserveTutorGeneration(s.ctx, input, owner, now, now.Add(time.Minute))
	if err != nil {
		return students.StudentTutorTurn{}, err
	}
	if r.Completed {
		return r.Turn, nil
	}
	input.ReservationOwner = owner
	return s.repo.AppendStudentTutorTurn(s.ctx, input, generation.TutorTurnBlueprint{Message: "Synthetic feedback", Provider: "emulator-evaluation"})
}

func (s *suite) cleanup() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var errs []error
	for _, f := range s.fixtures {
		ref := s.attemptRef(f)
		rows, err := ref.Collection("tutor_events").Limit(512).Documents(ctx).GetAll()
		if err != nil {
			errs = append(errs, err)
		}
		for _, row := range rows {
			if _, err := row.Ref.Delete(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		for _, ref := range []*firestore.DocumentRef{ref, s.client.Collection("users").Doc(f.user.ID), s.client.Collection("student_mission_completions").Doc(f.user.ID).Collection("missions").Doc(f.attempt.ID), s.client.Collection(students.ClassRefreshCollection).Doc(f.attempt.ClassID)} {
			if _, err := ref.Delete(ctx); err != nil {
				errs = append(errs, err)
			}
		}
	}
	for _, ref := range s.extra {
		if _, err := ref.Delete(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("could not remove %d fixture documents (prefix %s)", len(errs), s.prefix)
	}
	return nil
}
