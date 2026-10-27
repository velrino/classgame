package students

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"classgame-api/internal/auth"
	analyticsmodule "classgame-api/internal/platform/analytics"
	"classgame-api/internal/ratelimit"
	"classgame-api/internal/transport/httpx"
)

const (
	defaultStudentSessionTimeout          = 12 * time.Second
	defaultStudentSessionRateLimitRate    = 0.5
	defaultStudentSessionRateLimitBurst   = 10
	defaultStudentSessionRateLimitExpires = 5 * time.Minute
	defaultStudentLearningTimeout         = 40 * time.Second
	studentNameMaxLength                  = 40
)

var classCodePattern = regexp.MustCompile(`^[0-9]{6}$`)

type Authenticator = auth.Authenticator

type UsersService interface {
	UpsertStudent(
		context.Context,
		auth.Identity,
		string,
		string,
	) (UserDocument, error)
	ValidateStudentClassCode(
		context.Context,
		string,
	) error
	StartStudentMission(
		context.Context,
		auth.Identity,
		string,
		string,
		string,
	) (UserDocument, error)
	UpdateStudentPresence(
		context.Context,
		auth.Identity,
		string,
		string,
		string,
		string,
		string,
		string,
	) (UserDocument, error)
}

type StudentLearningService interface {
	SaveStudentLearningSnapshot(
		context.Context,
		auth.Identity,
		string,
		string,
		StudentLearningSnapshot,
		AttemptScope,
	) (StudentMissionAttemptDocument, error)
	GetCurrentStudentLearningAttempt(
		context.Context,
		auth.Identity,
		string,
	) (StudentMissionAttemptDocument, bool, error)
}

type StudentTutorService interface {
	SaveStudentTutorTurn(
		context.Context,
		auth.Identity,
		SaveStudentTutorTurnInput,
	) (StudentTutorTurn, error)
}

type TutorRateLimiters struct {
	PerIPHour        ratelimit.Limiter
	PerIPMinute      ratelimit.Limiter
	PerStudentHour   ratelimit.Limiter
	PerStudentMinute ratelimit.Limiter
}

type HandlerOptions struct {
	StudentClassReader             StudentClassReader
	Analytics                      analyticsmodule.Tracker
	Authenticator                  Authenticator
	StudentSessionTimeout          time.Duration
	StudentSessionRateLimitRate    float64
	StudentSessionRateLimitBurst   int
	StudentSessionRateLimitExpires time.Duration
	StudentLearningTimeout         time.Duration
	StudentLearningService         StudentLearningService
	StudentTutorService            StudentTutorService
	TutorRateLimiters              TutorRateLimiters
	UsersService                   UsersService
}

type Handler struct {
	studentClassReader     StudentClassReader
	analytics              analyticsmodule.Tracker
	authenticator          Authenticator
	studentSessionTimeout  time.Duration
	studentLearningTimeout time.Duration
	studentLearningService StudentLearningService
	studentTutorService    StudentTutorService
	tutorRateLimiters      TutorRateLimiters
	usersService           UsersService
}

type StudentSessionRequest struct {
	ClassCode   string `json:"class_code"`
	StudentName string `json:"student_name"`
}

type StudentClassCodeRequest struct {
	ClassCode string `json:"class_code"`
}

type StudentClassCodeResponse struct {
	Valid bool `json:"valid"`
}

type StudentSessionResponse struct {
	User    UserResponse           `json:"user"`
	Class   *StudentSessionClass   `json:"class,omitempty"`
	Mission *StudentSessionMission `json:"mission,omitempty"`
}

type StudentMissionRequest struct {
	ClassCode   string `json:"class_code"`
	MissionID   string `json:"mission_id"`
	StudentName string `json:"student_name"`
}

type StudentPresenceRequest struct {
	MissionID string `json:"mission_id"`
	ClassCode string `json:"class_code"`
	Subject   string `json:"subject"`
	StageID   string `json:"stage_id"`
	StageName string `json:"stage_name"`
	Status    string `json:"status"`
}

type StudentLearningSnapshotRequest struct {
	AttemptID         string                  `json:"attempt_id,omitempty"`
	PublishedRevision int64                   `json:"published_revision,omitempty"`
	MissionID         string                  `json:"mission_id"`
	Status            string                  `json:"status"`
	Snapshot          StudentLearningSnapshot `json:"snapshot"`
}

type StudentLearningResponse struct {
	Attempt *StudentLearningResumeResponse `json:"attempt"`
}

