import type {
  StudentLearningResumeAttemptDto,
  StudentLearningSnapshotDto,
  StudentLearningStatus,
} from "@/modules/learning/adapters/types/contracts";

import {
  createInitialMemory,
  normalizeSessionCheckpoint,
  normalizeStudentMemory,
  type StudentMemory,
  type StudentSessionCheckpoint,
} from "@/modules/learning/domain/memory";
import {
  slotForStageID,
  sourceQuestionID,
  sourceStageID,
  type GameMission,
} from "@/modules/learning/domain/published-mission";
import { type Session } from "@/modules/learning/domain/session-reducer";
import { POD_ORDER } from "@/modules/learning/domain/types";

export function learningStatus(
  memory: StudentMemory,
  phase: string | undefined,
): StudentLearningStatus {
  if (
    POD_ORDER.every((podId) => memory.progress[podId].status === "complete")
  ) {
    return "completed";
  }
  if (phase === "thinking") {
    return "thinking";
  }
  if (POD_ORDER.some((podId) => memory.progress[podId].status === "review")) {
    return "reviewing";
  }
  if (phase || memory.eventSequence > 0) {
    return "learning";
  }

  return "exploring";
}

export function checkpointFromSession(
  session: Session,
): StudentSessionCheckpoint {
  const lesson = session.lesson;
  return {
    stageId: lesson?.pod ?? session.tutorPod,
    questionId: lesson?.question.id ?? "",
    phase: lesson?.phase ?? "",
    stepIndex: lesson?.stepIndex ?? 0,
    explainIndex: lesson?.explainIndex ?? 0,
    lines: lesson ? [...lesson.lines] : [],
    wrongChoice: lesson?.wrong?.chosen ?? "",
    wrongMisconceptionId: lesson?.wrong?.misconception?.id ?? "",
    decisionAction: lesson?.decision?.action ?? "",
    decisionTarget: lesson?.decision?.target ?? "",
    decisionMessage: lesson?.decision?.message ?? "",
    decisionWhy: lesson?.decision?.why ?? "",
    tutorMessage: lesson?.tutorMessage ?? "",
    whyPrompt: lesson?.whyPrompt ?? "",
    clean: lesson?.clean ?? true,
    seated: session.seatedAt !== null && lesson !== null,
  };
}

export function toLearningSnapshot(
  session: Session,
  checkpoint: StudentSessionCheckpoint,
  revision: number,
  compact = false,
): StudentLearningSnapshotDto {
  const { memory, mission } = session;
  const notes = compact ? memory.notes.slice(0, 8) : memory.notes;
  const decisions = compact ? memory.decisions.slice(0, 8) : memory.decisions;
  const events = compact ? memory.events.slice(-16) : memory.events;
  return {
    revision,
    progress: POD_ORDER.map((podId) => ({
      stage_id: sourceStageID(mission, podId),
      status: memory.progress[podId].status,
      queue: [...memory.progress[podId].queue],
      pool: [...memory.progress[podId].pool],
      correct: memory.progress[podId].correct,
      required: memory.progress[podId].required,
    })),
    notes: notes.map((note) => ({
      id: note.id,
      misconception_id: note.misconceptionId ?? "",
      text: note.text,
      kind: note.kind,
      skill: sourceStageID(mission, note.skill),
      evidence: [...note.evidence],
      count: note.count,
      created_at: note.createdAt,
    })),
    preferences: {
      explanation_style: memory.preferences.explanationStyle,
      difficulty_bias: memory.preferences.difficultyBias,
    },
    decisions: decisions.map((decision) => ({
      id: decision.id,
      at: decision.at,
      action: decision.action,
      message: decision.message,
      why: decision.why,
    })),
    events: events.map((event) => ({
      id: event.id,
      sequence: event.sequence,
      type: event.type,
      stage_id: sourceStageID(mission, event.stageId, event.questionId),
      question_id: sourceQuestionID(mission, event.questionId),
      title: event.title,
      detail: event.detail,
      outcome: event.outcome,
      occurred_at: event.occurredAt,
    })),
    redirected: memory.redirected,
    return_to: memory.returnTo ? sourceStageID(mission, memory.returnTo) : "",
    last_pod: memory.lastPod ? sourceStageID(mission, memory.lastPod) : "",
    checkpoint: {
      stage_id: sourceStageID(
        mission,
        checkpoint.stageId,
        checkpoint.questionId,
      ),
      question_id: checkpoint.questionId,
      phase: checkpoint.phase,
      step_index: checkpoint.stepIndex,
      explain_index: checkpoint.explainIndex,
      lines: [...checkpoint.lines],
      wrong_choice: checkpoint.wrongChoice,
      wrong_misconception_id: checkpoint.wrongMisconceptionId,
      decision_action: checkpoint.decisionAction,
      decision_target: checkpoint.decisionTarget
        ? sourceStageID(mission, checkpoint.decisionTarget)
        : "",
      decision_message: checkpoint.decisionMessage,
      decision_why: checkpoint.decisionWhy,
      tutor_message: checkpoint.tutorMessage,
      why_prompt: checkpoint.whyPrompt,
      clean: checkpoint.clean,
      seated: checkpoint.seated,
    },
  };
}

