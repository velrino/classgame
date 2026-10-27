import type { MissionFlowDto } from "@/modules/missions";

export type StudentLearningStatus =
  | "exploring"
  | "learning"
  | "thinking"
  | "reviewing"
  | "paused"
  | "completed";

export interface StudentLearningSnapshotRequest {
  mission_id: string;
  status: StudentLearningStatus;
  snapshot: StudentLearningSnapshotDto;
}

export interface StudentLearningSnapshotDto {
  revision: number;
  progress: StudentStageProgressDto[];
  notes: StudentLearningNoteDto[];
  preferences: StudentPreferencesDto;
  decisions: StudentDecisionDto[];
  events: StudentLearningEventDto[];
  redirected: boolean;
  return_to: string;
  last_pod: string;
  checkpoint: StudentSessionCheckpointDto;
}

export interface StudentSessionCheckpointDto {
  stage_id: string;
  question_id: string;
  phase: string;
  step_index: number;
  explain_index: number;
  lines: string[];
  wrong_choice: string;
  wrong_misconception_id: string;
  decision_action: string;
  decision_target: string;
  decision_message: string;
  decision_why: string;
  tutor_message: string;
  why_prompt: string;
  clean: boolean;
  seated: boolean;
}

export interface StudentStageProgressDto {
  stage_id: string;
  status: "locked" | "ready" | "review" | "complete" | string;
  queue: string[];
  pool: string[];
  correct: number;
  required: number;
}

export interface StudentLearningNoteDto {
  id: string;
  misconception_id: string;
  text: string;
  kind: "misconception" | "habit" | string;
  skill: string;
  evidence: string[];
  count: number;
  created_at: number;
}

export interface StudentPreferencesDto {
  explanation_style: "steps" | "visual" | string;
  difficulty_bias: number;
}

export interface StudentDecisionDto {
  id: string;
  at: number;
  action: string;
  message: string;
  why: string;
}

export interface StudentLearningEventDto {
  id: string;
  sequence: number;
  type:
    | "question"
    | "answer"
    | "reasoning"
    | "feedback"
    | "adaptation"
    | "mastery"
    | string;
  stage_id: string;
  question_id: string;
  title: string;
  detail: string;
  outcome:
    | "neutral"
    | "correct"
    | "incorrect"
    | "support"
    | "complete"
    | string;
  occurred_at: number;
}

export interface StudentLearningResponse {
  attempt: StudentLearningResumeAttemptDto | null;
}

export interface StudentLearningResumeAttemptDto {
  id: string;
  mission_id: string;
  published_revision: number;
  status: StudentLearningStatus | "abandoned" | string;
  snapshot: {
    revision: number;
    progress: StudentStageProgressDto[];
    preferences: StudentPreferencesDto;
    redirected: boolean;
    return_to: string;
    last_pod: string;
    checkpoint: StudentSessionCheckpointDto;
  };
  last_event_sequence: number;
  started_at: string;
  last_activity_at: string;
}

export interface StudentTutorTurnRequest {
  mission_id: string;
  event_id: string;
  type: "answer" | "reasoning" | "feedback" | "resume";
  stage_id: string;
  question_id: string;
  prompt: string;
  student_response: string;
  expected_answer: string;
  outcome: "neutral" | "correct" | "incorrect" | "support" | "complete";
  explicit_reasoning: string;
  feedback: string;
}

export interface StudentTutorTurnResponse {
  turn: {
    turn_id: string;
    sequence: number;
    message: string;
    why_prompt: string;
    explanation_steps: string[];
    explanation_style: "steps" | "visual";
    difficulty_bias: -1 | 0 | 1;
    next_action:
      | "continue"
      | "ask_reasoning"
      | "explain"
      | "retry"
      | "complete";
  };
}

export interface StudentLearningAttemptDto {
  id: string;
  organization_id: string;
  district_id: string;
  school_id: string;
  student_id: string;
  student_name: string;
  class_id: string;
  mission_id: string;
  mission_code: string;
  published_revision: number;
  subject: string;
  status: StudentLearningStatus | string;
  snapshot: StudentLearningSnapshotDto;
  tutor_turns: StudentTutorTurnDto[];
  trace: StudentLearningTraceDto;
  trace_state: "live" | "pending" | "processing" | "ready" | "failed" | string;
  started_at: string;
  last_activity_at: string;
  completed_at: string;
  abandoned_at: string;
  created_at: string;
  updated_at: string;
}

export interface StudentTutorTurnDto {
  id: string;
  sequence: number;
  type: string;
  stage_id: string;
  question_id: string;
  prompt: string;
  student_response: string;
  expected_answer: string;
  outcome: string;
  explicit_reasoning: string;
  feedback: string;
  tutor: {
    message: string;
    why_prompt: string;
    explanation_steps: string[];
    explanation_style: string;
    difficulty_bias: number;
    next_action: string;
    teacher_observation: string;
    retrieved_chunk_ids: string[];
    evidence_event_ids: string[];
    generated_by_ai: boolean;
    provider: string;
    model: string;
    model_version: string;
    fallback_reason: string;
  };
  created_at: string;
}

export interface StudentLearningTraceDto {
  summary: string;
  strengths: string[];
  improvement_areas: string[];
  next_adaptation: string;
  flow: MissionFlowDto;
  finalized: boolean;
  event_count: number;
  generated_by_ai: boolean;
  generated_by_mock: boolean;
  provider: string;
  model: string;
  model_version: string;
  fallback_reason: string;
  usage: {
    prompt_token_count: number;
    candidates_token_count: number;
    thoughts_token_count: number;
    total_token_count: number;
  };
}
