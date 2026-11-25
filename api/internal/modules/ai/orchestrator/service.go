package orchestrator

import (
	"context"

	"classgame-api/internal/modules/ai/providers/genai"
)

type GenAIClient interface {
	genai.StructuredGenerator
	IsEnabled() bool
}

type AttachmentReader interface {
	ReadAttachment(context.Context, string, int64) ([]byte, error)
}

type Service struct {
	attachmentReader AttachmentReader
	genAI            GenAIClient
}

type Dependencies struct {
	Client      GenAIClient
	Attachments AttachmentReader
}

func NewService(dependencies Dependencies) *Service {
	return &Service{genAI: dependencies.Client, attachmentReader: dependencies.Attachments}
}
