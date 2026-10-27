package attachments

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode"

	"cloud.google.com/go/storage"

	"classgame-api/internal/uploadcare"
)

const (
	MaxAttachmentBytes      int64 = 20 * 1024 * 1024
	defaultHTTPTimeout            = 3 * time.Minute
	defaultWriterChunkBytes       = 2 * 1024 * 1024
	maximumRedirects              = 5
)

var ErrAttachmentTooLarge = errors.New("attachment is larger than 20 MB")

type CopyInput struct {
	ClassID           string
	MissionID         string
	Revision          int64
	UploadcareUUID    string
	SourceURL         string
	FileName          string
	MIMEType          string
	ExpectedSizeBytes int64
}

type StoredObject struct {
	GSURI      string
	Bucket     string
	ObjectPath string
	SizeBytes  int64
	MIMEType   string
}

type Copier interface {
	Copy(context.Context, CopyInput) (StoredObject, error)
}

type StorageCopierOptions struct {
	BucketName string
	Bucket     *storage.BucketHandle
	HTTPClient *http.Client
	MaxBytes   int64
}

type StorageCopier struct {
	bucketName string
	bucket     *storage.BucketHandle
	httpClient *http.Client
	maxBytes   int64
}

func NewStorageCopier(options StorageCopierOptions) (*StorageCopier, error) {
	bucketName := strings.TrimSpace(options.BucketName)
	if bucketName == "" {
		return nil, errors.New("mission attachment storage bucket name is required")
	}
	if options.Bucket == nil {
		return nil, errors.New("mission attachment storage bucket is required")
	}
	if options.Bucket.BucketName() != bucketName {
		return nil, errors.New("mission attachment bucket handle does not match its configured name")
	}

	maxBytes := options.MaxBytes
	if maxBytes <= 0 {
		maxBytes = MaxAttachmentBytes
	}

	baseHTTPClient := options.HTTPClient
	if baseHTTPClient == nil {
		baseHTTPClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	httpClient := cloneUploadcareHTTPClient(baseHTTPClient)

	return &StorageCopier{
		bucketName: bucketName,
		bucket:     options.Bucket,
		httpClient: httpClient,
		maxBytes:   maxBytes,
	}, nil
}

func (copier *StorageCopier) Copy(ctx context.Context, input CopyInput) (StoredObject, error) {
	if copier == nil || copier.bucket == nil {
		return StoredObject{}, errors.New("mission attachment storage copier is not configured")
	}
	if err := validateCopyInput(input, copier.maxBytes); err != nil {
		return StoredObject{}, MarkPermanent(err)
	}

	objectPath := buildObjectPath(input)
	object := copier.bucket.Object(objectPath)
	if existing, exists, err := copier.existingObject(ctx, object, input); err != nil {
		return StoredObject{}, err
	} else if exists {
		return existing, nil
	}
	uniform, err := copier.privateBucket(ctx)
	if err != nil {
		return StoredObject{}, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSpace(input.SourceURL), nil)
	if err != nil {
		return StoredObject{}, MarkPermanent(fmt.Errorf("create uploadcare attachment request: %w", err))
	}

	response, err := copier.httpClient.Do(request)
	if err != nil {
		return StoredObject{}, fmt.Errorf("download uploadcare attachment: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		err := fmt.Errorf("download uploadcare attachment returned status %d", response.StatusCode)
		if response.StatusCode != http.StatusRequestTimeout && response.StatusCode != http.StatusTooManyRequests && response.StatusCode < http.StatusInternalServerError {
			return StoredObject{}, MarkPermanent(err)
		}
		return StoredObject{}, err
	}
	if response.ContentLength > copier.maxBytes {
		return StoredObject{}, MarkPermanent(ErrAttachmentTooLarge)
	}

	contentType := resolveContentType(input.MIMEType, response.Header.Get("Content-Type"))
	conditionedObject := object.If(storage.Conditions{DoesNotExist: true})
	writer := conditionedObject.NewWriter(ctx)
	writer.ChunkSize = defaultWriterChunkBytes
	writer.ContentType = contentType
	writer.CacheControl = "private, no-store"
	// Fine-grained buckets may have public default object ACLs. Override them.
	// Uniform buckets forbid object ACLs and rely on the verified private IAM policy.
	if !uniform {
		writer.PredefinedACL = "private"
	}
	writer.ContentDisposition = mime.FormatMediaType("inline", map[string]string{
		"filename": strings.TrimSpace(input.FileName),
	})
	writer.Metadata = map[string]string{
		"class_id":        input.ClassID,
		"mission_id":      input.MissionID,
		"revision":        fmt.Sprintf("%d", input.Revision),
		"uploadcare_uuid": strings.TrimSpace(input.UploadcareUUID),
		"source_key":      sourceKey(input),
	}

	written, copyErr := io.Copy(writer, io.LimitReader(response.Body, copier.maxBytes+1))
	if copyErr != nil {
		_ = writer.CloseWithError(copyErr)
		return StoredObject{}, fmt.Errorf("stream attachment to cloud storage: %w", copyErr)
	}
	if written > copier.maxBytes {
		_ = writer.CloseWithError(ErrAttachmentTooLarge)
		return StoredObject{}, MarkPermanent(ErrAttachmentTooLarge)
	}
	if err := writer.Close(); err != nil {
		// A concurrent retry may have won the DoesNotExist precondition. Reading
		// and validating the final object makes that race idempotent.
		if existing, exists, attrsErr := copier.existingObject(ctx, object, input); attrsErr == nil && exists {
			return existing, nil
		}
		return StoredObject{}, fmt.Errorf("finish cloud storage attachment: %w", err)
	}

	stored, exists, err := copier.existingObject(ctx, object, input)
	if err != nil {
		return StoredObject{}, err
	}
	if !exists {
		return StoredObject{}, errors.New("stored attachment was not found after upload")
	}
	return stored, nil
}

func (copier *StorageCopier) existingObject(
	ctx context.Context,
	object *storage.ObjectHandle,
	input CopyInput,
) (StoredObject, bool, error) {
	attrs, err := object.Attrs(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return StoredObject{}, false, nil
	}
	if err != nil {
		return StoredObject{}, false, fmt.Errorf("read existing cloud storage attachment: %w", err)
	}
	uniform, err := copier.privateBucket(ctx)
	if err != nil {
		return StoredObject{}, false, err
	}
	if strings.TrimSpace(attrs.Metadata["firebaseStorageDownloadTokens"]) != "" {
		return StoredObject{}, false, MarkPermanent(errors.New("attachment has download-token access; operator review is required"))
	}
	if !uniform {
		// Attrs requests the full projection, keeping the ACL and metadata tied
		// to one object generation. An omitted ACL is not proof of private access.
		if len(attrs.ACL) == 0 {
			return StoredObject{}, false, errors.New("attachment access could not be verified; operator review is required")
		}
		for _, rule := range attrs.ACL {
			if rule.Entity == storage.AllUsers || rule.Entity == storage.AllAuthenticatedUsers {
				return StoredObject{}, false, MarkPermanent(errors.New("attachment still has public access; operator review is required"))
			}
		}
	}

	if attrs.Metadata["class_id"] != input.ClassID ||
		attrs.Metadata["mission_id"] != input.MissionID ||
		attrs.Metadata["revision"] != fmt.Sprintf("%d", input.Revision) ||
		attrs.Metadata["source_key"] != sourceKey(input) {
		return StoredObject{}, false, MarkPermanent(errors.New("cloud storage attachment path collision"))
	}

	return copier.toStoredObject(attrs.Name, attrs), true, nil
}

func (copier *StorageCopier) privateBucket(ctx context.Context) (bool, error) {
	attrs, err := copier.bucket.Attrs(ctx)
	if err != nil {
		return false, errors.New("attachment bucket metadata could not be verified")
	}
	policy, err := copier.bucket.IAM().V3().Policy(ctx)
	if err != nil {
		return false, errors.New("attachment bucket access could not be verified")
	}
	for _, binding := range policy.Bindings {
		for _, member := range binding.Members {
			if member == "allUsers" || member == "allAuthenticatedUsers" {
				return false, MarkPermanent(errors.New("attachment bucket has public access; use a private bucket"))
			}
		}
	}
	return attrs.UniformBucketLevelAccess.Enabled, nil
}

func (copier *StorageCopier) toStoredObject(objectPath string, attrs *storage.ObjectAttrs) StoredObject {
	return StoredObject{
		GSURI:      "gs://" + copier.bucketName + "/" + objectPath,
		Bucket:     copier.bucketName,
		ObjectPath: objectPath,
		SizeBytes:  attrs.Size,
		MIMEType:   attrs.ContentType,
	}
}

func validateCopyInput(input CopyInput, maxBytes int64) error {
	if err := validatePathID("class_id", input.ClassID, 40); err != nil {
		return err
	}
	if err := validatePathID("mission_id", input.MissionID, 120); err != nil {
		return err
	}
	if input.Revision <= 0 {
		return errors.New("revision must be greater than zero")
	}
	if strings.TrimSpace(input.FileName) == "" {
		return errors.New("attachment file name is required")
	}
	if input.ExpectedSizeBytes < 0 {
		return errors.New("attachment size cannot be negative")
	}
	if input.ExpectedSizeBytes > maxBytes {
		return ErrAttachmentTooLarge
	}
	if err := validateUploadcareURL(input.SourceURL); err != nil {
		return err
	}

	return nil
}

func validateUploadcareURL(rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return fmt.Errorf("parse uploadcare attachment url: %w", err)
	}
	if parsed.Scheme != "https" {
		return errors.New("uploadcare attachment url must use https")
	}
	if parsed.User != nil {
		return errors.New("uploadcare attachment url must not include user information")
	}
	if port := parsed.Port(); port != "" && port != "443" {
		return errors.New("uploadcare attachment url must use the default https port")
	}
	if !uploadcare.IsCDNHost(parsed.Hostname()) {
		return errors.New("uploadcare attachment url host is not allowed")
	}

	return nil
}

func cloneUploadcareHTTPClient(base *http.Client) *http.Client {
	cloned := *base
	previousCheckRedirect := cloned.CheckRedirect
	cloned.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) >= maximumRedirects {
			return errors.New("uploadcare attachment redirected too many times")
		}
		if err := validateUploadcareURL(request.URL.String()); err != nil {
			return err
		}
		if previousCheckRedirect != nil {
			return previousCheckRedirect(request, via)
		}
		return nil
	}
	if cloned.Timeout <= 0 {
		cloned.Timeout = defaultHTTPTimeout
	}

	return &cloned
}

