package main

import (
	"classgame-api/internal/auth"
	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/missions"
	"classgame-api/internal/modules/students"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"
)

func (s *suite) claim(f fixture, token string, now time.Time) (students.StudentMissionAttemptDocument, error) {
	a, err := s.load(f)
	if err != nil {
		return a, err
	}
	claimed, _, ok, err := s.repo.ClaimStudentFinalization(s.ctx, students.FinalizationJob(a), token, now, now.Add(time.Minute))
	if err != nil {
		return a, err
	}
	if !ok {
		return a, errors.New("expected a finalization claim")
	}
	return claimed, nil
}

func (s *suite) concurrentFinalization() error {
	f, err := s.seed("finalization-concurrent", "abandoned")
	if err != nil {
		return err
	}
	const count = 5
	start := make(chan struct{})
	errs := make([]error, count)
	claimed := make([]bool, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			now := time.Now()
			_, _, claimed[i], errs[i] = s.repo.ClaimStudentFinalization(s.ctx, students.FinalizationJob(f.attempt), fmt.Sprintf("finalizer-%d", i), now, now.Add(time.Minute))
		}(i)
	}
	close(start)
	wg.Wait()
	wins, busy := 0, 0
	for i, err := range errs {
		if err == nil && claimed[i] {
			wins++
		} else if errors.Is(err, students.ErrFinalizationBusy) {
			busy++
		} else {
			return fmt.Errorf("unexpected finalizer claim: claimed=%t err=%v", claimed[i], err)
		}
	}
	if wins != 1 || busy != count-1 {
		return fmt.Errorf("want one finalizer owner, got wins=%d busy=%d", wins, busy)
	}
	return nil
}

func (s *suite) staleFinalizer(change string) error {
	f, err := s.seed("finalization-"+change, "abandoned")
	if err != nil {
		return err
	}
	now := time.Now()
	a, err := s.claim(f, "old-finalizer", now)
	if err != nil {
		return err
	}
	fence := students.CaptureFinalizationFence(a)
	switch change {
	case "owner":
		if _, err = s.claim(f, "new-finalizer", now.Add(2*time.Minute)); err != nil {
			return err
		}
	case "evidence":
		input := s.input(f, "late-feedback")
		input.Type = "feedback"
		input.StudentResponse = "helpful"
		input.Feedback = "helpful"
		if _, err = s.repo.AppendStudentTutorTurn(s.ctx, input, generation.TutorTurnBlueprint{}); err != nil {
			return err
		}
	case "resume":
		mission := missions.MissionDocument{ID: f.attempt.MissionID, ClassID: f.attempt.ClassID, PublishedRevision: 1, Title: "Synthetic fractions", Stages: []generation.MissionStage{{ID: "synthetic-stage", Title: "Synthetic stage"}}, PreviewQuestions: []generation.PreviewQuestion{{ID: "synthetic-question", StageID: "synthetic-stage", Prompt: "What is one half?", AnswerType: "multiple_choice", Options: []string{"1/2", "1/3", "1/4"}, ExpectedAnswer: "1/2"}}}
		_, _, _, err = s.repo.SaveStudentLearningSnapshot(s.ctx, students.SaveStudentLearningSnapshotInput{Identity: auth.Identity{ProviderUserID: f.user.ID}, ExpectedAttemptID: f.attempt.ID, PublishedRevision: 1, Mission: mission, MissionID: f.attempt.MissionID, Status: "learning", Snapshot: students.StudentLearningSnapshot{Revision: 4}, Timestamp: now.Add(time.Second)})
		if err != nil {
			return err
		}
	}
	before, err := s.load(f)
	if err != nil {
		return err
	}
	if _, err = s.repo.FinalizeStudentLearningTrace(s.ctx, fence, generation.LearningTraceBlueprint{Summary: "stale synthetic result"}, students.StudentLearningMemory{}, now.Add(3*time.Minute)); !errors.Is(err, students.ErrStaleFinalization) {
		return fmt.Errorf("stale finalizer %s must be fenced: %v", change, err)
	}
	if err = s.repo.FailStudentFinalization(s.ctx, fence, now.Add(3*time.Minute)); err != nil {
		return err
	}
	after, err := s.load(f)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(before, after) {
		return fmt.Errorf("stale finalizer %s changed current state", change)
	}
	return nil
}

func (s *suite) finalizationReady() error {
	f, err := s.seed("finalization-ready", "abandoned")
	if err != nil {
		return err
	}
	now := time.Now()
	a, err := s.claim(f, "ready-owner", now)
	if err != nil {
		return err
	}
	fence := students.CaptureFinalizationFence(a)
	memory := students.StudentLearningMemory{LastMissionID: f.attempt.MissionID, EvidenceAt: f.attempt.LastActivityAt, NextAdaptation: "synthetic next step"}
	if _, err = s.repo.FinalizeStudentLearningTrace(s.ctx, fence, generation.LearningTraceBlueprint{Summary: "synthetic final result"}, memory, now); err != nil {
		return err
	}
	before, err := s.load(f)
	if err != nil {
		return err
	}
	if !before.Trace.Finalized || before.TraceState != "ready" {
		return errors.New("valid finalization did not commit")
	}
	if err = s.repo.FailStudentFinalization(s.ctx, fence, now.Add(time.Second)); err != nil {
		return err
	}
	if _, err = s.repo.FinalizeStudentLearningTrace(s.ctx, fence, generation.LearningTraceBlueprint{Summary: "late duplicate"}, memory, now.Add(time.Second)); err != nil {
		return err
	}
	after, err := s.load(f)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(before, after) {
		return errors.New("late failure or success changed ready result")
	}
	return nil
}
