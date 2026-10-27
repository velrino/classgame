package main

import (
	"classgame-api/internal/auth"
	"classgame-api/internal/datarecovery"
	"classgame-api/internal/modules/classrooms"
	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/missions"
	"classgame-api/internal/modules/students"
	"cloud.google.com/go/firestore"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type suite struct {
	ctx          context.Context
	store        *datarecovery.Store
	prefix       string
	refs         []*firestore.DocumentRef
	scope        datarecovery.Scope
	user         students.UserDocument
	attempt      students.StudentMissionAttemptDocument
	pending      students.StudentMissionAttemptDocument
	input        students.SaveStudentTutorTurnInput
	pendingInput students.SaveStudentTutorTurnInput
	fence        students.FinalizationFence
	turnID       string
}

func (s *suite) put(path string, value any) error {
	ref := s.store.Client.Doc(path)
	if _, e := ref.Create(s.ctx, value); e != nil {
		return e
	}
	s.refs = append(s.refs, ref)
	return nil
}
func (s *suite) seed() error {
	now := time.Now().UTC().Truncate(time.Microsecond)
	s.scope = datarecovery.Scope{Organization: s.prefix + "-org", District: s.prefix + "-district", School: s.prefix + "-school", Class: s.prefix + "-class"}
	tenant := map[string]any{"organization_id": s.scope.Organization, "district_id": s.scope.District, "school_id": s.scope.School, "class_id": s.scope.Class}
	scoped := func(extra map[string]any) map[string]any {
		out := map[string]any{}
		for k, v := range tenant {
			out[k] = v
		}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}
	class := classrooms.TeacherClassDocument{ID: s.scope.Class, ClassID: s.scope.Class, OrganizationID: s.scope.Organization, DistrictID: s.scope.District, SchoolID: s.scope.School, TeacherID: s.prefix + "-teacher", MissionCount: 9007199254740993, Status: "archived", CreatedAt: now, UpdatedAt: now}
	if e := s.put("teacher_classes/"+s.scope.Class, class); e != nil {
		return e
	}
	for _, name := range []string{"other-class", "other-tenant"} {
		if e := s.put("teacher_classes/"+s.prefix+"-"+name, map[string]any{"probe": "unchanged", "organization_id": "other-org", "class_id": s.prefix + "-" + name}); e != nil {
			return e
		}
	}
	mission := missions.MissionDocument{ID: s.prefix + "-mission", ClassID: s.scope.Class, OrganizationID: s.scope.Organization, DistrictID: s.scope.District, SchoolID: s.scope.School, TeacherID: class.TeacherID, PublishedRevision: 1, CreatedAt: now, UpdatedAt: now}
	if e := s.put("missions/"+mission.ID, mission); e != nil {
		return e
	}
	if e := s.put("missions/"+mission.ID+"/published_revisions/1", scoped(map[string]any{"mission_id": mission.ID, "revision": int64(1), "published_at": now})); e != nil {
		return e
	}
	if e := s.put("student_class_missions/"+s.scope.Class+"/missions/"+mission.ID, scoped(map[string]any{"mission_id": mission.ID})); e != nil {
		return e
	}
	s.attempt = students.StudentMissionAttemptDocument{ID: s.prefix + "-attempt", StudentID: s.prefix + "-student", ClassID: s.scope.Class, MissionID: mission.ID, OrganizationID: s.scope.Organization, DistrictID: s.scope.District, SchoolID: s.scope.School, PublishedRevision: 1, Status: "learning", EvidenceSchemaVersion: 1, TutorEventSequence: 9007199254740993, Snapshot: students.StudentLearningSnapshot{Revision: 3}, TraceState: "live", CreatedAt: now, UpdatedAt: now, LastActivityAt: now, CompletedAt: now}
	s.pending = s.attempt
	s.pending.ID = s.prefix + "-pending"
	s.pending.StudentID = s.prefix + "-pending-student"
	for _, attempt := range []students.StudentMissionAttemptDocument{s.attempt, s.pending} {
		if e := s.put("student_mission_attempts/"+attempt.ID, attempt); e != nil {
			return e
		}
		user := students.UserDocument{ID: attempt.StudentID, Student: students.StudentProfile{ClassID: s.scope.Class, MissionID: mission.ID, OrganizationID: s.scope.Organization, DistrictID: s.scope.District, SchoolID: s.scope.School}, Activity: students.UserActivity{CurrentGame: students.StudentGameSession{AttemptID: attempt.ID, MissionID: mission.ID, PublishedRevision: 1}}}
		if e := s.put("users/"+user.ID, user); e != nil {
			return e
		}
		if user.ID == s.attempt.StudentID {
			s.user = user
		}
	}
	input := func(a students.StudentMissionAttemptDocument, event string) students.SaveStudentTutorTurnInput {
		return students.SaveStudentTutorTurnInput{Identity: auth.Identity{ProviderUserID: a.StudentID}, ExpectedAttemptID: a.ID, ExpectedPublishedRevision: 1, RequestScope: students.AttemptScope{AttemptID: a.ID, PublishedRevision: 1}, MissionID: a.MissionID, EventID: event, Type: "answer", QuestionID: "q", StageID: "stage", StudentResponse: "1/2", ExpectedAnswer: "1/2", Timestamp: now}
	}
	repo := students.NewFirestoreRepository(s.store.Client)
	s.input = input(s.attempt, "saved-event")
	s.input.ReservationOwner = "original-provider"
	if _, e := repo.ReserveTutorGeneration(s.ctx, s.input, s.input.ReservationOwner, now, now.Add(time.Minute)); e != nil {
		return e
	}
	turn, e := repo.AppendStudentTutorTurn(s.ctx, s.input, generation.TutorTurnBlueprint{Message: "Synthetic learning feedback"})
	if e != nil {
		return e
	}
	s.turnID = turn.ID
	if _, e = s.store.Client.Doc("student_mission_attempts/"+s.attempt.ID).Update(s.ctx, []firestore.Update{{Path: "status", Value: "completed"}, {Path: "trace.finalized", Value: true}, {Path: "trace_state", Value: "ready"}}); e != nil {
		return e
	}
	s.pendingInput = input(s.pending, "processing-event")
	s.pendingInput.ReservationOwner = "old-provider"
	if _, e = repo.ReserveTutorGeneration(s.ctx, s.pendingInput, s.pendingInput.ReservationOwner, now, now.Add(time.Minute)); e != nil {
		return e
	}
	if _, e = s.store.Client.Doc("student_mission_attempts/"+s.pending.ID).Update(s.ctx, []firestore.Update{{Path: "status", Value: "completed"}, {Path: "trace.finalized", Value: false}, {Path: "trace_state", Value: "pending"}}); e != nil {
		return e
	}
	for i := range []int{0, 1} {
		a := &s.attempt
		if i == 1 {
			a = &s.pending
		}
		snap, e := s.store.Client.Doc("student_mission_attempts/" + a.ID).Get(s.ctx)
		if e != nil {
			return e
		}
		if e = snap.DataTo(a); e != nil {
			return e
		}
	}
	claimed, _, ok, e := repo.ClaimStudentFinalization(s.ctx, students.FinalizationJob(s.pending), "old-finalizer", now, now.Add(time.Minute))
	if e != nil || !ok {
		return fmt.Errorf("seed finalizer: %v", e)
	}
	s.fence = students.CaptureFinalizationFence(claimed)
	for path, value := range map[string]any{
		"student_mission_completions/" + s.user.ID + "/missions/" + s.attempt.ID: map[string]any{"student_id": s.user.ID, "class_id": s.scope.Class, "mission_id": mission.ID, "published_revision": int64(1), "status": "completed"},
		"class_learning_summaries/" + s.scope.Class:                              scoped(map[string]any{"attempt_count": int64(1), "evidence_updated_at": now}),
		"class_learning_students/" + s.scope.Class:                               scoped(map[string]any{"version": int64(1), "generation": "synthetic-generation", "next_cursor": "", "complete": true, "updated_at": now}),
		"class_learning_students/" + s.scope.Class + "/students/" + s.user.ID:    scoped(map[string]any{"version": int64(1), "attempt_id": s.attempt.ID, "student_id": s.user.ID, "student_name": "Synthetic student", "evidence_revision": s.attempt.EvidenceRevision, "snapshot_revision": s.attempt.Snapshot.Revision, "status": "completed", "mission_id": s.attempt.MissionID, "published_revision": int64(1), "evidence_at": now, "projected_at": now, "strengths": []string{"synthetic skill"}, "improvement_areas": []string{}}),
	} {
		if e := s.put(path, value); e != nil {
			return e
		}
	}
	return nil
}
func (s *suite) cleanup() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var all []error
	for _, a := range []students.StudentMissionAttemptDocument{s.attempt, s.pending} {
		if a.ID == "" {
			continue
		}
		rows, e := s.store.Client.Doc("student_mission_attempts/" + a.ID).Collection("tutor_events").Limit(10).Documents(ctx).GetAll()
		if e != nil {
			all = append(all, e)
		}
		for _, row := range rows {
			if _, e = row.Ref.Delete(ctx); e != nil {
				all = append(all, e)
			}
		}
	}
	for _, ref := range s.refs {
		if _, e := ref.Delete(ctx); e != nil {
			all = append(all, e)
		}
	}
	if s.scope.Class != "" {
		if _, e := s.store.Client.Collection(students.ClassRefreshCollection).Doc(s.scope.Class).Delete(ctx); e != nil {
			all = append(all, e)
		}
	}
	hash := sha256.Sum256([]byte(s.scope.Organization + "/" + s.scope.District + "/" + s.scope.School + "/" + s.scope.Class))
	if _, e := s.store.Client.Collection("data_recovery_holds").Doc(hex.EncodeToString(hash[:])).Delete(ctx); e != nil {
		all = append(all, e)
	}
	if len(all) > 0 {
		return fmt.Errorf("%d owned fixture deletes failed", len(all))
	}
	return nil
}
