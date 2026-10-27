package students

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"classgame-api/internal/modules/missions"
)

const assessmentEvidenceSchemaVersion = 1

var (
	ErrTutorEventConflict     = errors.New("tutor event ID was reused with different evidence")
	ErrStudentAttemptClosed   = errors.New("student attempt is closed")
	ErrTutorHistoryIncomplete = errors.New("legacy tutor budget history is incomplete")
	ErrTutorAdmissionHandled  = errors.New("tutor admission response was written")
	ErrTutorEventPending      = errors.New("tutor event generation is pending")
	ErrStaleFinalization      = errors.New("learning finalization evidence changed")
	ErrFinalizationBusy       = errors.New("learning finalization lease is active")
)

type AcceptedAnswerFact struct {
	QuestionID        string    `firestore:"question_id" json:"question_id"`
	StageID           string    `firestore:"stage_id" json:"stage_id"`
	StudentResponse   string    `firestore:"student_response" json:"student_response"`
	PublishedRevision int64     `firestore:"published_revision" json:"published_revision"`
	TurnID            string    `firestore:"turn_id" json:"turn_id"`
	AcceptedAt        time.Time `firestore:"accepted_at" json:"accepted_at"`
}

const tutorEventLeaseDuration = 2 * time.Minute

type TutorGenerationReservation struct {
	Completed bool
	Turn      StudentTutorTurn
}
type TutorEventDocument struct {
	State      string    `firestore:"state"`
	OwnerToken string    `firestore:"owner_token"`
	LeaseUntil time.Time `firestore:"lease_until"`

	StudentID   string           `firestore:"student_id"`
	EventID     string           `firestore:"event_id"`
	Fingerprint string           `firestore:"fingerprint"`
	Turn        StudentTutorTurn `firestore:"turn"`
	CreatedAt   time.Time        `firestore:"created_at"`
}

// TutorEventFingerprint includes client evidence and captured server scope,
// excluding client fields replaced by the pinned question and UI-only values.
func TutorEventFingerprint(input SaveStudentTutorTurnInput) string {
	input = normalizeStudentTutorTurnInput(input)
	response, reasoning, feedback := input.StudentResponse, input.ExplicitReasoning, input.Feedback
	canonical := struct {
		Version                                                                       int
		Actor, Attempt, Mission, Event, Kind, Question, Response, Reasoning, Feedback string
		Revision                                                                      int64
	}{1, input.Identity.ProviderUserID, input.ExpectedAttemptID, input.MissionID, input.EventID, input.Type, input.QuestionID, response, reasoning, feedback, input.ExpectedPublishedRevision}
	body, _ := json.Marshal(canonical)
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}
func ReplayTutorEvent(record TutorEventDocument, input SaveStudentTutorTurnInput) (StudentTutorTurn, error) {
	if record.EventID != input.EventID || record.Fingerprint != TutorEventFingerprint(input) {
		return StudentTutorTurn{}, ErrTutorEventConflict
	}
	if record.State != "complete" && !(record.State == "" && record.Turn.ID != "") {
		return StudentTutorTurn{}, ErrTutorEventPending
	}
	return record.Turn, nil
}
func fingerprintRecordedTurn(turn StudentTutorTurn, input SaveStudentTutorTurnInput) string {
	input.Type = turn.Type
	input.QuestionID = turn.QuestionID
	input.StudentResponse = turn.StudentResponse
	input.ExplicitReasoning = turn.ExplicitReasoning
	input.Feedback = turn.Feedback
	return TutorEventFingerprint(input)
}

