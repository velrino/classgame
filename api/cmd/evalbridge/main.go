// evalbridge runs synthetic offline cases through production policy boundaries.
// It never loads application configuration, credentials, or provider clients.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"classgame-api/internal/app"
	"classgame-api/internal/auth"
	"classgame-api/internal/modules/classrooms"
	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/missions"
	"classgame-api/internal/modules/students"
	"classgame-api/internal/platform/genai"
	"classgame-api/internal/platform/objectstorage"
)

type batch struct {
	TutorContext generation.BuildTutorTurnInput `json:"tutor_context"`
	Mission      missions.MissionDocument       `json:"mission"`
	Cases        []evalCase                     `json:"cases"`
}

type securityInput struct {
	Action         string                          `json:"action"`
	Identity       auth.Identity                   `json:"identity"`
	Emails         string                          `json:"emails"`
	Proxies        string                          `json:"proxies"`
	Peer           string                          `json:"peer"`
	Forwarded      string                          `json:"forwarded"`
	Class          classrooms.TeacherClassDocument `json:"class"`
	ExistingMember bool                            `json:"existing_member"`
	Now            time.Time                       `json:"now"`
}

type evalCase struct {
	Security     *securityInput     `json:"security,omitempty"`
	ID           string             `json:"id"`
	Kind         string             `json:"kind"`
	Tutor        *tutorInput        `json:"tutor,omitempty"`
	Auth         *authInput         `json:"auth,omitempty"`
	Projection   *projectionInput   `json:"projection,omitempty"`
	Learning     *learningInput     `json:"learning,omitempty"`
	Scope        *scopeInput        `json:"scope,omitempty"`
	Memory       *memoryInput       `json:"memory,omitempty"`
	ObjectPath   *objectPathInput   `json:"object_path,omitempty"`
	Evidence     *evidenceInput     `json:"evidence,omitempty"`
	TutorEvent   *tutorEventInput   `json:"tutor_event,omitempty"`
	Finalization *finalizationInput `json:"finalization,omitempty"`
}

type tutorInput struct {
	ProviderMode    string                        `json:"provider_mode"`
	RetrievedChunks *[]generation.KnowledgeChunk  `json:"retrieved_chunks,omitempty"`
	Response        generation.TutorTurnBlueprint `json:"response"`
}

type authInput struct {
	AllowedProviders string          `json:"allowed_providers"`
	Method           auth.AuthMethod `json:"method"`
	Disabled         bool            `json:"disabled"`
}

type projectionInput struct {
	Turn students.StudentTutorTurn `json:"turn"`
}

type learningInput struct {
	Snapshot        students.StudentLearningSnapshot `json:"snapshot"`
	Turns           []students.StudentTutorTurn      `json:"turns"`
	RequestedStatus string                           `json:"requested_status"`
	AcceptedAnswers []students.AcceptedAnswerFact    `json:"accepted_answers,omitempty"`
}

// These fixtures map private server state into production types. They do not
// reproduce the policies under evaluation or open a database connection.
type evidenceInput struct {
	Attempt             students.StudentMissionAttemptDocument `json:"attempt"`
	SchemaVersion       int                                    `json:"schema_version"`
	GeneratedCount      int64                                  `json:"generated_count"`
	LegacyIncomplete    bool                                   `json:"legacy_incomplete"`
	RecentFeedbackCount int                                    `json:"recent_feedback_count"`
}
type eventRequest struct {
	StudentID         string `json:"student_id"`
	AttemptID         string `json:"attempt_id"`
	MissionID         string `json:"mission_id"`
	EventID           string `json:"event_id"`
	Type              string `json:"type"`
	QuestionID        string `json:"question_id"`
	StudentResponse   string `json:"student_response"`
	ExplicitReasoning string `json:"explicit_reasoning"`
	Feedback          string `json:"feedback"`
	Revision          int64  `json:"revision"`
}