func buildObjectPath(input CopyInput) string {
	return path.Join(
		"classes",
		input.ClassID,
		"missions",
		input.MissionID,
		"revisions",
		fmt.Sprintf("%d", input.Revision),
		sourceKey(input),
		sanitizeFileName(input.FileName),
	)
}

func sourceKey(input CopyInput) string {
	if uploadcareUUID := sanitizePathSegment(input.UploadcareUUID); uploadcareUUID != "" {
		return uploadcareUUID
	}

	digest := sha256.Sum256([]byte(strings.TrimSpace(input.SourceURL)))
	return "source-" + hex.EncodeToString(digest[:12])
}

func sanitizeFileName(value string) string {
	value = path.Base(strings.ReplaceAll(strings.TrimSpace(value), "\\", "/"))
	if value == "" || value == "." || value == ".." {
		return "attachment"
	}

	characters := make([]rune, 0, len(value))
	lastSeparator := false
	for _, character := range value {
		switch {
		case unicode.IsLetter(character), unicode.IsDigit(character), character == '.', character == '_', character == '-':
			characters = append(characters, character)
			lastSeparator = false
		default:
			if !lastSeparator {
				characters = append(characters, '-')
				lastSeparator = true
			}
		}
		if len(characters) == 160 {
			break
		}
	}

	result := strings.Trim(string(characters), ".-")
	if result == "" {
		return "attachment"
	}

	return result
}

func sanitizePathSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." || strings.ContainsAny(value, `/\\`) {
		return ""
	}

	characters := make([]rune, 0, len(value))
	for _, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '-' || character == '_' {
			characters = append(characters, character)
		}
	}

	return string(characters)
}

func resolveContentType(input string, response string) string {
	for _, value := range []string{input, response} {
		mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
		if err == nil && mediaType != "" {
			return strings.ToLower(mediaType)
		}
	}

	return "application/octet-stream"
}
