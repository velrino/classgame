/**
 * The mock tutor. Every function here is pure and is the exact boundary the
 * real Gemini 3.5+-backed agent will replace: same inputs, same outputs, just async.
 */
import type {
  DecisionAction,
  StudentMemory,
  StudentNote,
  StudentPreferences,
} from "@/modules/learning/domain/memory";
import {
  GUESSED_ID,
  TUTOR_NAME,
  type Misconception,
  type Question,
} from "@/modules/learning/domain/mission";
import {
  getMissionMisconception,
  stageLabel,
  type GameMission,
} from "@/modules/learning/domain/published-mission";
import {
  POD_ORDER,
  type FeedbackValue,
  type PodId,
} from "@/modules/learning/domain/types";

export type Decision = {
  readonly action: DecisionAction;
  readonly target: PodId | null;
  readonly message: string;
  readonly why: string;
};

export type WhyOption = {
  readonly id: string;
  readonly label: string;
};

/** How long the hologram "thinks" before revealing a decision. Hides real latency later. */
export const THINKING_MS = 750;

function hashString(value: string) {
  let hash = 0;

  for (let index = 0; index < value.length; index += 1) {
    hash = (hash * 31 + value.charCodeAt(index)) | 0;
  }

  return Math.abs(hash);
}

/** Deterministic shuffle so the matching reason isn't always first. */
function orderBySeed<T>(items: readonly T[], seed: string) {
  return items
    .map((item, index) => ({ item, key: hashString(`${seed}:${index}`) }))
    .sort((a, b) => a.key - b.key)
    .map((entry) => entry.item);
}

/** The clarifying question: a few plausible reasons in the student's voice, plus guessing. */
export function buildWhyOptions(
  mission: GameMission,
  question: Question,
  chosenMisconception: Misconception | null,
): WhyOption[] {
  const skills = new Set<PodId>([question.skill]);

  if (question.skill !== "fractions" && question.kind === "steps") {
    skills.add("simple-equations");
  }

  const candidates = mission.misconceptions.filter(
    (item) =>
      item.id !== GUESSED_ID &&
      (skills.has(item.skill) || item.id === chosenMisconception?.id),
  );
  const picked = new Map<string, Misconception>();

  if (chosenMisconception) {
    picked.set(chosenMisconception.id, chosenMisconception);
  }

  for (const candidate of orderBySeed(candidates, question.id)) {
    if (picked.size >= 3) {
      break;
    }

    picked.set(candidate.id, candidate);
  }

  const options = orderBySeed(
    [...picked.values()].map((item) => ({
      id: item.id,
      label: item.reasoning,
    })),
    `${question.id}-why`,
  );

  options.push({ id: GUESSED_ID, label: "Honestly, I guessed" });

  return options;
}

export function noteFromReasoning(
  mission: GameMission,
  question: Question,
  reasoning: { misconceptionId: string } | { text: string },
  now: number,
): Omit<StudentNote, "count"> {
  if ("text" in reasoning) {
    return {
      id: `note-${question.id}-${now}`,
      misconceptionId: null,
      text: `In their own words: “${reasoning.text.trim()}”`,
      kind: "misconception",
      skill: question.skill,
      evidence: [question.id],
      createdAt: now,
    };
  }

  if (reasoning.misconceptionId === GUESSED_ID) {
    return {
      id: `note-${GUESSED_ID}-${question.skill}`,
      misconceptionId: GUESSED_ID,
      text: "Guesses when unsure — ask for the reason before showing an explanation",
      kind: "habit",
      skill: question.skill,
      evidence: [question.id],
      createdAt: now,
    };
  }

  const misconception = getMissionMisconception(
    mission,
    reasoning.misconceptionId,
  );

  if (!misconception) {
    throw new Error(`Unknown misconception: ${reasoning.misconceptionId}`);
  }

  return {
    id: `note-${misconception.id}`,
    misconceptionId: misconception.id,
    text: misconception.note,
    kind: misconception.kind,
    // "guessed" is a habit about this pod, not about fractions specifically.
    skill:
      misconception.id === GUESSED_ID ? question.skill : misconception.skill,
    evidence: [question.id],
    createdAt: now,
  };
}

export function applyFeedback(
  preferences: StudentPreferences,
  feedback: FeedbackValue,
): StudentPreferences {
  switch (feedback) {
    case "confusing":
      return { ...preferences, explanationStyle: "visual" };
    case "too-hard":
      return { ...preferences, difficultyBias: -1 };
    case "too-easy":
      return { ...preferences, difficultyBias: 1 };
    case "helpful":
      return preferences;
  }
}

