package main

import (
	"classgame-api/internal/auth"
	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/missions"
	"classgame-api/internal/modules/students"
	"cloud.google.com/go/firestore"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"
)

func (s *suite) concurrentReservation() error {
	f, err := s.seed("reservation", "learning")
	if err != nil {
		return err
	}
	input := s.input(f, "same-event")
	const count = 6
	start := make(chan struct{})
	errorsByWorker := make([]error, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			now := time.Now()
			_, errorsByWorker[i] = s.repo.ReserveTutorGeneration(s.ctx, input, fmt.Sprintf("owner-%d", i), now, now.Add(time.Minute))
		}(i)
	}
	close(start)
	wg.Wait()
	wins, pending, winner := 0, 0, 0
	for i, err := range errorsByWorker {
		if err == nil {
			wins++
			winner = i
		} else if errors.Is(err, students.ErrTutorEventPending) {
			pending++
		} else {
			return fmt.Errorf("unexpected reservation failure: %w", err)
		}
	}
	a, err := s.load(f)
	if err != nil {
		return err
	}
	if wins != 1 || pending != count-1 || a.GeneratedTurnCount != 1 {
		return fmt.Errorf("want one admission and %d busy results; got wins=%d busy=%d admissions=%d", count-1, wins, pending, a.GeneratedTurnCount)
	}
	input.ReservationOwner = fmt.Sprintf("owner-%d", winner)
	if _, err := s.repo.AppendStudentTutorTurn(s.ctx, input, generation.TutorTurnBlueprint{}); err != nil {
		return err
	}
	return nil
}

func (s *suite) durableEvidence() error {
	f, err := s.seed("history", "learning")
	if err != nil {
		return err
	}
	first := s.input(f, "answer-0")
	var firstTurn students.StudentTutorTurn
	for i := 0; i < 3; i++ {
		input := s.input(f, fmt.Sprintf("answer-%d", i))
		input.QuestionID = fmt.Sprintf("synthetic-question-%d", i)
		turn, err := s.reserveAndAppend(input, fmt.Sprintf("owner-%d", i))
		if err != nil {
			return err
		}
		if i == 0 {
			first = input
			firstTurn = turn
		}
	}
	for i := 0; i < 90; i++ {
		input := s.input(f, fmt.Sprintf("feedback-%d", i))
		input.Type = "feedback"
		input.Feedback = "helpful"
		input.StudentResponse = "helpful"
		input.ExpectedAnswer = ""
		if _, err := s.repo.AppendStudentTutorTurn(s.ctx, input, generation.TutorTurnBlueprint{}); err != nil {
			return fmt.Errorf("append feedback %d: %w", i, err)
		}
	}
	before, err := s.load(f)
	if err != nil {
		return err
	}
	if len(before.TutorTurns) != 80 || len(before.AcceptedAnswers) != 3 || before.GeneratedTurnCount != 3 || before.TutorEventSequence != 93 {
		return fmt.Errorf("history lost durable facts/count: turns=%d facts=%d admissions=%d sequence=%d", len(before.TutorTurns), len(before.AcceptedAnswers), before.GeneratedTurnCount, before.TutorEventSequence)
	}
	for _, turn := range before.TutorTurns {
		if turn.ID == firstTurn.ID {
			return errors.New("first answer did not leave the recent context window")
		}
	}
	mission := missions.MissionDocument{ID: f.attempt.MissionID, ClassID: f.attempt.ClassID, PublishedRevision: 1, Title: "Synthetic fractions", Stages: []generation.MissionStage{{ID: "synthetic-stage", Title: "Synthetic stage"}}}
	for i := 0; i < 3; i++ {
		mission.PreviewQuestions = append(mission.PreviewQuestions, generation.PreviewQuestion{ID: fmt.Sprintf("synthetic-question-%d", i), StageID: "synthetic-stage", Prompt: "What is one half?", AnswerType: "multiple_choice", Options: []string{"1/2", "1/3", "1/4"}, ExpectedAnswer: "1/2"})
	}
	completed, _, started, err := s.repo.SaveStudentLearningSnapshot(s.ctx, students.SaveStudentLearningSnapshotInput{Identity: auth.Identity{ProviderUserID: f.user.ID}, ExpectedAttemptID: f.attempt.ID, PublishedRevision: 1, Mission: mission, MissionID: f.attempt.MissionID, Status: "completed", Snapshot: students.StudentLearningSnapshot{Revision: 4}, Timestamp: time.Now()})
	if err != nil {
		return err
	}
	if completed.Status != "completed" || !started || len(completed.Snapshot.Progress) != 1 || completed.Snapshot.Progress[0].Correct != 3 {
		return errors.New("durable accepted facts did not authorize completion after recent-turn eviction")
	}
	before, err = s.load(f)
	if err != nil {
		return err
	}
	event, found, err := s.repo.GetStudentTutorEvent(s.ctx, f.user.ID, f.attempt.ID, first.EventID)
	if err != nil || !found {
		return fmt.Errorf("durable event lookup: found=%t error=%v", found, err)
	}
	replay, err := students.ReplayTutorEvent(event, first)
	if err != nil || replay.ID != firstTurn.ID {
		return fmt.Errorf("old event replay failed: %v", err)
	}
	reservation, err := s.repo.ReserveTutorGeneration(s.ctx, first, "new-owner", time.Now(), time.Now().Add(time.Minute))
	if err != nil || !reservation.Completed || reservation.Turn.ID != firstTurn.ID {
		return fmt.Errorf("exact replay was not free of a new reservation: %v", err)
	}
	turn, err := s.repo.AppendStudentTutorTurn(s.ctx, first, generation.TutorTurnBlueprint{})
	if err != nil || turn.ID != firstTurn.ID {
		return fmt.Errorf("append replay failed: %v", err)
	}
	changed := first
	changed.StudentResponse = "1/3"
	if _, err := s.repo.ReserveTutorGeneration(s.ctx, changed, "changed-owner", time.Now(), time.Now().Add(time.Minute)); !errors.Is(err, students.ErrTutorEventConflict) {
		return fmt.Errorf("changed event reservation must conflict, got %v", err)
	}
	if _, err := s.repo.AppendStudentTutorTurn(s.ctx, changed, generation.TutorTurnBlueprint{}); !errors.Is(err, students.ErrTutorEventConflict) {
		return fmt.Errorf("changed event append must conflict, got %v", err)
	}
	after, err := s.load(f)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(before, after) {
		return errors.New("replay or conflict mutated durable evidence")
	}
	return nil
}