type StudentTutorTurnRequest struct {
	AttemptID         string `json:"attempt_id,omitempty"`
	PublishedRevision int64  `json:"published_revision,omitempty"`
	MissionID         string `json:"mission_id"`
	EventID           string `json:"event_id"`
	Type              string `json:"type"`
	StageID           string `json:"stage_id"`
	QuestionID        string `json:"question_id"`
	Prompt            string `json:"prompt"`
	StudentResponse   string `json:"student_response"`
	ExpectedAnswer    string `json:"expected_answer"`
	Outcome           string `json:"outcome"`
	ExplicitReasoning string `json:"explicit_reasoning"`
	Feedback          string `json:"feedback"`
}

type StudentTutorTurnAPIResponse struct {
	Turn StudentTutorTurnResponse `json:"turn"`
}

type ErrorResponse = httpx.ErrorResponse

type ErrorResponseBody = httpx.ErrorResponseBody

func RegisterRoutes(server *echo.Echo, options HandlerOptions) {
	studentLearningService := options.StudentLearningService
	studentTutorService := options.StudentTutorService

	handler := &Handler{
		studentClassReader:     options.StudentClassReader,
		analytics:              options.Analytics,
		authenticator:          options.Authenticator,
		studentSessionTimeout:  resolveStudentSessionTimeout(options.StudentSessionTimeout),
		studentLearningTimeout: resolveStudentLearningTimeout(options.StudentLearningTimeout),
		studentLearningService: studentLearningService,
		studentTutorService:    studentTutorService,
		tutorRateLimiters:      options.TutorRateLimiters,
		usersService:           options.UsersService,
	}

	sessionRateLimiter := studentSessionRateLimiter(options)
	server.GET("/users/student-class", handler.GetStudentClass, studentPresenceRateLimiter())
	server.POST(
		"/users/student-session/validate",
		handler.PostStudentSessionValidation,
		sessionRateLimiter,
	)
	server.POST(
		"/users/student-session",
		handler.PostStudentSession,
		sessionRateLimiter,
	)
	server.POST(
		"/users/student-mission",
		handler.PostStudentMission,
		sessionRateLimiter,
	)
	server.POST(
		"/users/student-presence",
		handler.PostStudentPresence,
		studentPresenceRateLimiter(),
	)
	server.GET(
		"/users/student-learning",
		handler.GetStudentLearning,
		studentLearningRateLimiter(),
	)
	server.POST(
		"/users/student-learning",
		handler.PostStudentLearning,
		studentLearningRateLimiter(),
	)
	server.POST(
		"/users/student-tutor-turn",
		handler.PostStudentTutorTurn,
		studentLearningRateLimiter(),
	)
}

func (handler *Handler) PostStudentSessionValidation(ctx *echo.Context) error {
	var request StudentClassCodeRequest
	if err := ctx.Bind(&request); err != nil {
		return httpx.JSONError(ctx, http.StatusBadRequest, "bad_request", "Invalid JSON body.")
	}
	request.ClassCode = strings.TrimSpace(request.ClassCode)
	if err := request.Validate(); err != nil {
		return httpx.JSONError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}

	if handler.usersService == nil {
		err := errors.New("users student session dependencies are not configured")
		slog.Error("users student session validation not configured", "error", err)

		return httpx.JSONError(ctx, http.StatusInternalServerError, "not_configured", "Users are not configured.")
	}

	sessionContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.studentSessionTimeout)
	defer cancel()

	if err := handler.usersService.ValidateStudentClassCode(sessionContext, request.ClassCode); err != nil {
		if errors.Is(err, ErrClassCodeNotFound) {
			return httpx.JSONError(ctx, http.StatusNotFound, "class_not_found", "Active class code not found.")
		}

		slog.Error("student session validation failed", "error", err)
		return httpx.JSONError(ctx, http.StatusInternalServerError, "session_validation_failed", "Could not validate class code.")
	}

	return ctx.JSON(http.StatusOK, StudentClassCodeResponse{Valid: true})
}

func (handler *Handler) PostStudentSession(ctx *echo.Context) error {
	var request StudentSessionRequest
	if err := ctx.Bind(&request); err != nil {
		return httpx.JSONError(ctx, http.StatusBadRequest, "bad_request", "Invalid JSON body.")
	}
	request.ClassCode = strings.TrimSpace(request.ClassCode)
	if err := request.Validate(); err != nil {
		return httpx.JSONError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}

	if handler.authenticator == nil || handler.usersService == nil {
		err := errors.New("users student session dependencies are not configured")
		slog.Error("users student session not configured", "error", err)

		return httpx.JSONError(ctx, http.StatusInternalServerError, "not_configured", "Users are not configured.")
	}

	sessionContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.studentSessionTimeout)
	defer cancel()

	identity, ok, err := handler.authenticate(sessionContext, ctx)
	if err != nil || !ok {
		return err
	}

	user, err := handler.usersService.UpsertStudent(
		sessionContext,
		identity,
		request.ClassCode,
		request.StudentName,
	)
	if err != nil {
		if errors.Is(err, ErrStudentRequiresAnonymousAuth) {
			return httpx.JSONError(ctx, http.StatusForbidden, "forbidden", "Student sessions require anonymous Firebase auth.")
		}
		if errors.Is(err, ErrClassCodeNotFound) {
			return httpx.JSONError(ctx, http.StatusNotFound, "class_not_found", "Active class code not found.")
		}

		slog.Error("student session upsert failed", "error", err, "auth_method", identity.AuthMethod)
		return httpx.JSONError(ctx, http.StatusInternalServerError, "session_failed", "Could not create student session.")
	}
	analyticsmodule.Track(
		handler.analytics,
		sessionContext,
		analyticsmodule.EventGameJoined,
		identity.ProviderUserID,
		analyticsmodule.Properties{"class_id": user.SessionClass.ID},
	)

	return ctx.JSON(http.StatusOK, StudentSessionResponse{
		User:  ToResponse(user),
		Class: &user.SessionClass,
	})
}

