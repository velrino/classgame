package students

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"classgame-api/internal/auth"
	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/missions"
)

const maxGeneratedTutorTurnsPerAttempt = 36

func (service *Service) SaveStudentTutorTurn(
	ctx context.Context,
	identity auth.Identity,
	input SaveStudentTutorTurnInput,
) (StudentTutorTurn, error) {
	if identity.AuthMethod != auth.AuthMethodAnonymous {
		return StudentTutorTurn{}, ErrStudentRequiresAnonymousAuth
	}
	if service.learning == nil || service.sessions == nil {
		return StudentTutorTurn{}, errors.New("student tutor dependencies are not configured")
	}

	input = normalizeStudentTutorTurnInput(input)
	input.Identity = identity
	input.Timestamp = time.Now()
	attempt, user, exists, err := service.sessions.GetCurrentStudentLearningAttempt(
		ctx,
		identity.ProviderUserID,
		input.MissionID,
	)
	if err != nil {
		return StudentTutorTurn{}, err
	}
	if !exists {
		return StudentTutorTurn{}, ErrStudentAttemptNotFound
	}
	if err := ValidateAttemptRequestScope(input.RequestScope, attempt); err != nil {
		return StudentTutorTurn{}, err
	}
	input.ExpectedAttemptID = attempt.ID
	input.ExpectedPublishedRevision = attempt.PublishedRevision

	replay, found, err := service.learning.GetStudentTutorEvent(ctx, identity.ProviderUserID, attempt.ID, input.EventID)
	if err != nil {
		return StudentTutorTurn{}, err
	}
	if found {
		turn, complete, err := TutorReplayDecision(replay, input, time.Now())
		if err != nil {
			return StudentTutorTurn{}, err
		}
		if complete {
			return turn, nil
		}
	}
	if existing, found := existingStudentTutorTurn(attempt.TutorTurns, input.EventID); found {
		if fingerprintRecordedTurn(existing, input) != TutorEventFingerprint(input) {
			return StudentTutorTurn{}, ErrTutorEventConflict
		}
		return existing, nil
	}
	if attempt.Status == "completed" {
		return StudentTutorTurn{}, ErrStudentAttemptClosed
	}
	attempt = EffectiveTutorEvidence(attempt)
	if input.Type != "feedback" {
		if attempt.LegacyHistoryIncomplete {
			return StudentTutorTurn{}, ErrTutorHistoryIncomplete
		}
		if !CanGenerateTutorTurn(attempt) {
			return StudentTutorTurn{}, ErrStudentTutorTurnLimit
		}
		if service.tutorTurnGenerator == nil {
			return StudentTutorTurn{}, errors.New("student tutor generator is not configured")
		}
	}

	missionDocument, err := service.missionForAttempt(
		ctx,
		attempt,
	)
	if err != nil {
		return StudentTutorTurn{}, err
	}
	input, err = groundStudentTutorTurn(missionDocument, input)
	if err != nil {
		return StudentTutorTurn{}, err
	}

	if input.Type != "feedback" && input.BeforeGeneration != nil {
		if err := input.BeforeGeneration(ctx); err != nil {
			return StudentTutorTurn{}, err
		}
	}
	safeSnapshot, _ := AssessLearningSnapshotWithEvidence(missionDocument, attempt.Snapshot, attempt.TutorTurns, attempt.AcceptedAnswers, attempt.Status)

	retrievedChunks := retrieveMissionKnowledge(missionDocument, input, 5)
	classContext := generation.TutorClassContext{}
	if input.Type != "feedback" {
		classContext, err = service.classTutorContext(ctx, user.Student.ClassID)
		if err != nil {
			return StudentTutorTurn{}, err
		}
	}
	turnInput := generation.BuildTutorTurnInput{
		Mission: generation.TutorMissionContext{
			ID:           missionDocument.ID,
			Title:        missionDocument.Title,
			Subject:      missionDocument.Subject,
			GradeLevel:   missionDocument.GradeLevel,
			LearningGoal: missionDocument.LearningGoal,
			Stages:       missionDocument.Stages,
		},
		StudentMemory: generation.LearningTracePreviousMemory{
			Strengths:            user.Student.LearningMemory.Strengths,
			ImprovementAreas:     user.Student.LearningMemory.ImprovementAreas,
			PreferredExplanation: user.Student.LearningMemory.PreferredExplanation,
			NextAdaptation:       user.Student.LearningMemory.NextAdaptation,
		},
		ClassContext: classContext,
		Attempt: generation.TutorAttemptContext{
			Status:       attempt.Status,
			Progress:     toTutorProgress(safeSnapshot.Progress),
			RecentTurns:  toTutorTurnHistory(attempt.TutorTurns),
			RecentEvents: toTutorEvents(safeSnapshot.Events),
		},
		Turn: generation.TutorTurnEvidence{
			EventID:           input.EventID,
			Type:              input.Type,
			StageID:           input.StageID,
			QuestionID:        input.QuestionID,
			Prompt:            input.Prompt,
			StudentResponse:   input.StudentResponse,
			ExpectedAnswer:    input.ExpectedAnswer,
			Outcome:           input.Outcome,
			ExplicitReasoning: input.ExplicitReasoning,
			Feedback:          input.Feedback,
		},
		RetrievedChunks: retrievedChunks,
	}

	tutor := generation.BuildObservedTutorTurn(
		turnInput,
		"Feedback was recorded for the next tutor interaction without generating new AI content.",
	)
	if input.Type != "feedback" {
		ownerBytes := make([]byte, 16)
		if _, err := rand.Read(ownerBytes); err != nil {
			return StudentTutorTurn{}, err
		}
		input.ReservationOwner = hex.EncodeToString(ownerBytes)
		now := time.Now()
		reservation, reserveErr := service.learning.ReserveTutorGeneration(ctx, input, input.ReservationOwner, now, now.Add(tutorEventLeaseDuration))
		if reserveErr != nil {
			return StudentTutorTurn{}, reserveErr
		}
		if reservation.Completed {
			return reservation.Turn, nil
		}
		tutor, err = service.tutorTurnGenerator.BuildTutorTurn(ctx, turnInput)
		if err != nil {
			_ = service.learning.FailTutorGeneration(ctx, input, time.Now())
			return StudentTutorTurn{}, err
		}
	}
	input.Timestamp = time.Now()
	return service.learning.AppendStudentTutorTurn(ctx, input, tutor)
}