func (s *suite) concurrentBudget() error {
	f, err := s.seed("budget", "learning")
	if err != nil {
		return err
	}
	if _, err = s.attemptRef(f).Update(s.ctx, []firestore.Update{{Path: "generated_turn_count", Value: int64(35)}}); err != nil {
		return err
	}
	const count = 6
	start := make(chan struct{})
	errs := make([]error, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			now := time.Now()
			_, errs[i] = s.repo.ReserveTutorGeneration(s.ctx, s.input(f, fmt.Sprintf("budget-event-%d", i)), fmt.Sprintf("owner-%d", i), now, now.Add(time.Minute))
		}(i)
	}
	close(start)
	wg.Wait()
	wins, limited := 0, 0
	for _, err := range errs {
		if err == nil {
			wins++
		} else if errors.Is(err, students.ErrStudentTutorTurnLimit) {
			limited++
		} else {
			return err
		}
	}
	a, err := s.load(f)
	if err != nil {
		return err
	}
	if wins != 1 || limited != count-1 || a.GeneratedTurnCount != 36 {
		return fmt.Errorf("concurrent budget exceeded: wins=%d limited=%d count=%d", wins, limited, a.GeneratedTurnCount)
	}
	return nil
}

func (s *suite) reservationFence() error {
	f, err := s.seed("event-fence", "learning")
	if err != nil {
		return err
	}
	input := s.input(f, "expired-event")
	now := time.Now()
	if _, err = s.repo.ReserveTutorGeneration(s.ctx, input, "old-owner", now.Add(-3*time.Minute), now.Add(-time.Minute)); err != nil {
		return err
	}
	if _, err = s.repo.ReserveTutorGeneration(s.ctx, input, "new-owner", now, now.Add(time.Minute)); err != nil {
		return err
	}
	old := input
	old.ReservationOwner = "old-owner"
	if err = s.repo.FailTutorGeneration(s.ctx, old, now); err != nil {
		return err
	}
	if _, err = s.repo.AppendStudentTutorTurn(s.ctx, old, generation.TutorTurnBlueprint{}); !errors.Is(err, students.ErrTutorEventPending) {
		return fmt.Errorf("old owner append must remain blocked: %v", err)
	}
	event, found, err := s.repo.GetStudentTutorEvent(s.ctx, f.user.ID, f.attempt.ID, input.EventID)
	if err != nil || !found || event.OwnerToken != "new-owner" || event.State != "processing" {
		return fmt.Errorf("old failure replaced new owner: found=%t error=%v", found, err)
	}
	input.ReservationOwner = "new-owner"
	if _, err = s.repo.AppendStudentTutorTurn(s.ctx, input, generation.TutorTurnBlueprint{}); err != nil {
		return err
	}
	a, err := s.load(f)
	if err != nil {
		return err
	}
	if a.GeneratedTurnCount != 2 || a.TutorEventSequence != 1 {
		return errors.New("uncertain admissions or append count were not retained")
	}
	return nil
}
