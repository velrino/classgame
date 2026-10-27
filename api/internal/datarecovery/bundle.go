// Package datarecovery plans bounded class backups and guarded local recovery.
// Class bundles never own global users, auth accounts, or the tenant hierarchy.
package datarecovery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/firestore/apiv1/firestorepb"
	"google.golang.org/protobuf/encoding/protojson"
)

const MaxDocuments = 200
const MaxBundleBytes = 8 * 1024 * 1024

var ErrScope = errors.New("backup contains invalid or unproven class/tenant scope")
var ErrConflict = errors.New("recovery scope is busy or source data changed")

type Scope struct {
	Organization string `json:"organization"`
	District     string `json:"district"`
	School       string `json:"school"`
	Class        string `json:"class"`
}

func (s Scope) Valid() bool {
	for _, v := range []string{s.Organization, s.District, s.School, s.Class} {
		if !regexp.MustCompile(`^[A-Za-z0-9_-]{1,120}$`).MatchString(v) {
			return false
		}
	}
	return true
}

type Record struct {
	Path     string          `json:"path"`
	Document json.RawMessage `json:"document"`
}
type Bundle struct {
	Version   int       `json:"version"`
	Project   string    `json:"project"`
	Scope     Scope     `json:"scope"`
	CreatedAt time.Time `json:"created_at"`
	Owner     string    `json:"recovery_owner"`
	Records   []Record  `json:"records"`
	SHA256    string    `json:"sha256"`
}

func (b Bundle) Digest() string {
	b.SHA256 = ""
	raw, _ := json.Marshal(b)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}
func (b Bundle) Validate() error {
	if b.Version != 1 || !b.Scope.Valid() || b.Project == "" || len(b.Owner) != 32 || b.CreatedAt.IsZero() || len(b.Records) == 0 || len(b.Records) > MaxDocuments || b.SHA256 != b.Digest() {
		return ErrScope
	}
	raw, e := json.Marshal(b)
	if e != nil || len(raw) > MaxBundleBytes {
		return ErrScope
	}
	documents := map[string]*firestorepb.Document{}
	prefix := "projects/" + b.Project + "/databases/(default)/documents/"
	for _, r := range b.Records {
		var d firestorepb.Document
		if protojson.Unmarshal(r.Document, &d) != nil || d.Name != prefix+r.Path || documents[r.Path] != nil || d.UpdateTime == nil {
			return ErrScope
		}
		documents[r.Path] = &d
	}
	class := documents["teacher_classes/"+b.Scope.Class]
	if class == nil || !matchesTenant(class, b.Scope) {
		return ErrScope
	}
	for path, d := range documents {
		p := strings.Split(path, "/")
		if len(p) == 2 && p[0] == "student_mission_attempts" {
			if value := d.Fields["finalization_dispatch_generation"]; value != nil {
				if _, ok := value.ValueType.(*firestorepb.Value_IntegerValue); !ok || value.GetIntegerValue() < 0 || value.GetIntegerValue() == math.MaxInt64 {
					return ErrScope
				}
			}
		}
		valid := false
		if len(p) == 2 {
			switch p[0] {
			case "teacher_classes":
				valid = p[1] == b.Scope.Class && matchesTenant(d, b.Scope)
			case "missions", "mission_codes", "student_mission_attempts":
				valid = text(d, "class_id") == b.Scope.Class && matchesTenant(d, b.Scope)
			case "class_learning_summaries", "class_learning_students":
				valid = p[1] == b.Scope.Class && text(d, "class_id") == b.Scope.Class && matchesTenant(d, b.Scope)
			}
		}
		if len(p) == 4 {
			switch p[0] + "/" + p[2] {
			case "missions/published_revisions":
				parent := documents["missions/"+p[1]]
				valid = parent != nil && text(d, "class_id") == b.Scope.Class && matchesTenant(d, b.Scope)
			case "student_class_missions/missions":
				valid = p[1] == b.Scope.Class && documents["missions/"+p[3]] != nil && text(d, "class_id") == b.Scope.Class && matchesTenant(d, b.Scope)
			case "student_mission_attempts/tutor_events":
				parent := documents["student_mission_attempts/"+p[1]]
				valid = parent != nil && text(d, "student_id") == text(parent, "student_id")
			case "student_mission_completions/missions":
				parent := documents["student_mission_attempts/"+p[3]]
				valid = parent != nil && text(parent, "student_id") == p[1] && text(d, "student_id") == p[1] && text(d, "class_id") == b.Scope.Class
			case "class_learning_students/students":
				parent := documents["student_mission_attempts/"+text(d, "attempt_id")]
				valid = p[1] == b.Scope.Class && text(d, "student_id") == p[3] && text(d, "class_id") == b.Scope.Class && matchesTenant(d, b.Scope) && parent != nil && text(parent, "student_id") == p[3]
			}
		}
		if !valid {
			return ErrScope
		}
	}
	return nil
}
func text(d *firestorepb.Document, key string) string {
	if d == nil {
		return ""
	}
	return d.Fields[key].GetStringValue()
}
func matchesTenant(d *firestorepb.Document, s Scope) bool {
	return text(d, "organization_id") == s.Organization && text(d, "district_id") == s.District && text(d, "school_id") == s.School
}
