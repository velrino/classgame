// Command evalqueries measures bounded queries with owned synthetic emulator fixtures.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"cloud.google.com/go/firestore"
	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"classgame-api/internal/auth"
	"classgame-api/internal/modules/classrooms"
	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/insights"
	"classgame-api/internal/modules/students"
	"classgame-api/internal/pagination"
)

type counterStream struct {
	grpc.ClientStream
	count  *atomic.Int64
	points *atomic.Int64
}

func (s *counterStream) RecvMsg(value any) error {
	err := s.ClientStream.RecvMsg(value)
	if err == nil {
		if row, ok := value.(*pb.RunQueryResponse); ok && row.Document != nil {
			s.count.Add(1)
		}
		if row, ok := value.(*pb.BatchGetDocumentsResponse); ok && (row.GetFound() != nil || row.GetMissing() != "") {
			s.points.Add(1)
		}
	}
	return err
}

type result struct {
	ID             string `json:"id"`
	Passed         bool   `json:"passed"`
	Documents      int64  `json:"query_documents"`
	PointDocuments int64  `json:"point_documents"`
	DurationMS     int64  `json:"duration_ms"`
	Detail         string `json:"detail,omitempty"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	flags := flag.NewFlagSet("evalqueries", flag.ContinueOnError)
	optIn := flags.Bool("run", false, "allow owned synthetic emulator writes")
	project := flags.String("project", "demo-classgame-query", "demo- project ID")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	host, port, err := net.SplitHostPort(os.Getenv("FIRESTORE_EMULATOR_HOST"))
	ip := net.ParseIP(host)
	portNumber, portError := strconv.Atoi(port)
	if !*optIn || flags.NArg() != 0 || err != nil || ip == nil || !ip.IsLoopback() || portError != nil || portNumber < 1 || portNumber > 65535 || !regexp.MustCompile(`^demo-[a-z0-9][a-z0-9-]{0,50}$`).MatchString(*project) {
		return errors.New("require -run, a numeric loopback FIRESTORE_EMULATOR_HOST, and a demo- project")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	var documents atomic.Int64
	var pointDocuments atomic.Int64
	interceptor := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, options ...grpc.CallOption) (grpc.ClientStream, error) {
		stream, err := streamer(ctx, desc, cc, method, options...)
		if err != nil {
			return nil, err
		}
		return &counterStream{stream, &documents, &pointDocuments}, nil
	}
	conn, err := grpc.NewClient(net.JoinHostPort(host, port), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithStreamInterceptor(interceptor))
	if err != nil {
		return err
	}
	defer conn.Close()
	client, err := firestore.NewClient(ctx, *project, option.WithoutAuthentication(), option.WithGRPCConn(conn))
	if err != nil {
		return err
	}
	defer client.Close()
	entropy := make([]byte, 8)
	if _, err := rand.Read(entropy); err != nil {
		return err
	}
	prefix := "QS" + strings.ToUpper(hex.EncodeToString(entropy))
	teacherID := prefix + "teacher"
	now := time.Now().UTC()
	owned := []*firestore.DocumentRef{}
	cleanup := func() error {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for start := 0; start < len(owned); start += 400 {
			batch := client.Batch()
			for _, ref := range owned[start:min(start+400, len(owned))] {
				batch.Delete(ref)
			}
			if _, err := batch.Commit(cleanupCtx); err != nil {
				return err
			}
		}
		return nil
	}
	defer cleanup()
	class := classrooms.TeacherClassDocument{ID: prefix, ClassID: prefix, TeacherID: teacherID, Status: "active"}
	classRef := client.Collection("teacher_classes").Doc(prefix)
	owned = append(owned, classRef, client.Collection("teachers").Doc(teacherID))
	if _, err := classRef.Set(ctx, class); err != nil {
		return err
	}
	batch := client.Batch()
	writes := 0
	flush := func() error {
		if writes == 0 {
			return nil
		}
		_, err := batch.Commit(ctx)
		batch = client.Batch()
		writes = 0
		return err
	}
	var first students.StudentMissionAttemptDocument
	for u := 0; u < 120; u++ {
		studentID := fmt.Sprintf("%s-u%03d", prefix, u)
		ref := client.Collection("users").Doc(studentID)
		owned = append(owned, ref)
		batch.Set(ref, map[string]any{"id": studentID, "roles": []string{"student"}, "student": map[string]any{"class_id": prefix, "student_name": "Synthetic student"}, "activity": map[string]any{"last_seen_at": now}})
		writes++
		for a := 0; a < 12; a++ {
			id := fmt.Sprintf("%s-u%03d-a%03d", prefix, u, a)
			attempt := students.StudentMissionAttemptDocument{ID: id, StudentID: studentID, StudentName: "Synthetic student", ClassID: prefix, MissionID: "synthetic-mission", PublishedRevision: 1, Status: "completed", EvidenceSchemaVersion: 1, EvidenceRevision: int64(a + 1), LastActivityAt: now.Add(-time.Duration(a/3) * time.Minute), UpdatedAt: now, Trace: generation.LearningTraceBlueprint{Finalized: true, Strengths: []string{"synthetic-skill"}}}
			attempt.Snapshot.Revision = int64(a + 1)
			ref := client.Collection("student_mission_attempts").Doc(id)
			owned = append(owned, ref)
			batch.Set(ref, attempt)
			writes++
			if u == 0 && a == 0 {
				first = attempt
			}
			if writes >= 400 {
				if err := flush(); err != nil {
					return err
				}
			}
		}
	}
	if err := flush(); err != nil {
		return err
	}
	repo := insights.NewFirestoreRepository(client)
	learning := students.NewFirestoreRepository(client)
	identity := auth.Identity{ProviderUserID: teacherID, AuthMethod: auth.AuthMethodEmail}
	service := insights.NewService(repo, classrooms.NewService(classrooms.NewFirestoreRepository(client)), nil)
	owned = append(owned, client.Collection(students.ClassEvidenceCollection).Doc(prefix))
	for u := 0; u < 120; u++ {
		owned = append(owned, client.Collection(students.ClassEvidenceCollection).Doc(prefix).Collection("students").Doc(fmt.Sprintf("%s-u%03d", prefix, u)))
	}
	cases := []result{}
	check := func(id string, fn func() error) {
		documents.Store(0)
		pointDocuments.Store(0)
		r := result{ID: id, Passed: true}
		started := time.Now()
		if err := fn(); err != nil {
			r.Passed = false
			r.Detail = err.Error()
		}
		r.Documents = documents.Load()
		r.PointDocuments = pointDocuments.Load()
		r.DurationMS = time.Since(started).Milliseconds()
		cases = append(cases, r)
		fmt.Fprintln(os.Stderr, id, r.Passed, r.Documents)
	}
	check("fixed-fixture-legacy-read-baseline-v1", func() error {
		users, err := client.Collection("users").Where("student.class_id", "==", prefix).Limit(2000).Documents(ctx).GetAll()
		if err != nil {
			return err
		}
		attempts, err := client.Collection("student_mission_attempts").Where("class_id", "==", prefix).Limit(2000).Documents(ctx).GetAll()
		if err != nil {
			return err
		}
		if len(users) != 120 || len(attempts) != 1440 || documents.Load() != 1560 {
			return errors.New("unexpected fixed fixture baseline")
		}
		return nil
	})
	check("bounded-roster-and-latest-window-v1", func() error {
		page, err := service.GetOverviewPage(ctx, identity, prefix, pagination.Params{Limit: 25})
		if err != nil {
			return err
		}
		if len(page.Students) != 25 || page.Page == nil || page.Page.NextCursor == "" || page.Page.MetricsScope != "student_page" || documents.Load() > 51 {
			return fmt.Errorf("unexpected page/reads:%d/%d", len(page.Students), documents.Load())
		}
		for _, student := range page.Students {
			if len(student.Attempts) != 1 || len(student.Attempts[0].TutorTurns) != 0 {
				return errors.New("overview returned full history")
			}
		}
		return nil
	})
	check("full-roster-cursor-no-gaps-v1", func() error {
		params := pagination.Params{Limit: 25}
		seen := map[string]bool{}
		for {
			page, err := repo.ListStudentPage(ctx, prefix, params)
			if err != nil {
				return err
			}
			if page.ReadDocuments > 26 {
				return errors.New("page read bound")
			}
			for _, u := range page.Students {
				if seen[u.ID] {
					return errors.New("duplicate student")
				}
				seen[u.ID] = true
			}
			if page.NextCursor == "" {
				break
			}
			params, err = pagination.Parse(mapValues(page.NextCursor, 25))
			if err != nil {
				return err
			}
		}
		if len(seen) != 120 {
			return fmt.Errorf("roster missing:%d", len(seen))
		}
		return nil
	})
	check("full-history-pages-no-gaps-v1", func() error {
		params := pagination.Params{Limit: 5}
		seen := map[string]bool{}
		for {
			page, err := repo.ListAttemptPage(ctx, prefix, first.StudentID, params, false)
			if err != nil {
				return err
			}
			if page.ReadDocuments > 6 {
				return errors.New("history read bound")
			}
			for _, a := range page.Attempts {
				if seen[a.ID] {
					return errors.New("duplicate attempt")
				}
				seen[a.ID] = true
			}
			if page.NextCursor == "" {
				break
			}
			params, err = pagination.Parse(mapValues(page.NextCursor, 5))
			if err != nil {
				return err
			}
		}
		if len(seen) != 12 {
			return fmt.Errorf("history missing:%d", len(seen))
		}
		return nil
	})
	check("legacy-overview-fails-explicitly-at-cap-v1", func() error {
		_, err := service.GetOverview(ctx, identity, prefix)
		if !errors.Is(err, insights.ErrPaginationRequired) || documents.Load() > 101 {
			return fmt.Errorf("legacy cap:%v/%d", err, documents.Load())
		}
		return nil
	})
	check("small-legacy-overview-keeps-complete-response-v1", func() error {
		classID, uid := prefix+"SM", prefix+"compat-student"
		classRef := client.Collection("teacher_classes").Doc(classID)
		userRef := client.Collection("users").Doc(uid)
		owned = append(owned, classRef, userRef)
		smallClass := class
		smallClass.ID = classID
		smallClass.ClassID = classID
		if _, err := classRef.Set(ctx, smallClass); err != nil {
			return err
		}
		if _, err := userRef.Set(ctx, map[string]any{"roles": []string{"student"}, "student": map[string]any{"class_id": classID, "student_name": "Synthetic compatibility student"}}); err != nil {
			return err
		}
		for index := 0; index < 2; index++ {
			attempt := first
			attempt.ID = fmt.Sprintf("%s-compat-%d", prefix, index)
			attempt.StudentID = uid
			attempt.ClassID = classID
			ref := client.Collection("student_mission_attempts").Doc(attempt.ID)
			owned = append(owned, ref)
			if _, err := ref.Set(ctx, attempt); err != nil {
				return err
			}
		}
		page, err := service.GetOverview(ctx, identity, classID)
		if err != nil {
			return err
		}
		if page.Page != nil || len(page.Students) != 1 || len(page.Students[0].Attempts) != 2 {
			return errors.New("small legacy response changed meaning")
		}
		return nil
	})
	check("history-owner-and-cursor-scope-v1", func() error {
		other := identity
		other.ProviderUserID = "synthetic-other-owner"
		if _, err := service.GetStudentHistory(ctx, other, prefix, first.StudentID, pagination.Params{Limit: 5}); !errors.Is(err, classrooms.ErrClassOwnedByAnotherTeacher) {
			return errors.New("wrong owner accepted")
		}
		page, err := repo.ListAttemptPage(ctx, prefix, first.StudentID, pagination.Params{Limit: 5}, false)
		if err != nil {
			return err
		}
		params, err := pagination.Parse(mapValues(page.NextCursor, 5))
		if err != nil {
			return err
		}
		if _, err := repo.ListAttemptPage(ctx, prefix, "another-student", params, false); !errors.Is(err, insights.ErrInvalidPage) {
			return errors.New("cross-student cursor accepted")
		}
		return nil
	})
	check("partial-rebuild-keeps-legacy-context-and-rejects-skips-v1", func() error {
		page, err := learning.RebuildClassEvidencePage(ctx, prefix, "", 100, true)
		if err != nil {
			return err
		}
		if page.NextCursor == "" {
			return errors.New("fixture should require more rebuild pages")
		}
		if _, err := learning.RebuildClassEvidencePage(ctx, prefix, "foreign-cursor", 100, true); err == nil || !strings.Contains(err.Error(), "cursor") {
			return errors.New("out-of-order rebuild cursor accepted")
		}
		before := documents.Load()
		rows, err := learning.ListLatestFinalizedAttemptsByClassID(ctx, prefix, 40)
		if err != nil {
			return err
		}
		if len(rows) != 40 || documents.Load()-before > 160 {
			return fmt.Errorf("partial projection replaced legacy context:%d/%d", len(rows), documents.Load()-before)
		}
		return nil
	})
	check("bounded-rebuild-and-source-validated-projection-v1", func() error {
		after := ""
		for {
			page, err := learning.RebuildClassEvidencePage(ctx, prefix, after, 100, true)
			if err != nil {
				return err
			}
			if page.Scanned > 100 {
				return errors.New("rebuild scan cap")
			}
			if page.NextCursor == "" {
				break
			}
			after = page.NextCursor
		}

		return nil
	})
	check("bounded-projection-read-v1", func() error {
		rows, err := learning.ListLatestFinalizedAttemptsByClassID(ctx, prefix, 40)
		if err != nil {
			return err
		}
		if len(rows) != 40 || documents.Load() > 80 {
			return fmt.Errorf("projection read bound:%d/%d", len(rows), documents.Load())
		}
		return nil
	})
	check("resumed-source-invalidates-derived-evidence-v1", func() error {
		if _, err := client.Collection("student_mission_attempts").Doc(first.ID).Update(ctx, []firestore.Update{{Path: "trace.finalized", Value: false}, {Path: "evidence_revision", Value: first.EvidenceRevision + 1}}); err != nil {
			return err
		}
		rows, err := learning.ListLatestFinalizedAttemptsByClassID(ctx, prefix, 100)
		if err != nil {
			return err
		}
		for _, a := range rows {
			if a.ID == first.ID {
				return errors.New("stale finalized projection returned")
			}
		}
		return nil
	})
	var projectionCommitGapNS int64
	check("live-finalizer-maintains-private-projection-v1", func() error {
		uid := prefix + "-live"
		id := prefix + "-live-attempt"
		userRef := client.Collection("users").Doc(uid)
		attemptRef := client.Collection("student_mission_attempts").Doc(id)
		projectionRef := client.Collection(students.ClassEvidenceCollection).Doc(prefix).Collection("students").Doc(uid)
		owned = append(owned, userRef, attemptRef, projectionRef)
		user := students.UserDocument{ID: uid}
		user.Student.ClassID = prefix
		if _, err := userRef.Set(ctx, user); err != nil {
			return err
		}
		attempt := students.StudentMissionAttemptDocument{ID: id, StudentID: uid, ClassID: prefix, MissionID: "synthetic-mission", PublishedRevision: 1, Status: "completed", EvidenceSchemaVersion: 1, EvidenceRevision: 1, LastActivityAt: now.Add(time.Minute)}
		attempt.Snapshot.Revision = 1
		if _, err := attemptRef.Set(ctx, attempt); err != nil {
			return err
		}
		claimed, _, ok, err := learning.ClaimStudentFinalization(ctx, students.FinalizationJob(attempt), "synthetic-owner", now, now.Add(time.Minute))
		if err != nil || !ok {
			return fmt.Errorf("claim:%v", err)
		}
		_, err = learning.FinalizeStudentLearningTrace(ctx, students.CaptureFinalizationFence(claimed), generation.LearningTraceBlueprint{Finalized: true, Strengths: []string{"synthetic-live-skill"}}, students.StudentLearningMemory{}, now)
		if err != nil {
			return err
		}
		row, err := projectionRef.Get(ctx)
		if err != nil {
			return err
		}
		source, err := attemptRef.Get(ctx)
		if err != nil {
			return err
		}
		projectionCommitGapNS = row.UpdateTime.Sub(source.UpdateTime).Nanoseconds()
		if projectionCommitGapNS != 0 {
			return errors.New("source and projection did not share the finalization commit")
		}
		var entry students.ClassEvidenceEntry
		if err := row.DataTo(&entry); err != nil {
			return err
		}
		if entry.AttemptID != id || entry.StudentID != uid || entry.ClassID != prefix || entry.EvidenceRevision != 1 {
			return errors.New("finalization did not maintain projection")
		}
		return nil
	})
	if err := cleanup(); err != nil {
		return err
	}
	owned = nil
	passed := 0
	for _, r := range cases {
		if r.Passed {
			passed++
		}
	}
	out := map[string]any{"mode": "firestore-emulator", "project": *project, "fixture_prefix": prefix, "fixture_students": 120, "fixture_attempts": 1440, "projection_commit_gap_ns": projectionCommitGapNS, "projection_measurement_scope": "source and projection Firestore UpdateTime after one finalizer transaction; not worker wait or traffic latency", "real_provider_calls": 0, "cleanup_complete": true, "passed": passed, "total": len(cases), "results": cases}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		return err
	}
	if passed != len(cases) {
		return errors.New("query evaluations failed")
	}
	return nil
}
func mapValues(cursor string, limit int) map[string][]string {
	return map[string][]string{"cursor": {cursor}, "limit": {fmt.Sprint(limit)}}
}
