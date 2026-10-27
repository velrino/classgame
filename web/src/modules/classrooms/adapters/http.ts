import type { StudentClassDto } from "@/modules/classrooms/adapters/types/contracts";
import { httpClient } from "@/shared/http/client";
import { apiError } from "@/shared/http/errors";
import { collectPages } from "@/shared/http/pagination";
import { HttpMethod } from "@/shared/http/types";

import type {
  StudentPresenceRequest,
  StudentSessionRequest,
  StudentSessionResponse,
  StudentSessionValidationRequest,
  StudentSessionValidationResponse,
  TeacherClassResponse,
  TeacherClassesResponse,
  UpsertTeacherClassRequest,
} from "@/modules/classrooms/adapters/types/contracts";
export const classroomsApi = {
  async currentStudentClass(accessToken: string): Promise<{
    class: StudentClassDto & { status: string; active_mission_id: string };
  }> {
    const response = await httpClient.request({
      accessToken,
      method: HttpMethod.GET,
      path: "/users/student-class",
    });
    if (!response.ok)
      throw await apiError(response, "STUDENT_CLASS_GET_FAILED");
    return response.json();
  },
  async validateStudentSession(
    data: StudentSessionValidationRequest,
  ): Promise<StudentSessionValidationResponse> {
    const res = await httpClient.request({
      data,
      method: HttpMethod.POST,
      path: "/users/student-session/validate",
    });

    if (!res.ok) {
      throw await apiError(res, "STUDENT_SESSION_VALIDATION_FAILED");
    }

    return (await res.json()) as StudentSessionValidationResponse;
  },
  async studentSession(
    accessToken: string,
    data: StudentSessionRequest,
  ): Promise<StudentSessionResponse> {
    const res = await httpClient.request({
      accessToken,
      data,
      method: HttpMethod.POST,
      path: "/users/student-session",
    });

    if (!res.ok) {
      throw await apiError(res, "STUDENT_SESSION_FAILED");
    }

    return (await res.json()) as StudentSessionResponse;
  },
  async studentPresence(
    accessToken: string,
    data: StudentPresenceRequest,
  ): Promise<StudentSessionResponse> {
    const res = await httpClient.request({
      accessToken,
      data,
      method: HttpMethod.POST,
      path: "/users/student-presence",
    });

    if (!res.ok) {
      throw await apiError(res, "STUDENT_PRESENCE_FAILED");
    }

    return (await res.json()) as StudentSessionResponse;
  },
  async list(accessToken: string): Promise<TeacherClassesResponse> {
    const items = await collectPages(async (cursor) => {
      const response = await httpClient.request({
        accessToken,
        method: HttpMethod.GET,
        path: "/teacher-classes",
        data: { limit: 100, cursor },
      });
      if (!response.ok) throw await apiError(response, "LIST_FAILED");
      const page = (await response.json()) as TeacherClassesResponse;
      return { items: page.teacher_classes, nextCursor: page.next_cursor };
    });
    return { teacher_classes: items };
  },
  async get(
    accessToken: string,
    class_id: string,
  ): Promise<TeacherClassResponse> {
    const res = await httpClient.request({
      accessToken,
      method: HttpMethod.GET,
      path: `/teacher-classes/${encodeURIComponent(class_id)}`,
    });

    if (!res.ok) {
      throw await apiError(res, "TEACHER_CLASS_GET_FAILED");
    }

    return (await res.json()) as TeacherClassResponse;
  },
  async upsert(
    accessToken: string,
    data: UpsertTeacherClassRequest,
  ): Promise<TeacherClassResponse> {
    const res = await httpClient.request({
      accessToken,
      data,
      method: HttpMethod.POST,
      path: "/teacher-classes",
    });

    if (!res.ok) {
      throw await apiError(res, "TEACHER_CLASS_UPSERT_FAILED");
    }

    return (await res.json()) as TeacherClassResponse;
  },
};