func (fixture eventRequest) input() students.SaveStudentTutorTurnInput {
	return students.SaveStudentTutorTurnInput{Identity: auth.Identity{ProviderUserID: fixture.StudentID}, ExpectedAttemptID: fixture.AttemptID, ExpectedPublishedRevision: fixture.Revision, MissionID: fixture.MissionID, EventID: fixture.EventID, Type: fixture.Type, QuestionID: fixture.QuestionID, StudentResponse: fixture.StudentResponse, ExplicitReasoning: fixture.ExplicitReasoning, Feedback: fixture.Feedback}
}

type tutorEventInput struct {
	Stored       eventRequest              `json:"stored"`
	Request      *eventRequest             `json:"request,omitempty"`
	State        string                    `json:"state"`
	OwnerToken   string                    `json:"owner_token"`
	RequestOwner string                    `json:"request_owner"`
	LeaseUntil   time.Time                 `json:"lease_until"`
	Now          time.Time                 `json:"now"`
	Turn         students.StudentTutorTurn `json:"turn"`
}
type finalizationAttemptInput struct {
	Attempt            students.StudentMissionAttemptDocument `json:"attempt"`
	SchemaVersion      int                                    `json:"schema_version"`
	EvidenceRevision   int64                                  `json:"evidence_revision"`
	Token              string                                 `json:"token"`
	LeaseUntil         time.Time                              `json:"lease_until"`
	DispatchGeneration int64                                  `json:"dispatch_generation"`
}

func (fixture finalizationAttemptInput) document() students.StudentMissionAttemptDocument {
	attempt := fixture.Attempt
	attempt.EvidenceSchemaVersion, attempt.EvidenceRevision = fixture.SchemaVersion, fixture.EvidenceRevision
	attempt.FinalizationToken, attempt.FinalizationLeaseUntil = fixture.Token, fixture.LeaseUntil
	attempt.FinalizationDispatchGeneration = fixture.DispatchGeneration
	return attempt
}

type finalizationInput struct {
	Captured finalizationAttemptInput  `json:"captured"`
	Current  *finalizationAttemptInput `json:"current,omitempty"`
	Now      time.Time                 `json:"now"`
}

type scopeInput struct {
	StudentID                 string                                 `json:"student_id"`
	MissionID                 string                                 `json:"mission_id"`
	ExpectedAttemptID         string                                 `json:"expected_attempt_id"`
	ExpectedPublishedRevision int64                                  `json:"expected_published_revision"`
	Current                   students.StudentGameSession            `json:"current"`
	Attempt                   students.StudentMissionAttemptDocument `json:"attempt"`
}

type memoryInput struct {
	Profile students.StudentProfile                `json:"profile"`
	Attempt students.StudentMissionAttemptDocument `json:"attempt"`
	Memory  students.StudentLearningMemory         `json:"memory"`
}

type objectPathInput struct {
	Path string `json:"path"`
}

func main() {
	if len(os.Args) != 1 {
		fail(errors.New("evalbridge accepts a JSON batch on stdin and no arguments"))
	}
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, 2*1024*1024+1))
	if err != nil || len(raw) > 2*1024*1024 {
		fail(errors.New("batch is unreadable or exceeds 2 MiB"))
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var input batch
	if err := decoder.Decode(&input); err != nil {
		fail(fmt.Errorf("invalid batch: %w", err))
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		fail(errors.New("batch must contain one JSON object"))
	}
	if err := validateBatch(input); err != nil {
		fail(err)
	}
	results := make([]map[string]any, 0, len(input.Cases))
	for _, candidate := range input.Cases {
		observation, err := runCase(input, candidate)
		if err != nil {
			if observation == nil {
				observation = map[string]any{"id": candidate.ID, "kind": candidate.Kind}
			}
			observation["error"] = err.Error()
		}
		results = append(results, observation)
	}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"mode": "offline", "real_provider_calls": 0, "results": results,
	}); err != nil {
		fail(err)
	}
}

