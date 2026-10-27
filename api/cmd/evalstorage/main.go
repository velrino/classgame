// Command evalstorage checks the production copier through a scripted SDK transport.
// It has no socket transport, application configuration, or cloud credentials.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"classgame-api/internal/modules/attachments"
)

const bucketName = "demo-classgame-attachments"
const sdkHost = "127.0.0.1:1"

type scriptedStorage struct {
	uniform, publicIAM, bucketDenied, iamDenied bool
	existing, publicObject, omittedACL, token   bool
	publicAfterWrite, omitACLAfterWrite         bool
	defaultPublic, allAuthenticated, mismatch   bool
	stored                                      map[string]any
	uploads, downloads, unexpected              int
	predefinedACL, cacheControl                 string
}

func response(request *http.Request, status int, value any) *http.Response {
	body, _ := json.Marshal(value)
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(bytes.NewReader(body)), Request: request}
}

func (s *scriptedStorage) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Host != sdkHost {
		s.unexpected++
		return nil, errors.New("SDK attempted an endpoint outside the scripted transport")
	}
	p := request.URL.Path
	if request.Method == http.MethodGet && p == "/b/"+bucketName {
		if s.bucketDenied {
			return response(request, 403, map[string]any{"error": map[string]any{"code": 403}}), nil
		}
		return response(request, 200, map[string]any{"name": bucketName, "iamConfiguration": map[string]any{"uniformBucketLevelAccess": map[string]any{"enabled": s.uniform}}}), nil
	}
	if request.Method == http.MethodGet && p == "/b/"+bucketName+"/iam" {
		if s.iamDenied {
			return response(request, 403, map[string]any{"error": map[string]any{"code": 403}}), nil
		}
		member := "serviceAccount:synthetic@example.invalid"
		if s.publicIAM {
			member = "allUsers"
			if s.allAuthenticated {
				member = "allAuthenticatedUsers"
			}
		}
		return response(request, 200, map[string]any{"version": 3, "bindings": []any{map[string]any{"role": "roles/storage.objectViewer", "members": []string{member}}}}), nil
	}
	if request.Method == http.MethodGet && strings.HasPrefix(p, "/b/"+bucketName+"/o/") {
		if !s.existing && s.stored == nil {
			return response(request, 404, map[string]any{"error": map[string]any{"code": 404}}), nil
		}
		value := s.stored
		if value == nil {
			value = map[string]any{"name": strings.TrimPrefix(p, "/b/"+bucketName+"/o/"), "metadata": fixtureMetadata(), "size": "7", "contentType": "text/plain", "generation": "1"}
		}
		if s.token {
			value["metadata"].(map[string]any)["firebaseStorageDownloadTokens"] = "synthetic-token"
		}
		public, omitted := s.publicObject, s.omittedACL
		if s.uploads > 0 {
			public, omitted = s.publicAfterWrite || s.defaultPublic && s.predefinedACL != "private", s.omitACLAfterWrite
		}
		if !s.uniform && !omitted {
			entity := "user-synthetic-owner"
			if public {
				entity = "allUsers"
				if s.allAuthenticated {
					entity = "allAuthenticatedUsers"
				}
			}
			value["acl"] = []any{map[string]any{"entity": entity, "role": "OWNER"}}
		} else {
			delete(value, "acl")
		}
		return response(request, 200, value), nil
	}
	if request.Method == http.MethodPost && strings.Contains(p, "/b/"+bucketName+"/o") && request.URL.Query().Get("uploadType") == "multipart" {
		_, parameters, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
		if err != nil {
			return nil, err
		}
		reader := multipart.NewReader(request.Body, parameters["boundary"])
		metadata, err := reader.NextPart()
		if err != nil {
			return nil, err
		}
		if err := json.NewDecoder(metadata).Decode(&s.stored); err != nil {
			return nil, err
		}
		media, err := reader.NextPart()
		if err != nil {
			return nil, err
		}
		size, err := io.Copy(io.Discard, media)
		if err != nil {
			return nil, err
		}
		s.stored["size"] = fmt.Sprint(size)
		s.stored["generation"] = "1"
		s.uploads++
		s.predefinedACL = request.URL.Query().Get("predefinedAcl")
		s.cacheControl, _ = s.stored["cacheControl"].(string)
		if request.URL.Query().Get("ifGenerationMatch") != "0" {
			return nil, errors.New("upload lost the create-only precondition")
		}
		return response(request, 200, s.stored), nil
	}
	s.unexpected++
	return nil, fmt.Errorf("unexpected scripted SDK operation: %s %s", request.Method, p)
}

type sourceTransport struct{ storage *scriptedStorage }

func (t sourceTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Scheme != "https" || request.URL.Host != "ucarecdn.com" || request.Method != http.MethodGet {
		t.storage.unexpected++
		return nil, errors.New("unexpected source request")
	}
	t.storage.downloads++
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/plain"}}, Body: io.NopCloser(strings.NewReader("fixture")), ContentLength: 7, Request: request}, nil
}

