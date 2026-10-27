import type { TeacherDto } from "@/modules/classrooms";
import type { SuggestionDto } from "@/modules/insights";

export interface StartStudentMissionRequest {
  class_code: string;
  mission_id: string;
  student_name: string;
}

export interface StudentMissionDto {
  id: string;
  mission_code: string;
  published_revision: number;
  mission_name: string;
  title: string;
  subject: string;
  grade_level: string;
  learning_goal: string;
  stages: MissionStageDto[];
  preview_questions: MissionPreviewQuestionDto[];
}

export interface MissionAttachmentDto {
  uploadcare_uuid: string;
  cdn_url: string;
  uploadcare_cdn_url?: string;
  file_name: string;
  mime_type: string;
  size_bytes: number;
  storage?: {
    status: "pending" | "copying" | "stored" | "failed" | string;
    bucket?: string;
    object_path?: string;
    gs_uri?: string;
    public_url?: string;
    mime_type?: string;
    size_bytes?: number;
    attempts?: number;
    retryable?: boolean;
    error?: string;
    updated_at?: string;
    stored_at?: string;
  };
}

export interface CreateTeacherClassMissionRequest {
  mission_id?: string;
  expected_draft_revision?: number;
  class_id?: string;
  class_name: string;
  mission_name: string;
  subject: string;
  grade_level: string;
  learning_goal: string;
  teacher_prompt: string;
  audience_type: "all" | "selected";
  target_student_ids: string[];
  attachments: MissionAttachmentDto[];
}

export interface MissionPreviewProgressDto {
  stage: string;
  message: string;
  percent?: number;
  current?: number;
  total?: number;
  attachment?: string;
}

export interface MissionStageDto {
  id: string;
  title: string;
  goal: string;
  checks: string[];
  unlocks_when: string;
}

export interface MissionPreviewQuestionDto {
  id: string;
  stage_id: string;
  prompt: string;
  answer_type: "multiple_choice" | "next_step" | "short_text" | string;
  options: string[] | null;
  expected_answer: string;
  explanation_steps: string[] | null;
  misconception_traps: string[] | null;
}

export interface MissionKnowledgeChunkDto {
  id: string;
  source: string;
  title: string;
  content: string;
  keywords: string[];
}

export interface EducationGuardrailDto {
  allowed: boolean;
  reason: string;
  used_attachments: boolean;
  attachment_notes: string;
}

export interface MissionFlowDto {
  nodes: MissionFlowNodeDto[];
  edges: MissionFlowEdgeDto[];
}

export interface MissionFlowNodeDto {
  id: string;
  kind:
    | "start"
    | "setup"
    | "activity"
    | "question"
    | "decision"
    | "feedback"
    | "complete"
    | "guardrail"
    | "answer"
    | "misconception"
    | "adaptation"
    | "mastery"
    | string;
  label: string;
  description: string;
  evidence_event_ids: string[];
  position: {
    x: number;
    y: number;
  };
}

export interface MissionFlowEdgeDto {
  id: string;
  source: string;
  target: string;
  label: string;
}

export interface MissionOrchestrationDto {
  provider: string;
  model: string;
  model_version: string;
  generated_by_ai: boolean;
  generated_by_mock: boolean;
  fallback_reason: string;
  prompt_token_count: number;
  candidates_token_count: number;
  thoughts_token_count: number;
  total_token_count: number;
}

export interface TeacherClassMissionDto {
  id: string;
  mission_code: string;
  organization_id: string;
  district_id: string;
  school_id: string;
  teacher_id: string;
  class_id: string;
  class_code: string;
  class_name: string;
  mission_name: string;
  title: string;
  summary: string;
  subject: string;
  grade_level: string;
  learning_goal: string;
  teacher_prompt: string;
  audience_type: "all" | "selected";
  target_student_ids: string[];
  attachments: MissionAttachmentDto[];
  preview_questions: MissionPreviewQuestionDto[];
  knowledge_chunks: MissionKnowledgeChunkDto[];
  suggested_duration_minutes: number;
  stages: MissionStageDto[];
  suggestions: SuggestionDto[];
  positive_signals: string[];
  improvement_signals: string[];
  education_guardrail: EducationGuardrailDto;
  mission_flow: MissionFlowDto;
  orchestration: MissionOrchestrationDto;
  status: string;
  draft_revision?: number;
  published_revision?: number;
  has_unpublished_changes?: boolean;
  deleted_at?: string | null;
  created_by: TeacherDto;
  updated_by: TeacherDto;
  published_at: string;
  created_at: string;
  updated_at: string;
}

export interface TeacherClassMissionResponse {
  mission: TeacherClassMissionDto;
}

export interface ApproveTeacherClassMissionDraftDto {
  title: string;
  summary: string;
  preview_questions: MissionPreviewQuestionDto[];
}

export interface ApproveTeacherClassMissionRequest {
  expected_draft_revision?: number;
  draft?: ApproveTeacherClassMissionDraftDto;
}

export interface TeacherClassMissionsResponse {
  next_cursor?: string;
  missions: TeacherClassMissionDto[];
}