func (handler *Handler) PostStudentMission(ctx *echo.Context) error {
	var request StudentMissionRequest
	if err := ctx.Bind(&request); err != nil {
		return httpx.JSONError(ctx, http.StatusBadRequest, "bad_request", "Invalid JSON body.")
	}
	request.ClassCode = strings.TrimSpace(request.ClassCode)
	request.MissionID = strings.TrimSpace(request.MissionID)
	if err := request.Validate(); err != nil {
		return httpx.JSONError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}
	if handler.authenticator == nil || handler.usersService == nil {
		return httpx.JSONError(ctx, http.StatusInternalServerError, "not_configured", "Users are not configured.")
	}

	requestContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.studentSessionTimeout)
	defer cancel()
	identity, ok, err := handler.authenticate(requestContext, ctx)
	if err != nil || !ok {
		return err
	}
	user, err := handler.usersService.StartStudentMission(
		requestContext,
		identity,
		request.ClassCode,
		request.MissionID,
		request.StudentName,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrStudentRequiresAnonymousAuth):
			return httpx.JSONError(ctx, http.StatusForbidden, "forbidden", "Student missions require anonymous Firebase auth.")
		case errors.Is(err, ErrClassCodeNotFound):
			return httpx.JSONError(ctx, http.StatusNotFound, "class_not_found", "The class is no longer available.")
		case errors.Is(err, ErrPublishedMissionNotFound):
			return httpx.JSONError(ctx, http.StatusNotFound, "mission_not_found", "The published mission is no longer available.")
		default:
			slog.Error("student mission start failed", "error", err)
			return httpx.JSONError(ctx, http.StatusInternalServerError, "mission_start_failed", "Could not start this mission.")
		}
	}
	analyticsmodule.Track(
		handler.analytics,
		requestContext,
		analyticsmodule.EventMissionStarted,
		identity.ProviderUserID,
		analyticsmodule.Properties{
			"attempt_id": user.Activity.CurrentGame.AttemptID,
			"class_id":   user.SessionClass.ID,
			"mission_id": user.SessionMission.ID,
		},
	)

	return ctx.JSON(http.StatusOK, StudentSessionResponse{
		User:    ToResponse(user),
		Class:   &user.SessionClass,
		Mission: &user.SessionMission,
	})
}

func (handler *Handler) PostStudentPresence(ctx *echo.Context) error {
	var request StudentPresenceRequest
	if err := ctx.Bind(&request); err != nil {
		return httpx.JSONError(ctx, http.StatusBadRequest, "bad_request", "Invalid JSON body.")
	}
	if err := request.Validate(); err != nil {
		return httpx.JSONError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}
	if handler.authenticator == nil || handler.usersService == nil {
		slog.Error("student presence dependencies are not configured")
		return httpx.JSONError(ctx, http.StatusInternalServerError, "not_configured", "Users are not configured.")
	}

	requestContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.studentSessionTimeout)
	defer cancel()

	identity, ok, err := handler.authenticate(requestContext, ctx)
	if err != nil || !ok {
		return err
	}

	user, err := handler.usersService.UpdateStudentPresence(
		requestContext,
		identity,
		request.MissionID,
		request.ClassCode,
		request.Subject,
		request.StageID,
		request.StageName,
		request.Status,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrStudentRequiresAnonymousAuth):
			return httpx.JSONError(ctx, http.StatusForbidden, "forbidden", "Student presence requires anonymous Firebase auth.")
		case errors.Is(err, ErrStudentNotFound):
			return httpx.JSONError(ctx, http.StatusNotFound, "student_not_found", "Start a student session before entering the game.")
		case errors.Is(err, ErrClassCodeNotFound):
			return httpx.JSONError(ctx, http.StatusNotFound, "class_not_found", "The class is no longer available.")
		case errors.Is(err, ErrPublishedMissionNotFound):
			return httpx.JSONError(ctx, http.StatusNotFound, "mission_not_found", "The published mission is no longer available.")
		case errors.Is(err, ErrStudentMissionMismatch):
			return httpx.JSONError(ctx, http.StatusConflict, "mission_mismatch", "Presence does not match the active student mission.")
		default:
			slog.Error("student presence update failed", "error", err)
			return httpx.JSONError(ctx, http.StatusInternalServerError, "presence_failed", "Could not update student presence.")
		}
	}

	return ctx.JSON(http.StatusOK, StudentSessionResponse{User: ToResponse(user)})
}

