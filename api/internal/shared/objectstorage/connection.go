package objectstorage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

type Options struct {
	BucketName      string
	CredentialsFile string
	CredentialsJSON string
}

type Connection struct {
	client     *storage.Client
	bucket     *storage.BucketHandle
	bucketName string
}

func Connect(ctx context.Context, input Options) (*Connection, error) {
	resolved, err := resolveOptions(input)
	if err != nil {
		return nil, err
	}

	clientOptions, err := buildClientOptions(resolved)
	if err != nil {
		return nil, err
	}

	client, err := storage.NewClient(ctx, clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("create cloud storage client: %w", err)
	}

	connection := &Connection{
		client:     client,
		bucket:     client.Bucket(resolved.BucketName),
		bucketName: resolved.BucketName,
	}

	return connection, nil
}

func (connection *Connection) Bucket() *storage.BucketHandle {
	if connection == nil {
		return nil
	}

	return connection.bucket
}

func (connection *Connection) BucketName() string {
	if connection == nil {
		return ""
	}

	return connection.bucketName
}

func (connection *Connection) Ping(ctx context.Context) error {
	if connection == nil || connection.bucket == nil {
		return errors.New("cloud storage connection is not configured")
	}

	_, err := connection.bucket.Attrs(ctx)
	return err
}

func (connection *Connection) Close() error {
	if connection == nil || connection.client == nil {
		return nil
	}

	return connection.client.Close()
}

func resolveOptions(input Options) (Options, error) {
	input.BucketName = strings.TrimSpace(input.BucketName)
	input.CredentialsFile = strings.TrimSpace(input.CredentialsFile)
	input.CredentialsJSON = strings.TrimSpace(input.CredentialsJSON)

	if input.BucketName == "" {
		return Options{}, errors.New("cloud storage bucket name is required")
	}
	if strings.Contains(input.BucketName, "/") {
		return Options{}, errors.New("cloud storage bucket name must not contain a path")
	}
	if input.CredentialsFile != "" && input.CredentialsJSON != "" {
		return Options{}, errors.New("configure either cloud storage credentials file or json, not both")
	}

	return input, nil
}

func buildClientOptions(input Options) ([]option.ClientOption, error) {
	if input.CredentialsJSON != "" {
		return []option.ClientOption{option.WithCredentialsJSON([]byte(input.CredentialsJSON))}, nil
	}
	if input.CredentialsFile != "" {
		return []option.ClientOption{option.WithCredentialsFile(input.CredentialsFile)}, nil
	}

	return nil, nil
}

// ReadAttachment reads only objects in the configured private bucket, with a
// bounded allocation. It does not generate signed or public download URLs.
func (connection *Connection) ReadAttachment(ctx context.Context, gsURI string, maxBytes int64) ([]byte, error) {
	if connection == nil || connection.bucket == nil {
		return nil, errors.New("attachment storage is not configured")
	}
	prefix := "gs://" + connection.bucketName + "/"
	if !strings.HasPrefix(gsURI, prefix) || maxBytes <= 0 {
		return nil, errors.New("invalid attachment storage reference")
	}
	objectPath := strings.TrimPrefix(gsURI, prefix)
	if err := ValidateAttachmentObjectPath(objectPath); err != nil {
		return nil, err
	}

	reader, err := connection.bucket.Object(objectPath).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("open private attachment: %w", err)
	}
	defer reader.Close()
	if reader.Attrs.Size > maxBytes {
		return nil, errors.New("attachment exceeds size limit")
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read private attachment: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, errors.New("attachment exceeds size limit")
	}
	return data, nil
}

// ValidateAttachmentObjectPath rejects traversal segments while preserving
// legitimate worksheet filenames such as lesson..pdf.
func ValidateAttachmentObjectPath(objectPath string) error {
	if !strings.HasPrefix(objectPath, "classes/") || strings.Contains(objectPath, `\`) {
		return errors.New("invalid attachment object path")
	}
	for _, segment := range strings.Split(objectPath, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return errors.New("invalid attachment object path")
		}
	}
	return nil
}
