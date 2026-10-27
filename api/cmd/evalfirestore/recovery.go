package main

import (
	"classgame-api/internal/modules/students"
	"errors"
	"fmt"
	"time"
)

func (s *suite) recoveryPage() error {
	prefix := s.prefix + "-recovery-"
	now := time.Now()
	for i := 0; i < 7; i++ {
		a := students.StudentMissionAttemptDocument{ID: fmt.Sprintf("%s%02d", prefix, i), StudentID: "synthetic-recovery-student", Status: "abandoned", EvidenceSchemaVersion: 1, TraceState: "pending"}
		if i == 1 {
			a.TraceState = "processing"
			a.FinalizationLeaseUntil = now.Add(time.Minute)
		}
		if i == 3 {
			a.Status = "completed"
			a.EvidenceSchemaVersion = 0
		}
		ref := s.client.Collection("student_mission_attempts").Doc(a.ID)
		if _, err := ref.Create(s.ctx, a); err != nil {
			return err
		}
		s.extra = append(s.extra, ref)
	}
	first, err := s.repo.ListPendingFinalizations(s.ctx, prefix, 2, now)
	if err != nil {
		return err
	}
	if first.Scanned != 2 || len(first.Attempts) != 1 || first.Attempts[0].ID != prefix+"00" || first.NextCursor != prefix+"01" {
		return fmt.Errorf("unexpected first bounded page: scanned=%d candidates=%d cursor=%q", first.Scanned, len(first.Attempts), first.NextCursor)
	}
	second, err := s.repo.ListPendingFinalizations(s.ctx, first.NextCursor, 2, now)
	if err != nil {
		return err
	}
	if second.Scanned != 2 || len(second.Attempts) != 1 || second.Attempts[0].ID != prefix+"02" || second.NextCursor != prefix+"03" {
		return errors.New("cursor did not advance or legacy completion was regraded")
	}
	for _, limit := range []int{0, 101} {
		if _, err := s.repo.ListPendingFinalizations(s.ctx, prefix, limit, now); err == nil {
			return errors.New("invalid recovery limit accepted")
		}
	}
	if _, err := s.repo.ListPendingFinalizations(s.ctx, "invalid/cursor", 2, now); err == nil {
		return errors.New("invalid cursor accepted")
	}
	return nil
}
