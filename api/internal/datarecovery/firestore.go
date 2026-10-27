package datarecovery

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	firev1 "cloud.google.com/go/firestore/apiv1"
	"cloud.google.com/go/firestore/apiv1/firestorepb"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Store struct {
	Project  string
	Client   *firestore.Client
	raw      *firev1.Client
	emulator bool
}

func Open(ctx context.Context, project string) (*Store, error) {
	if !regexp.MustCompile(`^[a-z][a-z0-9-]{4,62}$`).MatchString(project) {
		return nil, ErrScope
	}
	var opts []option.ClientOption
	local := false
	if endpoint := os.Getenv("FIRESTORE_EMULATOR_HOST"); endpoint != "" {
		if e := RequireLocal(project); e != nil {
			return nil, e
		}
		local = true
		opts = []option.ClientOption{option.WithEndpoint(endpoint), option.WithoutAuthentication(), option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials()))}
	}
	client, e := firestore.NewClient(ctx, project, opts...)
	if e != nil {
		return nil, e
	}
	raw, e := firev1.NewClient(ctx, opts...)
	if e != nil {
		client.Close()
		return nil, e
	}
	return &Store{Project: project, Client: client, raw: raw, emulator: local}, nil
}
func (s *Store) Close() { s.Client.Close(); s.raw.Close() }
func (s *Store) document(path string) string {
	return "projects/" + s.Project + "/databases/(default)/documents/" + path
}
func (s *Store) Export(ctx context.Context, scope Scope) (Bundle, error) {
	b := Bundle{Version: 1, Project: s.Project, Scope: scope, CreatedAt: time.Now().UTC()}
	if !scope.Valid() {
		return b, ErrScope
	}
	owner := make([]byte, 16)
	if _, e := rand.Read(owner); e != nil {
		return b, e
	}
	b.Owner = hex.EncodeToString(owner)
	paths := map[string]bool{}
	encodedBytes := 0
	add := func(path string) error {
		if paths[path] {
			return nil
		}
		d, e := s.raw.GetDocument(ctx, &firestorepb.GetDocumentRequest{Name: s.document(path)})
		if status.Code(e) == codes.NotFound {
			return nil
		}
		if e != nil {
			return e
		}
		if len(b.Records) >= MaxDocuments {
			return fmt.Errorf("class backup exceeds %d documents; use a reviewed managed backup", MaxDocuments)
		}
		raw, e := protojson.Marshal(d)
		if e != nil {
			return e
		}
		if len(raw) > MaxBundleBytes-encodedBytes {
			return errors.New("class backup exceeds the byte limit; use a reviewed managed backup")
		}
		encodedBytes += len(raw)
		paths[path] = true
		b.Records = append(b.Records, Record{Path: path, Document: raw})
		return nil
	}
	if e := add("teacher_classes/" + scope.Class); e != nil {
		return b, e
	}
	if !paths["teacher_classes/"+scope.Class] {
		return b, ErrScope
	}
	list := func(query firestore.Query) error {
		rows, e := query.Select(firestore.DocumentID).Limit(MaxDocuments + 1).Documents(ctx).GetAll()
		if e != nil {
			return e
		}
		if len(rows) > MaxDocuments {
			return errors.New("backup page limit reached; no partial bundle is accepted")
		}
		for _, row := range rows {
			if e := add(row.Ref.Path[strings.Index(row.Ref.Path, "/documents/")+11:]); e != nil {
				return e
			}
		}
		return nil
	}
	for _, collection := range []string{"missions", "mission_codes", "student_mission_attempts"} {
		if e := list(s.Client.Collection(collection).Where("class_id", "==", scope.Class)); e != nil {
			return b, e
		}
	}
	roots := append([]Record(nil), b.Records...)
	for _, record := range roots {
		parts := strings.Split(record.Path, "/")
		if parts[0] == "missions" {
			if e := list(s.Client.Doc(record.Path).Collection("published_revisions").Query); e != nil {
				return b, e
			}
		}
		if parts[0] == "student_mission_attempts" {
			if e := list(s.Client.Doc(record.Path).Collection("tutor_events").Query); e != nil {
				return b, e
			}
			var d firestorepb.Document
			protojson.Unmarshal(record.Document, &d)
			if student := text(&d, "student_id"); student != "" {
				if e := add("student_mission_completions/" + student + "/missions/" + parts[1]); e != nil {
					return b, e
				}
			}
		}
	}
	for _, collection := range []string{"student_class_missions/" + scope.Class + "/missions", "class_learning_students/" + scope.Class + "/students"} {
		if e := list(s.Client.Collection(collection).Query); e != nil {
			return b, e
		}
	}
	if e := add("class_learning_students/" + scope.Class); e != nil {
		return b, e
	}
	if e := add("class_learning_summaries/" + scope.Class); e != nil {
		return b, e
	}
	if e := add("class_learning_refreshes/" + scope.Class); e != nil {
		return b, e
	}
	sort.Slice(b.Records, func(i, j int) bool { return b.Records[i].Path < b.Records[j].Path })
	b.SHA256 = b.Digest()
	return b, b.Validate()
}
func (s *Store) holdRef(scope Scope) *firestore.DocumentRef {
	raw := sha256.Sum256([]byte(scope.Organization + "/" + scope.District + "/" + scope.School + "/" + scope.Class))
	return s.Client.Collection("data_recovery_holds").Doc(hex.EncodeToString(raw[:]))
}