func (handler *Handler) GetStudentLearning(ctx *echo.Context) error {
	missionID := strings.TrimSpace(ctx.QueryParam("mission_id"))
	if err := validation.Validate(missionID, validation.Required, validation.Length(1, 120)); err != nil {
		return httpx.JSONError(ctx, http.StatusBadRequest, "validation_failed", "mission_id: "+err.Error())
	}
	if handler.authenticator == nil || handler.studentLearningService == nil {
		slog.Error("student learning dependencies are not configured")
		return httpx.JSONError(ctx, http.StatusInternalServerError, "not_configured", "Student learning is not configured.")
	}

	requestContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.studentLearningTimeout)
	defer cancel()
	identity, ok, err := handler.authenticate(requestContext, ctx)
	if err != nil || !ok {
		return err
	}

	attempt, exists, err := handler.studentLearningService.GetCurrentStudentLearningAttempt(requestContext, identity, missionID)
	if err != nil {
		return handler.studentLearningError(ctx, err)
	}
	if !exists {
		return ctx.JSON(http.StatusOK, StudentLearningResponse{Attempt: nil})
	}

	resume := ToStudentLearningResume(attempt)
	return ctx.JSON(http.StatusOK, StudentLearningResponse{Attempt: &resume})
}

func (handler *Handler) PostStudentLearning(ctx *echo.Context) error {
	var request StudentLearningSnapshotRequest
	if err := ctx.Bind(&request); err != nil {
		return httpx.JSONError(ctx, http.StatusBadRequest, "bad_request", "Invalid JSON body.")
	}
	if err := request.Validate(); err != nil {
		return httpx.JSONError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}
	if handler.authenticator == nil || handler.studentLearningService == nil {
		slog.Error("student learning dependencies are not configured")
		return httpx.JSONError(ctx, http.StatusInternalServerError, "not_configured", "Student learning is not configured.")
	}

	requestContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.studentLearningTimeout)
	defer cancel()
	identity, ok, err := handler.authenticate(requestContext, ctx)
	if err != nil || !ok {
		return err
	}

	attempt, err := handler.studentLearningService.SaveStudentLearningSnapshot(
		requestContext,
		identity,
		request.MissionID,
		request.Status,
		request.Snapshot,
		AttemptScope{AttemptID: request.AttemptID, PublishedRevision: request.PublishedRevision},
	)
	if err != nil {
		return handler.studentLearningError(ctx, err)
	}
	if strings.EqualFold(attempt.Status, "completed") {
		analyticsmodule.Track(
			handler.analytics,
			requestContext,
			analyticsmodule.EventMissionCompleted,
			identity.ProviderUserID,
			analyticsmodule.Properties{
				"attempt_id": attempt.ID,
				"class_id":   attempt.ClassID,
				"mission_id": attempt.MissionID,
			},
		)
	}

	resume := ToStudentLearningResume(attempt)
	return ctx.JSON(http.StatusOK, StudentLearningResponse{Attempt: &resume})
}

func (handler *Handler) PostStudentTutorTurn(ctx *echo.Context) error {
	var request StudentTutorTurnRequest
	if err := ctx.Bind(&request); err != nil {
		return httpx.JSONError(ctx, http.StatusBadRequest, "bad_request", "Invalid JSON body.")
	}
	if err := request.Validate(); err != nil {
		return httpx.JSONError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}
	if handler.authenticator == nil || handler.studentTutorService == nil {
		slog.Error("student tutor dependencies are not configured")
		return httpx.JSONError(ctx, http.StatusInternalServerError, "not_configured", "Student tutor is not configured.")
	}

	requestContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.studentLearningTimeout)
	defer cancel()
	identity, ok, err := handler.authenticate(requestContext, ctx)
	if err != nil || !ok {
		return err
	}
	limited, err := handler.limitTutorGeneration(requestContext, ctx, identity)
	if err != nil || limited {
		return err
	}

	turn, err := handler.studentTutorService.SaveStudentTutorTurn(
		requestContext,
		identity,
		SaveStudentTutorTurnInput{
			RequestScope:      AttemptScope{AttemptID: request.AttemptID, PublishedRevision: request.PublishedRevision},
			MissionID:         request.MissionID,
			EventID:           request.EventID,
			Type:              request.Type,
			StageID:           request.StageID,
			QuestionID:        request.QuestionID,
			Prompt:            request.Prompt,
			StudentResponse:   request.StudentResponse,
			ExpectedAnswer:    request.ExpectedAnswer,
			Outcome:           request.Outcome,
			ExplicitReasoning: request.ExplicitReasoning,
			Feedback:          request.Feedback,
		},
	)
	if err != nil {
		return handler.studentLearningError(ctx, err)
	}

	return ctx.JSON(http.StatusOK, StudentTutorTurnAPIResponse{
		Turn: ToStudentTutorTurnResponse(turn),
	})
}

