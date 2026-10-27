import {
  createInitialMemory,
  pickTutorPod,
  type LearningEvent,
  type PodProgress,
  type StudentMemory,
  type StudentNote,
  type StudentSessionCheckpoint,
} from "@/modules/learning/domain/memory";
import type {
  EquationStep,
  Misconception,
  Question,
  StepMove,
} from "@/modules/learning/domain/mission";
import {
  FALLBACK_GAME_MISSION,
  getMissionMisconception,
  getMissionQuestion,
  normalizeMissionAnswer,
  type GameMission,
} from "@/modules/learning/domain/published-mission";
import {
  applyFeedback,
  decideAfterFeedback,
  decideAfterPodComplete,
  noteFromReasoning,
  roamingLine,
  type Decision,
} from "@/modules/learning/domain/tutor-policy";
import {
  POD_ORDER,
  type FeedbackValue,
  type PodId,
} from "@/modules/learning/domain/types";

export type LessonPhase =
  | "question"
  | "correct"
  | "why"
  | "explain"
  | "feedback"
  | "thinking"
  | "decision";

export type Lesson = {
  readonly pod: PodId;
  readonly question: Question;
  readonly phase: LessonPhase;
  /** For step questions: which step we're on and the lines written so far. */
  readonly stepIndex: number;
  readonly lines: readonly string[];
  /** The move the student just got wrong, if any. */
  readonly wrong: {
    readonly chosen: string;
    readonly misconception: Misconception | null;
  } | null;
  readonly explainIndex: number;
  readonly decision: Decision | null;
  readonly tutorMessage: string;
  readonly whyPrompt: string;
  /** Reset to false the moment a wrong move happens; a question only counts when clean. */
  readonly clean: boolean;
};

export type Session = {
  readonly hydrated: boolean;
  readonly memory: StudentMemory;
  readonly mission: GameMission;
  readonly lesson: Lesson | null;
  readonly seatedAt: PodId | null;
  readonly tutorPod: PodId;
  readonly tutorLine: string;
  readonly walkRequest: { readonly token: number } | null;
};

export type SessionAction =
  | {
      type: "hydrate";
      memory: StudentMemory | null;
      mission: GameMission;
      checkpoint?: StudentSessionCheckpoint | null;
      now: number;
    }
  | { type: "sit" }
  | { type: "stand" }
  | { type: "walk" }
  | { type: "choose"; index: number; now: number }
  | { type: "write-answer"; text: string; now: number }
  | { type: "step"; index: number; now: number }
  | {
      type: "why";
      reasoning: { misconceptionId: string } | { text: string };
      now: number;
    }
  | { type: "explain-next" }
  | { type: "feedback"; value: FeedbackValue; now: number }
  | { type: "reveal-decision" }
  | { type: "continue"; now: number }
  | {
      type: "apply-tutor-turn";
      questionId: string;
      message: string;
      whyPrompt: string;
      explanationSteps: readonly string[];
      explanationStyle: "steps" | "visual";
      difficultyBias: -1 | 0 | 1;
    }
  | { type: "leave-mission" }
  | { type: "reset"; now: number };

export function createInitialSession(
  mission: GameMission = FALLBACK_GAME_MISSION,
): Session {
  return {
    hydrated: false,
    mission,
    memory: createInitialMemory(mission),
    lesson: null,
    seatedAt: null,
    tutorPod: POD_ORDER[0],
    tutorLine: "",
    walkRequest: null,
  };
}

// ── helpers ─────────────────────────────────────────────────────────────────

function sortByBias(
  mission: GameMission,
  ids: readonly string[],
  bias: -1 | 0 | 1,
) {
  if (bias === 0) {
    return [...ids];
  }

  return [...ids].sort(
    (a, b) =>
      (getMissionQuestion(mission, a).difficulty -
        getMissionQuestion(mission, b).difficulty) *
      bias,
  );
}

