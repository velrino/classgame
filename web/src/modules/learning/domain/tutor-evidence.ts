import type { Question } from "@/modules/learning/domain/mission";

export function questionPrompt(question: Question) {
  return question.kind === "choice" ? question.prompt : question.equation;
}

export function expectedAnswer(question: Question) {
  if (question.kind === "choice") {
    return question.options.find((option) => option.correct)?.text ?? "";
  }

  return question.steps.at(-1)?.correct.result ?? "";
}

export function learningEventID(now: number, sequence: number) {
  return `event-${now}-${sequence}`;
}
