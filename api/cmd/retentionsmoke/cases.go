package main

import (
	"classgame-api/internal/datarecovery"
	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/students"
	"cloud.google.com/go/firestore"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (s *suite) cases(artifact, keyPath, temp string) []result {
	ctx, store := s.ctx, s.store
	results := []result{}
	run := func(id string, work func() error) {
		r := result{ID: id, Passed: true}
		if e := work(); e != nil {
			r.Passed = false
			r.Detail = e.Error()
		}
		results = append(results, r)
		fmt.Fprintf(os.Stderr, "%s: %v\n", id, r.Passed)
	}
	var bundle datarecovery.Bundle
	run("scoped-export-includes-private-projection", func() error {
		var e error
		bundle, e = store.Export(ctx, s.scope)
		if e != nil {
			return e
		}
		found := map[string]bool{}
		for _, r := range bundle.Records {
			found[strings.Split(r.Path, "/")[0]] = true
			if strings.HasPrefix(r.Path, "users/") || strings.HasPrefix(r.Path, "tenants/") {
				return errors.New("global record entered class bundle")
			}
		}
		for _, family := range []string{"teacher_classes", "missions", "student_mission_attempts", "student_mission_completions", "class_learning_students", "class_learning_summaries", "student_class_missions"} {
			if !found[family] {
				return fmt.Errorf("missing %s", family)
			}
		}
		return nil
	})

	run("read-only-plan-command", func() error {
		args := []string{"-project", store.Project, "-organization", s.scope.Organization, "-district", s.scope.District, "-school", s.scope.School, "-class", s.scope.Class}
		raw, e := exec.CommandContext(ctx, os.Getenv("RETENTION_BACKUP_COMMAND"), args...).Output()
		if e != nil {
			return e
		}
		var response map[string]any
		if e = json.Unmarshal(raw, &response); e != nil {
			return e
		}
		if response["database_writes"] != float64(0) || response["global_identities_included"] != false {
			return errors.New("plan is not read-only/scoped")
		}
		return nil
	})
	run("read-only-encrypted-export-command", func() error {
		file := filepath.Join(temp, "cli.enc")
		args := []string{"-mode", "export", "-project", store.Project, "-organization", s.scope.Organization, "-district", s.scope.District, "-school", s.scope.School, "-class", s.scope.Class, "-file", file, "-key-file", keyPath}
		raw, e := exec.CommandContext(ctx, os.Getenv("RETENTION_BACKUP_COMMAND"), args...).Output()
		if e != nil {
			return e
		}
		var response map[string]any
		if e = json.Unmarshal(raw, &response); e != nil {
			return e
		}
		if response["database_writes"] != float64(0) || response["encrypted"] != true {
			return errors.New("export changed data or skipped encryption")
		}
		b, e := datarecovery.ReadEncrypted(file, keyPath)
		if e != nil {
			return e
		}
		if len(b.Records) != len(bundle.Records) {
			return errors.New("CLI export is incomplete")
		}
		return nil
	})
	run("encrypted-owner-only-artifact-roundtrip", func() error {
		if e := datarecovery.WriteEncrypted(artifact, keyPath, bundle); e != nil {
			return e
		}
		read, e := datarecovery.ReadEncrypted(artifact, keyPath)
		if e != nil {
			return e
		}
		if read.SHA256 != bundle.SHA256 || len(read.Records) != len(bundle.Records) {
			return errors.New("artifact differs")
		}
		return nil
	})
	run("tampered-ciphertext-and-wrong-key-rejected", func() error {
		raw, e := os.ReadFile(artifact)
		if e != nil {
			return e
		}
		var envelope map[string]any
		if e = json.Unmarshal(raw, &envelope); e != nil {
			return e
		}
		value := envelope["ciphertext"].(string)
		first := "A"
		if value[:1] == first {
			first = "B"
		}
		envelope["ciphertext"] = first + value[1:]
		changed, _ := json.Marshal(envelope)
		os.WriteFile(artifact, changed, 0600)
		_, bad := datarecovery.ReadEncrypted(artifact, keyPath)
		os.WriteFile(artifact, raw, 0600)
		wrong := filepath.Join(temp, "wrong-key")
		os.WriteFile(wrong, make([]byte, 32), 0600)
		_, badKey := datarecovery.ReadEncrypted(artifact, wrong)
		if bad == nil || badKey == nil {
			return errors.New("invalid artifact accepted")
		}
		return nil
	})
	run("foreign-scope-and-duplicate-records-rejected", func() error {
		bad := bundle
		bad.Scope.Organization = "foreign-org"
		bad.SHA256 = bad.Digest()
		if bad.Validate() == nil {
			return errors.New("foreign tenant accepted")
		}
		bad = bundle
		bad.Records = append(append([]datarecovery.Record(nil), bundle.Records...), bundle.Records[0])
		bad.SHA256 = bad.Digest()
		if bad.Validate() == nil {
			return errors.New("duplicate accepted")
		}
		return nil
	})
	run("delete-requires-current-owner-hold", func() error {
		if _, e := store.Delete(ctx, bundle); e == nil {
			return errors.New("delete ran without hold")
		}
		if e := store.AcquireHold(ctx, bundle); e != nil {
			return e
		}
		other := bundle
		other.Owner = strings.Repeat("f", 32)
		other.SHA256 = other.Digest()
		if e := store.AcquireHold(ctx, other); !errors.Is(e, datarecovery.ErrConflict) {
			return fmt.Errorf("foreign owner accepted: %v", e)
		}
		return nil
	})
	run("changed-source-aborts-entire-delete", func() error {
		ref := store.Client.Doc("student_mission_attempts/" + s.attempt.ID)
		if _, e := ref.Update(ctx, []firestore.Update{{Path: "snapshot.revision", Value: int64(9)}}); e != nil {
			return e
		}
		if _, e := store.Delete(ctx, bundle); e == nil {
			return errors.New("changed source deleted")
		}
		if _, e := store.Client.Doc("teacher_classes/" + s.scope.Class).Get(ctx); e != nil {
			return errors.New("atomic delete partly applied")
		}
		if e := store.ReleaseHold(ctx, bundle); e != nil {
			return e
		}
		var e error
		bundle, e = store.Export(ctx, s.scope)
		return e
	})
	run("scoped-delete-preserves-other-classes-and-users", func() error {
		if e := store.AcquireHold(ctx, bundle); e != nil {
			return e
		}
		changed, e := store.Delete(ctx, bundle)
		if e != nil || !changed {
			return fmt.Errorf("delete: %v", e)
		}
		for _, r := range bundle.Records {
			if _, e := store.Client.Doc(r.Path).Get(ctx); status.Code(e) != codes.NotFound {
				return errors.New("class record remains")
			}
		}
		for _, name := range []string{"other-class", "other-tenant"} {
			snap, e := store.Client.Doc("teacher_classes/" + s.prefix + "-" + name).Get(ctx)
			if e != nil || snap.Data()["probe"] != "unchanged" {
				return errors.New("foreign class changed")
			}
		}
		if _, e := store.Client.Doc("users/" + s.user.ID).Get(ctx); e != nil {
			return errors.New("global identity deleted")
		}
		return nil
	})
	run("restore-preserves-timestamps-integers-and-projection", func() error {
		changed, e := store.Restore(ctx, bundle)
		if e != nil || !changed {
			return fmt.Errorf("restore: %v", e)
		}
		snap, e := store.Client.Doc("student_mission_attempts/" + s.attempt.ID).Get(ctx)
		if e != nil {
			return e
		}
		var a students.StudentMissionAttemptDocument
		if e = snap.DataTo(&a); e != nil {
			return e
		}
		if a.TutorEventSequence != s.attempt.TutorEventSequence || !a.CreatedAt.Equal(s.attempt.CreatedAt) || len(a.AcceptedAnswers) != 1 {
			return errors.New("typed evidence changed")
		}
		row, e := store.Client.Doc("class_learning_students/" + s.scope.Class + "/students/" + s.user.ID).Get(ctx)
		if e != nil || row.Data()["attempt_id"] != s.attempt.ID {
			return errors.New("projection missing")
		}
		return nil
	})
	run("nonrepresentable-int64-restores-exactly-v1", func() error {
		snapshot, e := store.Client.Doc("teacher_classes/" + s.scope.Class).Get(ctx)
		if e != nil {
			return e
		}
		value, ok := snapshot.Data()["mission_count"].(int64)
		if !ok || value != 9007199254740993 {
			return errors.New("integer type or precision changed through backup")
		}
		return nil
	})
	run("old-finalizer-and-provider-owners-are-fenced", func() error {
		repo := students.NewFirestoreRepository(store.Client)
		if _, e := repo.FinalizeStudentLearningTrace(ctx, s.fence, generation.LearningTraceBlueprint{Finalized: true}, students.StudentLearningMemory{}, time.Now()); !errors.Is(e, students.ErrStaleFinalization) {
			return fmt.Errorf("old finalizer result: %v", e)
		}
		if e := repo.FailStudentFinalization(ctx, s.fence, time.Now()); e != nil {
			return e
		}
		if _, e := repo.AppendStudentTutorTurn(ctx, s.pendingInput, generation.TutorTurnBlueprint{Message: "late synthetic provider"}); e == nil {
			return errors.New("old provider committed")
		}
		return nil
	})
	run("completed-event-replay-survives-restore", func() error {
		repo := students.NewFirestoreRepository(store.Client)
		r, e := repo.ReserveTutorGeneration(ctx, s.input, "new-owner", time.Now(), time.Now().Add(time.Minute))
		if e != nil || !r.Completed || r.Turn.ID != s.turnID {
			return fmt.Errorf("saved replay lost: %v", e)
		}
		return nil
	})
	run("repeat-restore-does-not-overwrite-new-writes", func() error {
		ref := store.Client.Doc("teacher_classes/" + s.scope.Class)
		if _, e := ref.Update(ctx, []firestore.Update{{Path: "class_name", Value: "Updated after restore"}}); e != nil {
			return e
		}
		changed, e := store.Restore(ctx, bundle)
		if e != nil || changed {
			return fmt.Errorf("repeat restore changed state: %v", e)
		}
		snap, e := ref.Get(ctx)
		if e != nil || snap.Data()["class_name"] != "Updated after restore" {
			return errors.New("new data overwritten")
		}
		return nil
	})
	return results
}
