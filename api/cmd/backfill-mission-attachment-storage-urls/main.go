// This legacy entrypoint audits by default. Explicit reviewed manifests may
// remove public access; it never rewrites URLs or grants public storage access.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"github.com/joho/godotenv"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func main() {
	_ = godotenv.Load()
	project := flag.String("project", os.Getenv("FIREBASE_PROJECT_ID"), "Firebase project to audit (read only)")
	credentials := flag.String("credentials-file", os.Getenv("FIREBASE_CREDENTIALS_FILE"), "Local Google credentials file")
	inspectACL := flag.Bool("inspect-acl", false, "Read object ACLs and bucket IAM policies")
	manifestOut := flag.String("manifest-out", "", "Write a private generation-pinned review manifest; no remote writes")
	applyManifest := flag.String("apply-manifest", "", "Apply only the public-reader removals in this reviewed manifest")
	revokePublic := flag.Bool("revoke-public", false, "Explicitly allow reviewed public-reader removal; never delete objects")
	confirmProject := flag.String("confirm-project", "", "Required exact project confirmation for access changes")
	bucket := flag.String("bucket", os.Getenv("FIREBASE_STORAGE_BUCKET"), "Only this explicit application bucket may appear in a manifest")
	flag.Parse()
	if strings.TrimSpace(*project) == "" {
		log.Fatal("project is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	options := []option.ClientOption{}
	if *credentials != "" {
		options = append(options, option.WithCredentialsFile(*credentials))
	}
	if *applyManifest != "" {
		if !*revokePublic || *confirmProject != *project || *bucket == "" || *manifestOut != "" {
			log.Fatal("apply requires --revoke-public, --confirm-project and the application --bucket")
		}
		storageClient, err := storage.NewClient(ctx, options...)
		if err != nil {
			log.Fatal("initialize private access client: ", err)
		}
		defer storageClient.Close()
		if err := applyAccessManifest(ctx, storageClient, *applyManifest, *project, *bucket); err != nil {
			log.Fatal(err)
		}
		return
	}
	if *revokePublic {
		log.Fatal("--revoke-public requires a reviewed --apply-manifest")
	}
	client, err := firestore.NewClient(ctx, *project, options...)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	iter := client.CollectionGroup("missions").Documents(ctx)
	defer iter.Stop()
	objects := map[string]map[string]bool{}
	legacyURLs, documents := 0, 0
	for {
		snapshot, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		documents++
		data := snapshot.Data()
		for _, view := range []map[string]any{data, object(data["draft"]), object(data["published"]), object(data["snapshot"])} {
			attachments, _ := view["attachments"].([]any)
			for _, value := range attachments {
				attachment := object(value)
				if strings.HasPrefix(text(attachment["cdn_url"]), "https://storage.googleapis.com/") {
					legacyURLs++
				}
				stored := object(attachment["storage"])
				bucket, path := text(stored["bucket"]), text(stored["object_path"])
				if gs := text(stored["gs_uri"]); (bucket == "" || path == "") && strings.HasPrefix(gs, "gs://") {
					bucket, path, _ = strings.Cut(strings.TrimPrefix(gs, "gs://"), "/")
				}
				if bucket == "" || !strings.HasPrefix(path, "classes/") {
					continue
				}
				if objects[bucket] == nil {
					objects[bucket] = map[string]bool{}
				}
				objects[bucket][path] = true
			}
		}
	}
	count := 0
	for _, paths := range objects {
		count += len(paths)
	}
	fmt.Printf("Read-only audit: documents=%d stored_objects=%d legacy_public_url_references=%d\n", documents, count, legacyURLs)
	if *manifestOut != "" {
		if *bucket == "" {
			log.Fatal("manifest requires the explicit application --bucket")
		}
		storageClient, err := storage.NewClient(ctx, options...)
		if err != nil {
			log.Fatal(err)
		}
		defer storageClient.Close()
		if err := writeAccessManifest(ctx, storageClient, objects, *project, *bucket, *manifestOut); err != nil {
			log.Fatal(err)
		}
		return
	}
	if !*inspectACL {
		fmt.Println("ACL/IAM was not inspected. Use --inspect-acl for a read-only access audit.")
		return
	}
	storageClient, err := storage.NewClient(ctx, options...)
	if err != nil {
		log.Fatal(err)
	}
	defer storageClient.Close()
	publicObjects, publicBuckets, unverified := 0, 0, 0
	for bucket, paths := range objects {
		handle := storageClient.Bucket(bucket)
		policy, err := handle.IAM().Policy(ctx)
		if err != nil {
			unverified++
		} else {
			public := false
			for _, role := range policy.Roles() {
				for _, member := range policy.Members(role) {
					if member == "allUsers" || member == "allAuthenticatedUsers" {
						public = true
					}
				}
			}
			if public {
				publicBuckets++
			}
		}
		for path := range paths {
			rules, err := handle.Object(path).ACL().List(ctx)
			if err != nil {
				unverified++
				continue
			}
			for _, rule := range rules {
				if rule.Entity == storage.AllUsers || rule.Entity == storage.AllAuthenticatedUsers {
					publicObjects++
					break
				}
			}
		}
	}
	fmt.Printf("Read-only access audit: public_object_acls=%d public_bucket_bindings=%d unverified_checks=%d\n", publicObjects, publicBuckets, unverified)
	fmt.Println("Existing public ACLs and bucket IAM require a separately approved operational remediation. No writes were performed.")
}
func object(value any) map[string]any { result, _ := value.(map[string]any); return result }
func text(value any) string           { result, _ := value.(string); return strings.TrimSpace(result) }
