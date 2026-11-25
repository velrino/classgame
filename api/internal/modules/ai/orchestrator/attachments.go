package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"classgame-api/internal/modules/ai"
	"classgame-api/internal/modules/ai/agents/materials"
	"classgame-api/internal/modules/ai/materials/uploadcare"
	"classgame-api/internal/modules/ai/providers/genai"
)

const maxInlineAttachmentBytes = 20 * 1024 * 1024

const maxAttachmentSummaryCharacters = 1400

func (service *Service) prepareMissionInput(
	ctx context.Context,
	input ai.BuildMissionInput,
) (ai.BuildMissionInput, ai.GenAIUsage) {
	if len(input.Attachments) == 0 {
		reportMissionProgress(input.Progress, ai.MissionProgress{
			Stage:   "preparing_context",
			Message: "Preparing the mission context.",
			Percent: 30,
		})
		return input, ai.GenAIUsage{}
	}

	output := input
	output.Attachments = make([]ai.AttachmentInput, 0, len(input.Attachments))
	usage := ai.GenAIUsage{}
	total := len(input.Attachments)
	reportMissionProgress(input.Progress, ai.MissionProgress{
		Stage:   "reading_attachments",
		Message: "Reading the uploaded materials.",
		Percent: 15,
		Total:   total,
	})

	for index, attachment := range input.Attachments {
		progress := 20 + int(float64(index)/float64(max(total, 1))*40)
		reportMissionProgress(input.Progress, ai.MissionProgress{
			Stage:      "summarizing_attachment",
			Message:    "Summarizing " + attachment.FileName + ".",
			Percent:    progress,
			Current:    index + 1,
			Total:      total,
			Attachment: attachment.FileName,
		})

		summary, summaryUsage, err := service.summarizeAttachment(ctx, attachment)
		usage = addGenAIUsage(usage, summaryUsage)
		if err != nil {
			slog.Warn("skip genai attachment summary", "file_name", attachment.FileName, "error", err)
			if attachment.Summary == "" {
				attachment.Summary = "Attachment content could not be read during preview generation. Use filename and teacher prompt only."
			}
			output.Attachments = append(output.Attachments, attachment)
			continue
		}

		attachment.Summary = summary
		output.Attachments = append(output.Attachments, attachment)
		reportMissionProgress(input.Progress, ai.MissionProgress{
			Stage:      "summarized_attachment",
			Message:    "Summarized " + attachment.FileName + ".",
			Percent:    20 + int(float64(index+1)/float64(max(total, 1))*40),
			Current:    index + 1,
			Total:      total,
			Attachment: attachment.FileName,
		})
	}

	reportMissionProgress(input.Progress, ai.MissionProgress{
		Stage:   "preparing_context",
		Message: "Combining the prompt and attachment summaries.",
		Percent: 65,
	})

	return output, usage
}

func (service *Service) summarizeAttachment(
	ctx context.Context,
	attachment ai.AttachmentInput,
) (string, ai.GenAIUsage, error) {
	mimeType := normalizeMIMEType(attachment.MimeType)
	if !isSupportedInlineMIMEType(mimeType) {
		return "", ai.GenAIUsage{}, fmt.Errorf("unsupported inline attachment MIME type %q", mimeType)
	}

	data, err := service.fetchAttachment(ctx, attachment)
	if err != nil {
		return "", ai.GenAIUsage{}, err
	}

	summary, result, err := materials.Run(ctx, service.genAI, attachment.FileName, mimeType, data)
	if err != nil {
		return "", ai.GenAIUsage{}, err
	}

	text := formatAttachmentSummary(attachment.FileName, summary)
	if text == "" {
		text = "No useful classroom learning content was found in this attachment."
	}

	return truncateText(text, maxAttachmentSummaryCharacters), ai.GenAIUsage{
		PromptTokenCount:     result.PromptTokenCount,
		CandidatesTokenCount: result.CandidatesTokenCount,
		ThoughtsTokenCount:   result.ThoughtsTokenCount,
		TotalTokenCount:      result.TotalTokenCount,
	}, nil
}

func formatAttachmentSummary(fileName string, summary materials.Summary) string {
	parts := []string{}
	if summary.Summary != "" {
		parts = append(parts, summary.Summary)
	}
	if len(summary.KeyConcepts) > 0 {
		parts = append(parts, "Key concepts: "+strings.Join(summary.KeyConcepts, "; "))
	}
	if len(summary.SuggestedQuestion) > 0 {
		parts = append(parts, "Question ideas: "+strings.Join(summary.SuggestedQuestion, "; "))
	}
	if len(summary.Cautions) > 0 {
		parts = append(parts, "Cautions: "+strings.Join(summary.Cautions, "; "))
	}
	if !summary.UsableForMission {
		parts = append(parts, "The model marked this attachment as not directly usable for the mission.")
	}
	if len(parts) == 0 {
		return ""
	}

	return fileName + ": " + strings.Join(parts, " ")
}

func buildAttachmentFileParts(ctx context.Context, attachments []ai.AttachmentInput) []genai.FilePart {
	fileParts := make([]genai.FilePart, 0, len(attachments))
	for _, attachment := range attachments {
		mimeType := normalizeMIMEType(attachment.MimeType)
		if !isSupportedInlineMIMEType(mimeType) {
			continue
		}

		data, err := fetchAttachment(ctx, attachment)
		if err != nil {
			slog.Warn("skip genai attachment", "file_name", attachment.FileName, "error", err)
			continue
		}

		fileParts = append(fileParts, genai.FilePart{
			Data:     data,
			MIMEType: mimeType,
		})
	}

	return fileParts
}

