import type { MissionPreviewQuestionDto } from "@/modules/missions/adapters/types/contracts";

export const studentPreviewQuestionLimit = 3;

export function normalizeMissionPreviewQuestions(
  value: unknown,
): MissionPreviewQuestionDto[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .slice(0, studentPreviewQuestionLimit)
    .map(normalizeMissionPreviewQuestion);
}

export function normalizeMissionPreviewQuestion(
  value: unknown,
): MissionPreviewQuestionDto {
  const question = isRecord(value) ? value : {};

  return {
    id: safeText(question.id),
    stage_id: safeText(question.stage_id),
    prompt: safeText(question.prompt),
    answer_type: "multiple_choice",
    options: normalizeMultipleChoiceOptions(
      safeText(question.expected_answer),
      question.options,
      question.misconception_traps,
    ),
    expected_answer: safeText(question.expected_answer),
    explanation_steps: safeStringList(question.explanation_steps),
    misconception_traps: safeStringList(question.misconception_traps),
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function safeStringList(value: unknown): string[] {
  return Array.isArray(value)
    ? value
        .filter((item): item is string => typeof item === "string")
        .map((item) => item.trim())
        .filter(Boolean)
    : [];
}

function safeText(value: unknown) {
  return typeof value === "string" ? value.trim() : "";
}

function normalizeMultipleChoiceOptions(
  expectedAnswer: string,
  options: unknown,
  traps: unknown,
) {
  const expected = expectedAnswer.trim();
  const result: string[] = [];
  const seen = new Set<string>();
  const add = (value: string) => {
    if (result.length >= 4) {
      return;
    }
    const text = value.trim();
    const key = text.toLocaleLowerCase();
    if (!text || seen.has(key)) {
      return;
    }
    seen.add(key);
    result.push(text);
  };

  safeStringList(options).forEach(add);
  if (expected && !seen.has(expected.toLocaleLowerCase())) {
    if (result.length >= 4) {
      seen.delete(result[result.length - 1]?.toLocaleLowerCase() ?? "");
      result[result.length - 1] = expected;
      seen.add(expected.toLocaleLowerCase());
    } else {
      add(expected);
    }
  }
  safeStringList(traps).forEach(add);
  add("I need another clue");
  add("Not enough information");

  return result;
}
