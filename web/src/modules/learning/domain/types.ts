export type PodId = "fractions" | "simple-equations" | "two-step-equations";

export const POD_ORDER: readonly PodId[] = [
  "fractions",
  "simple-equations",
  "two-step-equations",
];

export const POD_LABELS: Record<PodId, string> = {
  fractions: "Fractions",
  "simple-equations": "Simple equations",
  "two-step-equations": "Two-step equations",
};

/** locked → ready → complete; "review" is a completed pod reopened by the tutor. */
export type PodStatus = "locked" | "ready" | "review" | "complete";

export type FeedbackValue = "helpful" | "confusing" | "too-easy" | "too-hard";

export type ExplanationStyle = "steps" | "visual";