func (handler *Handler) limitTutorGeneration(
	ctx context.Context,
	echoCtx *echo.Context,
	identity auth.Identity,
) (bool, error) {
	studentIdentifier := string(identity.Provider) + ":" + identity.ProviderUserID
	ipIdentifier := "ip:" + echoCtx.RealIP()
	policy, err := ratelimit.Evaluate(ctx,
		ratelimit.Rule{Limiter: handler.tutorRateLimiters.PerStudentMinute, Identifier: "student:" + studentIdentifier},
		ratelimit.Rule{Limiter: handler.tutorRateLimiters.PerStudentHour, Identifier: "student:" + studentIdentifier},
		ratelimit.Rule{Limiter: handler.tutorRateLimiters.PerIPMinute, Identifier: ipIdentifier},
		ratelimit.Rule{Limiter: handler.tutorRateLimiters.PerIPHour, Identifier: ipIdentifier},
	)
	if err != nil {
		slog.Error("student tutor rate limiter failed", "error", err)
		return true, httpx.JSONError(echoCtx, http.StatusServiceUnavailable, "rate_limit_unavailable", "Student tutor is temporarily unavailable.")
	}
	if !policy.Evaluated {
		return false, nil
	}

	httpx.SetRateLimitHeaders(echoCtx.Response().Header(), policy.Decision)
	if !policy.Decision.Allowed {
		return true, httpx.JSONError(echoCtx, http.StatusTooManyRequests, "tutor_rate_limited", "Too many tutor requests. Wait a moment and try again.")
	}
	return false, nil
}

func (handler *Handler) studentLearningError(ctx *echo.Context, err error) error {
	switch {
	case errors.Is(err, ErrStudentRequiresAnonymousAuth):
		return httpx.JSONError(ctx, http.StatusForbidden, "forbidden", "Student learning requires anonymous Firebase auth.")
	case errors.Is(err, ErrStudentNotFound):
		return httpx.JSONError(ctx, http.StatusNotFound, "student_not_found", "Start a student session before entering the game.")
	case errors.Is(err, ErrStudentMissionMismatch):
		return httpx.JSONError(ctx, http.StatusConflict, "mission_mismatch", "The learning snapshot does not match the active student mission.")
	case errors.Is(err, ErrStudentAttemptNotFound):
		return httpx.JSONError(ctx, http.StatusNotFound, "attempt_not_found", "The student learning attempt was not found.")
	case errors.Is(err, ErrStudentQuestionNotFound):
		return httpx.JSONError(ctx, http.StatusBadRequest, "question_not_found", "The question is not part of the published mission.")
	case errors.Is(err, ErrMissionRevisionNotFound):
		return httpx.JSONError(ctx, http.StatusConflict, "mission_revision_not_found", "The published mission revision for this attempt is unavailable.")
	case errors.Is(err, ErrStudentTutorTurnLimit):
		return httpx.JSONError(ctx, http.StatusTooManyRequests, "tutor_turn_limit_reached", "This mission has reached its tutor request limit.")
	default:
		slog.Error("student learning request failed", "error", err)
		return httpx.JSONError(ctx, http.StatusInternalServerError, "student_learning_failed", "Could not save student learning.")
	}
}

func (handler *Handler) authenticate(
	ctx context.Context,
	echoCtx *echo.Context,
) (auth.Identity, bool, error) {
	return httpx.Authenticate(ctx, echoCtx, handler.authenticator)
}

func (request StudentSessionRequest) Validate() error {
	return validation.ValidateStruct(&request,
		validation.Field(&request.ClassCode,
			validation.By(validateClassCode),
		),
		validation.Field(&request.StudentName, validation.By(validateStudentName)),
	)
}

func (request StudentClassCodeRequest) Validate() error {
	return validation.ValidateStruct(&request,
		validation.Field(&request.ClassCode,
			validation.By(validateClassCode),
		),
	)
}

