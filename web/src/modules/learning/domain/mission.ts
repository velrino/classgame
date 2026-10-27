import type { PodId } from "@/modules/learning/domain/types";

export const TUTOR_NAME = "Mia";

export type NoteKind = "misconception" | "habit";

/** A named way of thinking the tutor can recognise, ask about and write down. */
export type Misconception = {
  readonly id: string;
  readonly skill: PodId;
  /** Shown as a "why did you pick that?" option, in the student's voice. */
  readonly reasoning: string;
  /** What the tutor writes in the notebook. */
  readonly note: string;
  readonly kind: NoteKind;
};

export const GUESSED_ID = "guessed";

export const MISCONCEPTIONS: readonly Misconception[] = [
  {
    id: "simplify-parts-separately",
    skill: "fractions",
    reasoning: "I divided only the top number",
    note: "Simplifies the numerator and denominator by different numbers",
    kind: "misconception",
  },
  {
    id: "add-across",
    skill: "fractions",
    reasoning: "I added the tops together and the bottoms together",
    note: "Adds numerators and denominators straight across",
    kind: "misconception",
  },
  {
    id: "bigger-denominator-bigger",
    skill: "fractions",
    reasoning: "A bigger bottom number means a bigger fraction",
    note: "Thinks a bigger denominator makes a bigger fraction",
    kind: "misconception",
  },
  {
    id: "divide-to-clear-denominator",
    skill: "fractions",
    reasoning: "To get rid of the /3 I divided by 3",
    note: "Divides instead of multiplying to clear a denominator",
    kind: "misconception",
  },
  {
    id: "sign-not-flipped",
    skill: "simple-equations",
    reasoning: "I moved the number across without changing its sign",
    note: "Moves a term across the equals sign without flipping the sign",
    kind: "misconception",
  },
  {
    id: "multiply-confused-with-add",
    skill: "simple-equations",
    reasoning: "3x looked like 3 + x to me",
    note: "Reads 3x as 3 + x",
    kind: "misconception",
  },
  {
    id: "isolate-x-first",
    skill: "two-step-equations",
    reasoning: "I wanted x alone right away, so I divided first",
    note: "Divides to isolate x before clearing the constant term",
    kind: "misconception",
  },
  {
    id: GUESSED_ID,
    skill: "fractions",
    reasoning: "Honestly, I guessed",
    note: "Guesses when unsure — ask for the reasoning before the answer",
    kind: "habit",
  },
];

export type Visual =
  | {
      readonly kind: "fraction-bars";
      readonly bars: readonly {
        readonly num: number;
        readonly den: number;
        readonly label: string;
      }[];
    }
  | { readonly kind: "chain"; readonly lines: readonly string[] };

export type Explanation = {
  readonly steps: readonly string[];
  readonly visual: Visual;
};

export type ChoiceOption = {
  readonly text: string;
  readonly correct?: boolean;
  readonly misconception?: string;
};

export type ChoiceQuestion = {
  readonly kind: "choice";
  readonly id: string;
  readonly skill: PodId;
  readonly difficulty: 1 | 2 | 3;
  readonly prompt: string;
  readonly options: readonly ChoiceOption[];
  readonly explanation: Explanation;
};

export type StepMove = {
  readonly op: string;
  readonly result: string;
  readonly misconception?: string;
};

export type EquationStep = {
  readonly correct: StepMove;
  readonly distractors: readonly StepMove[];
};

export type StepQuestion = {
  readonly kind: "steps";
  readonly id: string;
  readonly skill: PodId;
  readonly difficulty: 1 | 2 | 3;
  readonly equation: string;
  readonly steps: readonly EquationStep[];
  readonly explanation: Explanation;
};

export type Question = ChoiceQuestion | StepQuestion;