func validateBatch(input batch) error {
	if len(input.Cases) == 0 || len(input.Cases) > 100 {
		return errors.New("batch requires 1 to 100 cases")
	}
	seen := map[string]bool{}
	for _, candidate := range input.Cases {
		if candidate.ID == "" || seen[candidate.ID] {
			return errors.New("case IDs must be nonempty and unique")
		}
		seen[candidate.ID] = true
		count := 0
		for _, present := range []bool{candidate.Tutor != nil, candidate.Auth != nil, candidate.Projection != nil, candidate.Learning != nil, candidate.Scope != nil, candidate.Memory != nil, candidate.ObjectPath != nil, candidate.Evidence != nil, candidate.TutorEvent != nil, candidate.Finalization != nil, candidate.Security != nil} {
			if present {
				count++
			}
		}
		if count != 1 {
			return fmt.Errorf("case %s requires exactly one input kind", candidate.ID)
		}
		switch candidate.Kind {
		case "tutor":
			if candidate.Tutor == nil || (candidate.Tutor.ProviderMode != "respond" && candidate.Tutor.ProviderMode != "disabled" && candidate.Tutor.ProviderMode != "timeout") {
				return fmt.Errorf("case %s has an invalid scripted provider mode", candidate.ID)
			}
		case "auth":
			if candidate.Auth == nil || candidate.Auth.AllowedProviders == "" {
				return fmt.Errorf("case %s requires provider policy", candidate.ID)
			}
		case "projection":
			if candidate.Projection == nil {
				return fmt.Errorf("case %s requires a projection input", candidate.ID)
			}
		case "learning":
			if candidate.Learning == nil || len(input.Mission.PreviewQuestions) != 3 {
				return fmt.Errorf("case %s requires a three-question published fixture", candidate.ID)
			}
		case "scope":
			if candidate.Scope == nil {
				return fmt.Errorf("case %s requires tutor scope", candidate.ID)
			}
		case "memory":
			if candidate.Memory == nil {
				return fmt.Errorf("case %s requires learning memory", candidate.ID)
			}
		case "object_path":
			if candidate.ObjectPath == nil {
				return fmt.Errorf("case %s requires an object path", candidate.ID)
			}
		case "evidence":
			if candidate.Evidence == nil || candidate.Evidence.RecentFeedbackCount < 0 || candidate.Evidence.RecentFeedbackCount > 100 || candidate.Evidence.GeneratedCount < 0 {
				return fmt.Errorf("case %s requires bounded evidence fixtures", candidate.ID)
			}
		case "tutor_event":
			if candidate.TutorEvent == nil || candidate.TutorEvent.Now.IsZero() {
				return fmt.Errorf("case %s requires a tutor event and fixed clock", candidate.ID)
			}
		case "security":
			if candidate.Security == nil || (candidate.Security.Action != "teacher" && candidate.Security.Action != "proxy" && candidate.Security.Action != "enrollment") {
				return fmt.Errorf("invalid security fixture")
			}
		case "finalization":
			if candidate.Finalization == nil || candidate.Finalization.Now.IsZero() {
				return fmt.Errorf("case %s requires finalization state and fixed clock", candidate.ID)
			}
		default:
			return fmt.Errorf("unknown case kind %q", candidate.Kind)
		}
	}
	return nil
}

