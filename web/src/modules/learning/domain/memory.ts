import type { NoteKind } from "@/modules/learning/domain/mission";
import {
  FALLBACK_GAME_MISSION,
  type GameMission,
} from "@/modules/learning/domain/published-mission";
import {
  POD_ORDER,
  type ExplanationStyle,
  type PodId,
  type PodStatus,
} from "@/modules/learning/domain/types";

/** One thing the tutor wrote down about how this student thinks. */
export type StudentNote = {
  readonly id: string;
  readonly misconceptionId: string | null;
  readonly text: string;
  readonly kind: NoteKind;
  readonly skill: PodId;
  /** Question ids that produced this note. */
  readonly evidence: readonly string[];
  readonly count: number;
  readonly createdAt: number;
};

export type PodProgress = {
  readonly status: PodStatus;
  /** Question ids still to answer; the first one is the current question. */
  readonly queue: readonly string[];
  /** Spare questions for retries and reviews. */
  readonly pool: readonly string[];
  readonly correct: number;
  readonly required: number;
};

export type DecisionAction =
  | "next"
  | "complete"
  | "redirect"
  | "mission-complete"
  | "erase-note";

export type DecisionEntry = {
  readonly id: string;
  readonly at: number;
  readonly action: DecisionAction;
  readonly message: string;
  readonly why: string;
};

export type StudentPreferences = {
  readonly explanationStyle: ExplanationStyle;
  readonly difficultyBias: -1 | 0 | 1;
};

export type LearningEventType =
  | "question"
  | "answer"
  | "reasoning"
  | "feedback"
  | "adaptation"
  | "mastery";

export type LearningEventOutcome =
  | "neutral"
  | "correct"
  | "incorrect"
  | "support"
  | "complete";

/** Observable evidence used to build the teacher-facing learning flow. */
export type LearningEvent = {
  readonly id: string;
  readonly sequence: number;
  readonly type: LearningEventType;
  readonly stageId: PodId;
  readonly questionId: string;
  readonly title: string;
  readonly detail: string;
  readonly outcome: LearningEventOutcome;
  readonly occurredAt: number;
};

/** Full server evidence for one student. Only recoverable progress/preferences are cached in the browser. */
export type StudentMemory = {
  readonly version: 2;
  readonly notes: readonly StudentNote[];
  readonly preferences: StudentPreferences;
  readonly progress: Record<PodId, PodProgress>;
  readonly decisions: readonly DecisionEntry[];
  readonly events: readonly LearningEvent[];
  readonly eventSequence: number;
  /** The tutor redirects at most once per mission, so the demo never ping-pongs. */
  readonly redirected: boolean;
  /** Pod to go back to once a review is done. */
  readonly returnTo: PodId | null;
  readonly lastPod: PodId | null;
};

export type StudentSessionCheckpoint = {
  readonly stageId: PodId;
  readonly questionId: string;
  readonly phase: string;
  readonly stepIndex: number;
  readonly explainIndex: number;
  readonly lines: readonly string[];
  readonly wrongChoice: string;
  readonly wrongMisconceptionId: string;
  readonly decisionAction: DecisionAction | "";
  readonly decisionTarget: PodId | "";
  readonly decisionMessage: string;
  readonly decisionWhy: string;
  readonly tutorMessage: string;
  readonly whyPrompt: string;
  readonly clean: boolean;
  readonly seated: boolean;
};

export type StudentRecovery = {
  readonly memory: StudentMemory;
  readonly checkpoint: StudentSessionCheckpoint | null;
  readonly revision: number;
};

