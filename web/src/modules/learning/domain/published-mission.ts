import type { StudentMissionDto } from "@/modules/missions";

import {
  MISCONCEPTIONS,
  QUESTIONS,
  type Misconception,
  type Question,
} from "@/modules/learning/domain/mission";
import { POD_ORDER, type PodId } from "@/modules/learning/domain/types";

const studentPreviewQuestionLimit = 3;
const fallbackQuestions = QUESTIONS.filter(
  (question) => question.kind === "choice",
).slice(0, studentPreviewQuestionLimit);

export type GameStage = {
  readonly slot: PodId;
  readonly sourceId: string;
  readonly title: string;
  readonly goal: string;
  readonly tagline: string;
  readonly primary: readonly string[];
  readonly reinforcement: readonly string[];
  readonly required: number;
};

export type GameMission = {
  readonly id: string;
  readonly publishedRevision: number;
  readonly title: string;
  readonly subject: string;
  readonly gradeLevel: string;
  readonly learningGoal: string;
  readonly stages: Record<PodId, GameStage>;
  readonly questions: readonly Question[];
  readonly sourceQuestionIds: Readonly<Record<string, string>>;
  readonly sourceQuestionStageIds: Readonly<Record<string, string>>;
  readonly sourceStageSlots: Readonly<Record<string, PodId>>;
  readonly misconceptions: readonly Misconception[];
};

export const FALLBACK_GAME_MISSION: GameMission = {
  id: "fallback-mission",
  publishedRevision: 0,
  title: "Class mission",
  subject: "Classroom learning",
  gradeLevel: "",
  learningGoal: "Complete the mission from your teacher.",
  stages: {
    fractions: {
      slot: "fractions",
      sourceId: "fractions",
      title: "Question 1",
      goal: "Answer one mission question.",
      tagline: "Answer one mission question.",
      primary: [fallbackQuestions[0]?.id ?? "f1"],
      reinforcement: [],
      required: 1,
    },
    "simple-equations": {
      slot: "simple-equations",
      sourceId: "simple-equations",
      title: "Question 2",
      goal: "Answer one mission question.",
      tagline: "Answer one mission question.",
      primary: [fallbackQuestions[1]?.id ?? "f2"],
      reinforcement: [],
      required: 1,
    },
    "two-step-equations": {
      slot: "two-step-equations",
      sourceId: "two-step-equations",
      title: "Question 3",
      goal: "Answer one mission question.",
      tagline: "Answer one mission question.",
      primary: [fallbackQuestions[2]?.id ?? "f3"],
      reinforcement: [],
      required: 1,
    },
  },
  questions: fallbackQuestions,
  sourceQuestionIds: Object.fromEntries(
    fallbackQuestions.map((question) => [question.id, question.id]),
  ),
  sourceQuestionStageIds: Object.fromEntries(
    fallbackQuestions.map((question, index) => [
      question.id,
      POD_ORDER[index] ?? question.skill,
    ]),
  ),
  sourceStageSlots: Object.fromEntries(POD_ORDER.map((slot) => [slot, slot])),
  misconceptions: MISCONCEPTIONS,
};