// EffectiveTutorEvidence never reconstructs credit or counts from a browser
// snapshot. Truncated pre-upgrade histories have only a known lower-bound count.
func EffectiveTutorEvidence(attempt StudentMissionAttemptDocument) StudentMissionAttemptDocument {
	if attempt.EvidenceSchemaVersion > 0 {
		return attempt
	}
	attempt.EvidenceSchemaVersion = assessmentEvidenceSchemaVersion
	knownCount := int64(0)
	for _, turn := range attempt.TutorTurns {
		if turn.Type != "feedback" {
			knownCount++
		}
		if int64(turn.Sequence) > attempt.TutorEventSequence {
			attempt.TutorEventSequence = int64(turn.Sequence)
		}
		if turn.Type == "answer" && turn.Outcome == "correct" && normalizedStudentAnswer(turn.StudentResponse) == normalizedStudentAnswer(turn.ExpectedAnswer) {
			attempt.AcceptedAnswers = mergeAcceptedAnswer(attempt.AcceptedAnswers, AcceptedAnswerFact{QuestionID: turn.QuestionID, StageID: turn.StageID, StudentResponse: turn.StudentResponse, PublishedRevision: attempt.PublishedRevision, TurnID: turn.ID, AcceptedAt: turn.CreatedAt})
		}
	}
	if knownCount > attempt.GeneratedTurnCount {
		attempt.GeneratedTurnCount = knownCount
	}
	attempt.LegacyHistoryIncomplete = len(attempt.TutorTurns) >= maxTutorTurnsPerAttempt
	return attempt
}
func CanGenerateTutorTurn(attempt StudentMissionAttemptDocument) bool {
	attempt = EffectiveTutorEvidence(attempt)
	return !attempt.LegacyHistoryIncomplete && attempt.GeneratedTurnCount < maxGeneratedTutorTurnsPerAttempt
}
func mergeAcceptedAnswer(facts []AcceptedAnswerFact, incoming AcceptedAnswerFact) []AcceptedAnswerFact {
	for _, fact := range facts {
		if fact.QuestionID == incoming.QuestionID {
			return facts
		}
	}
	return append(append([]AcceptedAnswerFact{}, facts...), incoming)
}
func acceptedAnswerFacts(document missions.MissionDocument, turns []StudentTutorTurn) []AcceptedAnswerFact {
	facts := []AcceptedAnswerFact{}
	questions := studentSessionMissionFromDocument(document).PreviewQuestions
	for _, turn := range turns {
		if turn.Type != "answer" {
			continue
		}
		for _, q := range questions {
			if q.ID == turn.QuestionID && normalizedStudentAnswer(turn.StudentResponse) == normalizedStudentAnswer(q.ExpectedAnswer) {
				facts = mergeAcceptedAnswer(facts, AcceptedAnswerFact{QuestionID: q.ID, StageID: q.StageID, StudentResponse: turn.StudentResponse, PublishedRevision: document.PublishedRevision, TurnID: turn.ID, AcceptedAt: turn.CreatedAt})
			}
		}
	}
	return facts
}

// TutorReplayDecision distinguishes durable results from active work and expired
// reservations without spending provider quota. It is used before reservation.
func TutorReplayDecision(record TutorEventDocument, input SaveStudentTutorTurnInput, now time.Time) (StudentTutorTurn, bool, error) {
	if record.EventID != input.EventID || record.Fingerprint != TutorEventFingerprint(input) {
		return StudentTutorTurn{}, false, ErrTutorEventConflict
	}
	if record.State == "complete" || (record.State == "" && record.Turn.ID != "") {
		return record.Turn, true, nil
	}
	if record.State == "processing" && record.LeaseUntil.After(now) {
		return StudentTutorTurn{}, false, ErrTutorEventPending
	}
	return StudentTutorTurn{}, false, nil
}
func MatchesTutorReservation(record TutorEventDocument, input SaveStudentTutorTurnInput, now time.Time) bool {
	return input.ReservationOwner != "" && record.OwnerToken == input.ReservationOwner && record.State == "processing" && record.LeaseUntil.After(now) && record.StudentID == input.Identity.ProviderUserID && record.EventID == input.EventID && record.Fingerprint == TutorEventFingerprint(input)
}