/** JSON and older local records can contain null for empty fields. */
export function normalizeSessionCheckpoint(value: unknown): StudentSessionCheckpoint | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  const checkpoint = value as Record<string, unknown>;
  const text = (key: string) => typeof checkpoint[key] === "string" ? checkpoint[key] as string : "";
  const index = (key: string) => typeof checkpoint[key] === "number" && Number.isFinite(checkpoint[key])
    ? Math.max(0, Math.floor(checkpoint[key] as number)) : 0;
  const stageId = text("stageId") as PodId;
  if (!POD_ORDER.includes(stageId) || !text("questionId") || checkpoint.seated !== true) return null;
  const decisionAction = text("decisionAction");
  const decisionTarget = text("decisionTarget") as PodId;
  return {
    stageId,
    questionId: text("questionId"),
    phase: text("phase"),
    stepIndex: index("stepIndex"),
    explainIndex: index("explainIndex"),
    lines: Array.isArray(checkpoint.lines) ? checkpoint.lines.filter((line): line is string => typeof line === "string") : [],
    wrongChoice: text("wrongChoice"),
    wrongMisconceptionId: text("wrongMisconceptionId"),
    decisionAction: ["next", "complete", "redirect", "mission-complete", "erase-note"].includes(decisionAction)
      ? decisionAction as DecisionAction : "",
    decisionTarget: POD_ORDER.includes(decisionTarget) ? decisionTarget : "",
    decisionMessage: text("decisionMessage"),
    decisionWhy: text("decisionWhy"),
    tutorMessage: text("tutorMessage"),
    whyPrompt: text("whyPrompt"),
    clean: checkpoint.clean === true,
    seated: true,
  };
}

export function createInitialMemory(
  mission: GameMission = FALLBACK_GAME_MISSION,
): StudentMemory {
  const progress = {} as Record<PodId, PodProgress>;

  for (const podId of POD_ORDER) {
    const plan = mission.stages[podId];

    progress[podId] = {
      status: podId === POD_ORDER[0] ? "ready" : "locked",
      queue: [...plan.primary],
      pool: [...plan.reinforcement],
      correct: 0,
      required: plan.required,
    };
  }

  return {
    version: 2,
    notes: [],
    preferences: { explanationStyle: "steps", difficultyBias: 0 },
    progress,
    decisions: [],
    events: [],
    eventSequence: 0,
    redirected: false,
    returnTo: null,
    lastPod: null,
  };
}

export function normalizeStudentMemory(
  value: Partial<StudentMemory>,
  mission: GameMission = FALLBACK_GAME_MISSION,
): StudentMemory {
  const initial = createInitialMemory(mission);
  const progress = { ...initial.progress };

  for (const podId of POD_ORDER) {
    const storedProgress = value.progress?.[podId];
    if (storedProgress) {
      progress[podId] = {
        status: storedProgress.status,
        queue: [...storedProgress.queue],
        pool: [...storedProgress.pool],
        correct: storedProgress.correct,
        required: storedProgress.required,
      };
    }
  }

  return {
    version: 2,
    notes: Array.isArray(value.notes) ? value.notes : [],
    preferences: value.preferences ?? initial.preferences,
    progress,
    decisions: Array.isArray(value.decisions) ? value.decisions : [],
    events: Array.isArray(value.events) ? value.events : [],
    eventSequence:
      typeof value.eventSequence === "number"
        ? value.eventSequence
        : (value.events?.at(-1)?.sequence ?? 0),
    redirected: value.redirected ?? false,
    returnTo: value.returnTo ?? null,
    lastPod: value.lastPod ?? null,
  };
}

// Older local checkpoints could mark remediation as complete without a correct
// answer. Reopen that practice; server-confirmed historical state uses its own mapper.
export function repairUnansweredCompletion(
  memory: StudentMemory,
  mission: GameMission,
): StudentMemory {
  const progress = { ...memory.progress };
  for (const pod of POD_ORDER) {
    const current = progress[pod];
    if (current.status === "complete" && current.correct < current.required) {
      progress[pod] = {
        ...current,
        status: "ready",
        queue: current.queue.length
          ? [...current.queue]
          : [...mission.stages[pod].primary],
      };
    }
  }
  return { ...memory, progress };
}

/** The pod the tutor should be standing at for this memory. */
export function pickTutorPod(memory: StudentMemory): PodId {
  const review = POD_ORDER.find(
    (podId) => memory.progress[podId].status === "review",
  );

  if (review) {
    return review;
  }

  return (
    POD_ORDER.find((podId) => memory.progress[podId].status === "ready") ??
    POD_ORDER[POD_ORDER.length - 1]
  );
}

export function isMissionComplete(memory: StudentMemory) {
  return POD_ORDER.every(
    (podId) => memory.progress[podId].status === "complete",
  );
}