type FeedbackContext = {
  readonly mission: GameMission;
  readonly memory: StudentMemory;
  readonly pod: PodId;
  readonly feedback: FeedbackValue;
  readonly misconception: Misconception | null;
  readonly previousStyle: StudentPreferences["explanationStyle"];
};

/** After the student rates an explanation: change route, change style, or carry on. */
export function decideAfterFeedback(context: FeedbackContext): Decision {
  const { mission, memory, pod, feedback, misconception, previousStyle } =
    context;

  const canRedirect =
    misconception !== null &&
    misconception.id !== GUESSED_ID &&
    misconception.skill !== pod &&
    !memory.redirected &&
    memory.progress[misconception.skill].status === "complete";

  if (canRedirect && misconception) {
    const target = misconception.skill;

    return {
      action: "redirect",
      target,
      message: `That slip connects to ${stageLabel(mission, target).toLowerCase()}. Let's fix that first.`,
      why: `Spotted: “${misconception.note}”. Reopening ${stageLabel(mission, target)} for a short review before we come back.`,
    };
  }

  const current = memory.progress[pod];
  if (current.queue.length <= 1 && current.pool.length === 0 &&
      current.correct < current.required) {
    return {
      action: "next",
      target: null,
      message: "Take another look, then try this question again.",
      why: "The explanation helps you practice. A correct answer is still needed.",
    };
  }

  switch (feedback) {
    case "confusing":
      return {
        action: "next",
        target: null,
        message:
          previousStyle === "visual"
            ? "Okay — I'll slow down and draw the next one too."
            : "Got it. From now on I draw it instead of explaining it.",
        why: "You marked the explanation as confusing → switching to visual explanations.",
      };
    case "too-hard":
      return {
        action: "next",
        target: null,
        message: "Let's take a smaller step.",
        why: "You marked it too hard → picking easier questions next.",
      };
    case "too-easy":
      return {
        action: "next",
        target: null,
        message: "Then I'll push a little harder.",
        why: "You marked it too easy → picking harder questions next.",
      };
    case "helpful":
      return {
        action: "next",
        target: null,
        message: "Good. One more like that.",
        why: "You marked it helpful → keeping this explanation style.",
      };
  }
}

/** After the last clean answer of a pod. */
export function decideAfterPodComplete(
  mission: GameMission,
  memory: StudentMemory,
  pod: PodId,
): Decision {
  const progress = memory.progress[pod];
  const cleanAnswers = `${progress.correct} clean answers`;

  if (memory.returnTo) {
    return {
      action: "complete",
      target: memory.returnTo,
      message: `${stageLabel(mission, pod)} is solid again. Let's go back to where we stopped.`,
      why: `Review passed with clean answers → ${stageLabel(mission, pod)} reinforced. Returning to ${stageLabel(mission, memory.returnTo)}.`,
    };
  }

  const next = POD_ORDER.find((candidate) =>
    candidate !== pod && memory.progress[candidate].status !== "complete",
  );

  if (!next) {
    return {
      action: "mission-complete",
      target: null,
      message: `Mission complete. You reached the learning goal: ${mission.learningGoal}`,
      why: `${cleanAnswers} → ${stageLabel(mission, pod)} completed. Every mission step is done.`,
    };
  }

  return {
    action: "complete",
    target: next,
    message: `${stageLabel(mission, pod)}: done. Next up: ${stageLabel(mission, next)}.`,
    why: `${cleanAnswers} → ${stageLabel(mission, pod)} completed. Unlocking ${stageLabel(mission, next)}: ${mission.stages[next].tagline}`,
  };
}

/** What the hologram says while you're walking around. */
export function roamingLine(
  mission: GameMission,
  memory: StudentMemory,
  tutorPod: PodId,
  resumed: boolean,
): string {
  const status = memory.progress[tutorPod].status;

  if (status === "review") {
    return `Sit back down — a quick ${stageLabel(mission, tutorPod).toLowerCase()} review, then we go back.`;
  }

  if (
    POD_ORDER.every((podId) => memory.progress[podId].status === "complete")
  ) {
    return `Mission complete. Your teacher has your progress.`;
  }

  if (resumed) {
    return `Welcome back. Last time you were on ${stageLabel(mission, tutorPod)} — take your desk and we pick up there.`;
  }

  if (tutorPod === POD_ORDER[0]) {
    return `Hi, I'm ${TUTOR_NAME}. Take the desk in front of mine — we start with ${stageLabel(mission, tutorPod)}.`;
  }

  return `Next: ${stageLabel(mission, tutorPod)}. Sit down when you're ready.`;
}