func (request StudentMissionRequest) Validate() error {
	return validation.ValidateStruct(&request,
		validation.Field(&request.ClassCode, validation.By(validateClassCode)),
		validation.Field(&request.MissionID, validation.Required, validation.Length(1, 120)),
		validation.Field(&request.StudentName, validation.By(validateStudentName)),
	)
}

func (request StudentPresenceRequest) Validate() error {
	return validation.ValidateStruct(&request,
		validation.Field(&request.MissionID, validation.Required, validation.Length(1, 120)),
		validation.Field(&request.ClassCode, validation.By(validateClassCode)),
		validation.Field(&request.Subject, validation.Required, validation.Length(2, 80)),
		validation.Field(&request.StageID, validation.Required, validation.Length(1, 80)),
		validation.Field(&request.StageName, validation.Required, validation.Length(1, 120)),
		validation.Field(&request.Status,
			validation.Required,
			validation.In("exploring", "learning", "thinking", "reviewing", "paused", "completed"),
		),
	)
}

func (request StudentLearningSnapshotRequest) Validate() error {
	if err := validateAttemptScopeFields(request.AttemptID, request.PublishedRevision); err != nil {
		return err
	}
	if err := validation.ValidateStruct(&request,
		validation.Field(&request.MissionID, validation.Required, validation.Length(1, 120)),
		validation.Field(&request.Status,
			validation.Required,
			validation.In("exploring", "learning", "thinking", "reviewing", "paused", "completed"),
		),
	); err != nil {
		return err
	}
	if len(request.Snapshot.Progress) == 0 || len(request.Snapshot.Progress) > 12 {
		return errors.New("snapshot.progress must contain between 1 and 12 stages")
	}
	if len(request.Snapshot.Notes) > 40 {
		return errors.New("snapshot.notes cannot contain more than 40 notes")
	}
	if len(request.Snapshot.Decisions) > 40 {
		return errors.New("snapshot.decisions cannot contain more than 40 decisions")
	}
	if len(request.Snapshot.Events) > 160 {
		return errors.New("snapshot.events cannot contain more than 160 events")
	}
	if request.Snapshot.Revision <= 0 {
		return errors.New("snapshot.revision must be greater than zero")
	}
	if len(request.Snapshot.Checkpoint.Lines) > 12 {
		return errors.New("snapshot.checkpoint.lines cannot contain more than 12 lines")
	}
	checkpoint := request.Snapshot.Checkpoint
	for field, value := range map[string]string{
		"stage_id":               checkpoint.StageID,
		"question_id":            checkpoint.QuestionID,
		"wrong_choice":           checkpoint.WrongChoice,
		"wrong_misconception_id": checkpoint.WrongMisconceptionID,
		"decision_target":        checkpoint.DecisionTarget,
	} {
		if err := validation.Validate(value, validation.Length(0, 120)); err != nil {
			return fmt.Errorf("snapshot.checkpoint.%s: %w", field, err)
		}
	}
	for field, value := range map[string]string{
		"decision_message": checkpoint.DecisionMessage,
		"decision_why":     checkpoint.DecisionWhy,
		"tutor_message":    checkpoint.TutorMessage,
		"why_prompt":       checkpoint.WhyPrompt,
	} {
		if err := validation.Validate(value, validation.Length(0, 500)); err != nil {
			return fmt.Errorf("snapshot.checkpoint.%s: %w", field, err)
		}
	}
	if err := validation.Validate(
		checkpoint.Phase,
		validation.In("", "question", "correct", "why", "explain", "feedback", "thinking", "decision"),
	); err != nil {
		return fmt.Errorf("snapshot.checkpoint.phase: %w", err)
	}
	if err := validation.Validate(
		checkpoint.DecisionAction,
		validation.In("", "next", "complete", "redirect", "mission-complete", "erase-note"),
	); err != nil {
		return fmt.Errorf("snapshot.checkpoint.decision_action: %w", err)
	}
	if request.Snapshot.Preferences.DifficultyBias < -1 || request.Snapshot.Preferences.DifficultyBias > 1 {
		return errors.New("snapshot.preferences.difficulty_bias must be -1, 0, or 1")
	}
	if err := validation.Validate(
		request.Snapshot.Preferences.ExplanationStyle,
		validation.Required,
		validation.In("steps", "visual"),
	); err != nil {
		return fmt.Errorf("snapshot.preferences.explanation_style: %w", err)
	}
	if err := validation.Validate(request.Snapshot.ReturnTo, validation.Length(0, 80)); err != nil {
		return fmt.Errorf("snapshot.return_to: %w", err)
	}
	if err := validation.Validate(request.Snapshot.LastPod, validation.Length(0, 80)); err != nil {
		return fmt.Errorf("snapshot.last_pod: %w", err)
	}

	seenStages := make(map[string]struct{}, len(request.Snapshot.Progress))
	for index, stage := range request.Snapshot.Progress {
		if err := validation.Validate(stage.StageID, validation.Required, validation.Length(1, 80)); err != nil {
			return fmt.Errorf("snapshot.progress[%d].stage_id: %w", index, err)
		}
		if _, exists := seenStages[stage.StageID]; exists {
			return fmt.Errorf("snapshot.progress[%d].stage_id is duplicated", index)
		}
		seenStages[stage.StageID] = struct{}{}
		if err := validation.Validate(stage.Status, validation.Required, validation.In("locked", "ready", "review", "complete")); err != nil {
			return fmt.Errorf("snapshot.progress[%d].status: %w", index, err)
		}
		if stage.Correct < 0 || stage.Correct > 100 || stage.Required < 0 || stage.Required > 100 {
			return fmt.Errorf("snapshot.progress[%d] counts are outside the allowed range", index)
		}
		if len(stage.Queue) > 40 || len(stage.Pool) > 40 {
			return fmt.Errorf("snapshot.progress[%d] contains too many queued questions", index)
		}
	}

	for index, note := range request.Snapshot.Notes {
		if err := validation.Validate(note.ID, validation.Required, validation.Length(1, 100)); err != nil {
			return fmt.Errorf("snapshot.notes[%d].id: %w", index, err)
		}
		if err := validation.Validate(note.Text, validation.Required, validation.Length(1, 320)); err != nil {
			return fmt.Errorf("snapshot.notes[%d].text: %w", index, err)
		}
		if err := validation.Validate(note.Kind, validation.Required, validation.In("misconception", "habit")); err != nil {
			return fmt.Errorf("snapshot.notes[%d].kind: %w", index, err)
		}
		if len(note.Evidence) > 12 {
			return fmt.Errorf("snapshot.notes[%d].evidence contains too many items", index)
		}
	}

	for index, decision := range request.Snapshot.Decisions {
		if err := validation.Validate(decision.ID, validation.Required, validation.Length(1, 100)); err != nil {
			return fmt.Errorf("snapshot.decisions[%d].id: %w", index, err)
		}
		if err := validation.Validate(decision.Action, validation.Required, validation.In("next", "complete", "redirect", "mission-complete", "erase-note")); err != nil {
			return fmt.Errorf("snapshot.decisions[%d].action: %w", index, err)
		}
		if len([]rune(decision.Message)) > 240 || len([]rune(decision.Why)) > 320 {
			return fmt.Errorf("snapshot.decisions[%d] text is too long", index)
		}
	}

	allowedEventTypes := map[string]struct{}{
		"question":   {},
		"answer":     {},
		"reasoning":  {},
		"feedback":   {},
		"adaptation": {},
		"mastery":    {},
	}
	allowedOutcomes := map[string]struct{}{
		"neutral":   {},
		"correct":   {},
		"incorrect": {},
		"support":   {},
		"complete":  {},
	}
	seenEvents := make(map[string]struct{}, len(request.Snapshot.Events))
	for index, event := range request.Snapshot.Events {
		if err := validation.Validate(event.ID, validation.Required, validation.Length(1, 100)); err != nil {
			return fmt.Errorf("snapshot.events[%d].id: %w", index, err)
		}
		if _, exists := seenEvents[event.ID]; exists {
			return fmt.Errorf("snapshot.events[%d].id is duplicated", index)
		}
		seenEvents[event.ID] = struct{}{}
		if _, allowed := allowedEventTypes[event.Type]; !allowed {
			return fmt.Errorf("snapshot.events[%d].type is invalid", index)
		}
		if _, allowed := allowedOutcomes[event.Outcome]; !allowed {
			return fmt.Errorf("snapshot.events[%d].outcome is invalid", index)
		}
		if event.Sequence < 1 || event.Sequence > 10000 {
			return fmt.Errorf("snapshot.events[%d].sequence is outside the allowed range", index)
		}
		if len([]rune(event.Title)) > 160 || len([]rune(event.Detail)) > 420 {
			return fmt.Errorf("snapshot.events[%d] text is too long", index)
		}
	}

	return nil
}

