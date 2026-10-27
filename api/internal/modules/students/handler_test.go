package students

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"classgame-api/internal/auth"
	"classgame-api/internal/ratelimit"
)

func TestStudentSessionRequestAcceptsNumericClassCode(t *testing.T) {
	request := StudentSessionRequest{
		ClassCode:   "123456",
		StudentName: "Alex",
	}

	if err := request.Validate(); err != nil {
		t.Fatalf("validate class code: %v", err)
	}
}

func TestStudentSessionRequestRejectsAlphanumericClassCode(t *testing.T) {
	request := StudentSessionRequest{
		ClassCode:   "ABC234",
		StudentName: "Alex",
	}

	if err := request.Validate(); err == nil {
		t.Fatal("expected alphanumeric class code to fail validation")
	}
}

func TestStudentSessionRequestRejectsClassStyleID(t *testing.T) {
	request := StudentSessionRequest{
		ClassCode:   "class-with-long-id",
		StudentName: "Alex",
	}

	if err := request.Validate(); err == nil {
		t.Fatal("expected class-style id to fail class code validation")
	}
}

func TestStudentSessionRequestRejectsStudentNameOverMaxLength(t *testing.T) {
	request := StudentSessionRequest{
		ClassCode:   "123456",
		StudentName: strings.Repeat("A", studentNameMaxLength+1),
	}

	if err := request.Validate(); err == nil {
		t.Fatal("expected long student name to fail validation")
	}
}

func TestNormalizeStudentNameLimitsLength(t *testing.T) {
	name := normalizeStudentName(strings.Repeat("A", studentNameMaxLength+1))

	if len([]rune(name)) != studentNameMaxLength {
		t.Fatalf("name length = %d, want %d", len([]rune(name)), studentNameMaxLength)
	}
}

func TestPostStudentSessionValidationDoesNotRequireFirebaseAuth(t *testing.T) {
	service := &fakeUsersService{}
	recorder := postStudentSessionValidation(t, &Handler{
		usersService: service,
	}, `{"class_code":"123456"}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if service.validatedClassCode != "123456" {
		t.Fatalf("validated class code = %q, want %q", service.validatedClassCode, "123456")
	}
	if service.upsertCalled {
		t.Fatal("student validation should not create or update a user")
	}
}

func TestPostStudentSessionValidationReturnsClassNotFound(t *testing.T) {
	recorder := postStudentSessionValidation(t, &Handler{
		usersService: &fakeUsersService{validateErr: ErrClassCodeNotFound},
	}, `{"class_code":"123456"}`)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d; body = %s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"code":"class_not_found"`) {
		t.Fatalf("expected class_not_found response, got %s", recorder.Body.String())
	}
}

