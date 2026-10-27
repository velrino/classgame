package students

import (
	"context"
	"errors"
	"testing"

	"classgame-api/internal/auth"
	"classgame-api/internal/modules/classrooms"
	"classgame-api/internal/modules/missions"
	"classgame-api/internal/ratelimit"
)

func TestValidateStudentClassCodeAcceptsActiveClass(t *testing.T) {
	reader := &fakeClassReader{
		teacherClass: classrooms.TeacherClassDocument{
			ClassID:   "123456",
			ClassCode: "123456",
			Status:    "active",
		},
		exists: true,
	}
	service := NewService(Dependencies{Classes: reader})

	if err := service.ValidateStudentClassCode(context.Background(), " 123456 "); err != nil {
		t.Fatalf("validate student class code: %v", err)
	}
	if reader.code != "123456" {
		t.Fatalf("class code = %q, want %q", reader.code, "123456")
	}
}

func TestValidateStudentClassCodeRejectsMissingClass(t *testing.T) {
	service := NewService(Dependencies{Classes: &fakeClassReader{}})

	err := service.ValidateStudentClassCode(context.Background(), "123456")
	if !errors.Is(err, ErrClassCodeNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrClassCodeNotFound)
	}
}

func TestValidateStudentClassCodeRejectsInactiveClass(t *testing.T) {
	service := NewService(Dependencies{Classes: &fakeClassReader{
		teacherClass: classrooms.TeacherClassDocument{
			ClassID:   "123456",
			ClassCode: "123456",
			Status:    "deleted",
		},
		exists: true,
	}})

	err := service.ValidateStudentClassCode(context.Background(), "123456")
	if !errors.Is(err, ErrClassCodeNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrClassCodeNotFound)
	}
}

func TestUpdateStudentPresenceRejectsDeletedMission(t *testing.T) {
	service := NewService(Dependencies{Missions: &fakeMissionReader{}, Classes: activeFakeClassReader()})

	_, err := service.UpdateStudentPresence(
		context.Background(),
		anonymousStudentIdentity(),
		"mission-1",
		"123456",
		"Mathematics",
		"stage-1",
		"Stage 1",
		"learning",
	)
	if !errors.Is(err, ErrPublishedMissionNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrPublishedMissionNotFound)
	}
}

func TestUpdateStudentPresenceRejectsMissingClass(t *testing.T) {
	service := NewService(Dependencies{Missions: &fakeMissionReader{}, Classes: &fakeClassReader{}})

	_, err := service.UpdateStudentPresence(
		context.Background(),
		anonymousStudentIdentity(),
		"mission-1",
		"123456",
		"Mathematics",
		"stage-1",
		"Stage 1",
		"learning",
	)
	if !errors.Is(err, ErrClassCodeNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrClassCodeNotFound)
	}
}

func TestExistingStudentTutorTurnMakesEventIDIdempotent(t *testing.T) {
	existing := StudentTutorTurn{
		ID:       tutorTurnID("event-1"),
		Sequence: 7,
	}

	turn, exists := existingStudentTutorTurn([]StudentTutorTurn{existing}, "event-1")
	if !exists {
		t.Fatal("expected existing tutor turn")
	}
	if turn.Sequence != existing.Sequence {
		t.Fatalf("sequence = %d, want %d", turn.Sequence, existing.Sequence)
	}
}

func TestStudentTutorTurnLimitMatchesThreeQuestionMissionBudget(t *testing.T) {
	turns := make([]StudentTutorTurn, maxGeneratedTutorTurnsPerAttempt)
	if !studentTutorTurnLimitReached(turns) {
		t.Fatalf("expected %d tutor turns to reach the attempt limit", maxGeneratedTutorTurnsPerAttempt)
	}
	if studentTutorTurnLimitReached(turns[:len(turns)-1]) {
		t.Fatalf("expected fewer than %d tutor turns to remain allowed", maxGeneratedTutorTurnsPerAttempt)
	}
}

func TestLearningGenerationQuotaFailsClosedByClass(t *testing.T) {
	limiter := &fakeLearningGenerationLimiter{
		decision: ratelimit.Decision{Allowed: false},
	}
	service := &Service{learningGenerationLimiter: limiter}

	if service.allowLearningGeneration(context.Background(), "class-1") {
		t.Fatal("expected denied class quota to disable GenAI generation")
	}
	if len(limiter.identifiers) != 1 || limiter.identifiers[0] != "class:class-1" {
		t.Fatalf("limiter identifiers = %#v", limiter.identifiers)
	}
}

func TestLearningGenerationQuotaFailsClosedWhenRedisIsUnavailable(t *testing.T) {
	service := &Service{learningGenerationLimiter: &fakeLearningGenerationLimiter{
		err: errors.New("redis unavailable"),
	}}

	if service.allowLearningGeneration(context.Background(), "class-1") {
		t.Fatal("expected Redis failure to disable GenAI generation")
	}
}

type fakeMissionReader struct {
	classID   string
	code      string
	err       error
	exists    bool
	mission   missions.MissionDocument
	missionID string
}

type fakeClassReader struct {
	code         string
	err          error
	exists       bool
	teacherClass classrooms.TeacherClassDocument
}

type fakeLearningGenerationLimiter struct {
	decision    ratelimit.Decision
	err         error
	identifiers []string
}

func (limiter *fakeLearningGenerationLimiter) Allow(
	_ context.Context,
	identifier string,
) (ratelimit.Decision, error) {
	limiter.identifiers = append(limiter.identifiers, identifier)
	return limiter.decision, limiter.err
}

func (reader *fakeMissionReader) GetByCode(
	_ context.Context,
	missionCode string,
) (missions.MissionDocument, bool, error) {
	reader.code = missionCode
	return reader.mission, reader.exists, reader.err
}

func (reader *fakeMissionReader) Get(
	_ context.Context,
	classID string,
	missionID string,
) (missions.MissionDocument, bool, error) {
	reader.classID = classID
	reader.missionID = missionID
	return reader.mission, reader.exists, reader.err
}

func (reader *fakeClassReader) Get(
	_ context.Context,
	classCode string,
) (classrooms.TeacherClassDocument, bool, error) {
	reader.code = classCode
	return reader.teacherClass, reader.exists, reader.err
}

func activeFakeClassReader() *fakeClassReader {
	return &fakeClassReader{
		teacherClass: classrooms.TeacherClassDocument{
			ClassID:   "123456",
			ClassCode: "123456",
			Status:    "active",
		},
		exists: true,
	}
}

func anonymousStudentIdentity() auth.Identity {
	return auth.Identity{
		Provider:       auth.IdentityProviderFirebase,
		ProviderUserID: "student-1",
		AuthMethod:     auth.AuthMethodAnonymous,
	}
}