export function memoryFromLearningAttempt(
  attempt: StudentLearningResumeAttemptDto,
  mission: GameMission,
): StudentMemory {
  const initial = createInitialMemory(mission);
  const progress = { ...initial.progress };
  for (const stage of attempt.snapshot.progress ?? []) {
    const pod = slotForStageID(mission, stage.stage_id ?? "");
    if (!pod) {
      continue;
    }
    progress[pod] = {
      status: isPodStatus(stage.status) ? stage.status : progress[pod].status,
      queue: [...new Set((stage.queue ?? []).flatMap((id) =>
        mission.questions
          .filter((question) => question.skill === pod &&
            (question.id === id || mission.sourceQuestionIds[question.id] === id))
          .map((question) => question.id),
      ))],
      // The API's pool is the full allowed question set, including accepted
      // answers; it is not a fresh-question queue and must not award repeat credit.
      pool: [],
      correct: stage.correct,
      required: stage.required,
    };
  }

  return normalizeStudentMemory(
    {
      version: 2,
      progress,
      preferences: {
        explanationStyle:
          attempt.snapshot.preferences.explanation_style === "visual"
            ? "visual"
            : "steps",
        difficultyBias: normalizeDifficultyBias(
          attempt.snapshot.preferences.difficulty_bias,
        ),
      },
      eventSequence: attempt.last_event_sequence,
      redirected: attempt.snapshot.redirected,
      returnTo: slotForStageID(mission, attempt.snapshot.return_to ?? ""),
      lastPod: slotForStageID(mission, attempt.snapshot.last_pod ?? ""),
    },
    mission,
  );
}

export function checkpointFromLearningAttempt(
  attempt: StudentLearningResumeAttemptDto,
  mission: GameMission,
): StudentSessionCheckpoint | null {
  const checkpoint = attempt.snapshot.checkpoint;
  if (!checkpoint) return null;
  const stage = slotForStageID(mission, checkpoint.stage_id ?? "");
  if (!stage || !checkpoint.seated || !checkpoint.question_id) {
    return null;
  }

  const questionId = mission.questions.find((question) =>
    question.skill === stage &&
    (question.id === checkpoint.question_id ||
      mission.sourceQuestionIds[question.id] === checkpoint.question_id),
  )?.id;
  if (!questionId) return null;
  return normalizeSessionCheckpoint({
    stageId: stage,
    questionId,
    phase: checkpoint.phase,
    stepIndex: checkpoint.step_index,
    explainIndex: checkpoint.explain_index,
    lines: checkpoint.lines,
    wrongChoice: checkpoint.wrong_choice,
    wrongMisconceptionId: checkpoint.wrong_misconception_id,
    decisionAction: isDecisionAction(checkpoint.decision_action)
      ? checkpoint.decision_action
      : "",
    decisionTarget: slotForStageID(mission, checkpoint.decision_target ?? "") ?? "",
    decisionMessage: checkpoint.decision_message,
    decisionWhy: checkpoint.decision_why,
    tutorMessage: checkpoint.tutor_message,
    whyPrompt: checkpoint.why_prompt,
    clean: checkpoint.clean,
    seated: checkpoint.seated,
  });
}

export function nextRevision(reference: { current: number }) {
  const revision = Math.max(Date.now(), reference.current + 1);
  reference.current = revision;
  return revision;
}

export function isPodStatus(
  value: string,
): value is StudentMemory["progress"][(typeof POD_ORDER)[number]]["status"] {
  return ["locked", "ready", "review", "complete"].includes(value);
}

export function isDecisionAction(
  value: string,
): value is Exclude<StudentSessionCheckpoint["decisionAction"], ""> {
  return [
    "next",
    "complete",
    "redirect",
    "mission-complete",
    "erase-note",
  ].includes(value);
}

export function normalizeDifficultyBias(value: number): -1 | 0 | 1 {
  if (value < 0) {
    return -1;
  }
  if (value > 0) {
    return 1;
  }

  return 0;
}