export function buildGameMission(
  source: StudentMissionDto | undefined,
): GameMission {
  const publishedStages = Array.isArray(source?.stages) ? source.stages : [];
  const previewQuestions = source
    ? playablePreviewQuestions(source, publishedStages)
    : [];

  if (
    !source ||
    publishedStages.length === 0 ||
    previewQuestions.length === 0
  ) {
    return FALLBACK_GAME_MISSION;
  }

  const sourceStages = POD_ORDER.map(
    (_, index) => publishedStages[Math.min(index, publishedStages.length - 1)],
  );
  const slotByStageId = new Map<string, PodId>();
  const sourceStageSlots: Record<string, PodId> = {};
  publishedStages.forEach((stage, index) => {
    const slot = POD_ORDER[Math.min(index, POD_ORDER.length - 1)];
    slotByStageId.set(normalizeKey(stage.id), slot);
    sourceStageSlots[normalizeKey(stage.id)] = slot;
  });

  const questions: Question[] = [];
  const sourceQuestionIds: Record<string, string> = {};
  const sourceQuestionStageIds: Record<string, string> = {};
  const misconceptions: Misconception[] = [];
  const questionIDsBySlot = new Map<PodId, string[]>(
    POD_ORDER.map((slot) => [slot, []]),
  );

  previewQuestions.forEach((question, index) => {
    const slot =
      slotByStageId.get(normalizeKey(question.stage_id)) ??
      POD_ORDER[index % POD_ORDER.length];
    const converted = convertQuestion(question, slot, misconceptions, index);
    questions.push(converted);
    sourceQuestionIds[converted.id] = question.id;
    sourceQuestionStageIds[converted.id] =
      question.stage_id || sourceStages[POD_ORDER.indexOf(slot)].id || slot;
    questionIDsBySlot.get(slot)?.push(converted.id);
  });

  // A mission always has three visible progress steps. Legacy missions may not
  // have stage_id on questions, so ensure every step has at least one grounded
  // question by cloning the nearest published question into the empty slot.
  for (const [index, slot] of POD_ORDER.entries()) {
    const ids = questionIDsBySlot.get(slot);
    if (ids && ids.length > 0) {
      continue;
    }
    const sourceQuestion = previewQuestions[index % previewQuestions.length];
    const converted = convertQuestion(
      sourceQuestion,
      slot,
      misconceptions,
      questions.length,
      `-${index + 1}`,
    );
    questions.push(converted);
    sourceQuestionIds[converted.id] = sourceQuestion.id;
    sourceQuestionStageIds[converted.id] =
      sourceQuestion.stage_id || sourceStages[index].id || slot;
    questionIDsBySlot.set(slot, [converted.id]);
  }

  const stages = {} as Record<PodId, GameStage>;
  POD_ORDER.forEach((slot, index) => {
    const stage = sourceStages[index];
    const ids = questionIDsBySlot.get(slot) ?? [];
    stages[slot] = {
      slot,
      sourceId: stage.id || slot,
      title: stage.title || `Step ${index + 1}`,
      goal: stage.goal || source.learning_goal,
      tagline: stage.goal || source.learning_goal,
      primary: ids,
      reinforcement: [],
      required: Math.max(1, Math.min(2, ids.length)),
    };
  });

  return {
    id: source.id,
    publishedRevision: source.published_revision,
    title: source.mission_name || source.title || "Class mission",
    subject: source.subject || "Classroom learning",
    gradeLevel: source.grade_level,
    learningGoal: source.learning_goal,
    stages,
    questions,
    sourceQuestionIds,
    sourceQuestionStageIds,
    sourceStageSlots,
    misconceptions,
  };
}

function playablePreviewQuestions(
  source: StudentMissionDto,
  publishedStages: StudentMissionDto["stages"],
): StudentMissionDto["preview_questions"] {
  const questions = Array.isArray(source.preview_questions)
    ? source.preview_questions
    : [];
  if (!hasLegacyGenericFallbackQuestion(questions)) {
    return ensureThreePreviewQuestions(
      source,
      publishedStages,
      questions.slice(0, studentPreviewQuestionLimit),
    );
  }

  if (isGradeFiveMathMission(source)) {
    return gradeFiveMathPreviewQuestions(publishedStages);
  }

  return ensureThreePreviewQuestions(source, publishedStages, [
    ...genericPreviewQuestions(publishedStages),
    ...questions
      .filter((question) => !isLegacyGenericFallbackQuestion(question))
      .map((question, index) => ({
        ...question,
        stage_id:
          cleanText(question.stage_id) || stageIdAt(publishedStages, index + 1),
        options: cleanTextList(question.options),
        explanation_steps: cleanTextList(question.explanation_steps),
        misconception_traps: cleanTextList(question.misconception_traps),
      })),
  ]);
}

function gradeFiveMathPreviewQuestions(
  publishedStages: StudentMissionDto["stages"],
): StudentMissionDto["preview_questions"] {
  return [
    {
      id: "place-value-build",
      stage_id: stageIdAt(publishedStages, 0),
      prompt: "Which number is built from 50,000 + 3,000 + 200 + 40 + 6?",
      answer_type: "multiple_choice",
      options: ["53,246", "50,324", "5,346", "53,426"],
      expected_answer: "53,246",
      explanation_steps: [
        "Each part fills a place-value position.",
        "50,000 + 3,000 + 200 + 40 + 6 equals 53,246.",
      ],
      misconception_traps: [
        "Mixes up the ten-thousands and thousands places.",
        "Reorders digits without matching each part to its place value.",
      ],
    },
    {
      id: "missing-power-of-ten-factor",
      stage_id: stageIdAt(publishedStages, 1),
      prompt: "What missing factor makes 47 × ? = 47,000?",
      answer_type: "multiple_choice",
      options: ["1,000", "100", "10", "47"],
      expected_answer: "1,000",
      explanation_steps: [
        "47 becomes 47,000 by shifting three place-value positions.",
        "Multiplying by 1,000 adds three zeros.",
      ],
      misconception_traps: [
        "Counts only one or two place-value shifts.",
        "Uses the starting number as the factor.",
      ],
    },
    {
      id: "acute-angle-classification",
      stage_id: stageIdAt(publishedStages, 2),
      prompt: "An angle that measures less than 90° is called what?",
      answer_type: "multiple_choice",
      options: ["Acute angle", "Right angle", "Obtuse angle", "Straight angle"],
      expected_answer: "Acute angle",
      explanation_steps: [
        "A right angle is exactly 90°.",
        "Any angle smaller than 90° is acute.",
      ],
      misconception_traps: [
        "Confuses smaller-than-90° angles with right angles.",
        "Uses obtuse for angles that are too small.",
      ],
    },
  ];
}

