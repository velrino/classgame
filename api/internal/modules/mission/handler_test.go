package mission

import (
	"classgame-api/internal/pagination"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"classgame-api/internal/modules/orchestrator"
	sharedauth "classgame-api/internal/shared/auth"
	sharedratelimit "classgame-api/internal/shared/ratelimit"
)

func TestCreateMissionRequestAllowsEveryoneAudience(t *testing.T) {
	request := validCreateMissionRequest()
	request.AudienceType = AudienceAll

	if err := request.Validate(); err != nil {
		t.Fatalf("validate everyone audience: %v", err)
	}
}

func TestCreateMissionRequestRequiresSelectedStudents(t *testing.T) {
	request := validCreateMissionRequest()
	request.AudienceType = AudienceSelected

	if err := request.Validate(); err == nil {
		t.Fatal("expected selected audience without students to fail validation")
	}
}

func TestMissionPreviewStopsBeforeGenerationWhenRedisQuotaIsExceeded(t *testing.T) {
	service := &fakeMissionService{}
	server := echo.New()
	RegisterRoutes(server, HandlerOptions{
		Authenticator: fakeMissionAuthenticator{},
		GenerationRateLimiters: GenerationRateLimiters{
			PerTeacherMinute: fakeMissionGenerationLimiter{decision: sharedratelimit.Decision{
				Allowed:    false,
				Limit:      6,
				Remaining:  0,
				RetryAfter: 30 * time.Second,
			}},
		},
		MissionService: service,
	})
	request := httptest.NewRequest(
		http.MethodPost,
		"/teacher-classes/AB/missions/preview",
		strings.NewReader(`{"mission_name":"Fractions review","subject":"Mathematics","learning_goal":"Review fraction operations","teacher_prompt":"Help students explain each fraction step."}`),
	)
	request.Header.Set(echo.HeaderAuthorization, "Bearer teacher-token")
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status code = %d, want %d; body = %s", recorder.Code, http.StatusTooManyRequests, recorder.Body.String())
	}
	if service.previewCalled {
		t.Fatal("rate-limited request should not generate a mission preview")
	}
	if got := recorder.Header().Get("Retry-After"); got != "30" {
		t.Fatalf("Retry-After = %q, want %q", got, "30")
	}
}

func TestMissionPreviewStreamEmitsProgressAndDone(t *testing.T) {
	service := &fakeMissionService{}
	server := echo.New()
	RegisterRoutes(server, HandlerOptions{
		Authenticator:  fakeMissionAuthenticator{},
		MissionService: service,
	})
	request := httptest.NewRequest(
		http.MethodPost,
		"/teacher-classes/AB/missions/preview/stream",
		strings.NewReader(`{"class_name":"Period 3","mission_name":"Fractions review","subject":"Mathematics","learning_goal":"Review fraction operations","teacher_prompt":"Help students explain each fraction step."}`),
	)
	request.Header.Set(echo.HeaderAuthorization, "Bearer teacher-token")
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if got := recorder.Header().Get(echo.HeaderContentType); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}
	if !service.previewCalled {
		t.Fatal("stream request should generate a mission preview")
	}

	body := recorder.Body.String()
	for _, expected := range []string{
		"event: progress",
		`"stage":"queued"`,
		`"stage":"generating_mission"`,
		"event: done",
		`"id":"mission-1"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("stream body missing %q; body = %s", expected, body)
		}
	}
}

func validCreateMissionRequest() CreateMissionRequest {
	return CreateMissionRequest{
		MissionName:   "Fractions review",
		Subject:       "Mathematics",
		LearningGoal:  "Review fraction operations",
		TeacherPrompt: "Help students explain each fraction step.",
	}
}

type fakeMissionAuthenticator struct{}

func (fakeMissionAuthenticator) AuthenticateBearerToken(
	context.Context,
	string,
) (sharedauth.Identity, error) {
	return sharedauth.Identity{
		Provider:       sharedauth.IdentityProviderFirebase,
		ProviderUserID: "teacher-1",
		AuthMethod:     sharedauth.AuthMethodGoogle,
	}, nil
}

type fakeMissionGenerationLimiter struct {
	decision sharedratelimit.Decision
}

func (limiter fakeMissionGenerationLimiter) Allow(
	context.Context,
	string,
) (sharedratelimit.Decision, error) {
	return limiter.decision, nil
}

type fakeMissionService struct {
	previewCalled bool
}

func (*fakeMissionService) Approve(
	context.Context,
	sharedauth.Identity,
	string,
	string,
	ApproveMissionInput,
) (MissionDocument, error) {
	return MissionDocument{}, nil
}

func (*fakeMissionService) Delete(
	context.Context,
	sharedauth.Identity,
	string,
	string,
) error {
	return nil
}

func (*fakeMissionService) ListPage(context.Context, sharedauth.Identity, string, pagination.Params) ([]MissionDocument, string, error) {
	return nil, "", nil
}

func (service *fakeMissionService) Preview(
	_ context.Context,
	_ sharedauth.Identity,
	input CreateMissionInput,
) (MissionDocument, error) {
	service.previewCalled = true
	if input.Progress != nil {
		input.Progress(orchestrator.MissionProgress{
			Stage:   "generating_mission",
			Message: "Building the playable mission preview.",
			Percent: 72,
		})
	}
	return fakePreviewMission(), nil
}

func fakePreviewMission() MissionDocument {
	now := time.Date(2026, 8, 29, 23, 45, 0, 0, time.UTC)

	return MissionDocument{
		ID:                    "mission-1",
		ClassID:               "AB",
		ClassName:             "Period 3",
		MissionName:           "Fractions review",
		Title:                 "Fractions Review",
		Summary:               "Students practice explaining each step in fraction operations.",
		Subject:               "Mathematics",
		LearningGoal:          "Review fraction operations",
		TeacherPrompt:         "Help students explain each fraction step.",
		AudienceType:          AudienceAll,
		Status:                StatusDraft,
		DraftRevision:         1,
		HasUnpublishedChanges: true,
		CreatedAt:             now,
		UpdatedAt:             now,
		PreviewQuestions: []orchestrator.PreviewQuestion{
			{
				ID:             "question-1",
				Prompt:         "What is 1/4 + 1/4?",
				Options:        []string{"1/2", "1/4", "2/8", "2/4"},
				ExpectedAnswer: "1/2",
			},
		},
	}
}
