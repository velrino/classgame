package users

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"classgame-api/internal/shared/requestmeta"
)

const (
	cloudTasksTaskNameHeader = "X-CloudTasks-TaskName"
	maximumLearningTaskBytes = 32 * 1024
)

type LearningTaskRequestAuthenticator interface {
	Authenticate(context.Context, *http.Request) error
}

type LearningJobProcessor interface {
	ProcessLearningJob(context.Context, LearningJob) error
}

type LearningTaskHTTPHandler struct {
	authenticator LearningTaskRequestAuthenticator
	processor     LearningJobProcessor
}

func NewLearningTaskHTTPHandler(
	authenticator LearningTaskRequestAuthenticator,
	processor LearningJobProcessor,
) (*LearningTaskHTTPHandler, error) {
	if authenticator == nil {
		return nil, errors.New("student learning task authenticator is required")
	}
	if processor == nil {
		return nil, errors.New("student learning task processor is required")
	}

	return &LearningTaskHTTPHandler{
		authenticator: authenticator,
		processor:     processor,
	}, nil
}

func (handler *LearningTaskHTTPHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(request.Header.Get(cloudTasksTaskNameHeader)) == "" {
		response.WriteHeader(http.StatusUnauthorized)
		return
	}
	if err := handler.authenticator.Authenticate(request.Context(), request); err != nil {
		slog.Warn("reject student learning task", "error", err)
		response.WriteHeader(http.StatusUnauthorized)
		return
	}

	request.Body = http.MaxBytesReader(response, request.Body, maximumLearningTaskBytes)
	var job LearningJob
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&job); err != nil {
		slog.Error("discard invalid student learning task", "error", err)
		response.WriteHeader(http.StatusNoContent)
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		slog.Error("discard student learning task with trailing JSON", "error", err)
		response.WriteHeader(http.StatusNoContent)
		return
	}
	if err := job.Validate(); err != nil {
		slog.Error("discard invalid student learning job", "error", err)
		response.WriteHeader(http.StatusNoContent)
		return
	}

	slog.InfoContext(request.Context(), "process student learning task",
		"request_id", requestmeta.RequestID(request.Context()),
		"trace_id", requestmeta.TraceID(request.Context()),
		"kind", job.Kind, "attempt_id", job.AttemptID,
	)
	if err := handler.processor.ProcessLearningJob(request.Context(), job); err != nil {
		slog.Error("retry student learning task",
			"kind", job.Kind,
			"student_id", job.StudentID,
			"attempt_id", job.AttemptID,
			"error", err,
		)
		response.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}
