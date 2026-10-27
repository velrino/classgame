package mission

import (
	"classgame-api/internal/pagination"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"classgame-api/internal/modules/orchestrator"
	teacherclass "classgame-api/internal/modules/teacher-class"
	sharedanalytics "classgame-api/internal/shared/analytics"
	sharedcontentpolicy "classgame-api/internal/shared/contentpolicy"
	sharedfirebase "classgame-api/internal/shared/google/firebase"
	sharedratelimit "classgame-api/internal/shared/ratelimit"
	shareduploadcare "classgame-api/internal/shared/uploadcare"
)

const (
	defaultMissionTimeout          = 3 * time.Minute
	defaultMissionRateLimitRate    = 0.2
	defaultMissionRateLimitBurst   = 8
	defaultMissionRateLimitExpires = 5 * time.Minute
	maxMissionAttachments          = 5
	maxMissionAttachmentSizeBytes  = 20 * 1024 * 1024
)

var classIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{1,39}$`)

type Authenticator interface {
	AuthenticateBearerToken(context.Context, string) (sharedfirebase.Identity, error)
}

type GenerationRateLimiters struct {
	PerIPHour        sharedratelimit.Limiter
	PerTeacherHour   sharedratelimit.Limiter
	PerTeacherMinute sharedratelimit.Limiter
}

type MissionService interface {
	Approve(context.Context, sharedfirebase.Identity, string, string, ApproveMissionInput) (MissionDocument, error)
	Delete(context.Context, sharedfirebase.Identity, string, string) error
	ListPage(context.Context, sharedfirebase.Identity, string, pagination.Params) ([]MissionDocument, string, error)
	Preview(context.Context, sharedfirebase.Identity, CreateMissionInput) (MissionDocument, error)
}

type HandlerOptions struct {
	Analytics               sharedanalytics.Tracker
	Authenticator           Authenticator
	GenerationRateLimiters  GenerationRateLimiters
	MissionTimeout          time.Duration
	MissionRateLimitRate    float64
	MissionRateLimitBurst   int
	MissionRateLimitExpires time.Duration
	MissionService          MissionService
}

type Handler struct {
	analytics          sharedanalytics.Tracker
	missionService     MissionService
	missionTimeout     time.Duration
	authenticator      Authenticator
	generationLimiters GenerationRateLimiters
}

type CreateMissionRequest struct {
	MissionID             string                     `json:"mission_id"`
	ExpectedDraftRevision int64                      `json:"expected_draft_revision"`
	ClassName             string                     `json:"class_name"`
	MissionName           string                     `json:"mission_name"`
	Subject               string                     `json:"subject"`
	GradeLevel            string                     `json:"grade_level"`
	LearningGoal          string                     `json:"learning_goal"`
	TeacherPrompt         string                     `json:"teacher_prompt"`
	AudienceType          string                     `json:"audience_type"`
	TargetStudentIDs      []string                   `json:"target_student_ids"`
	Attachments           []MissionAttachmentRequest `json:"attachments"`
}

type ApproveMissionRequest struct {
	ExpectedDraftRevision int64                       `json:"expected_draft_revision"`
	Draft                 *ApproveMissionDraftRequest `json:"draft"`
}

type ApproveMissionDraftRequest struct {
	Title            string                         `json:"title"`
	Summary          string                         `json:"summary"`
	PreviewQuestions []orchestrator.PreviewQuestion `json:"preview_questions"`
}

type MissionAttachmentRequest struct {
	UploadcareUUID   string `json:"uploadcare_uuid"`
	CDNURL           string `json:"cdn_url"`
	UploadcareCDNURL string `json:"uploadcare_cdn_url"`
	FileName         string `json:"file_name"`
	MimeType         string `json:"mime_type"`
	SizeBytes        int64  `json:"size_bytes"`
}

type ErrorResponse struct {
	Error ErrorResponseBody `json:"error"`
}

type ErrorResponseBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func RegisterRoutes(server *echo.Echo, options HandlerOptions) {
	handler := &Handler{
		analytics:          options.Analytics,
		missionService:     options.MissionService,
		missionTimeout:     resolveMissionTimeout(options.MissionTimeout),
		authenticator:      options.Authenticator,
		generationLimiters: options.GenerationRateLimiters,
	}

	group := server.Group("/teacher-classes/:class_id/missions")
	generationRateLimiter := missionRateLimiter(options)
	group.GET("", handler.ListMissions)
	group.POST("/preview", handler.PostMissionPreview, generationRateLimiter)
	group.POST("/preview/stream", handler.PostMissionPreviewStream, generationRateLimiter)
	group.POST("/:mission_id/approve", handler.PostMissionApprove)
	group.DELETE("/:mission_id", handler.DeleteMission)
}

func (handler *Handler) PostMissionPreview(ctx *echo.Context) error {
	return handler.createMission(ctx)
}

func (handler *Handler) PostMissionPreviewStream(ctx *echo.Context) error {
	if handler.authenticator == nil || handler.missionService == nil {
		slog.Error("teacher mission dependencies are not configured")
		return jsonError(ctx, http.StatusInternalServerError, "not_configured", "Teacher missions are not configured.")
	}

	classID := ctx.Param("class_id")
	if err := validateClassID(classID); err != nil {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}

	var request CreateMissionRequest
	if err := ctx.Bind(&request); err != nil {
		return jsonError(ctx, http.StatusBadRequest, "bad_request", "Invalid JSON body.")
	}
	if err := request.Validate(); err != nil {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}

	requestContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.missionTimeout)
	defer cancel()

	identity, ok, err := handler.authenticate(requestContext, ctx)
	if err != nil || !ok {
		return err
	}
	limited, err := handler.limitGeneration(requestContext, ctx, identity)
	if err != nil || limited {
		return err
	}

	response := ctx.Response()
	response.Header().Set(echo.HeaderContentType, "text/event-stream")
	response.Header().Set(echo.HeaderCacheControl, "no-cache")
	response.Header().Set(echo.HeaderConnection, "keep-alive")
	response.Header().Set("X-Accel-Buffering", "no")
	response.WriteHeader(http.StatusOK)

	input := request.toInput(classID)
	var streamErr error
	input.Progress = func(progress orchestrator.MissionProgress) {
		if streamErr != nil {
			return
		}
		streamErr = writeSSE(ctx, "progress", progress)
	}

	if err := writeSSE(ctx, "progress", orchestrator.MissionProgress{
		Stage:   "queued",
		Message: "Starting mission preview.",
		Percent: 1,
	}); err != nil {
		return err
	}

	mission, err := handler.missionService.Preview(requestContext, identity, input)
	if streamErr != nil {
		return nil
	}
	if err != nil {
		slog.Error("teacher mission stream request failed", "error", err)
		_ = writeSSE(ctx, "error", ErrorResponse{
			Error: ErrorResponseBody{
				Code:    "teacher_mission_failed",
				Message: "Could not process teacher mission.",
			},
		})
		return nil
	}
	if strings.TrimSpace(request.MissionID) == "" {
		sharedanalytics.Track(
			handler.analytics,
			requestContext,
			sharedanalytics.EventMissionCreated,
			identity.ProviderUserID,
			sharedanalytics.Properties{
				"class_id":   mission.ClassID,
				"mission_id": mission.ID,
			},
		)
	}

	if err := writeSSE(ctx, "done", map[string]any{
		"mission": ToResponse(mission),
	}); err != nil {
		return nil
	}

	return nil
}

func (handler *Handler) PostMissionApprove(ctx *echo.Context) error {
	if handler.authenticator == nil || handler.missionService == nil {
		slog.Error("teacher mission dependencies are not configured")
		return jsonError(ctx, http.StatusInternalServerError, "not_configured", "Teacher missions are not configured.")
	}

	classID := ctx.Param("class_id")
	if err := validateClassID(classID); err != nil {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}

	missionID := ctx.Param("mission_id")
	if err := validateMissionID(missionID); err != nil {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}

	var request ApproveMissionRequest
	if ctx.Request().Body != nil && ctx.Request().ContentLength != 0 {
		if err := ctx.Bind(&request); err != nil {
			return jsonError(ctx, http.StatusBadRequest, "bad_request", "Invalid JSON body.")
		}
	}
	if request.ExpectedDraftRevision <= 0 {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", "expected_draft_revision must be greater than zero")
	}
	if err := request.Validate(); err != nil {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}

	requestContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.missionTimeout)
	defer cancel()

	identity, ok, err := handler.authenticate(requestContext, ctx)
	if err != nil || !ok {
		return err
	}

	mission, err := handler.missionService.Approve(
		requestContext,
		identity,
		classID,
		missionID,
		ApproveMissionInput{
			ExpectedDraftRevision: request.ExpectedDraftRevision,
			DraftEdits:            request.Draft.toInput(),
		},
	)
	if err != nil {
		return handler.serviceError(ctx, err)
	}
	sharedanalytics.Track(
		handler.analytics,
		requestContext,
		sharedanalytics.EventMissionPublished,
		identity.ProviderUserID,
		sharedanalytics.Properties{
			"class_id":   mission.ClassID,
			"mission_id": mission.ID,
		},
	)

	return ctx.JSON(http.StatusOK, map[string]any{
		"mission": ToResponse(mission),
	})
}

func (handler *Handler) DeleteMission(ctx *echo.Context) error {
	if handler.authenticator == nil || handler.missionService == nil {
		slog.Error("teacher mission dependencies are not configured")
		return jsonError(ctx, http.StatusInternalServerError, "not_configured", "Teacher missions are not configured.")
	}

	classID := ctx.Param("class_id")
	if err := validateClassID(classID); err != nil {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}
	missionID := ctx.Param("mission_id")
	if err := validateMissionID(missionID); err != nil {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}

	requestContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.missionTimeout)
	defer cancel()

	identity, ok, err := handler.authenticate(requestContext, ctx)
	if err != nil || !ok {
		return err
	}
	if err := handler.missionService.Delete(requestContext, identity, classID, missionID); err != nil {
		return handler.serviceError(ctx, err)
	}

	return ctx.NoContent(http.StatusNoContent)
}

func (handler *Handler) ListMissions(ctx *echo.Context) error {
	if handler.authenticator == nil || handler.missionService == nil {
		slog.Error("teacher mission dependencies are not configured")
		return jsonError(ctx, http.StatusInternalServerError, "not_configured", "Teacher missions are not configured.")
	}

	classID := ctx.Param("class_id")
	if err := validateClassID(classID); err != nil {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}

	requestContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.missionTimeout)
	defer cancel()

	identity, ok, err := handler.authenticate(requestContext, ctx)
	if err != nil || !ok {
		return err
	}

	page, err := pagination.Parse(ctx.QueryParams())
	if err != nil {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}
	missions, next, err := handler.missionService.ListPage(requestContext, identity, classID, page)
	if err != nil {
		return handler.serviceError(ctx, err)
	}

	return ctx.JSON(http.StatusOK, map[string]any{
		"missions":    ToResponses(missions),
		"next_cursor": next,
	})
}

func (handler *Handler) createMission(ctx *echo.Context) error {
	if handler.authenticator == nil || handler.missionService == nil {
		slog.Error("teacher mission dependencies are not configured")
		return jsonError(ctx, http.StatusInternalServerError, "not_configured", "Teacher missions are not configured.")
	}

	classID := ctx.Param("class_id")
	if err := validateClassID(classID); err != nil {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}

	var request CreateMissionRequest
	if err := ctx.Bind(&request); err != nil {
		return jsonError(ctx, http.StatusBadRequest, "bad_request", "Invalid JSON body.")
	}
	if err := request.Validate(); err != nil {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}

	requestContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.missionTimeout)
	defer cancel()

	identity, ok, err := handler.authenticate(requestContext, ctx)
	if err != nil || !ok {
		return err
	}
	limited, err := handler.limitGeneration(requestContext, ctx, identity)
	if err != nil || limited {
		return err
	}

	input := request.toInput(classID)

	mission, err := handler.missionService.Preview(requestContext, identity, input)
	if err != nil {
		return handler.serviceError(ctx, err)
	}
	if strings.TrimSpace(request.MissionID) == "" {
		sharedanalytics.Track(
			handler.analytics,
			requestContext,
			sharedanalytics.EventMissionCreated,
			identity.ProviderUserID,
			sharedanalytics.Properties{
				"class_id":   mission.ClassID,
				"mission_id": mission.ID,
			},
		)
	}

	return ctx.JSON(http.StatusOK, map[string]any{
		"mission": ToResponse(mission),
	})
}

func (request CreateMissionRequest) toInput(classID string) CreateMissionInput {
	return CreateMissionInput{
		MissionID:             request.MissionID,
		ExpectedDraftRevision: request.ExpectedDraftRevision,
		ClassID:               classID,
		ClassName:             request.ClassName,
		MissionName:           request.MissionName,
		Subject:               request.Subject,
		GradeLevel:            request.GradeLevel,
		LearningGoal:          request.LearningGoal,
		TeacherPrompt:         request.TeacherPrompt,
		AudienceType:          request.AudienceType,
		TargetStudentIDs:      request.TargetStudentIDs,
		Attachments:           toMissionAttachments(request.Attachments),
		Status:                StatusDraft,
	}
}

func writeSSE(ctx *echo.Context, event string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ctx.Response(), "event: %s\ndata: %s\n\n", event, payload); err != nil {
		return err
	}
	if flusher, ok := ctx.Response().(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

func (handler *Handler) authenticate(
	ctx context.Context,
	echoCtx *echo.Context,
) (sharedfirebase.Identity, bool, error) {
	identity, err := handler.authenticator.AuthenticateBearerToken(
		ctx,
		echoCtx.Request().Header.Get(echo.HeaderAuthorization),
	)
	if err == nil {
		return identity, true, nil
	}

	if errors.Is(err, sharedfirebase.ErrMissingBearerToken) ||
		errors.Is(err, sharedfirebase.ErrInvalidBearerToken) {
		return sharedfirebase.Identity{}, false, jsonError(echoCtx, http.StatusUnauthorized, "unauthorized", "Invalid Firebase ID token.")
	}

	slog.Warn("teacher mission auth failed", "error", err)
	return sharedfirebase.Identity{}, false, jsonError(echoCtx, http.StatusUnauthorized, "unauthorized", "Invalid Firebase ID token.")
}

func (handler *Handler) serviceError(ctx *echo.Context, err error) error {
	if errors.Is(err, teacherclass.ErrTeacherRequiresGoogleAuth) {
		return jsonError(ctx, http.StatusForbidden, "forbidden", "Teacher missions require Google sign in.")
	}
	if errors.Is(err, teacherclass.ErrClassOwnedByAnotherTeacher) {
		return jsonError(ctx, http.StatusForbidden, "forbidden", "This class code already belongs to another teacher.")
	}
	if errors.Is(err, teacherclass.ErrClassNotFound) || errors.Is(err, ErrMissionNotFound) {
		return jsonError(ctx, http.StatusNotFound, "not_found", "Teacher mission not found.")
	}
	if errors.Is(err, ErrMissionDraftConflict) {
		return jsonError(ctx, http.StatusConflict, "draft_conflict", "The mission draft changed. Generate the preview again before approving.")
	}
	if errors.Is(err, ErrMissionDeleted) {
		return jsonError(ctx, http.StatusGone, "mission_deleted", "This mission was deleted.")
	}
	if errors.Is(err, ErrMissionDraftRevisionRequired) {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", "expected_draft_revision must be greater than zero")
	}
	if errors.Is(err, ErrMissionAttachmentQueue) {
		return jsonError(ctx, http.StatusServiceUnavailable, "attachment_queue_unavailable", "Mission file storage is temporarily unavailable. Try publishing again.")
	}
	if errors.Is(err, ErrMissionGuardrailBlocked) {
		return jsonError(ctx, http.StatusUnprocessableEntity, "mission_guardrail_blocked", "This preview cannot be published. Add relevant grade-level educational material and generate it again.")
	}

	slog.Error("teacher mission request failed", "error", err)
	return jsonError(ctx, http.StatusInternalServerError, "teacher_mission_failed", "Could not process teacher mission.")
}

func (handler *Handler) limitGeneration(
	ctx context.Context,
	echoCtx *echo.Context,
	identity sharedfirebase.Identity,
) (bool, error) {
	teacherIdentifier := "teacher:" + string(identity.Provider) + ":" + identity.ProviderUserID
	policy, err := sharedratelimit.Evaluate(ctx,
		sharedratelimit.Rule{Limiter: handler.generationLimiters.PerTeacherMinute, Identifier: teacherIdentifier},
		sharedratelimit.Rule{Limiter: handler.generationLimiters.PerTeacherHour, Identifier: teacherIdentifier},
		sharedratelimit.Rule{Limiter: handler.generationLimiters.PerIPHour, Identifier: "ip:" + echoCtx.RealIP()},
	)
	if err != nil {
		slog.Error("teacher mission generation rate limiter failed", "error", err)
		return true, jsonError(echoCtx, http.StatusServiceUnavailable, "rate_limit_unavailable", "Mission generation is temporarily unavailable.")
	}
	if !policy.Evaluated {
		return false, nil
	}

	sharedratelimit.SetResponseHeaders(echoCtx.Response().Header(), policy.Decision)
	if policy.Decision.Allowed {
		return false, nil
	}

	return true, jsonError(echoCtx, http.StatusTooManyRequests, "generation_rate_limited", "Too many mission previews. Wait a moment and try again.")
}

func (request ApproveMissionRequest) Validate() error {
	if err := validation.ValidateStruct(&request,
		validation.Field(&request.ExpectedDraftRevision,
			validation.Required,
			validation.Min(int64(1)),
		),
	); err != nil {
		return err
	}
	if request.Draft == nil {
		return nil
	}

	return request.Draft.Validate()
}

func (request *ApproveMissionDraftRequest) Validate() error {
	if request == nil {
		return nil
	}
	if err := validation.ValidateStruct(request,
		validation.Field(&request.Title,
			validation.Required,
			validation.Length(2, 160),
		),
		validation.Field(&request.Summary,
			validation.Required,
			validation.Length(10, 2000),
		),
		validation.Field(&request.PreviewQuestions,
			validation.Required,
			validation.Length(1, 8),
			validation.By(validatePreviewQuestions),
		),
	); err != nil {
		return err
	}

	return nil
}

func (request *ApproveMissionDraftRequest) toInput() MissionDraftEdits {
	if request == nil {
		return MissionDraftEdits{}
	}

	return MissionDraftEdits{
		Title:            request.Title,
		Summary:          request.Summary,
		PreviewQuestions: request.PreviewQuestions,
		HasChanges:       true,
	}
}

func (request CreateMissionRequest) Validate() error {
	if err := validation.ValidateStruct(&request,
		validation.Field(&request.MissionID,
			validation.Length(0, 120),
			validation.By(validateOptionalMissionID),
		),
		validation.Field(&request.ExpectedDraftRevision,
			validation.Min(int64(0)),
		),
		validation.Field(&request.ClassName,
			validation.Length(0, 120),
		),
		validation.Field(&request.MissionName,
			validation.Required,
			validation.Length(2, 120),
		),
		validation.Field(&request.Subject,
			validation.Length(0, 80),
		),
		validation.Field(&request.GradeLevel,
			validation.Length(0, 80),
		),
		validation.Field(&request.LearningGoal,
			validation.Required,
			validation.Length(6, 240),
		),
		validation.Field(&request.TeacherPrompt,
			validation.Required,
			validation.Length(10, 1000),
		),
		validation.Field(&request.AudienceType,
			validation.In("", AudienceAll, AudienceSelected),
		),
		validation.Field(&request.TargetStudentIDs,
			validation.By(validateTargetStudentIDs),
		),
		validation.Field(&request.Attachments,
			validation.By(validateAttachments),
		),
	); err != nil {
		return err
	}

	for _, field := range []struct {
		label string
		value string
	}{
		{label: "Mission name", value: request.MissionName},
		{label: "Learning goal", value: request.LearningGoal},
		{label: "Teacher prompt", value: request.TeacherPrompt},
	} {
		if err := validateEducationalField(field.label, field.value); err != nil {
			return err
		}
	}
	if !sharedcontentpolicy.HasEducationalContext(
		request.MissionName,
		request.LearningGoal,
		request.TeacherPrompt,
	) {
		return errors.New("mission name, learning goal, and teacher prompt must focus on grade-appropriate education")
	}

	if request.AudienceType == AudienceSelected && len(request.TargetStudentIDs) == 0 {
		return errors.New("target_student_ids is required when audience_type is selected")
	}
	if request.MissionID == "" && request.ExpectedDraftRevision > 0 {
		return errors.New("mission_id is required with expected_draft_revision")
	}
	if request.MissionID != "" && request.ExpectedDraftRevision <= 0 {
		return errors.New("expected_draft_revision must be greater than zero when mission_id is provided")
	}

	return nil
}

func validateEducationalField(label string, value string) error {
	decision := sharedcontentpolicy.ReviewEducationalText(value)
	if decision.Allowed {
		return nil
	}

	return fmt.Errorf("%s was rejected: %s", label, decision.Message)
}

func validateOptionalMissionID(value any) error {
	missionID, ok := value.(string)
	if !ok || missionID == "" {
		return nil
	}

	return validateMissionID(missionID)
}

func validateTargetStudentIDs(value any) error {
	studentIDs, ok := value.([]string)
	if !ok {
		return nil
	}
	if len(studentIDs) > 100 {
		return errors.New("target_student_ids cannot contain more than 100 students")
	}

	for _, studentID := range studentIDs {
		if err := validation.Validate(studentID, validation.Required, validation.Length(1, 128)); err != nil {
			return errors.New("target_student_ids contains an invalid student id")
		}
	}

	return nil
}

func validatePreviewQuestions(value any) error {
	questions, ok := value.([]orchestrator.PreviewQuestion)
	if !ok {
		return nil
	}
	if len(questions) > orchestrator.StudentPreviewQuestionLimit() {
		return fmt.Errorf("preview_questions cannot contain more than %d questions", orchestrator.StudentPreviewQuestionLimit())
	}
	for _, question := range questions {
		if err := validation.Validate(question.ID, validation.Length(0, 128)); err != nil {
			return errors.New("preview_questions contains an invalid id")
		}
		if err := validation.Validate(question.StageID, validation.Length(0, 128)); err != nil {
			return errors.New("preview_questions contains an invalid stage_id")
		}
		if err := validation.Validate(question.Prompt, validation.Required, validation.Length(2, 1000)); err != nil {
			return errors.New("preview_questions contains an invalid prompt")
		}
		if err := validation.Validate(question.AnswerType, validation.Length(0, 80)); err != nil {
			return errors.New("preview_questions contains an invalid answer_type")
		}
		if err := validatePreviewQuestionTextList(question.Options, 4, 240, "options"); err != nil {
			return err
		}
		if err := validation.Validate(question.ExpectedAnswer, validation.Required, validation.Length(1, 1000)); err != nil {
			return errors.New("preview_questions contains an invalid expected_answer")
		}
		if err := validatePreviewQuestionTextList(question.ExplanationSteps, 8, 400, "explanation_steps"); err != nil {
			return err
		}
		if err := validatePreviewQuestionTextList(question.MisconceptionTraps, 8, 400, "misconception_traps"); err != nil {
			return err
		}
	}

	return nil
}

func validatePreviewQuestionTextList(
	values []string,
	maxItems int,
	maxLength int,
	field string,
) error {
	if len(values) > maxItems {
		return errors.New("preview_questions contains too many " + field)
	}
	for _, value := range values {
		if err := validation.Validate(value, validation.Length(0, maxLength)); err != nil {
			return errors.New("preview_questions contains an invalid " + field + " item")
		}
	}

	return nil
}

func validateAttachments(value any) error {
	attachments, ok := value.([]MissionAttachmentRequest)
	if !ok {
		return nil
	}
	if len(attachments) > maxMissionAttachments {
		return errors.New("attachments cannot contain more than 5 files")
	}

	seen := make(map[string]struct{}, len(attachments))
	for _, attachment := range attachments {
		attachmentUUID := strings.TrimSpace(attachment.UploadcareUUID)
		if attachment.SizeBytes < 0 || attachment.SizeBytes > maxMissionAttachmentSizeBytes {
			return errors.New("each attachment must be 20 MB or smaller")
		}
		if attachmentUUID == "" || strings.TrimSpace(attachment.CDNURL) == "" {
			return errors.New("each attachment requires uploadcare_uuid and cdn_url")
		}
		if _, exists := seen[attachmentUUID]; exists {
			return errors.New("attachments cannot contain duplicate uploadcare_uuid values")
		}
		seen[attachmentUUID] = struct{}{}
		if err := validateMissionAttachmentURL(attachment.CDNURL, attachmentUUID); err != nil {
			return err
		}
		if err := validation.Validate(attachment.FileName, validation.Required, validation.Length(1, 255)); err != nil {
			return errors.New("each attachment requires a valid file_name")
		}
		if err := validation.Validate(attachment.MimeType, validation.Length(0, 160)); err != nil {
			return errors.New("each attachment requires a valid mime_type")
		}
	}

	return nil
}

func validateMissionAttachmentURL(rawURL string, uploadcareUUID string) error {
	parsedURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsedURL.Scheme != "https" || parsedURL.User != nil {
		return errors.New("each attachment requires a valid HTTPS URL")
	}
	if port := parsedURL.Port(); port != "" && port != "443" {
		return errors.New("each attachment requires a valid HTTPS URL")
	}
	if shareduploadcare.IsCDNHost(parsedURL.Hostname()) {
		return validateMissionUploadcareURL(parsedURL, uploadcareUUID)
	}
	if parsedURL.Hostname() == "storage.googleapis.com" {
		return validateMissionPublicStorageURL(parsedURL, uploadcareUUID)
	}

	return errors.New("each attachment URL must use Uploadcare CDN or Cloud Storage")
}

func validateMissionUploadcareURL(parsedURL *url.URL, uploadcareUUID string) error {
	path := "/" + strings.Trim(parsedURL.EscapedPath(), "/") + "/"
	if !strings.Contains(path, "/"+url.PathEscape(uploadcareUUID)+"/") {
		return errors.New("attachment uploadcare_uuid does not match its CDN URL")
	}

	return nil
}

func validateMissionPublicStorageURL(parsedURL *url.URL, uploadcareUUID string) error {
	segments := strings.Split(strings.Trim(parsedURL.EscapedPath(), "/"), "/")
	if len(segments) < 2 || segments[0] == "" || segments[1] == "" {
		return errors.New("each attachment requires a valid Cloud Storage URL")
	}

	path := "/" + strings.Trim(parsedURL.EscapedPath(), "/") + "/"
	if !strings.Contains(path, "/"+url.PathEscape(uploadcareUUID)+"/") {
		return errors.New("attachment uploadcare_uuid does not match its Cloud Storage URL")
	}

	return nil
}

func validateClassID(value string) error {
	return validation.Validate(value,
		validation.Required,
		validation.Length(2, 40),
		validation.Match(classIDPattern),
	)
}

func validateMissionID(value string) error {
	return validation.Validate(value,
		validation.Required,
		validation.Length(4, 120),
	)
}

func toMissionAttachments(input []MissionAttachmentRequest) []MissionAttachment {
	attachments := make([]MissionAttachment, 0, len(input))
	for _, attachment := range input {
		attachments = append(attachments, MissionAttachment{
			UploadcareUUID:   attachment.UploadcareUUID,
			CDNURL:           attachment.CDNURL,
			UploadcareCDNURL: attachment.UploadcareCDNURL,
			FileName:         attachment.FileName,
			MimeType:         attachment.MimeType,
			SizeBytes:        attachment.SizeBytes,
		})
	}

	return attachments
}

func missionRateLimiter(options HandlerOptions) echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:      resolveMissionRateLimitRate(options.MissionRateLimitRate),
			Burst:     resolveMissionRateLimitBurst(options.MissionRateLimitBurst),
			ExpiresIn: resolveMissionRateLimitExpires(options.MissionRateLimitExpires),
		}),
		IdentifierExtractor: func(ctx *echo.Context) (string, error) {
			return "teacher_mission:" + ctx.RealIP(), nil
		},
		DenyHandler: func(ctx *echo.Context, _ string, _ error) error {
			return jsonError(ctx, http.StatusTooManyRequests, "rate_limited", "Too many teacher mission requests.")
		},
	})
}

func resolveMissionTimeout(value time.Duration) time.Duration {
	if value <= 0 {
		return defaultMissionTimeout
	}

	return value
}

func resolveMissionRateLimitRate(value float64) float64 {
	if value <= 0 {
		return defaultMissionRateLimitRate
	}

	return value
}

func resolveMissionRateLimitBurst(value int) int {
	if value <= 0 {
		return defaultMissionRateLimitBurst
	}

	return value
}

func resolveMissionRateLimitExpires(value time.Duration) time.Duration {
	if value <= 0 {
		return defaultMissionRateLimitExpires
	}

	return value
}

func jsonError(ctx *echo.Context, statusCode int, code string, message string) error {
	return ctx.JSON(statusCode, ErrorResponse{
		Error: ErrorResponseBody{
			Code:    code,
			Message: message,
		},
	})
}