/** Make sure there's a question at the front of the queue, pulling from the pool when needed. */
function ensureQueue(
  mission: GameMission,
  progress: PodProgress,
  bias: -1 | 0 | 1,
): PodProgress {
  if (progress.queue.length > 0 || progress.pool.length === 0) {
    return progress;
  }

  const [next, ...rest] = sortByBias(mission, progress.pool, bias);
  return { ...progress, queue: [next], pool: rest };
}

function startLesson(
  mission: GameMission,
  memory: StudentMemory,
  pod: PodId,
): { lesson: Lesson | null; memory: StudentMemory } {
  const progress = ensureQueue(
    mission,
    memory.progress[pod],
    memory.preferences.difficultyBias,
  );
  const questionId = progress.queue[0];

  if (!questionId) {
    return { lesson: null, memory };
  }

  const question = getMissionQuestion(mission, questionId);

  return {
    memory: {
      ...memory,
      progress: { ...memory.progress, [pod]: progress },
      lastPod: pod,
    },
    lesson: {
      pod,
      question,
      phase: "question",
      stepIndex: 0,
      lines: question.kind === "steps" ? [question.equation] : [],
      wrong: null,
      explainIndex: 0,
      decision: null,
      tutorMessage: "",
      whyPrompt: "",
      clean: true,
    },
  };
}

function restoreLesson(
  mission: GameMission,
  checkpoint: StudentSessionCheckpoint | null | undefined,
): Lesson | null {
  if (!checkpoint?.seated || !isPodId(checkpoint.stageId)) {
    return null;
  }

  const question = mission.questions.find(
    (item) => item.id === checkpoint.questionId,
  );
  if (!question || question.skill !== checkpoint.stageId) {
    return null;
  }

  const phase = isLessonPhase(checkpoint.phase) ? checkpoint.phase : "question";
  const misconception = getMissionMisconception(
    mission,
    checkpoint.wrongMisconceptionId,
  );
  const decision = isDecisionAction(checkpoint.decisionAction)
    ? {
        action: checkpoint.decisionAction,
        target: isPodId(checkpoint.decisionTarget)
          ? checkpoint.decisionTarget
          : null,
        message: checkpoint.decisionMessage,
        why: checkpoint.decisionWhy,
      }
    : null;

  return {
    pod: checkpoint.stageId,
    question,
    phase:
      (phase === "thinking" || phase === "decision") && !decision
        ? "question"
        : phase,
    stepIndex: Math.max(0, checkpoint.stepIndex),
    lines:
      checkpoint.lines.length > 0
        ? checkpoint.lines
        : question.kind === "steps"
          ? [question.equation]
          : [],
    wrong: checkpoint.wrongChoice
      ? { chosen: checkpoint.wrongChoice, misconception }
      : null,
    explainIndex: Math.max(0, checkpoint.explainIndex),
    decision,
    tutorMessage: checkpoint.tutorMessage,
    whyPrompt: checkpoint.whyPrompt,
    clean: checkpoint.clean,
  };
}

/** A wrong answer sends the question to the back of the line; a fresh one comes forward if it's the only one. */
function requeueCurrent(
  mission: GameMission,
  progress: PodProgress,
  bias: -1 | 0 | 1,
): PodProgress {
  const [current, ...rest] = progress.queue;

  if (!current) {
    return progress;
  }

  if (rest.length === 0 && progress.pool.length > 0) {
    const [fresh, ...pool] = sortByBias(mission, progress.pool, bias);
    return { ...progress, queue: [fresh, current], pool };
  }

  return { ...progress, queue: [...rest, current] };
}

function mergeNote(
  notes: readonly StudentNote[],
  draft: Omit<StudentNote, "count">,
): StudentNote[] {
  const existing = draft.misconceptionId
    ? notes.find((note) => note.misconceptionId === draft.misconceptionId)
    : null;

  if (!existing) {
    return [{ ...draft, count: 1 }, ...notes];
  }

  return notes.map((note) =>
    note.id === existing.id
      ? {
          ...note,
          count: note.count + 1,
          evidence: [...new Set([...note.evidence, ...draft.evidence])],
        }
      : note,
  );
}