func runCase(input batch, candidate evalCase) (map[string]any, error) {
	observation := map[string]any{"id": candidate.ID, "kind": candidate.Kind, "scripted_provider_calls": 0}
	switch candidate.Kind {
	case "tutor":
		// Every case receives its own copy so normalization cannot contaminate later cases.
		encoded, err := json.Marshal(input.TutorContext)
		if err != nil {
			return nil, err
		}
		var tutorContext generation.BuildTutorTurnInput
		if err := json.Unmarshal(encoded, &tutorContext); err != nil {
			return nil, err
		}
		if candidate.Tutor.RetrievedChunks != nil {
			tutorContext.RetrievedChunks = *candidate.Tutor.RetrievedChunks
		}
		originalPrivate, _ := json.Marshal([]any{tutorContext.StudentMemory, tutorContext.ClassContext})
		provider := &scriptedProvider{mode: candidate.Tutor.ProviderMode, response: candidate.Tutor.Response}
		service := generation.NewService(generation.Dependencies{Client: provider})
		tutor, err := service.BuildTutorTurn(context.Background(), tutorContext)
		observation["scripted_provider_calls"] = provider.calls
		observation["provider_input"] = provider.instruction + "\n" + provider.prompt
		if err != nil {
			return observation, err
		}
		observation["tutor"] = tutor
		remainingPrivate, _ := json.Marshal([]any{tutorContext.StudentMemory, tutorContext.ClassContext})
		observation["teacher_context_preserved"] = bytes.Equal(originalPrivate, remainingPrivate)
		if provider.calls > 0 {
			start := strings.Index(provider.prompt, "{")
			var supplied map[string]any
			if start < 0 {
				return observation, errors.New("provider prompt has no inspectable context object")
			}
			if err := json.NewDecoder(strings.NewReader(provider.prompt[start:])).Decode(&supplied); err != nil {
				return observation, fmt.Errorf("inspect provider context: %w", err)
			}
			observation["provider_context"] = supplied
		}
	case "security":
		spec := candidate.Security
		switch spec.Action {
		case "teacher":
			allowed, err := auth.ParseTeacherEmails(spec.Emails)
			observation["configuration_valid"] = err == nil
			observation["allowed"] = err == nil && auth.TeacherEmailAllowed(spec.Identity, allowed)
		case "proxy":
			extractor, err := app.ClientIP(spec.Proxies)
			observation["configuration_valid"] = err == nil
			if err == nil {
				request := httptest.NewRequest("GET", "/", nil)
				request.RemoteAddr = spec.Peer
				request.Header.Set("X-Forwarded-For", spec.Forwarded)
				observation["client_ip"] = extractor(request)
			}
		case "enrollment":
			observation["allowed"] = classrooms.MayEnroll(spec.Class, spec.ExistingMember, spec.Now)
			observation["configuration_valid"] = classrooms.ValidateEnrollment(spec.Class.Enrollment, spec.Now) == nil
		}
	case "auth":
		verifier := &scriptedAuthenticator{identity: auth.Identity{AuthMethod: candidate.Auth.Method, Disabled: candidate.Auth.Disabled, ProviderUserID: "synthetic-user"}}
		policy, err := auth.RestrictProviders(verifier, candidate.Auth.AllowedProviders)
		if err != nil {
			observation["auth_allowed"] = false
			observation["auth_configuration_error"] = true
		} else {
			_, err = policy.AuthenticateBearerToken(context.Background(), "Bearer synthetic-eval-token")
			observation["auth_allowed"] = err == nil
			observation["auth_configuration_error"] = false
		}
		observation["auth_verifier_calls"] = verifier.calls
	case "projection":
		observation["student_response"] = students.ToStudentTutorTurnResponse(candidate.Projection.Turn)
	case "learning":
		snapshot, status := students.AssessLearningSnapshotWithEvidence(input.Mission, candidate.Learning.Snapshot, candidate.Learning.Turns, candidate.Learning.AcceptedAnswers, candidate.Learning.RequestedStatus)
		observation["learning_snapshot"] = snapshot
		observation["learning_status"] = status
	case "scope":
		spec := candidate.Scope
		err := students.ValidateTutorTurnScope(students.SaveStudentTutorTurnInput{
			Identity: auth.Identity{ProviderUserID: spec.StudentID}, MissionID: spec.MissionID,
			ExpectedAttemptID: spec.ExpectedAttemptID, ExpectedPublishedRevision: spec.ExpectedPublishedRevision,
		}, spec.Current, spec.Attempt)
		observation["scope_allowed"] = err == nil
	case "memory":
		spec := candidate.Memory
		observation["memory_update_allowed"] = students.CanUpdateLearningMemory(spec.Profile, spec.Attempt, spec.Memory)
	case "object_path":
		observation["object_path_allowed"] = objectstorage.ValidateAttachmentObjectPath(candidate.ObjectPath.Path) == nil
	case "evidence":
		spec := candidate.Evidence
		attempt := spec.Attempt
		attempt.EvidenceSchemaVersion, attempt.GeneratedTurnCount, attempt.LegacyHistoryIncomplete = spec.SchemaVersion, spec.GeneratedCount, spec.LegacyIncomplete
		for index := 0; index < spec.RecentFeedbackCount; index++ {
			attempt.TutorTurns = append(attempt.TutorTurns, students.StudentTutorTurn{ID: fmt.Sprintf("synthetic-feedback-%d", index), Sequence: index + 1, Type: "feedback"})
		}
		effective := students.EffectiveTutorEvidence(attempt)
		observation["generation_allowed"] = students.CanGenerateTutorTurn(effective)
		observation["generated_count"] = effective.GeneratedTurnCount
		observation["history_incomplete"] = effective.LegacyHistoryIncomplete
	case "tutor_event":
		spec := candidate.TutorEvent
		stored, requested := spec.Stored.input(), spec.Stored.input()
		if spec.Request != nil {
			requested = spec.Request.input()
		}
		requested.ReservationOwner = spec.RequestOwner
		record := students.TutorEventDocument{State: spec.State, StudentID: stored.Identity.ProviderUserID, EventID: stored.EventID, Fingerprint: students.TutorEventFingerprint(stored), OwnerToken: spec.OwnerToken, LeaseUntil: spec.LeaseUntil, Turn: spec.Turn}
		turn, complete, err := students.TutorReplayDecision(record, requested, spec.Now)
		observation["replay_complete"], observation["replay_turn_id"] = complete, turn.ID
		observation["event_conflict"], observation["event_pending"] = errors.Is(err, students.ErrTutorEventConflict), errors.Is(err, students.ErrTutorEventPending)
		observation["reservation_matches"] = students.MatchesTutorReservation(record, requested, spec.Now)
	case "finalization":
		spec := candidate.Finalization
		captured, current := spec.Captured.document(), spec.Captured.document()
		if spec.Current != nil {
			current = spec.Current.document()
		}
		observation["fence_matches"] = students.MatchesFinalizationFence(current, students.CaptureFinalizationFence(captured))
		observation["job_matches"] = students.FinalizationJobMatches(current, students.FinalizationJob(captured))
		observation["eligible"] = students.FinalizationEligible(current)
		observation["due"] = students.FinalizationDue(current, spec.Now)
	}
	return observation, nil
}

