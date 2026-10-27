package generation

import (
	"context"

	"classgame-api/internal/platform/genai"
)

type GenAIClient interface {
	GenerateStructured(context.Context, genai.StructuredRequest, any) (genai.StructuredResult, error)
	IsEnabled() bool
	DefaultModel() string
	Provider() string
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
