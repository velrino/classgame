export { learningApi } from "./adapters/http";
export {
  clearMemory,
  loadStudentRecovery,
  saveMemory,
} from "./adapters/recovery-storage";
export type * from "./adapters/types/contracts";
export { createInitialMemory, normalizeStudentMemory } from "./domain/memory";
export type {
  PodProgress,
  StudentMemory,
  StudentSessionCheckpoint,
} from "./domain/memory";
export { TUTOR_NAME } from "./domain/mission";
export type { Explanation, Question, Visual } from "./domain/mission";
export {
  buildGameMission,
  normalizeMissionAnswer,
  slotForStageID,
  sourceQuestionID,
  sourceStageID,
  stageLabel,
} from "./domain/published-mission";
export type { GameMission } from "./domain/published-mission";
export {
  createInitialSession,
  sessionReducer,
  shuffledMoves,
} from "./domain/session-reducer";
export type { Lesson, Session } from "./domain/session-reducer";
export { THINKING_MS, buildWhyOptions } from "./domain/tutor-policy";
export { POD_LABELS, POD_ORDER } from "./domain/types";
export type {
  ExplanationStyle,
  FeedbackValue,
  PodId,
  PodStatus,
} from "./domain/types";
export { useLearningSession } from "./hooks/use-learning-session";
