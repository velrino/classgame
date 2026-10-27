package mission

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"

	sharedauth "classgame-api/internal/shared/auth"
	"github.com/labstack/echo/v5"
)

const maximumAttachmentDownloadBytes int64 = 20 * 1024 * 1024

type AttachmentReader interface {
	ReadAttachment(context.Context, string, int64) ([]byte, error)
}
type AttachmentService interface {
	DownloadAttachment(context.Context, sharedauth.Identity, string, string, string, int64) (AttachmentDownload, error)
}
type AttachmentDownload struct {
	Data     []byte
	FileName string
	MIMEType string
}

func WithAttachmentReader(reader AttachmentReader) ServiceOption {
	return func(service *Service) { service.attachmentReader = reader }
}

func (service *Service) DownloadAttachment(ctx context.Context, identity sharedauth.Identity, classID, missionID, uuid string, revision int64) (AttachmentDownload, error) {
	_, exists, err := service.teacherClassService.Get(ctx, identity, classID)
	if err != nil {
		return AttachmentDownload{}, err
	}
	if !exists {
		return AttachmentDownload{}, ErrMissionNotFound
	}
	document, exists, err := service.repository.Get(ctx, classID, missionID)
	if err != nil {
		return AttachmentDownload{}, err
	}
	if !exists {
		return AttachmentDownload{}, ErrMissionNotFound
	}
	attachments := draftView(document).Attachments
	if revision > 0 {

		published, exists, err := service.repository.GetPublishedRevision(ctx, classID, missionID, revision)
		if err != nil {
			return AttachmentDownload{}, err
		}
		if !exists {
			return AttachmentDownload{}, ErrMissionNotFound
		}
		attachments = published.Snapshot.Attachments
	}
	for _, attachment := range attachments {
		if attachment.UploadcareUUID != uuid {
			continue
		}
		uri := AttachmentStorageURI(attachment.Storage)
		if uri == "" {
			return AttachmentDownload{}, ErrMissionNotFound
		}
		_, objectPath := parseGSURI(uri)
		if !strings.HasPrefix(objectPath, "classes/"+classID+"/missions/"+missionID+"/revisions/") {
			return AttachmentDownload{}, errors.New("attachment storage scope mismatch")
		}
		if service.attachmentReader == nil {
			return AttachmentDownload{}, ErrMissionAttachmentQueue
		}
		data, err := service.attachmentReader.ReadAttachment(ctx, uri, maximumAttachmentDownloadBytes)
		if err != nil {
			return AttachmentDownload{}, fmt.Errorf("download private attachment: %w", err)
		}
		return AttachmentDownload{Data: data, FileName: attachment.FileName, MIMEType: attachment.MimeType}, nil
	}
	return AttachmentDownload{}, ErrMissionNotFound
}

func (handler *Handler) GetMissionAttachment(ctx *echo.Context) error {
	classID, missionID, uuid := ctx.Param("class_id"), ctx.Param("mission_id"), ctx.Param("uploadcare_uuid")
	if err := validateClassID(classID); err != nil {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", "Invalid class ID.")
	}
	if err := validateMissionID(missionID); err != nil {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", "Invalid mission ID.")
	}
	if len(uuid) > 120 || strings.ContainsAny(uuid, "/\\") || strings.TrimSpace(uuid) == "" {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", "Invalid attachment ID.")
	}
	revision := int64(0)
	if value := ctx.QueryParam("revision"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed < 1 {
			return jsonError(ctx, http.StatusBadRequest, "validation_failed", "Invalid published revision.")
		}
		revision = parsed
	}
	requestContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.missionTimeout)
	defer cancel()
	identity, ok, err := handler.authenticate(requestContext, ctx)
	if err != nil || !ok {
		return err
	}
	if handler.attachmentService == nil {
		return jsonError(ctx, http.StatusServiceUnavailable, "not_configured", "Attachment delivery is not configured.")
	}
	result, err := handler.attachmentService.DownloadAttachment(requestContext, identity, classID, missionID, uuid, revision)
	if err != nil {
		return handler.serviceError(ctx, err)
	}
	contentType := result.MIMEType
	if _, _, err := mime.ParseMediaType(contentType); err != nil {
		contentType = "application/octet-stream"
	}
	ctx.Response().Header().Set("Cache-Control", "private, no-store")
	ctx.Response().Header().Set("X-Content-Type-Options", "nosniff")
	ctx.Response().Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": result.FileName}))
	return ctx.Blob(http.StatusOK, contentType, result.Data)
}