function logDecision(
  memory: StudentMemory,
  decision: Decision,
  now: number,
): StudentMemory {
  const nextMemory: StudentMemory = {
    ...memory,
    decisions: [
      {
        id: `decision-${now}`,
        at: now,
        action: decision.action,
        message: decision.message,
        why: decision.why,
      },
      ...memory.decisions,
    ].slice(0, 40),
  };

  return appendLearningEvent(nextMemory, {
    type:
      decision.action === "complete" || decision.action === "mission-complete"
        ? "mastery"
        : "adaptation",
    stageId: memory.lastPod ?? decision.target ?? POD_ORDER[0],
    questionId: "",
    title: limitedEventText(decision.message, 150),
    detail: limitedEventText(decision.why),
    outcome:
      decision.action === "complete" || decision.action === "mission-complete"
        ? "complete"
        : decision.action === "redirect"
          ? "support"
          : "neutral",
    occurredAt: now,
  });
}

function appendLearningEvent(
  memory: StudentMemory,
  event: Omit<LearningEvent, "id" | "sequence">,
): StudentMemory {
  const sequence = memory.eventSequence + 1;
  const nextEvent: LearningEvent = {
    ...event,
    id: `event-${event.occurredAt}-${sequence}`,
    sequence,
  };

  return {
    ...memory,
    events: [...memory.events, nextEvent].slice(-160),
    eventSequence: sequence,
  };
}

function questionLabel(question: Question) {
  return question.kind === "choice" ? question.prompt : question.equation;
}

function limitedEventText(value: string, length = 400) {
  return [...value].slice(0, length).join("");
}

function correctChoiceText(question: Question) {
  return question.kind === "choice"
    ? (question.options.find((option) => option.correct)?.text ?? "")
    : "";
}

function choiceAnswerIsCorrect(question: Question, answer: string) {
  const expected = correctChoiceText(question);

  return (
    expected !== "" &&
    normalizeMissionAnswer(answer) === normalizeMissionAnswer(expected)
  );
}

function leaveSeat(
  session: Session,
  memory: StudentMemory,
  tutorPod: PodId,
  tutorLine: string,
): Session {
  return {
    ...session,
    memory,
    lesson: null,
    seatedAt: null,
    tutorPod,
    tutorLine,
  };
}

/** Stay at the desk and open the next stage right away; stand up only if it has nothing to ask. */
function continueSeated(
  session: Session,
  memory: StudentMemory,
  tutorPod: PodId,
): Session {
  const started = startLesson(session.mission, memory, tutorPod);
  const tutorLine = roamingLine(
    session.mission,
    started.memory,
    tutorPod,
    false,
  );

  if (!started.lesson) {
    return leaveSeat(session, started.memory, tutorPod, tutorLine);
  }

  return {
    ...session,
    memory: started.memory,
    lesson: started.lesson,
    seatedAt: tutorPod,
    tutorPod,
    tutorLine,
  };
}

// ── reducer ─────────────────────────────────────────────────────────────────