type hold struct {
	Owner  string    `firestore:"owner"`
	Digest string    `firestore:"digest"`
	State  string    `firestore:"state"`
	Until  time.Time `firestore:"until"`
}

func (s *Store) AcquireHold(ctx context.Context, b Bundle) error {
	if !s.emulator {
		return errors.New("destructive recovery is restricted to a demo loopback emulator")
	}
	if e := b.Validate(); e != nil {
		return e
	}
	if b.Project != s.Project {
		return ErrScope
	}
	return s.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		ref := s.holdRef(b.Scope)
		snapshot, e := tx.Get(ref)
		now := time.Now().UTC()
		var previous hold
		if e == nil {
			if e = snapshot.DataTo(&previous); e != nil {
				return e
			}
			now = snapshot.ReadTime
			if previous.Owner != b.Owner && (previous.State == "deleted" || previous.Until.After(now)) {
				return ErrConflict
			}
			if previous.Owner == b.Owner && previous.Digest != b.SHA256 {
				return ErrConflict
			}
		} else if status.Code(e) != codes.NotFound {
			return e
		} else {
			class, classErr := tx.Get(s.Client.Doc("teacher_classes/" + b.Scope.Class))
			if classErr != nil {
				return classErr
			}
			fields := class.Data()
			if fields["organization_id"] != b.Scope.Organization || fields["district_id"] != b.Scope.District || fields["school_id"] != b.Scope.School {
				return ErrScope
			}
			now = class.ReadTime
		}
		state := "held"
		if previous.Owner == b.Owner && previous.Digest == b.SHA256 {
			state = previous.State
		}
		return tx.Set(ref, hold{Owner: b.Owner, Digest: b.SHA256, State: state, Until: now.Add(15 * time.Minute)})
	})
}
func (s *Store) apply(ctx context.Context, b Bundle, restoring bool) (bool, error) {
	if !s.emulator {
		return false, errors.New("destructive recovery is restricted to a demo loopback emulator")
	}
	if e := b.Validate(); e != nil {
		return false, e
	}
	if b.Project != s.Project {
		return false, ErrScope
	}
	ref := s.holdRef(b.Scope)
	snap, e := ref.Get(ctx)
	if e != nil {
		return false, e
	}
	var current hold
	if e = snap.DataTo(&current); e != nil {
		return false, e
	}
	if current.Owner != b.Owner || current.Digest != b.SHA256 || !current.Until.After(snap.ReadTime) {
		return false, ErrConflict
	}
	if (restoring && current.State == "restored") || (!restoring && current.State == "deleted") {
		return false, nil
	}
	if (!restoring && current.State != "held") || (restoring && current.State != "deleted") {
		return false, ErrConflict
	}
	writes := []*firestorepb.Write{}
	refreshPath := "class_learning_refreshes/" + b.Scope.Class
	hasRefresh := false
	for _, record := range b.Records {
		hasRefresh = hasRefresh || record.Path == refreshPath
		var d firestorepb.Document
		if e = protojson.Unmarshal(record.Document, &d); e != nil {
			return false, e
		}
		if restoring {
			d.CreateTime = nil
			d.UpdateTime = nil
			resetClaims(record.Path, &d)
			writes = append(writes, &firestorepb.Write{Operation: &firestorepb.Write_Update{Update: &d}, CurrentDocument: &firestorepb.Precondition{ConditionType: &firestorepb.Precondition_Exists{Exists: false}}})
		} else {
			writes = append(writes, &firestorepb.Write{Operation: &firestorepb.Write_Delete{Delete: d.Name}, CurrentDocument: &firestorepb.Precondition{ConditionType: &firestorepb.Precondition_UpdateTime{UpdateTime: d.UpdateTime}}})
		}
	}
	if !hasRefresh {
		// An older bundle must not leave an untracked current intent behind.
		// The absent-document precondition is checked with the recovery batch.
		writes = append(writes, &firestorepb.Write{Operation: &firestorepb.Write_Delete{Delete: s.document(refreshPath)}, CurrentDocument: &firestorepb.Precondition{ConditionType: &firestorepb.Precondition_Exists{Exists: false}}})
	}
	next := "deleted"
	if restoring {
		next = "restored"
	}
	fields := map[string]*firestorepb.Value{"owner": str(b.Owner), "digest": str(b.SHA256), "state": str(next), "until": {ValueType: &firestorepb.Value_TimestampValue{TimestampValue: timestamppb.New(current.Until)}}}
	writes = append(writes, &firestorepb.Write{Operation: &firestorepb.Write_Update{Update: &firestorepb.Document{Name: s.document("data_recovery_holds/" + ref.ID), Fields: fields}}, CurrentDocument: &firestorepb.Precondition{ConditionType: &firestorepb.Precondition_UpdateTime{UpdateTime: timestamppb.New(snap.UpdateTime)}}})
	_, e = s.raw.Commit(ctx, &firestorepb.CommitRequest{Database: "projects/" + s.Project + "/databases/(default)", Writes: writes})
	if e != nil {
		return false, fmt.Errorf("atomic recovery precondition failed: %w", e)
	}
	return true, nil
}
func (s *Store) Delete(ctx context.Context, b Bundle) (bool, error)  { return s.apply(ctx, b, false) }
func (s *Store) Restore(ctx context.Context, b Bundle) (bool, error) { return s.apply(ctx, b, true) }
func str(value string) *firestorepb.Value {
	return &firestorepb.Value{ValueType: &firestorepb.Value_StringValue{StringValue: value}}
}
func resetClaims(path string, d *firestorepb.Document) {
	parts := strings.Split(path, "/")
	zero := &firestorepb.Value{ValueType: &firestorepb.Value_TimestampValue{TimestampValue: timestamppb.New(time.Time{})}}
	if len(parts) == 2 && parts[0] == "student_mission_attempts" {
		d.Fields["finalization_token"] = str("")
		d.Fields["finalization_lease_until"] = zero
		generation := d.Fields["finalization_dispatch_generation"].GetIntegerValue()
		d.Fields["finalization_dispatch_generation"] = &firestorepb.Value{ValueType: &firestorepb.Value_IntegerValue{IntegerValue: generation + 1}}
		finalized := d.Fields["trace"].GetMapValue().GetFields()["finalized"].GetBooleanValue()
		if !finalized && (text(d, "status") == "completed" || text(d, "status") == "abandoned") {
			d.Fields["trace_state"] = str("pending")
		}
	}
	if len(parts) == 2 && parts[0] == "class_learning_refreshes" {
		d.Fields["owner_token"] = str("")
		d.Fields["lease_until"] = zero
		for _, key := range []string{"generation", "dispatch_generation"} {
			generation := d.Fields[key].GetIntegerValue()
			d.Fields[key] = &firestorepb.Value{ValueType: &firestorepb.Value_IntegerValue{IntegerValue: generation + 1}}
		}
	}
	if len(parts) == 4 && parts[0] == "student_mission_attempts" && parts[2] == "tutor_events" && text(d, "state") == "processing" {
		d.Fields["state"] = str("failed")
		d.Fields["owner_token"] = str("")
		d.Fields["lease_until"] = zero
	}
}

