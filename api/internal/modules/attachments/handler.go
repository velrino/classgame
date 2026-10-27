package attachments

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"google.golang.org/api/idtoken"

	"classgame-api/internal/platform/requestmeta"
)

const (
	cloudTasksTaskNameHeader = "X-CloudTasks-TaskName"
	maximumTaskBodyBytes     = 64 * 1024
)

type RequestAuthenticator interface {
	Authenticate(context.Context, *http.Request) error
}

type OIDCAuthenticator struct {
	validator            *idtoken.Validator
	audience             string
	expectedServiceEmail string
}

func NewOIDCAuthenticator(
	ctx context.Context,
	audience string,
	expectedServiceEmail string,
) (*OIDCAuthenticator, error) {
	audience = strings.TrimSpace(audience)
	expectedServiceEmail = strings.ToLower(strings.TrimSpace(expectedServiceEmail))
	if audience == "" {
		return nil, errors.New("cloud tasks oidc audience is required")
	}
	if expectedServiceEmail == "" {
		return nil, errors.New("cloud tasks oidc service account email is required")
	}

	validator, err := idtoken.NewValidator(ctx)
	if err != nil {
		return nil, err
	}

	return &OIDCAuthenticator{
		validator:            validator,
		audience:             audience,
		expectedServiceEmail: expectedServiceEmail,
	}, nil
}

func (authenticator *OIDCAuthenticator) Authenticate(ctx context.Context, request *http.Request) error {
	if authenticator == nil || authenticator.validator == nil {
		return errors.New("cloud tasks oidc authenticator is not configured")
	}

	token, err := bearerToken(request.Header.Get("Authorization"))
	if err != nil {
		return err
	}
	payload, err := authenticator.validator.Validate(ctx, token, authenticator.audience)
	if err != nil {
		return errors.New("cloud tasks oidc token is invalid")
	}

	email, _ := payload.Claims["email"].(string)
	if !strings.EqualFold(strings.TrimSpace(email), authenticator.expectedServiceEmail) {
		return errors.New("cloud tasks oidc service account is not allowed")
	}

	return nil
}

type HTTPHandler struct {
	authenticator RequestAuthenticator
	processor     JobProcessor
}

func NewHTTPHandler(authenticator RequestAuthenticator, processor JobProcessor) (*HTTPHandler, error) {
	if authenticator == nil {
		return nil, errors.New("mission attachment task authenticator is required")
	}
	if processor == nil {
		return nil, errors.New("mission attachment task processor is required")
	}

	return &HTTPHandler{
		authenticator: authenticator,
		processor:     processor,
	}, nil
}

func (handler *HTTPHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(request.Header.Get(cloudTasksTaskNameHeader)) == "" {
		response.WriteHeader(http.StatusUnauthorized)
		return
	}
	if err := handler.authenticator.Authenticate(request.Context(), request); err != nil {
		slog.Warn("reject mission attachment task", "error", err)
		response.WriteHeader(http.StatusUnauthorized)
		return
	}

	request.Body = http.MaxBytesReader(response, request.Body, maximumTaskBodyBytes)
	var job Job
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&job); err != nil {
		slog.Error("discard invalid mission attachment task", "error", err)
		response.WriteHeader(http.StatusNoContent)
		return
	}
	if err := ensureJSONEnd(decoder); err != nil {
		slog.Error("discard invalid mission attachment task", "error", err)
		response.WriteHeader(http.StatusNoContent)
		return
	}
	if err := job.Validate(); err != nil {
		slog.Error("discard invalid mission attachment task", "error", err)
		response.WriteHeader(http.StatusNoContent)
		return
	}

	slog.InfoContext(request.Context(), "process mission attachment task",
		"request_id", requestmeta.RequestID(request.Context()),
		"trace_id", requestmeta.TraceID(request.Context()),
		"class_id", job.ClassID, "mission_id", job.MissionID, "revision", job.Revision,
	)
	err := handler.processor.Process(request.Context(), job)
	switch {
	case err == nil, errors.Is(err, ErrRevisionObsolete), IsPermanent(err):
		if err != nil {
			slog.Warn("discard permanent mission attachment task failure",
				"class_id", job.ClassID,
				"mission_id", job.MissionID,
				"revision", job.Revision,
				"error", err,
			)
		}
		response.WriteHeader(http.StatusNoContent)
	case errors.Is(err, ErrRevisionPending):
		response.Header().Set("Retry-After", "5")
		response.WriteHeader(http.StatusServiceUnavailable)
	default:
		slog.Error("retry mission attachment task",
			"class_id", job.ClassID,
			"mission_id", job.MissionID,
			"revision", job.Revision,
			"error", err,
		)
		response.WriteHeader(http.StatusServiceUnavailable)
	}
}

func bearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", errors.New("cloud tasks bearer token is required")
	}

	return parts[1], nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("cloud task body must contain one JSON object")
	}

	return err
}