func fixtureMetadata() map[string]any {
	return map[string]any{"class_id": "SYNTHETIC", "mission_id": "synthetic-mission", "revision": "1", "source_key": "synthetic-source", "uploadcare_uuid": "synthetic-source"}
}

type result struct {
	ID     string `json:"id"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	flags := flag.NewFlagSet("evalstorage", flag.ContinueOnError)
	optIn := flags.Bool("run", false, "run synthetic SDK transport checks; no sockets or cloud calls")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	if !*optIn || flags.NArg() != 0 {
		return errors.New("require -run with no positional arguments")
	}
	cases := []struct {
		id        string
		state     scriptedStorage
		wantError string
		writes    int
	}{
		{"fine-grained-overrides-public-default-acl-v1", scriptedStorage{defaultPublic: true}, "", 1},
		{"uniform-private-without-object-acl-v1", scriptedStorage{uniform: true}, "", 1},
		{"public-bucket-rejected-before-download-v1", scriptedStorage{publicIAM: true}, "bucket has public access", 0},
		{"authenticated-public-bucket-rejected-v1", scriptedStorage{publicIAM: true, allAuthenticated: true}, "bucket has public access", 0},
		{"uniform-public-bucket-rejected-v1", scriptedStorage{uniform: true, publicIAM: true}, "bucket has public access", 0},
		{"bucket-metadata-unverified-v1", scriptedStorage{bucketDenied: true}, "bucket metadata could not be verified", 0},
		{"bucket-iam-unverified-v1", scriptedStorage{iamDenied: true}, "bucket access could not be verified", 0},
		{"existing-private-idempotent-v1", scriptedStorage{existing: true}, "", 0},
		{"existing-uniform-private-v1", scriptedStorage{existing: true, uniform: true}, "", 0},
		{"existing-public-requires-review-v1", scriptedStorage{existing: true, publicObject: true}, "still has public access", 0},
		{"existing-authenticated-public-requires-review-v1", scriptedStorage{existing: true, publicObject: true, allAuthenticated: true}, "still has public access", 0},
		{"existing-acl-unverified-v1", scriptedStorage{existing: true, omittedACL: true}, "access could not be verified", 0},
		{"existing-download-token-requires-review-v1", scriptedStorage{existing: true, token: true}, "download-token access", 0},
		{"uniform-download-token-requires-review-v1", scriptedStorage{existing: true, uniform: true, token: true}, "download-token access", 0},
		{"bucket-handle-name-mismatch-v1", scriptedStorage{mismatch: true}, "does not match", 0},
		{"post-upload-public-never-ready-v1", scriptedStorage{publicAfterWrite: true}, "still has public access", 1},
		{"post-upload-unverified-never-ready-v1", scriptedStorage{omitACLAfterWrite: true}, "access could not be verified", 1},
	}
	results := []result{}
	for _, c := range cases {
		state := c.state
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		client, err := storage.NewClient(ctx, option.WithEndpoint("http://"+sdkHost), option.WithHTTPClient(&http.Client{Transport: &state}), option.WithoutAuthentication())
		if err != nil {
			cancel()
			return err
		}
		configuredName := bucketName
		if state.mismatch {
			configuredName = "other-synthetic-bucket"
		}
		copier, err := attachments.NewStorageCopier(attachments.StorageCopierOptions{BucketName: configuredName, Bucket: client.Bucket(bucketName), HTTPClient: &http.Client{Transport: sourceTransport{&state}}})
		if err == nil {
			_, err = copier.Copy(ctx, attachments.CopyInput{ClassID: "SYNTHETIC", MissionID: "synthetic-mission", Revision: 1, UploadcareUUID: "synthetic-source", SourceURL: "https://ucarecdn.com/00000000-0000-4000-8000-000000000000/", FileName: "worksheet.txt", MIMEType: "text/plain"})
		}
		_ = client.Close()
		cancel()
		r := result{ID: c.id, Passed: true}
		if c.wantError == "" && err != nil || c.wantError != "" && (err == nil || !strings.Contains(err.Error(), c.wantError)) {
			r.Passed = false
			r.Detail = fmt.Sprintf("unexpected outcome: %v", err)
		}
		if state.uploads != c.writes || state.unexpected != 0 || c.writes == 0 && state.downloads != 0 {
			r.Passed = false
			r.Detail += fmt.Sprintf(" calls uploads=%d downloads=%d unexpected=%d", state.uploads, state.downloads, state.unexpected)
		}
		if c.writes > 0 && (state.cacheControl != "private, no-store" || state.uniform && state.predefinedACL != "" || !state.uniform && state.predefinedACL != "private") {
			r.Passed = false
			r.Detail += " unsafe upload metadata or ACL"
		}
		results = append(results, r)
	}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{"mode": "scripted-sdk-transport", "real_network_calls": 0, "real_provider_calls": 0, "cloud_writes": 0, "results": results}); err != nil {
		return err
	}
	for _, r := range results {
		if !r.Passed {
			return errors.New("storage evaluation failed")
		}
	}
	return nil
}