// ReleaseHold aborts a local plan that has not deleted data.
func (s *Store) ReleaseHold(ctx context.Context, b Bundle) error {
	if !s.emulator {
		return errors.New("hold writes require a demo loopback emulator")
	}
	if e := b.Validate(); e != nil {
		return e
	}
	if b.Project != s.Project {
		return ErrScope
	}
	ref := s.holdRef(b.Scope)
	snap, e := ref.Get(ctx)
	if e != nil {
		return e
	}
	var current hold
	if e = snap.DataTo(&current); e != nil {
		return e
	}
	if current.Owner != b.Owner || current.Digest != b.SHA256 || current.State != "held" {
		return ErrConflict
	}
	_, e = ref.Delete(ctx, firestore.LastUpdateTime(snap.UpdateTime))
	return e
}

// RequireLocal rejects hosted writes before credentials or clients are loaded.
func RequireLocal(project string) error {
	endpoint := os.Getenv("FIRESTORE_EMULATOR_HOST")
	host, port, e := net.SplitHostPort(endpoint)
	number, parseErr := strconv.Atoi(port)
	if e != nil || parseErr != nil || number < 1 || number > 65535 || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() || !strings.HasPrefix(project, "demo-") {
		return errors.New("local recovery requires a numeric loopback emulator and demo- project")
	}
	return nil
}