func (request StudentTutorTurnRequest) Validate() error {
	if err := validateAttemptScopeFields(request.AttemptID, request.PublishedRevision); err != nil {
		return err
	}
	if err := validation.ValidateStruct(&request,
		validation.Field(&request.MissionID, validation.Required, validation.Length(1, 120)),
		validation.Field(&request.EventID, validation.Required, validation.Length(1, 120)),
		validation.Field(&request.Type,
			validation.Required,
			validation.In("answer", "reasoning", "feedback", "resume"),
		),
		validation.Field(&request.StageID, validation.Required, validation.Length(1, 120)),
		validation.Field(&request.QuestionID, validation.Length(0, 120)),
		validation.Field(&request.Prompt, validation.Required, validation.Length(1, 500)),
		validation.Field(&request.StudentResponse, validation.Length(0, 800)),
		validation.Field(&request.ExpectedAnswer, validation.Length(0, 500)),
		validation.Field(&request.Outcome,
			validation.Required,
			validation.In("neutral", "correct", "incorrect", "support", "complete"),
		),
		validation.Field(&request.ExplicitReasoning, validation.Length(0, 800)),
		validation.Field(&request.Feedback, validation.Length(0, 80)),
	); err != nil {
		return err
	}
	if request.Type == "reasoning" && strings.TrimSpace(request.ExplicitReasoning) == "" {
		return errors.New("explicit_reasoning is required for a reasoning turn")
	}
	if request.Type == "feedback" {
		if err := validation.Validate(
			request.Feedback,
			validation.Required,
			validation.In("helpful", "confusing", "too-easy", "too-hard"),
		); err != nil {
			return fmt.Errorf("feedback: %w", err)
		}
	}

	return nil
}

