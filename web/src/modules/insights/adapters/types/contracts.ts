import type { TeacherClassDto, TeacherDto } from "@/modules/classrooms";
import type { StudentLearningAttemptDto } from "@/modules/learning";
import type { TeacherClassMissionDto } from "@/modules/missions";

export interface DashboardOverviewRequest {
  class_id?: string;
  limit?: number;
  cursor?: string;
}

export interface SuggestionDto {
  id: string;
  title: string;
  reason: string;
  activity: string;
  questions: string[];
  focus_skills: string[];
}

export interface DashboardMetricsDto {
  total_students: number;
  active_students: number;
  completed_students: number;
  average_progress: number;
  needs_attention: number;
  abandoned_students: number;
  notebook_notes: number;
}

export interface StudentProgressDto {
  completed_questions: number;
  total_questions: number;
  percentage: number;
  status:
    | "completed"
    | "in_progress"
    | "needs_attention"
    | "not_started"
    | string;
}

export interface DashboardStudentDto {
  id: string;
  student_name: string;
  class_id: string;
  progress: StudentProgressDto;
  strengths: string[];
  improvement_areas: string[];
  last_activity_at: string;
  notebook_notes: number;
  attempts: StudentLearningAttemptDto[];
  session?: {
    mission_id: string;
    subject: string;
    stage_id: string;
    stage_name: string;
    status: string;
    is_online: boolean;
    started_at: string;
    last_heartbeat_at: string;
  };
}

export interface ClassLearningPatternDto {
  label: string;
  count: number;
  student_ids: string[];
  student_names: string[];
}

export interface ClassLearningSummaryDto {
  total_attempts: number;
  completed_attempts: number;
  abandoned_attempts: number;
  students_with_evidence: number;
  common_strengths: ClassLearningPatternDto[];
  common_needs: ClassLearningPatternDto[];
  next_adaptation: string;
}

export interface DashboardPageDto {
  limit: number;
  returned: number;
  next_cursor: string;
  read_at: string;
  metrics_scope: "student_page";
  history_scope: "latest_attempt_summary";
}
export interface StudentHistoryResponse {
  attempts: StudentLearningAttemptDto[];
  next_cursor: string;
  read_at: string;
}
export interface DashboardOverviewResponse {
  page?: DashboardPageDto;
  organization_id: string;
  district_id: string;
  school_id: string;
  class_id: string;
  class_code: string;
  teacher_class: TeacherClassDto | null;
  active_mission: TeacherClassMissionDto | null;
  metrics: DashboardMetricsDto;
  students: DashboardStudentDto[];
  class_learning: ClassLearningSummaryDto;
  suggestions: SuggestionDto[];
  generated_by_ai: boolean;
  generated_by_mock: boolean;
  teacher: TeacherDto;
}