function genericPreviewQuestions(
  publishedStages: StudentMissionDto["stages"],
): StudentMissionDto["preview_questions"] {
  return [
    {
      id: "mission-first-step",
      stage_id: stageIdAt(publishedStages, 0),
      prompt: "What should you do first in this mission?",
      answer_type: "multiple_choice",
      options: [
        "Use the mission goal and solve one task carefully",
        "Guess from a keyword",
        "Skip the directions",
      ],
      expected_answer: "Use the mission goal and solve one task carefully",
      explanation_steps: [
        "Start with the mission goal.",
        "Use the teacher's instructions before choosing an answer.",
      ],
      misconception_traps: [
        "Starts from a keyword without checking the actual task.",
        "Moves ahead without using the teacher's instructions.",
      ],
    },
    {
      id: "apply-the-idea",
      stage_id: stageIdAt(publishedStages, 1),
      prompt: "Which example best uses this idea?",
      answer_type: "multiple_choice",
      options: [
        "A direct application",
        "A related but incorrect pattern",
        "A memorized definition only",
      ],
      expected_answer: "A direct application",
      explanation_steps: [
        "Use the mission goal to check the example.",
        "A direct application uses the idea in a new task.",
      ],
      misconception_traps: [
        "Chooses a familiar-looking answer without checking the reasoning.",
      ],
    },
    {
      id: "explain-after-answer",
      stage_id: stageIdAt(publishedStages, 2),
      prompt: "What should you do after choosing an answer?",
      answer_type: "multiple_choice",
      options: [
        "Explain the reasoning that supports it",
        "Stop without checking",
        "Change it randomly",
        "Ignore feedback",
      ],
      expected_answer: "Explain the reasoning that supports it",
      explanation_steps: [
        "Mia learns from the reason you give.",
        "Your explanation helps the teacher see what to review next.",
      ],
      misconception_traps: [
        "Treats the answer as finished without reasoning.",
        "Changes strategy without using feedback.",
      ],
    },
  ];
}

function ensureThreePreviewQuestions(
  source: StudentMissionDto,
  publishedStages: StudentMissionDto["stages"],
  questions: StudentMissionDto["preview_questions"],
): StudentMissionDto["preview_questions"] {
  const result = questions.slice(0, studentPreviewQuestionLimit);

  if (result.length >= studentPreviewQuestionLimit) {
    return result;
  }

  const fallback = isGradeFiveMathMission(source)
    ? gradeFiveMathPreviewQuestions(publishedStages)
    : genericPreviewQuestions(publishedStages);
  const seen = new Set(
    result.map((question) => cleanText(question.id).toLocaleLowerCase()),
  );

  fallback.forEach((question) => {
    if (result.length >= studentPreviewQuestionLimit) {
      return;
    }
    const id = cleanText(question.id).toLocaleLowerCase();
    result.push(
      seen.has(id)
        ? { ...question, id: `${question.id}-${result.length + 1}` }
        : question,
    );
    seen.add(id);
  });

  return result;
}

function hasLegacyGenericFallbackQuestion(
  questions: StudentMissionDto["preview_questions"],
) {
  return questions.some(isLegacyGenericFallbackQuestion);
}

function isLegacyGenericFallbackQuestion(
  question: StudentMissionDto["preview_questions"][number],
) {
  const id = cleanText(question.id).toLocaleLowerCase();
  const prompt = cleanText(question.prompt).toLocaleLowerCase();
  const expected = cleanText(question.expected_answer).toLocaleLowerCase();

  return (
    id === "main-idea-recall" ||
    prompt.includes("main idea from today's lesson") ||
    expected.includes("student-friendly explanation of the lesson") ||
    expected === "any honest student preference."
  );
}

function isGradeFiveMathMission(source: StudentMissionDto) {
  const text = [
    source.subject,
    source.grade_level,
    source.learning_goal,
    source.title,
    source.mission_name,
  ]
    .map(cleanText)
    .join(" ")
    .toLocaleLowerCase();
  const hasGradeFive =
    text.includes("grade 5") ||
    text.includes("5th grade") ||
    text.includes("fifth grade");
  const hasMath = text.includes("math") || text.includes("mathematics");
  const hasGradeFiveMathTopic =
    text.includes("5-digit") ||
    text.includes("place value") ||
    text.includes("angle") ||
    text.includes("multiplying by 10") ||
    text.includes("100 or 1,000");

  return hasGradeFiveMathTopic || (hasGradeFive && hasMath);
}