export const QUESTIONS: readonly Question[] = [
  // ── Fractions ────────────────────────────────────────────────────────────
  {
    kind: "choice",
    id: "f1",
    skill: "fractions",
    difficulty: 1,
    prompt: "Simplify 6/9.",
    options: [
      { text: "2/3", correct: true },
      { text: "6/3", misconception: "simplify-parts-separately" },
      { text: "3/2" },
      { text: "1/9" },
    ],
    explanation: {
      steps: [
        "6 and 9 share a factor: 3.",
        "Divide BOTH by 3 → 6 ÷ 3 = 2 and 9 ÷ 3 = 3.",
        "So 6/9 = 2/3. Top and bottom shrink by the same number.",
      ],
      visual: {
        kind: "fraction-bars",
        bars: [
          { num: 6, den: 9, label: "6/9" },
          { num: 2, den: 3, label: "2/3" },
        ],
      },
    },
  },
  {
    kind: "choice",
    id: "f2",
    skill: "fractions",
    difficulty: 2,
    prompt: "1/2 + 1/3 = ?",
    options: [
      { text: "5/6", correct: true },
      { text: "2/5", misconception: "add-across" },
      { text: "2/6" },
      { text: "1/5" },
    ],
    explanation: {
      steps: [
        "Different denominators can't be added yet.",
        "Rewrite both over 6 → 1/2 = 3/6 and 1/3 = 2/6.",
        "Now add only the tops → 3/6 + 2/6 = 5/6.",
      ],
      visual: {
        kind: "fraction-bars",
        bars: [
          { num: 3, den: 6, label: "1/2 = 3/6" },
          { num: 2, den: 6, label: "1/3 = 2/6" },
          { num: 5, den: 6, label: "5/6" },
        ],
      },
    },
  },
  {
    kind: "choice",
    id: "f3",
    skill: "fractions",
    difficulty: 1,
    prompt: "Which is bigger: 1/4 or 1/8?",
    options: [
      { text: "1/4", correct: true },
      { text: "1/8", misconception: "bigger-denominator-bigger" },
      { text: "They're equal" },
    ],
    explanation: {
      steps: [
        "Cut one bar into 4 pieces and another into 8.",
        "More pieces → each piece is smaller.",
        "So 1/4 is bigger than 1/8.",
      ],
      visual: {
        kind: "fraction-bars",
        bars: [
          { num: 1, den: 4, label: "1/4" },
          { num: 1, den: 8, label: "1/8" },
        ],
      },
    },
  },
  {
    kind: "choice",
    id: "f4",
    skill: "fractions",
    difficulty: 2,
    prompt: "Simplify 8/12.",
    options: [
      { text: "2/3", correct: true },
      { text: "8/3", misconception: "simplify-parts-separately" },
      { text: "4/6" },
      { text: "2/4" },
    ],
    explanation: {
      steps: [
        "8 and 12 share a factor: 4.",
        "Divide BOTH by 4 → 8 ÷ 4 = 2 and 12 ÷ 4 = 3.",
        "So 8/12 = 2/3.",
      ],
      visual: {
        kind: "fraction-bars",
        bars: [
          { num: 8, den: 12, label: "8/12" },
          { num: 2, den: 3, label: "2/3" },
        ],
      },
    },
  },
  {
    kind: "choice",
    id: "f5",
    skill: "fractions",
    difficulty: 3,
    prompt: "3/4 − 1/2 = ?",
    options: [
      { text: "1/4", correct: true },
      { text: "2/2", misconception: "add-across" },
      { text: "2/4" },
      { text: "1/2" },
    ],
    explanation: {
      steps: [
        "Match the denominators first → 1/2 = 2/4.",
        "Now subtract only the tops → 3/4 − 2/4 = 1/4.",
      ],
      visual: {
        kind: "fraction-bars",
        bars: [
          { num: 3, den: 4, label: "3/4" },
          { num: 2, den: 4, label: "1/2 = 2/4" },
          { num: 1, den: 4, label: "1/4" },
        ],
      },
    },
  },
  {
    kind: "choice",
    id: "f6",
    skill: "fractions",
    difficulty: 1,
    prompt: "Which is bigger: 2/3 or 2/5?",
    options: [
      { text: "2/3", correct: true },
      { text: "2/5", misconception: "bigger-denominator-bigger" },
      { text: "They're equal" },
    ],
    explanation: {
      steps: [
        "Same number of pieces (2) each time.",
        "Thirds are bigger pieces than fifths.",
        "So 2/3 is bigger.",
      ],
      visual: {
        kind: "fraction-bars",
        bars: [
          { num: 2, den: 3, label: "2/3" },
          { num: 2, den: 5, label: "2/5" },
        ],
      },
    },
  },

  // ── Simple equations (one move) ─────────────────────────────────────────
  {
    kind: "steps",
    id: "s1",
    skill: "simple-equations",
    difficulty: 1,
    equation: "x + 4 = 9",
    steps: [
      {
        correct: { op: "− 4 on both sides", result: "x = 5" },
        distractors: [
          {
            op: "+ 4 on both sides",
            result: "x + 8 = 13",
            misconception: "sign-not-flipped",
          },
          {
            op: "÷ 4 on both sides",
            result: "x/4 + 1 = 9/4",
            misconception: "isolate-x-first",
          },
          { op: "× 4 on both sides", result: "4x + 16 = 36" },
        ],
      },
    ],
    explanation: {
      steps: [
        "x has a + 4 stuck to it.",
        "Undo + 4 with − 4, on BOTH sides.",
        "Left: x. Right: 9 − 4 = 5.",
      ],
      visual: {
        kind: "chain",
        lines: ["x + 4 = 9", "− 4 on both sides", "x = 5"],
      },
    },
  },
  {
    kind: "steps",
    id: "s2",
    skill: "simple-equations",
    difficulty: 2,
    equation: "3x = 12",
    steps: [
      {
        correct: { op: "÷ 3 on both sides", result: "x = 4" },
        distractors: [
          {
            op: "− 3 on both sides",
            result: "3x − 3 = 9",
            misconception: "multiply-confused-with-add",
          },
          { op: "× 3 on both sides", result: "9x = 36" },
          {
            op: "+ 3 on both sides",
            result: "3x + 3 = 15",
            misconception: "multiply-confused-with-add",
          },
        ],
      },
    ],
    explanation: {
      steps: [
        "3x means 3 × x, not 3 + x.",
        "Undo × 3 with ÷ 3, on BOTH sides.",
        "Left: x. Right: 12 ÷ 3 = 4.",
      ],
      visual: {
        kind: "chain",
        lines: ["3x = 12", "÷ 3 on both sides", "x = 4"],
      },
    },
  },
  {
    kind: "steps",
    id: "s3",
    skill: "simple-equations",
    difficulty: 1,
    equation: "x − 7 = 2",
    steps: [
      {
        correct: { op: "+ 7 on both sides", result: "x = 9" },
        distractors: [
          {
            op: "− 7 on both sides",
            result: "x − 14 = −5",
            misconception: "sign-not-flipped",
          },
          { op: "× 7 on both sides", result: "7x − 49 = 14" },
          {
            op: "÷ 7 on both sides",
            result: "x/7 − 1 = 2/7",
            misconception: "isolate-x-first",
          },
        ],
      },
    ],
    explanation: {
      steps: [
        "x has a − 7 stuck to it.",
        "Undo − 7 with + 7, on BOTH sides.",
        "Left: x. Right: 2 + 7 = 9.",
      ],
      visual: {
        kind: "chain",
        lines: ["x − 7 = 2", "+ 7 on both sides", "x = 9"],
      },
    },
  },

  // ── Two-step equations ──────────────────────────────────────────────────
  {
    kind: "steps",
    id: "t1",
    skill: "two-step-equations",
    difficulty: 1,
    equation: "2x + 1 = 5",
    steps: [
      {
        correct: { op: "− 1 on both sides", result: "2x = 4" },
        distractors: [
          {
            op: "+ 1 on both sides",
            result: "2x + 2 = 6",
            misconception: "sign-not-flipped",
          },
          {
            op: "÷ 2 on both sides",
            result: "x + 1/2 = 5/2",
            misconception: "isolate-x-first",
          },
          { op: "× 2 on both sides", result: "4x + 2 = 10" },
        ],
      },
      {
        correct: { op: "÷ 2 on both sides", result: "x = 2" },
        distractors: [
          { op: "× 2 on both sides", result: "4x = 8" },
          {
            op: "− 2 on both sides",
            result: "2x − 2 = 2",
            misconception: "multiply-confused-with-add",
          },
          { op: "+ 2 on both sides", result: "2x + 2 = 6" },
        ],
      },
    ],
    explanation: {
      steps: [
        "Clear what's ADDED first: − 1 on both sides → 2x = 4.",
        "Then clear what's MULTIPLIED: ÷ 2 on both sides → x = 2.",
        "Undo in reverse order: add/subtract first, multiply/divide last.",
      ],
      visual: {
        kind: "chain",
        lines: [
          "2x + 1 = 5",
          "− 1 on both sides",
          "2x = 4",
          "÷ 2 on both sides",
          "x = 2",
        ],
      },
    },
  },
  {
    kind: "steps",
    id: "t2",
    skill: "two-step-equations",
    difficulty: 3,
    equation: "2x/3 + 1 = 5",
    steps: [
      {
        correct: { op: "− 1 on both sides", result: "2x/3 = 4" },
        distractors: [
          {
            op: "+ 1 on both sides",
            result: "2x/3 + 2 = 6",
            misconception: "sign-not-flipped",
          },
          {
            op: "÷ 2 on both sides",
            result: "x/3 + 1/2 = 5/2",
            misconception: "isolate-x-first",
          },
          { op: "× 3 on both sides", result: "2x + 3 = 15" },
        ],
      },
      {
        correct: { op: "× 3 on both sides", result: "2x = 12" },
        distractors: [
          {
            op: "÷ 3 on both sides",
            result: "2x/9 = 4/3",
            misconception: "divide-to-clear-denominator",
          },
          { op: "− 3 on both sides", result: "2x/3 − 3 = 1" },
          { op: "+ 3 on both sides", result: "2x/3 + 3 = 7" },
        ],
      },
      {
        correct: { op: "÷ 2 on both sides", result: "x = 6" },
        distractors: [
          { op: "× 2 on both sides", result: "4x = 24" },
          {
            op: "− 2 on both sides",
            result: "2x − 2 = 10",
            misconception: "multiply-confused-with-add",
          },
          { op: "+ 2 on both sides", result: "2x + 2 = 14" },
        ],
      },
    ],
    explanation: {
      steps: [
        "Clear the + 1 first → 2x/3 = 4.",
        "The /3 is a division. Undo it by MULTIPLYING by 3 → 2x = 12.",
        "Finally undo × 2 with ÷ 2 → x = 6.",
      ],
      visual: {
        kind: "chain",
        lines: [
          "2x/3 + 1 = 5",
          "− 1 on both sides",
          "2x/3 = 4",
          "× 3 on both sides",
          "2x = 12",
          "÷ 2 on both sides",
          "x = 6",
        ],
      },
    },
  },
  {
    kind: "steps",
    id: "t3",
    skill: "two-step-equations",
    difficulty: 2,
    equation: "x/2 − 3 = 1",
    steps: [
      {
        correct: { op: "+ 3 on both sides", result: "x/2 = 4" },
        distractors: [
          {
            op: "− 3 on both sides",
            result: "x/2 − 6 = −2",
            misconception: "sign-not-flipped",
          },
          {
            op: "÷ 2 on both sides",
            result: "x/4 − 3/2 = 1/2",
            misconception: "isolate-x-first",
          },
          { op: "× 3 on both sides", result: "3x/2 − 9 = 3" },
        ],
      },
      {
        correct: { op: "× 2 on both sides", result: "x = 8" },
        distractors: [
          {
            op: "÷ 2 on both sides",
            result: "x/4 = 2",
            misconception: "divide-to-clear-denominator",
          },
          { op: "+ 2 on both sides", result: "x/2 + 2 = 6" },
          { op: "− 2 on both sides", result: "x/2 − 2 = 2" },
        ],
      },
    ],
    explanation: {
      steps: [
        "Clear the − 3 first → x/2 = 4.",
        "x/2 means x ÷ 2. Undo it by MULTIPLYING by 2 → x = 8.",
      ],
      visual: {
        kind: "chain",
        lines: [
          "x/2 − 3 = 1",
          "+ 3 on both sides",
          "x/2 = 4",
          "× 2 on both sides",
          "x = 8",
        ],
      },
    },
  },
];

