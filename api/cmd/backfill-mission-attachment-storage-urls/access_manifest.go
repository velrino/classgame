package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"classgame-api/internal/platform/objectstorage"
	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
)

type accessManifest struct {
	Version   int            `json:"version"`
	Project   string         `json:"project"`
	Bucket    string         `json:"bucket"`
	CreatedAt time.Time      `json:"created_at"`
	Objects   []accessObject `json:"objects"`
}
type accessObject struct {
	Problem        string   `json:"problem,omitempty"`
	Path           string   `json:"path"`
	Generation     int64    `json:"generation"`
	Metageneration int64    `json:"metageneration"`
	ACLHash        string   `json:"acl_hash"`
	State          string   `json:"state"`
	Public         []string `json:"public,omitempty"`
}

func aclDigest(rules []storage.ACLRule) string {
	rows := make([]string, 0, len(rules))
	for _, rule := range rules {
		rows = append(rows, string(rule.Entity)+":"+string(rule.Role))
	}
	sort.Strings(rows)
	body, _ := json.Marshal(rows)
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}
func publicEntities(rules []storage.ACLRule) []string {
	public := []string{}
	for _, rule := range rules {
		if rule.Entity == storage.AllUsers || rule.Entity == storage.AllAuthenticatedUsers {
			public = append(public, string(rule.Entity))
		}
	}
	sort.Strings(public)
	return public
}
func publicBucket(ctx context.Context, bucket *storage.BucketHandle) (bool, error) {
	policy, err := bucket.IAM().Policy(ctx)
	if err != nil {
		return false, err
	}
	for _, role := range policy.Roles() {
		for _, member := range policy.Members(role) {
			if member == "allUsers" || member == "allAuthenticatedUsers" {
				return true, nil
			}
		}
	}
	return false, nil
}
func writeAccessManifest(ctx context.Context, client *storage.Client, objects map[string]map[string]bool, project, bucket, output string) error {
	if len(objects) != 1 || objects[bucket] == nil {
		return errors.New("manifest bucket must exactly match all referenced application objects")
	}
	handle := client.Bucket(bucket)
	public, err := publicBucket(ctx, handle)
	if err != nil {
		return errors.New("bucket access could not be verified; no plan written")
	}
	if public {
		return errors.New("public bucket binding needs a separate bucket policy change")
	}
	manifest := accessManifest{Version: 1, Project: project, Bucket: bucket, CreatedAt: time.Now().UTC()}
	paths := make([]string, 0, len(objects[bucket]))
	for path := range objects[bucket] {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	counts := map[string]int{}
	for _, path := range paths {
		if !strings.HasPrefix(path, "classes/") || objectstorage.ValidateAttachmentObjectPath(path) != nil {
			return errors.New("invalid application object path")
		}
		item := accessObject{Path: path}
		attrs, err := handle.Object(path).Attrs(ctx)
		if errors.Is(err, storage.ErrObjectNotExist) {
			item.State = "missing"
		} else if err != nil {
			item.State = "unverified"
			item.Problem = accessProblem("metadata", err)
		} else if strings.TrimSpace(attrs.Metadata["firebaseStorageDownloadTokens"]) != "" {
			item.State = "unverified"
			item.Problem = "download_token_requires_review"
		} else {
			rules, aclErr := handle.Object(path).ACL().List(ctx)
			if aclErr != nil {
				item.State = "unverified"
				item.Problem = accessProblem("acl", aclErr)
			} else {
				item.Generation, item.Metageneration, item.ACLHash = attrs.Generation, attrs.Metageneration, aclDigest(rules)
				item.Public = publicEntities(rules)
				item.State = "private"
				if len(item.Public) > 0 {
					item.State = "public"
				}
			}
		}
		counts[item.State]++
		manifest.Objects = append(manifest.Objects, item)
	}
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		return err
	}
	fmt.Printf("Read-only plan: objects=%d public=%d private=%d missing=%d; local manifest mode=0600\n", len(paths), counts["public"], counts["private"], counts["missing"])
	if counts["unverified"] > 0 {
		return fmt.Errorf("%d objects unverified; manifest is audit-only and cannot be applied", counts["unverified"])
	}
	return nil
}