function stageIdAt(stages: StudentMissionDto["stages"], index: number) {
  return cleanText(stages[index]?.id) || POD_ORDER[index % POD_ORDER.length];
}

export function getMissionQuestion(mission: GameMission, id: string): Question {
  const question = mission.questions.find((item) => item.id === id);
  if (!question) {
    throw new Error(`Unknown mission question: ${id}`);
  }

  return question;
}

export function getMissionMisconception(
  mission: GameMission,
  id: string | undefined,
): Misconception | null {
  return id
    ? (mission.misconceptions.find((item) => item.id === id) ?? null)
    : null;
}

export function stageLabel(mission: GameMission, slot: PodId) {
  return mission.stages[slot].title;
}

export function sourceStageID(
  mission: GameMission,
  slot: PodId,
  questionID?: string,
) {
  return (
    (questionID ? mission.sourceQuestionStageIds[questionID] : undefined) ??
    mission.stages[slot].sourceId
  );
}

export function sourceQuestionID(mission: GameMission, questionID: string) {
  return mission.sourceQuestionIds[questionID] ?? questionID;
}

export function slotForStageID(
  mission: GameMission,
  stageID: string,
): PodId | null {
  const normalized = normalizeKey(stageID);

  if (mission.sourceStageSlots[normalized]) {
    return mission.sourceStageSlots[normalized];
  }

  return (
    POD_ORDER.find(
      (slot) =>
        normalizeKey(slot) === normalized ||
        normalizeKey(mission.stages[slot].sourceId) === normalized,
    ) ?? null
  );
}

function convertQuestion(
  source: StudentMissionDto["preview_questions"][number],
  slot: PodId,
  misconceptions: Misconception[],
  index: number,
  idSuffix = "",
): Question {
  const id = `${safeID(source.id || `question-${index + 1}`)}${idSuffix}`;
  const expected = cleanText(source.expected_answer);
  const sourceOptions = cleanTextList(source.options);
  const sourceTraps = cleanTextList(source.misconception_traps);
  const options = uniqueStrings([...sourceOptions, expected, ...sourceTraps])
    .filter(Boolean)
    .slice(0, 4);
  if (options.length < 2) {
    options.push("I need another clue");
  }

  const choiceOptions = options.map((text, optionIndex) => {
    const correct =
      normalizeMissionAnswer(text) === normalizeMissionAnswer(expected);
    if (correct) {
      return { text, correct: true as const };
    }
    const misconceptionId = `${id}-choice-${optionIndex + 1}`;
    const trap =
      sourceTraps[optionIndex] ??
      `Chose “${text}” instead of checking the mission evidence.`;
    misconceptions.push({
      id: misconceptionId,
      skill: slot,
      reasoning: `“${text}” looked like the best match`,
      note: trap,
      kind: "misconception",
    });
    return { text, misconception: misconceptionId };
  });

  const explanationSteps = cleanTextList(source.explanation_steps);
  const steps = explanationSteps.length
    ? explanationSteps
    : [
        `Connect the question to the learning goal.`,
        `The expected answer is: ${expected}`,
      ];

  return {
    kind: "choice",
    id,
    skill: slot,
    difficulty: difficultyForIndex(index),
    prompt: cleanText(source.prompt) || "Answer the mission question.",
    options: choiceOptions,
    explanation: {
      steps,
      visual: {
        kind: "chain",
        lines: [cleanText(source.prompt), ...steps, expected].filter(Boolean),
      },
    },
  };
}

function cleanText(value: unknown) {
  return typeof value === "string" ? value.trim() : "";
}

function cleanTextList(value: unknown) {
  return Array.isArray(value) ? value.map(cleanText).filter(Boolean) : [];
}

function difficultyForIndex(index: number): 1 | 2 | 3 {
  return ((index % 3) + 1) as 1 | 2 | 3;
}

export function normalizeMissionAnswer(value: string) {
  return value
    .trim()
    .toLocaleLowerCase()
    .replace(/[.!?]+$/g, "");
}

function normalizeKey(value: string) {
  return value
    .trim()
    .toLocaleLowerCase()
    .replace(/[_\s]+/g, "-");
}

function safeID(value: string) {
  return normalizeKey(value).replace(/[^a-z0-9-]/g, "") || "question";
}

function uniqueStrings(values: readonly string[]) {
  const seen = new Set<string>();
  return values.filter((value) => {
    const key = normalizeMissionAnswer(value);
    if (!key || seen.has(key)) {
      return false;
    }
    seen.add(key);
    return true;
  });
}