func normalizeStudentTutorTurnInput(input SaveStudentTutorTurnInput) SaveStudentTutorTurnInput {
	input.MissionID = normalizeText(input.MissionID)
	input.EventID = normalizeText(input.EventID)
	input.Type = strings.ToLower(normalizeText(input.Type))
	input.StageID = normalizeText(input.StageID)
	input.QuestionID = normalizeText(input.QuestionID)
	input.Prompt = truncateStudentText(input.Prompt, 500)
	input.StudentResponse = truncateStudentText(input.StudentResponse, 800)
	input.ExpectedAnswer = truncateStudentText(input.ExpectedAnswer, 500)
	input.Outcome = strings.ToLower(normalizeText(input.Outcome))
	input.ExplicitReasoning = truncateStudentText(input.ExplicitReasoning, 800)
	input.Feedback = strings.ToLower(truncateStudentText(input.Feedback, 80))

	return input
}

func groundStudentTutorTurn(
	document missions.MissionDocument,
	input SaveStudentTutorTurnInput,
) (SaveStudentTutorTurnInput, error) {
	if input.Type == "resume" && input.QuestionID == "" {
		input.StageID = "mission-resume"
		input.Prompt = "Resume the published mission: " + document.Title
		input.ExpectedAnswer = ""
		input.Outcome = "neutral"
		return input, nil
	}

	var publishedQuestion *generation.PreviewQuestion
	for index := range document.PreviewQuestions {
		question := &document.PreviewQuestions[index]
		if question.ID == input.QuestionID {
			publishedQuestion = question
			break
		}
	}
	if publishedQuestion == nil {
		return SaveStudentTutorTurnInput{}, ErrStudentQuestionNotFound
	}

	input.StageID = publishedQuestion.StageID
	input.Prompt = publishedQuestion.Prompt
	input.ExpectedAnswer = publishedQuestion.ExpectedAnswer
	switch input.Type {
	case "answer":
		if normalizedStudentAnswer(input.StudentResponse) == normalizedStudentAnswer(publishedQuestion.ExpectedAnswer) {
			input.Outcome = "correct"
		} else {
			input.Outcome = "incorrect"
		}
	case "reasoning", "feedback":
		input.Outcome = "support"
	case "resume":
		input.Outcome = "neutral"
	}

	return input, nil
}

func normalizedStudentAnswer(value string) string {
	value = strings.ToLower(normalizeText(value))
	return strings.Trim(value, " .,!?:;\"'")
}

func existingStudentTutorTurn(
	turns []StudentTutorTurn,
	eventID string,
) (StudentTutorTurn, bool) {
	turnID := tutorTurnID(eventID)
	for _, turn := range turns {
		if turn.ID == turnID {
			return turn, true
		}
	}

	return StudentTutorTurn{}, false
}

func studentTutorTurnLimitReached(turns []StudentTutorTurn) bool {
	generatedTurns := 0
	for _, turn := range turns {
		if turn.Type != "feedback" {
			generatedTurns++
		}
	}

	return generatedTurns >= maxGeneratedTutorTurnsPerAttempt
}
