"use client";

import { useEffect, useMemo, useReducer } from "react";

import {
  normalizeMissionAnswer,
  sourceQuestionID,
  sourceStageID,
} from "@/modules/learning/domain/published-mission";
import {
  createInitialSession,
  sessionReducer,
  shuffledMoves,
} from "@/modules/learning/domain/session-reducer";
import { THINKING_MS } from "@/modules/learning/domain/tutor-policy";
import type { FeedbackValue } from "@/modules/learning/domain/types";

import { useLearningRecovery } from "@/modules/learning/adapters/hooks/use-learning-recovery";
import {
  expectedAnswer,
  learningEventID,
  questionPrompt,
} from "@/modules/learning/domain/tutor-evidence";
export function useLearningSession() {
  const [session, dispatch] = useReducer(
    sessionReducer,
    undefined,
    createInitialSession,
  );
  const {
    requestTutorTurn,
    resetRecovery,
    clearCurrentRecovery,
    recoveryError,
    evidenceState,
    completionAcknowledged,
    retryEvidence,
    deleteUnsentEvidence,
  } = useLearningRecovery(session, dispatch);
  // The hologram pauses before revealing a decision.
  useEffect(() => {
    if (session.lesson?.phase !== "thinking") {
      return;
    }

    const timer = window.setTimeout(
      () => dispatch({ type: "reveal-decision" }),
      THINKING_MS,
    );
    return () => window.clearTimeout(timer);
  }, [session.lesson?.decision, session.lesson?.phase]);

  const handlers = useMemo(
    () => ({
      sit: () => dispatch({ type: "sit" }),
      stand: () => dispatch({ type: "stand" }),
      choose: (index: number) => {
        const lesson = session.lesson;
        if (lesson?.question.kind !== "choice") {
          return;
        }
        const option = lesson.question.options[index];
        if (!option) {
          return;
        }
        const now = Date.now();
        const outcome = option.correct ? "correct" : "incorrect";
        const eventID = learningEventID(now, session.memory.eventSequence + 1);

        void requestTutorTurn(
          lesson.question.id,
          {
            event_id: eventID,
            type: "answer",
            stage_id: sourceStageID(
              session.mission,
              lesson.pod,
              lesson.question.id,
            ),
            question_id: sourceQuestionID(session.mission, lesson.question.id),
            prompt: questionPrompt(lesson.question),
            student_response: option.text,
            expected_answer: expectedAnswer(lesson.question),
            outcome,
            explicit_reasoning: "",
            feedback: "",
          },
          () =>
            dispatch({
              type: "choose",
              index,
              now,
              eventId: eventID,
              questionId: lesson.question.id,
            }),
        );
      },
      writeAnswer: (text: string) => {
        const lesson = session.lesson;
        const answer = text.trim();
        if (lesson?.question.kind !== "choice" || answer.length === 0) {
          return;
        }
        const now = Date.now();
        const expected = expectedAnswer(lesson.question);
        const outcome =
          expected !== "" &&
          normalizeMissionAnswer(answer) === normalizeMissionAnswer(expected)
            ? "correct"
            : "incorrect";
        const eventID = learningEventID(now, session.memory.eventSequence + 1);

        void requestTutorTurn(
          lesson.question.id,
          {
            event_id: eventID,
            type: "answer",
            stage_id: sourceStageID(
              session.mission,
              lesson.pod,
              lesson.question.id,
            ),
            question_id: sourceQuestionID(session.mission, lesson.question.id),
            prompt: questionPrompt(lesson.question),
            student_response: answer,
            expected_answer: expected,
            outcome,
            explicit_reasoning: "",
            feedback: "",
          },
          () =>
            dispatch({
              type: "write-answer",
              text: answer,
              now,
              eventId: eventID,
              questionId: lesson.question.id,
            }),
        );
      },
      step: (index: number) => {
        const lesson = session.lesson;
        if (lesson?.question.kind !== "steps") {
          return;
        }
        const expectedStep = lesson.question.steps[lesson.stepIndex];
        const move = shuffledMoves(
          lesson.question.id,
          lesson.stepIndex,
          expectedStep,
        )[index];
        if (!move || !expectedStep) {
          return;
        }
        const now = Date.now();
        const outcome = move === expectedStep.correct ? "correct" : "incorrect";
        const eventID = learningEventID(now, session.memory.eventSequence + 1);

        void requestTutorTurn(
          lesson.question.id,
          {
            event_id: eventID,
            type: "answer",
            stage_id: sourceStageID(
              session.mission,
              lesson.pod,
              lesson.question.id,
            ),
            question_id: sourceQuestionID(session.mission, lesson.question.id),
            prompt: questionPrompt(lesson.question),
            student_response: `${move.op} → ${move.result}`,
            expected_answer: `${expectedStep.correct.op} → ${expectedStep.correct.result}`,
            outcome,
            explicit_reasoning: "",
            feedback: "",
          },
          () =>
            dispatch({
              type: "step",
              index,
              now,
              eventId: eventID,
              questionId: lesson.question.id,
            }),
        );
      },
      why: (reasoning: { misconceptionId: string } | { text: string }) => {
        const lesson = session.lesson;
        if (!lesson || lesson.phase !== "why") {
          return;
        }
        const now = Date.now();
        const reasoningText =
          "text" in reasoning
            ? reasoning.text
            : (session.mission.misconceptions.find(
                (item) => item.id === reasoning.misconceptionId,
              )?.reasoning ?? reasoning.misconceptionId);
        const eventID = learningEventID(now, session.memory.eventSequence + 1);

        void requestTutorTurn(
          lesson.question.id,
          {
            event_id: eventID,
            type: "reasoning",
            stage_id: sourceStageID(
              session.mission,
              lesson.pod,
              lesson.question.id,
            ),
            question_id: sourceQuestionID(session.mission, lesson.question.id),
            prompt:
              lesson.whyPrompt ||
              `Why did you choose ${lesson.wrong?.chosen ?? "that answer"}?`,
            student_response: reasoningText,
            expected_answer: expectedAnswer(lesson.question),
            outcome: "support",
            explicit_reasoning: reasoningText,
            feedback: "",
          },
          () =>
            dispatch({
              type: "why",
              reasoning,
              now,
              eventId: eventID,
              questionId: lesson.question.id,
            }),
        );
      },
      explainNext: () => dispatch({ type: "explain-next" }),
      feedback: (value: FeedbackValue) => {
        const lesson = session.lesson;
        if (!lesson || lesson.phase !== "feedback") {
          return;
        }
        const now = Date.now();
        const eventID = learningEventID(now, session.memory.eventSequence + 1);

        void requestTutorTurn(
          lesson.question.id,
          {
            event_id: eventID,
            type: "feedback",
            stage_id: sourceStageID(
              session.mission,
              lesson.pod,
              lesson.question.id,
            ),
            question_id: sourceQuestionID(session.mission, lesson.question.id),
            prompt: questionPrompt(lesson.question),
            student_response: value,
            expected_answer: expectedAnswer(lesson.question),
            outcome: "support",
            explicit_reasoning: "",
            feedback: value,
          },
          () =>
            dispatch({
              type: "feedback",
              value,
              now,
              eventId: eventID,
              questionId: lesson.question.id,
            }),
        );
      },
      continue: () => dispatch({ type: "continue", now: Date.now() }),
      leaveMission: () => {
        resetRecovery();
        dispatch({ type: "leave-mission" });
      },
      reset: () => {
        clearCurrentRecovery();
        dispatch({ type: "reset", now: Date.now() });
      },
    }),
    [clearCurrentRecovery, requestTutorTurn, resetRecovery, session],
  );

  return {
    session,
    handlers,
    recoveryError,
    evidenceState,
    completionAcknowledged,
    retryEvidence,
    deleteUnsentEvidence,
  };
}