export type PodPlan = {
  readonly id: PodId;
  readonly tagline: string;
  /** Questions every student sees, in order. */
  readonly primary: readonly string[];
  /** Extra questions the tutor can pull for retries or a review. */
  readonly reinforcement: readonly string[];
  /** Clean answers needed to mark the pod complete. */
  readonly required: number;
};

export const POD_PLANS: Record<PodId, PodPlan> = {
  fractions: {
    id: "fractions",
    tagline: "Simplify, compare and add fractions.",
    primary: ["f1", "f2", "f3"],
    reinforcement: ["f4", "f6", "f5"],
    required: 3,
  },
  "simple-equations": {
    id: "simple-equations",
    tagline: "Get x alone with one move.",
    primary: ["s1", "s2"],
    reinforcement: ["s3"],
    required: 2,
  },
  "two-step-equations": {
    id: "two-step-equations",
    tagline: "Two moves, in the right order.",
    primary: ["t1", "t2"],
    reinforcement: ["t3"],
    required: 2,
  },
};

const questionIndex = new Map(
  QUESTIONS.map((question) => [question.id, question]),
);
const misconceptionIndex = new Map(
  MISCONCEPTIONS.map((item) => [item.id, item]),
);

export function getQuestion(id: string): Question {
  const question = questionIndex.get(id);

  if (!question) {
    throw new Error(`Unknown question: ${id}`);
  }

  return question;
}

export function getMisconception(id: string | undefined): Misconception | null {
  return id ? (misconceptionIndex.get(id) ?? null) : null;
}
