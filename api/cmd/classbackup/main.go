// Command classbackup plans private class retention and exports encrypted bundles.
// Destructive modes only work on a demo project in a numeric-loopback emulator.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"classgame-api/internal/datarecovery"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	flags := flag.NewFlagSet("classbackup", flag.ContinueOnError)
	mode := flags.String("mode", "plan", "plan, export, delete-local, restore-local, or release-local")
	project := flags.String("project", "", "explicit project ID")
	org := flags.String("organization", "", "organization")
	district := flags.String("district", "", "district")
	school := flags.String("school", "", "school")
	class := flags.String("class", "", "class ID")
	path := flags.String("file", "", "private encrypted file outside Git")
	key := flags.String("key-file", "", "owner-only 32-byte key outside Git")
	apply := flags.Bool("apply-local", false, "explicitly allow guarded emulator writes")
	archived := flags.String("archived-at", "", "teacher-approved archive timestamp (RFC3339), for plan only")
	reviewDays := flags.Int("archive-days", 90, "days after approved archive before a manual review")
	if e := flags.Parse(os.Args[1:]); e != nil {
		return e
	}
	if flags.NArg() != 0 || *project == "" || *reviewDays < 1 || *reviewDays > 3650 {
		return errors.New("explicit project and valid bounded arguments are required")
	}
	scope := datarecovery.Scope{Organization: *org, District: *district, School: *school, Class: *class}
	if !scope.Valid() {
		return datarecovery.ErrScope
	}
	destructive := *mode == "delete-local" || *mode == "restore-local" || *mode == "release-local"
	if *mode != "plan" && *mode != "export" && !destructive {
		return errors.New("unknown backup mode")
	}
	if destructive != *apply {
		return errors.New("apply-local is required only for local delete/restore")
	}
	if *mode != "plan" && (*path == "" || *key == "") {
		return errors.New("encrypted file and key-file are required")
	}
	if *mode != "plan" && *archived != "" {
		return errors.New("archived-at only applies to the retention plan")
	}
	if destructive {
		if e := datarecovery.RequireLocal(*project); e != nil {
			return e
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	store, e := datarecovery.Open(ctx, *project)
	if e != nil {
		return e
	}
	defer store.Close()
	if destructive {
		b, e := datarecovery.ReadEncrypted(*path, *key)
		if e != nil {
			return e
		}
		if b.Scope != scope || b.Project != *project {
			return datarecovery.ErrScope
		}
		if *mode == "release-local" {
			if e := store.ReleaseHold(ctx, b); e != nil {
				return e
			}
			return json.NewEncoder(os.Stdout).Encode(map[string]any{"mode": *mode, "released": true})
		}
		if e = store.AcquireHold(ctx, b); e != nil {
			return e
		}
		var changed bool
		if *mode == "delete-local" {
			changed, e = store.Delete(ctx, b)
		} else {
			changed, e = store.Restore(ctx, b)
		}
		if e != nil {
			return e
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"mode": *mode, "changed": changed, "sha256": b.SHA256, "documents": len(b.Records)})
	}
	bundle, e := store.Export(ctx, scope)
	if e != nil {
		return e
	}
	if *mode == "export" {
		if e = datarecovery.WriteEncrypted(*path, *key, bundle); e != nil {
			return e
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"mode": "export", "documents": len(bundle.Records), "sha256": bundle.SHA256, "encrypted": true, "database_writes": 0})
	}
	eligible := false
	reason := "manual archive approval is required"
	if *archived != "" {
		when, e := time.Parse(time.RFC3339, *archived)
		if e != nil || when.After(time.Now()) {
			return errors.New("archived-at must be a past RFC3339 timestamp")
		}
		eligible = time.Since(when) >= time.Duration(*reviewDays)*24*time.Hour
		reason = "archive review date has not arrived"
		if eligible {
			reason = "ready for owner review; this does not authorize deletion"
		}
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"mode": "plan", "scope": scope, "documents": len(bundle.Records), "archive_review_days": *reviewDays, "eligible_for_review": eligible, "reason": reason, "database_writes": 0, "global_identities_included": false, "stored_objects_included": false})
}