export function sessionReducer(
  session: Session,
  action: SessionAction,
): Session {
  switch (action.type) {
    case "hydrate": {
      const memory = action.memory ?? createInitialMemory(action.mission);
      const restoredLesson = restoreLesson(action.mission, action.checkpoint);
      const tutorPod = restoredLesson?.pod ?? pickTutorPod(memory);

      return {
        ...session,
        hydrated: true,
        mission: action.mission,
        memory,
        lesson: restoredLesson,
        seatedAt: restoredLesson?.pod ?? null,
        tutorPod,
        tutorLine: roamingLine(
          action.mission,
          memory,
          tutorPod,
          action.memory !== null,
        ),
        walkRequest: null,
      };
    }

    case "leave-mission":
      return createInitialSession();

    case "walk":
      return {
        ...session,
        walkRequest: { token: (session.walkRequest?.token ?? 0) + 1 },
      };

    // One desk for the whole mission: sitting down resumes whatever stage the tutor is on.
    case "sit": {
      const pod = session.tutorPod;
      const status = session.memory.progress[pod].status;

      if (status === "locked" || status === "complete") {
        return session;
      }

      const started = startLesson(session.mission, session.memory, pod);

      if (!started.lesson) {
        return session;
      }

      return {
        ...session,
        memory: started.memory,
        lesson: started.lesson,
        seatedAt: pod,
      };
    }

    case "stand":
      return { ...session, lesson: null, seatedAt: null };

    case "choose": {
      const { lesson } = session;

      if (
        !lesson ||
        lesson.phase !== "question" ||
        lesson.question.kind !== "choice"
      ) {
        return session;
      }

      const option = lesson.question.options[action.index];

      if (!option) {
        return session;
      }

      if (option.correct) {
        return {
          ...session,
          memory: appendLearningEvent(session.memory, {
            type: "answer",
            stageId: lesson.pod,
            questionId: lesson.question.id,
            title: limitedEventText(questionLabel(lesson.question), 150),
            detail: limitedEventText(`Selected "${option.text}".`),
            outcome: "correct",
            occurredAt: action.now,
          }),
          lesson: { ...lesson, phase: "correct" },
        };
      }

      return {
        ...session,
        memory: appendLearningEvent(session.memory, {
          type: "answer",
          stageId: lesson.pod,
          questionId: lesson.question.id,
          title: limitedEventText(questionLabel(lesson.question), 150),
          detail: limitedEventText(`Selected "${option.text}".`),
          outcome: "incorrect",
          occurredAt: action.now,
        }),
        lesson: {
          ...lesson,
          phase: "why",
          clean: false,
          wrong: {
            chosen: option.text,
            misconception: getMissionMisconception(
              session.mission,
              option.misconception,
            ),
          },
        },
      };
    }

    case "write-answer": {
      const { lesson } = session;
      const answer = action.text.trim();

      if (
        !lesson ||
        lesson.phase !== "question" ||
        lesson.question.kind !== "choice" ||
        answer.length === 0
      ) {
        return session;
      }

      if (choiceAnswerIsCorrect(lesson.question, answer)) {
        return {
          ...session,
          memory: appendLearningEvent(session.memory, {
            type: "answer",
            stageId: lesson.pod,
            questionId: lesson.question.id,
            title: limitedEventText(questionLabel(lesson.question), 150),
            detail: limitedEventText(`Wrote "${answer}".`),
            outcome: "correct",
            occurredAt: action.now,
          }),
          lesson: { ...lesson, phase: "correct" },
        };
      }

      return {
        ...session,
        memory: appendLearningEvent(session.memory, {
          type: "answer",
          stageId: lesson.pod,
          questionId: lesson.question.id,
          title: limitedEventText(questionLabel(lesson.question), 150),
          detail: limitedEventText(`Wrote "${answer}".`),
          outcome: "incorrect",
          occurredAt: action.now,
        }),
        lesson: {
          ...lesson,
          phase: "why",
          clean: false,
          wrong: {
            chosen: answer,
            misconception: null,
          },
        },
      };
    }

    case "step": {
      const { lesson } = session;

      if (
        !lesson ||
        lesson.phase !== "question" ||
        lesson.question.kind !== "steps"
      ) {
        return session;
      }

      const step = lesson.question.steps[lesson.stepIndex];
      const moves = shuffledMoves(lesson.question.id, lesson.stepIndex, step);
      const move = moves[action.index];

      if (!step || !move) {
        return session;
      }

      if (move === step.correct) {
        const lines = [...lesson.lines, move.result];
        const isLast = lesson.stepIndex + 1 >= lesson.question.steps.length;

        return {
          ...session,
          memory: appendLearningEvent(session.memory, {
            type: "answer",
            stageId: lesson.pod,
            questionId: lesson.question.id,
            title: limitedEventText(questionLabel(lesson.question), 150),
            detail: limitedEventText(
              `Applied "${move.op}" and reached "${move.result}".`,
            ),
            outcome: "correct",
            occurredAt: action.now,
          }),
          lesson: {
            ...lesson,
            lines,
            stepIndex: lesson.stepIndex + 1,
            phase: isLast ? "correct" : "question",
          },
        };
      }

      return {
        ...session,
        memory: appendLearningEvent(session.memory, {
          type: "answer",
          stageId: lesson.pod,
          questionId: lesson.question.id,
          title: limitedEventText(questionLabel(lesson.question), 150),
          detail: limitedEventText(
            `Applied "${move.op}" and reached "${move.result}".`,
          ),
          outcome: "incorrect",
          occurredAt: action.now,
        }),
        lesson: {
          ...lesson,
          lines: [...lesson.lines, move.result],
          phase: "why",
          clean: false,
          wrong: {
            chosen: move.op,
            misconception: getMissionMisconception(
              session.mission,
              move.misconception,
            ),
          },
        },
      };
    }

    case "why": {
      const { lesson } = session;

      if (!lesson || lesson.phase !== "why") {
        return session;
      }

      const draft = noteFromReasoning(
        session.mission,
        lesson.question,
        action.reasoning,
        action.now,
      );
      const notes = mergeNote(session.memory.notes, draft);
      const misconception =
        "misconceptionId" in action.reasoning
          ? getMissionMisconception(
              session.mission,
              action.reasoning.misconceptionId,
            )
          : (lesson.wrong?.misconception ?? null);

      const reasoningText =
        "misconceptionId" in action.reasoning
          ? (misconception?.reasoning ?? action.reasoning.misconceptionId)
          : action.reasoning.text;
      return {
        ...session,
        memory: appendLearningEvent(
          { ...session.memory, notes },
          {
            type: "reasoning",
            stageId: lesson.pod,
            questionId: lesson.question.id,
            title: "Student explained the selected answer",
            detail: limitedEventText(reasoningText),
            outcome: "support",
            occurredAt: action.now,
          },
        ),
        lesson: {
          ...lesson,
          phase: "explain",
          explainIndex: 0,
          wrong: lesson.wrong
            ? { ...lesson.wrong, misconception }
            : lesson.wrong,
        },
      };
    }

    case "explain-next": {
      const { lesson } = session;

      if (!lesson || lesson.phase !== "explain") {
        return session;
      }

      const next = lesson.explainIndex + 1;

      if (next >= lesson.question.explanation.steps.length) {
        return { ...session, lesson: { ...lesson, phase: "feedback" } };
      }

      return { ...session, lesson: { ...lesson, explainIndex: next } };
    }

    case "feedback": {
      const { lesson } = session;

      if (!lesson || lesson.phase !== "feedback") {
        return session;
      }

      const previousStyle = session.memory.preferences.explanationStyle;
      const preferences = applyFeedback(
        session.memory.preferences,
        action.value,
      );
      const memory = appendLearningEvent(
        { ...session.memory, preferences },
        {
          type: "feedback",
          stageId: lesson.pod,
          questionId: lesson.question.id,
          title: "Student rated Mia's explanation",
          detail: `Feedback: ${action.value}.`,
          outcome: action.value === "helpful" ? "correct" : "support",
          occurredAt: action.now,
        },
      );
      const decision = decideAfterFeedback({
        mission: session.mission,
        memory,
        pod: lesson.pod,
        feedback: action.value,
        misconception: lesson.wrong?.misconception ?? null,
        previousStyle,
      });

      return {
        ...session,
        memory: logDecision(memory, decision, action.now),
        lesson: { ...lesson, phase: "thinking", decision },
      };
    }

    case "reveal-decision": {
      const { lesson } = session;

      if (!lesson || lesson.phase !== "thinking" || !lesson.decision) {
        return session;
      }

      return { ...session, lesson: { ...lesson, phase: "decision" } };
    }

    case "continue": {
      const { lesson, memory } = session;

      if (!lesson) {
        return session;
      }

      const bias = memory.preferences.difficultyBias;

      // A clean answer: count it, and either move on or let the tutor close the pod.
      if (lesson.phase === "correct") {
        const progress = memory.progress[lesson.pod];
        const [, ...rest] = progress.queue;
        const correct = lesson.clean ? progress.correct + 1 : progress.correct;
        let updated: PodProgress = { ...progress, queue: rest, correct };

        if (correct >= progress.required) {
          const nextMemory: StudentMemory = {
            ...memory,
            progress: { ...memory.progress, [lesson.pod]: updated },
          };
          const decision = decideAfterPodComplete(
            session.mission,
            nextMemory,
            lesson.pod,
          );

          return {
            ...session,
            memory: logDecision(nextMemory, decision, action.now),
            lesson: { ...lesson, phase: "thinking", decision },
          };
        }

        updated = ensureQueue(session.mission, updated, bias);
        const nextMemory: StudentMemory = {
          ...memory,
          progress: { ...memory.progress, [lesson.pod]: updated },
        };
        const started = startLesson(session.mission, nextMemory, lesson.pod);

        return started.lesson
          ? { ...session, memory: started.memory, lesson: started.lesson }
          : leaveSeat(session, nextMemory, session.tutorPod, session.tutorLine);
      }

      if (lesson.phase !== "decision" || !lesson.decision) {
        return session;
      }

      const { decision } = lesson;

      switch (decision.action) {
        case "next": {
          const current = memory.progress[lesson.pod];
          const hasAlternative =
            current.queue.length > 1 || current.pool.length > 0;

          // A generated mission currently has one question per visible stage.
          // Once Mia has explained a missed answer, repeating that same question
          // only loops over an answer the student has already seen. Record the
          // remediation and advance, preserving the incorrect result as evidence.
          if (!hasAlternative) {
            const nextIndex = POD_ORDER.indexOf(lesson.pod) + 1;
            const target = memory.returnTo ?? POD_ORDER[nextIndex] ?? null;
            const progress = {
              ...memory.progress,
              [lesson.pod]: {
                ...current,
                status: "complete" as const,
                queue: [],
              },
            };

            if (target && progress[target].status === "locked") {
              progress[target] = { ...progress[target], status: "ready" };
            }

            const nextMemory = appendLearningEvent(
              { ...memory, progress, returnTo: null },
              {
                type: "adaptation",
                stageId: lesson.pod,
                questionId: lesson.question.id,
                title: "Guided remediation completed",
                detail:
                  "Moving on after explaining the stage's only question; the incorrect answer remains recorded.",
                outcome: "support",
                occurredAt: action.now,
              },
            );

            if (!target) {
              return leaveSeat(
                session,
                nextMemory,
                lesson.pod,
                roamingLine(session.mission, nextMemory, lesson.pod, false),
              );
            }

            return continueSeated(session, nextMemory, target);
          }

          const progress = requeueCurrent(session.mission, current, bias);
          const nextMemory: StudentMemory = {
            ...memory,
            progress: { ...memory.progress, [lesson.pod]: progress },
          };
          const started = startLesson(session.mission, nextMemory, lesson.pod);

          return started.lesson
            ? { ...session, memory: started.memory, lesson: started.lesson }
            : leaveSeat(
                session,
                nextMemory,
                session.tutorPod,
                session.tutorLine,
              );
        }

        case "redirect": {
          const target = decision.target ?? POD_ORDER[0];
          const review = memory.progress[target];
          const [first, second, ...pool] = sortByBias(
            session.mission,
            review.pool,
            bias,
          );
          const reviewQueue = [first, second].filter((id): id is string =>
            Boolean(id),
          );
          const nextMemory: StudentMemory = {
            ...memory,
            redirected: true,
            returnTo: lesson.pod,
            // The failed question stays at the front: when the student comes back, they retry exactly what stopped them.
            progress: {
              ...memory.progress,
              [target]: {
                ...review,
                status: "review",
                queue: reviewQueue,
                pool,
                required: review.correct + reviewQueue.length,
              },
            },
          };

          return continueSeated(session, nextMemory, target);
        }

        case "complete":
        case "mission-complete": {
          const target = decision.target;
          const progress = {
            ...memory.progress,
            [lesson.pod]: {
              ...memory.progress[lesson.pod],
              status: "complete" as const,
            },
          };

          if (target && progress[target].status === "locked") {
            progress[target] = { ...progress[target], status: "ready" };
          }

          const nextMemory: StudentMemory = {
            ...memory,
            progress,
            returnTo: null,
          };

          if (!target) {
            return leaveSeat(
              session,
              nextMemory,
              lesson.pod,
              roamingLine(session.mission, nextMemory, lesson.pod, false),
            );
          }

          return continueSeated(session, nextMemory, target);
        }

        case "erase-note":
          return session;
      }

      return session;
    }

    case "apply-tutor-turn": {
      const lesson = session.lesson;
      const preferences = {
        explanationStyle: action.explanationStyle,
        difficultyBias: action.difficultyBias,
      } as const;

      if (!lesson || lesson.question.id !== action.questionId) {
        return {
          ...session,
          memory: { ...session.memory, preferences },
          tutorLine: action.message || session.tutorLine,
        };
      }

      const explanationSteps = action.explanationSteps.filter(Boolean);
      const question =
        explanationSteps.length > 0
          ? {
              ...lesson.question,
              explanation: {
                ...lesson.question.explanation,
                steps: explanationSteps,
              },
            }
          : lesson.question;
      const decision = lesson.decision
        ? {
            ...lesson.decision,
            message: action.message || lesson.decision.message,
          }
        : null;

      return {
        ...session,
        memory: { ...session.memory, preferences },
        lesson: {
          ...lesson,
          question,
          decision,
          tutorMessage: action.message,
          whyPrompt: action.whyPrompt,
        },
      };
    }

    case "reset": {
      const memory = createInitialMemory(session.mission);
      const tutorPod = POD_ORDER[0];

      return {
        ...createInitialSession(session.mission),
        hydrated: true,
        memory,
        tutorPod,
        tutorLine: roamingLine(session.mission, memory, tutorPod, false),
        walkRequest: null,
      };
    }
  }
}

function isPodId(value: string): value is PodId {
  return POD_ORDER.some((podId) => podId === value);
}

function isLessonPhase(value: string): value is LessonPhase {
  return [
    "question",
    "correct",
    "why",
    "explain",
    "feedback",
    "thinking",
    "decision",
  ].includes(value);
}

function isDecisionAction(value: string): value is Decision["action"] {
  return [
    "next",
    "complete",
    "redirect",
    "mission-complete",
    "erase-note",
  ].includes(value);
}

// ── derived views used by the board ─────────────────────────────────────────

function hashString(value: string) {
  let hash = 0;

  for (let index = 0; index < value.length; index += 1) {
    hash = (hash * 31 + value.charCodeAt(index)) | 0;
  }

  return Math.abs(hash);
}

/** Stable shuffle so the correct move isn't always in the same slot. */
export function shuffledMoves(
  questionId: string,
  stepIndex: number,
  step: EquationStep | undefined,
): StepMove[] {
  if (!step) {
    return [];
  }

  const moves = [step.correct, ...step.distractors];

  return moves
    .map((move, index) => ({
      move,
      key: hashString(`${questionId}:${stepIndex}:${index}`),
    }))
    .sort((a, b) => a.key - b.key)
    .map((entry) => entry.move);
}
