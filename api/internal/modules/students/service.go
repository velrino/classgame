package students

import (
	"context"
	"errors"
	"time"

	"classgame-api/internal/modules/classrooms"
	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/missions"
	"classgame-api/internal/ratelimit"
)

var ErrStudentRequiresAnonymousAuth = errors.New("student session requires anonymous auth")

var ErrStudentNotFound = errors.New("student user not found")

var ErrClassCodeNotFound = errors.New("active class code not found")

var ErrPublishedMissionNotFound = errors.New("published class mission not found")

var ErrMissionCodeNotFound = errors.New("published mission code not found")

var ErrStudentMissionMismatch = errors.New("student mission does not match current session")

var ErrStudentAttemptNotFound = errors.New("student learning attempt not found")

var ErrStudentQuestionNotFound = errors.New("student mission question not found")

var ErrMissionRevisionNotFound = errors.New("published student mission revision not found")

var ErrStudentTutorTurnLimit = errors.New("student tutor turn limit reached")

type MissionReader interface {
	GetByCode(context.Context, string) (missions.MissionDocument, bool, error)
	Get(context.Context, string, string) (missions.MissionDocument, bool, error)
}

type ClassReader interface {
	Get(context.Context, string) (classrooms.TeacherClassDocument, bool, error)
}

type MissionRevisionReader interface {
	GetPublishedRevision(
		context.Context,
		string,
		string,
		int64,
	) (missions.MissionPublishedRevisionDocument, bool, error)
}

type LearningTraceGenerator interface {
	BuildLearningTrace(context.Context, generation.BuildLearningTraceInput) (generation.LearningTraceBlueprint, error)
}

type TutorTurnGenerator interface {
	BuildTutorTurn(context.Context, generation.BuildTutorTurnInput) (generation.TutorTurnBlueprint, error)
}

type ClassLearningGenerator interface {
	BuildClassLearningSynthesis(context.Context, generation.BuildClassLearningSynthesisInput) (generation.ClassLearningSynthesis, error)
}

type Dependencies struct {
	Sessions          SessionStore
	Learning          LearningStore
	ClassLearning     ClassLearningStore
	Missions          MissionReader
	Revisions         MissionRevisionReader
	Classes           ClassReader
	TraceGenerator    LearningTraceGenerator
	TutorGenerator    TutorTurnGenerator
	ClassGenerator    ClassLearningGenerator
	Tasks             LearningTaskEnqueuer
	GenerationLimiter ratelimit.Limiter
	AbandonmentWindow time.Duration
}

type Service struct {
	sessions                  SessionStore
	learning                  LearningStore
	classLearning             ClassLearningStore
	abandonmentWindow         time.Duration
	learningTaskEnqueuer      LearningTaskEnqueuer
	learningGenerationLimiter ratelimit.Limiter
	learningTraceGenerator    LearningTraceGenerator
	tutorTurnGenerator        TutorTurnGenerator
	classLearningGenerator    ClassLearningGenerator
	classReader               ClassReader
	missionReader             MissionReader
	revisionReader            MissionRevisionReader
}

func NewService(dependencies Dependencies) *Service {
	window := dependencies.AbandonmentWindow
	if window <= 0 {
		window = defaultStudentAbandonmentWindow
	}
	return &Service{sessions: dependencies.Sessions, learning: dependencies.Learning, classLearning: dependencies.ClassLearning,
		abandonmentWindow: window, learningTaskEnqueuer: dependencies.Tasks, learningGenerationLimiter: dependencies.GenerationLimiter,
		learningTraceGenerator: dependencies.TraceGenerator, tutorTurnGenerator: dependencies.TutorGenerator, classLearningGenerator: dependencies.ClassGenerator,
		classReader: dependencies.Classes, missionReader: dependencies.Missions, revisionReader: dependencies.Revisions}
}