func validateClassCode(value any) error {
	return validation.Validate(value,
		validation.Required,
		validation.Length(6, 6),
		validation.Match(classCodePattern),
	)
}

func validateStudentName(value any) error {
	studentName, ok := value.(string)
	if !ok {
		return errors.New("student_name must be a string")
	}

	return validation.Validate(
		normalizeText(studentName),
		validation.Required,
		validation.Length(2, studentNameMaxLength),
	)
}

func studentSessionRateLimiter(options HandlerOptions) echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:      resolveStudentSessionRateLimitRate(options.StudentSessionRateLimitRate),
			Burst:     resolveStudentSessionRateLimitBurst(options.StudentSessionRateLimitBurst),
			ExpiresIn: resolveStudentSessionRateLimitExpires(options.StudentSessionRateLimitExpires),
		}),
		IdentifierExtractor: func(ctx *echo.Context) (string, error) {
			return "student_session:" + ctx.RealIP(), nil
		},
		DenyHandler: func(ctx *echo.Context, _ string, _ error) error {
			return httpx.JSONError(ctx, http.StatusTooManyRequests, "rate_limited", "Too many student session attempts.")
		},
	})
}

func studentPresenceRateLimiter() echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:      1,
			Burst:     12,
			ExpiresIn: 5 * time.Minute,
		}),
		IdentifierExtractor: func(ctx *echo.Context) (string, error) {
			tokenHash := sha256.Sum256([]byte(ctx.Request().Header.Get(echo.HeaderAuthorization)))
			return fmt.Sprintf("student_presence:%x", tokenHash[:8]), nil
		},
		DenyHandler: func(ctx *echo.Context, _ string, _ error) error {
			return httpx.JSONError(ctx, http.StatusTooManyRequests, "rate_limited", "Too many presence updates.")
		},
	})
}

func studentLearningRateLimiter() echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:      2,
			Burst:     24,
			ExpiresIn: 5 * time.Minute,
		}),
		IdentifierExtractor: func(ctx *echo.Context) (string, error) {
			tokenHash := sha256.Sum256([]byte(ctx.Request().Header.Get(echo.HeaderAuthorization)))
			return fmt.Sprintf("student_learning:%x", tokenHash[:8]), nil
		},
		DenyHandler: func(ctx *echo.Context, _ string, _ error) error {
			return httpx.JSONError(ctx, http.StatusTooManyRequests, "rate_limited", "Too many student learning updates.")
		},
	})
}

func resolveStudentSessionTimeout(value time.Duration) time.Duration {
	if value <= 0 {
		return defaultStudentSessionTimeout
	}

	return value
}

func resolveStudentLearningTimeout(value time.Duration) time.Duration {
	if value <= 0 {
		return defaultStudentLearningTimeout
	}

	return value
}

func resolveStudentSessionRateLimitRate(value float64) float64 {
	if value <= 0 {
		return defaultStudentSessionRateLimitRate
	}

	return value
}

func resolveStudentSessionRateLimitBurst(value int) int {
	if value <= 0 {
		return defaultStudentSessionRateLimitBurst
	}

	return value
}

func resolveStudentSessionRateLimitExpires(value time.Duration) time.Duration {
	if value <= 0 {
		return defaultStudentSessionRateLimitExpires
	}

	return value
}

func validateAttemptScopeFields(attemptID string, revision int64) error {
	if revision < 0 {
		return errors.New("published_revision must be zero or greater")
	}
	if len(attemptID) > 120 {
		return errors.New("attempt_id is too long")
	}
	if trimmed := strings.TrimSpace(attemptID); trimmed != "" {
		matched, _ := regexp.MatchString(`^[A-Za-z0-9_-]{1,120}$`, trimmed)
		if !matched {
			return errors.New("attempt_id is invalid")
		}
	}
	return nil
}