type scriptedProvider struct {
	mode        string
	response    generation.TutorTurnBlueprint
	calls       int
	prompt      string
	instruction string
}

func (provider *scriptedProvider) IsEnabled() bool      { return provider.mode != "disabled" }
func (provider *scriptedProvider) DefaultModel() string { return "synthetic-scripted-v1" }
func (provider *scriptedProvider) Provider() string     { return "scripted-offline" }
func (provider *scriptedProvider) GenerateStructured(_ context.Context, request genai.StructuredRequest, target any) (genai.StructuredResult, error) {
	provider.calls++
	provider.prompt, provider.instruction = request.Prompt, request.SystemInstruction
	if provider.mode == "timeout" {
		return genai.StructuredResult{}, context.DeadlineExceeded
	}
	if provider.mode != "respond" {
		return genai.StructuredResult{}, errors.New("disabled scripted provider was invoked")
	}
	encoded, err := json.Marshal(provider.response)
	if err != nil {
		return genai.StructuredResult{}, err
	}
	if err := json.Unmarshal(encoded, target); err != nil {
		return genai.StructuredResult{}, err
	}
	return genai.StructuredResult{Provider: provider.Provider(), Model: provider.DefaultModel(), ModelVersion: "v1"}, nil
}

type scriptedAuthenticator struct {
	identity auth.Identity
	calls    int
}

func (verifier *scriptedAuthenticator) AuthenticateBearerToken(context.Context, string) (auth.Identity, error) {
	verifier.calls++
	return verifier.identity, nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "evalbridge:", err)
	os.Exit(2)
}
