import type { StudentMissionDto } from "@/modules/missions";

export interface StudentSessionRequest {
  class_code: string;
  student_name: string;
}

export interface StudentSessionValidationRequest {
  class_code: string;
}

export interface StudentSessionValidationResponse {
  valid: boolean;
}

export interface StudentPresenceRequest {
  mission_id: string;
  class_code: string;
  subject: string;
  stage_id: string;
  stage_name: string;
  status:
    | "exploring"
    | "learning"
    | "thinking"
    | "reviewing"
    | "paused"
    | "completed";
}

export interface StudentSessionUserDto {
  id: string;
  identity: {
    provider: string;
    provider_user_id: string;
    auth_method: string;
    sign_in_provider: string;
  };
  profile: {
    display_name: string;
    email: string;
    email_verified: boolean;
    photo_url: string;
  };
  student: {
    organization_id: string;
    district_id: string;
    school_id: string;
    class_id: string;
    class_code: string;
    mission_id: string;
    mission_code: string;
    student_name: string;
  };
  activity: {
    first_seen_at: string;
    last_seen_at: string;
    login_count: number;
    current_game: {
      attempt_id: string;
      mission_id: string;
      published_revision: number;
      subject: string;
      stage_id: string;
      stage_name: string;
      status: string;
      started_at: string;
      last_heartbeat_at: string;
    };
  };
  roles: string[];
  is_guest: boolean;
  deleted_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface StudentSessionResponse {
  user: StudentSessionUserDto;
  class?: StudentClassDto;
  mission?: StudentMissionDto;
}

export interface StudentClassDto {
  id: string;
  class_code: string;
  class_name: string;
  subject: string;
  grade_level: string;
}

export interface EnrollmentPolicy {
  open: boolean;
  expires_at: string;
}

export interface UpsertTeacherClassRequest {
  enrollment?: EnrollmentPolicy;
  class_id?: string;
  class_name: string;
  subject: string;
  grade_level: string;
}

export interface TeacherDto {
  provider_user_id: string;
  display_name: string;
  email: string;
  photo_url: string;
}

export interface TeacherClassDto {
  enrollment?: EnrollmentPolicy;
  id: string;
  organization_id: string;
  district_id: string;
  school_id: string;
  class_id: string;
  class_code: string;
  class_name: string;
  subject: string;
  grade_level: string;
  teacher_id: string;
  teacher: TeacherDto;
  active_mission_id: string;
  mission_count: number;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface TeacherClassResponse {
  teacher_class: TeacherClassDto;
}

export interface TeacherClassesResponse {
  next_cursor?: string;
  teacher_classes: TeacherClassDto[];
}