func TestPostStudentPresenceReturnsMissionNotFound(t *testing.T) {
	recorder := postStudentPresence(t, &Handler{
		authenticator:         fakeStudentAuthenticator{},
		studentSessionTimeout: time.Second,
		usersService:          &fakeUsersService{presenceErr: ErrPublishedMissionNotFound},
	})

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d; body = %s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"code":"mission_not_found"`) {
		t.Fatalf("expected mission_not_found response, got %s", recorder.Body.String())
	}
}

func TestPostStudentTutorTurnRejectsDistributedRateLimit(t *testing.T) {
	limiter := &fakeTutorRateLimiter{decision: ratelimit.Decision{
		Allowed:    false,
		Limit:      12,
		Remaining:  0,
		RetryAfter: 42 * time.Second,
	}}
	service := &fakeStudentTutorService{}
	recorder := postStudentTutorTurn(t, &Handler{
		authenticator:          fakeStudentAuthenticator{},
		studentLearningTimeout: time.Second,
		studentTutorService:    service,
		tutorRateLimiters: TutorRateLimiters{
			PerStudentMinute: limiter,
		},
	})

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status code = %d, want %d; body = %s", recorder.Code, http.StatusTooManyRequests, recorder.Body.String())
	}
	if service.called {
		t.Fatal("rate-limited request should not call the tutor service")
	}
	if got := recorder.Header().Get("Retry-After"); got != "42" {
		t.Fatalf("Retry-After = %q, want %q", got, "42")
	}
	if !strings.Contains(recorder.Body.String(), `"code":"tutor_rate_limited"`) {
		t.Fatalf("expected tutor_rate_limited response, got %s", recorder.Body.String())
	}
}

func TestPostStudentTutorTurnChecksStudentAndIPQuotas(t *testing.T) {
	studentMinute := allowTutorRateLimiter()
	studentHour := allowTutorRateLimiter()
	ipMinute := allowTutorRateLimiter()
	ipHour := allowTutorRateLimiter()
	service := &fakeStudentTutorService{}
	recorder := postStudentTutorTurn(t, &Handler{
		authenticator:          fakeStudentAuthenticator{},
		studentLearningTimeout: time.Second,
		studentTutorService:    service,
		tutorRateLimiters: TutorRateLimiters{
			PerIPHour:        ipHour,
			PerIPMinute:      ipMinute,
			PerStudentHour:   studentHour,
			PerStudentMinute: studentMinute,
		},
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if !service.called {
		t.Fatal("allowed request should call the tutor service")
	}
	for _, limiter := range []*fakeTutorRateLimiter{studentMinute, studentHour} {
		if len(limiter.identifiers) != 1 || limiter.identifiers[0] != "student:firebase:student-1" {
			t.Fatalf("student limiter identifiers = %#v", limiter.identifiers)
		}
	}
	for _, limiter := range []*fakeTutorRateLimiter{ipMinute, ipHour} {
		if len(limiter.identifiers) != 1 || limiter.identifiers[0] != "ip:192.0.2.1" {
			t.Fatalf("IP limiter identifiers = %#v", limiter.identifiers)
		}
	}
}

func postStudentSessionValidation(t *testing.T, handler *Handler, body string) *httptest.ResponseRecorder {
	t.Helper()

	server := echo.New()
	request := httptest.NewRequest(http.MethodPost, "/users/student-session/validate", strings.NewReader(body))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	ctx := server.NewContext(request, recorder)

	if err := handler.PostStudentSessionValidation(ctx); err != nil {
		t.Fatalf("post student session validation: %v", err)
	}

	return recorder
}

func postStudentTutorTurn(t *testing.T, handler *Handler) *httptest.ResponseRecorder {
	t.Helper()

	server := echo.New()
	request := httptest.NewRequest(
		http.MethodPost,
		"/users/student-tutor-turn",
		strings.NewReader(`{"mission_id":"mission-1","event_id":"event-1","type":"answer","stage_id":"stage-1","question_id":"question-1","prompt":"What is the answer?","student_response":"A","expected_answer":"A","outcome":"correct"}`),
	)
	request.Header.Set(echo.HeaderAuthorization, "Bearer student-token")
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	request.RemoteAddr = "192.0.2.1:4242"
	recorder := httptest.NewRecorder()
	ctx := server.NewContext(request, recorder)

	if err := handler.PostStudentTutorTurn(ctx); err != nil {
		t.Fatalf("post student tutor turn: %v", err)
	}

	return recorder
}

func postStudentPresence(t *testing.T, handler *Handler) *httptest.ResponseRecorder {
	t.Helper()

	server := echo.New()
	request := httptest.NewRequest(
		http.MethodPost,
		"/users/student-presence",
		strings.NewReader(`{"mission_id":"mission-1","class_code":"123456","subject":"Mathematics","stage_id":"stage-1","stage_name":"Stage 1","status":"learning"}`),
	)
	request.Header.Set(echo.HeaderAuthorization, "Bearer student-token")
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	ctx := server.NewContext(request, recorder)

	if err := handler.PostStudentPresence(ctx); err != nil {
		t.Fatalf("post student presence: %v", err)
	}

	return recorder
}

func allowTutorRateLimiter() *fakeTutorRateLimiter {
	return &fakeTutorRateLimiter{decision: ratelimit.Decision{
		Allowed:   true,
		Limit:     12,
		Remaining: 11,
	}}
}

type fakeStudentAuthenticator struct{}

func (fakeStudentAuthenticator) AuthenticateBearerToken(
	context.Context,
	string,
) (auth.Identity, error) {
	return auth.Identity{
		Provider:       auth.IdentityProviderFirebase,
		ProviderUserID: "student-1",
		AuthMethod:     auth.AuthMethodAnonymous,
	}, nil
}

type fakeTutorRateLimiter struct {
	decision    ratelimit.Decision
	err         error
	identifiers []string
}

func (limiter *fakeTutorRateLimiter) Allow(
	_ context.Context,
	identifier string,
) (ratelimit.Decision, error) {
	limiter.identifiers = append(limiter.identifiers, identifier)
	return limiter.decision, limiter.err
}

type fakeStudentTutorService struct {
	called bool
}

func (service *fakeStudentTutorService) SaveStudentTutorTurn(
	ctx context.Context,
	_ auth.Identity,
	input SaveStudentTutorTurnInput,
) (StudentTutorTurn, error) {
	if input.BeforeGeneration != nil {
		if err := input.BeforeGeneration(ctx); err != nil {
			return StudentTutorTurn{}, err
		}
	}
	service.called = true
	return StudentTutorTurn{}, nil
}

type fakeUsersService struct {
	validateErr        error
	validatedClassCode string
	upsertCalled       bool
	presenceErr        error
}

func (service *fakeUsersService) UpsertStudent(
	context.Context,
	auth.Identity,
	string,
	string,
) (UserDocument, error) {
	service.upsertCalled = true
	return UserDocument{}, nil
}

func (service *fakeUsersService) ValidateStudentClassCode(
	_ context.Context,
	classCode string,
) error {
	service.validatedClassCode = classCode
	return service.validateErr
}

func (service *fakeUsersService) StartStudentMission(
	context.Context,
	auth.Identity,
	string,
	string,
	string,
) (UserDocument, error) {
	return UserDocument{}, nil
}

func (service *fakeUsersService) UpdateStudentPresence(
	context.Context,
	auth.Identity,
	string,
	string,
	string,
	string,
	string,
	string,
) (UserDocument, error) {
	return UserDocument{}, service.presenceErr
}
