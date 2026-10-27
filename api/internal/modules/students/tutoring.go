package students

import (
	"context"
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
	if service.learning == nil || service.sessions == nil || service.tutorTurnGenerator == nil {
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
	input.ExpectedAttemptID = attempt.ID
	input.ExpectedPublishedRevision = attempt.PublishedRevision
	if existingTurn, exists := existingStudentTutorTurn(attempt.TutorTurns, input.EventID); exists {
		return existingTurn, nil
	}
	if input.Type != "feedback" && studentTutorTurnLimitReached(attempt.TutorTurns) {
		return StudentTutorTurn{}, ErrStudentTutorTurnLimit
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
			Progress:     toTutorProgress(attempt.Snapshot.Progress),
			RecentTurns:  toTutorTurnHistory(attempt.TutorTurns),
			RecentEvents: toTutorEvents(attempt.Snapshot.Events),
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
		tutor, err = service.tutorTurnGenerator.BuildTutorTurn(ctx, turnInput)
		if err != nil {
			return StudentTutorTurn{}, err
		}
	}

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