func accessProblem(step string, err error) string {
	var apiError *googleapi.Error
	if errors.As(err, &apiError) {
		return fmt.Sprintf("%s_http_%d", step, apiError.Code)
	}
	return step + "_unavailable"
}

func applyAccessManifest(ctx context.Context, client *storage.Client, input, project, bucket string) error {
	file, err := os.Open(input)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 1024*1024))
	decoder.DisallowUnknownFields()
	var manifest accessManifest
	if err := decoder.Decode(&manifest); err != nil {
		return errors.New("invalid access manifest")
	}
	if !errors.Is(decoder.Decode(new(any)), io.EOF) || manifest.Version != 1 || manifest.Project != project || manifest.Bucket != bucket || len(manifest.Objects) > 1000 || len(manifest.Objects) == 0 || manifest.CreatedAt.After(time.Now()) || time.Since(manifest.CreatedAt) > 24*time.Hour {
		return errors.New("manifest scope, age, or version is invalid")
	}
	handle := client.Bucket(bucket)
	public, err := publicBucket(ctx, handle)
	if err != nil || public {
		return errors.New("bucket policy is unverified or public; no object updates")
	}
	type change struct {
		item  accessObject
		rules []storage.ACLRule
	}
	changes := []change{}
	seen := map[string]bool{}
	for _, item := range manifest.Objects {
		if seen[item.Path] || !strings.HasPrefix(item.Path, "classes/") || objectstorage.ValidateAttachmentObjectPath(item.Path) != nil {
			return errors.New("invalid or repeated object path")
		}
		seen[item.Path] = true
		if item.State == "missing" {
			if _, err := handle.Object(item.Path).Attrs(ctx); !errors.Is(err, storage.ErrObjectNotExist) {
				return errors.New("previously missing object changed; create a new plan")
			}
			continue
		}
		if item.State != "private" && item.State != "public" {
			return errors.New("manifest contains unverified objects")
		}
		attrs, err := handle.Object(item.Path).Attrs(ctx)
		if err != nil {
			return errors.New("object changed or cannot be verified; create a new plan")
		}
		if strings.TrimSpace(attrs.Metadata["firebaseStorageDownloadTokens"]) != "" {
			return errors.New("download-token access needs a separate review; no object updates")
		}
		rules, err := handle.Object(item.Path).ACL().List(ctx)
		if err != nil {
			return errors.New("object ACL cannot be verified")
		}
		// Safe repeated execution may observe already-private objects.
		if len(publicEntities(rules)) == 0 {
			continue
		}
		if item.State != "public" || attrs.Generation != item.Generation || attrs.Metageneration != item.Metageneration || aclDigest(rules) != item.ACLHash {
			return errors.New("object generation or access changed; create a new plan")
		}
		keep := []storage.ACLRule{}
		for _, rule := range rules {
			if rule.Entity != storage.AllUsers && rule.Entity != storage.AllAuthenticatedUsers {
				keep = append(keep, rule)
			}
		}
		changes = append(changes, change{item, keep})
	}
	for _, change := range changes {
		item := change.item
		// ACLHandle.Delete does not carry object conditions. Replace only the
		// reviewed ACL, preserving every other entry, with atomic generation fences.
		object := handle.Object(item.Path).If(storage.Conditions{GenerationMatch: item.Generation, MetagenerationMatch: item.Metageneration})
		if _, err := object.Update(ctx, storage.ObjectAttrsToUpdate{ACL: change.rules}); err != nil {
			return errors.New("access update stopped; completed objects stay private, rerun read-only audit")
		}
		rules, err := handle.Object(item.Path).ACL().List(ctx)
		if err != nil || len(publicEntities(rules)) != 0 {
			return errors.New("private access verification failed; stop and inspect")
		}
	}
	fmt.Printf("Access changes complete: objects_privatized=%d objects_deleted=0 firestore_writes=0\n", len(changes))
	return nil
}