func (service *Service) fetchAttachment(ctx context.Context, attachment ai.AttachmentInput) ([]byte, error) {
	if strings.HasPrefix(strings.TrimSpace(attachment.StorageURL), "gs://") {
		if service.attachmentReader == nil {
			return nil, errors.New("private attachment reader is not configured")
		}
		return service.attachmentReader.ReadAttachment(ctx, attachment.StorageURL, maxInlineAttachmentBytes)
	}
	return fetchAttachment(ctx, attachment)
}

func fetchAttachment(ctx context.Context, attachment ai.AttachmentInput) ([]byte, error) {
	if strings.TrimSpace(attachment.StorageURL) != "" {
		data, err := fetchPublicStorageAttachment(ctx, attachment.StorageURL)
		if err == nil {
			return data, nil
		}
		if strings.TrimSpace(attachment.CDNURL) == "" {
			return nil, err
		}

		slog.Warn("fallback to uploadcare attachment", "file_name", attachment.FileName, "error", err)
	}

	if isPublicStorageURL(attachment.CDNURL) {
		return fetchPublicStorageAttachment(ctx, attachment.CDNURL)
	}

	return fetchUploadcareAttachment(ctx, attachment.CDNURL)
}

func fetchPublicStorageAttachment(ctx context.Context, rawURL string) ([]byte, error) {
	return fetchHTTPAttachment(ctx, rawURL, validatePublicStorageURL, "storage")
}

func fetchUploadcareAttachment(ctx context.Context, rawURL string) ([]byte, error) {
	return fetchHTTPAttachment(ctx, rawURL, validateUploadcareURL, "uploadcare")
}

func fetchHTTPAttachment(
	ctx context.Context,
	rawURL string,
	validate func(*url.URL) error,
	sourceLabel string,
) ([]byte, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("parse %s url: %w", sourceLabel, err)
	}
	if err := validate(parsedURL); err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create %s request: %w", sourceLabel, err)
	}

	client := http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many %s redirects", sourceLabel)
			}

			return validate(request.URL)
		},
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download %s attachment: %w", sourceLabel, err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("download %s attachment returned status %d", sourceLabel, response.StatusCode)
	}
	if response.ContentLength > maxInlineAttachmentBytes {
		return nil, fmt.Errorf("%s attachment is larger than %d bytes", sourceLabel, maxInlineAttachmentBytes)
	}

	data, err := io.ReadAll(io.LimitReader(response.Body, maxInlineAttachmentBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read %s attachment: %w", sourceLabel, err)
	}
	if len(data) > maxInlineAttachmentBytes {
		return nil, fmt.Errorf("%s attachment is larger than %d bytes", sourceLabel, maxInlineAttachmentBytes)
	}

	return data, nil
}

func validateUploadcareURL(parsedURL *url.URL) error {
	return uploadcare.ValidateCDNURL(parsedURL)
}

func validatePublicStorageURL(parsedURL *url.URL) error {
	if parsedURL == nil ||
		parsedURL.Scheme != "https" ||
		parsedURL.User != nil ||
		parsedURL.Hostname() != "storage.googleapis.com" {
		return fmt.Errorf("invalid public storage URL")
	}
	if port := parsedURL.Port(); port != "" && port != "443" {
		return fmt.Errorf("invalid public storage URL")
	}

	segments := strings.Split(strings.Trim(parsedURL.EscapedPath(), "/"), "/")
	if len(segments) < 2 || segments[0] == "" || segments[1] == "" {
		return fmt.Errorf("invalid public storage URL")
	}

	return nil
}

func isPublicStorageURL(rawURL string) bool {
	parsedURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}

	return validatePublicStorageURL(parsedURL) == nil
}

func isSupportedInlineMIMEType(value string) bool {
	switch normalizeMIMEType(value) {
	case "application/pdf",
		"image/gif",
		"image/jpeg",
		"image/png",
		"image/webp",
		"text/csv",
		"text/markdown",
		"text/plain":
		return true
	default:
		return false
	}
}

func normalizeMIMEType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if semicolonIndex := strings.Index(value, ";"); semicolonIndex >= 0 {
		value = strings.TrimSpace(value[:semicolonIndex])
	}

	return value
}

func normalizeAttachments(attachments []ai.AttachmentInput) []ai.AttachmentInput {
	normalized := make([]ai.AttachmentInput, 0, len(attachments))
	for _, attachment := range attachments {
		fileName := normalizeText(attachment.FileName)
		mimeType := normalizeText(attachment.MimeType)
		cdnURL := strings.TrimSpace(attachment.CDNURL)
		storageURL := strings.TrimSpace(attachment.StorageURL)
		if fileName == "" && cdnURL == "" && storageURL == "" {
			continue
		}

		normalized = append(normalized, ai.AttachmentInput{
			FileName:   fileName,
			MimeType:   mimeType,
			CDNURL:     cdnURL,
			StorageURL: storageURL,
			Summary:    truncateText(normalizeText(attachment.Summary), maxAttachmentSummaryCharacters),
		})
	}

	return normalized
}
