import type { StudentSessionResponse } from "@/modules/classrooms";
import type { AttachmentAccess } from "@/modules/missions/adapters/types/attachment-access";
import { readBoundedBlob } from "@/shared/http/blob";
import { httpClient } from "@/shared/http/client";
import { apiError } from "@/shared/http/errors";
import { collectPages } from "@/shared/http/pagination";
import { HttpMethod } from "@/shared/http/types";

import { readMissionPreviewEventStream } from "@/modules/missions/adapters/preview-stream";
import type {
  ApproveTeacherClassMissionRequest,
  CreateTeacherClassMissionRequest,
  MissionPreviewProgressDto,
  StartStudentMissionRequest,
  TeacherClassMissionResponse,
  TeacherClassMissionsResponse,
} from "@/modules/missions/adapters/types/contracts";
export const missionsApi = {
  async downloadAttachment(
    accessToken: string,
    access: AttachmentAccess,
    uploadcareUuid: string,
    signal?: AbortSignal,
  ): Promise<Blob> {
    const response = await httpClient.request({
      accessToken,
      method: HttpMethod.GET,
      path: `/teacher-classes/${encodeURIComponent(access.classId)}/missions/${encodeURIComponent(access.missionId)}/attachments/${encodeURIComponent(uploadcareUuid)}`,
      data: access.revision ? { revision: access.revision } : undefined,
      signal,
    });
    if (!response.ok)
      throw await apiError(response, "MISSION_ATTACHMENT_DOWNLOAD_FAILED");
    return readBoundedBlob(response, 20 * 1024 * 1024);
  },
  async startStudentMission(
    accessToken: string,
    data: StartStudentMissionRequest,
    signal?: AbortSignal,
  ): Promise<StudentSessionResponse> {
    const res = await httpClient.request({
      accessToken,
      data,
      method: HttpMethod.POST,
      path: "/users/student-mission",
      signal,
    });

    if (!res.ok) {
      throw await apiError(res, "STUDENT_MISSION_START_FAILED");
    }

    return (await res.json()) as StudentSessionResponse;
  },
  async preview(
    accessToken: string,
    class_id: string,
    data: CreateTeacherClassMissionRequest,
  ): Promise<TeacherClassMissionResponse> {
    const res = await httpClient.request({
      accessToken,
      data,
      method: HttpMethod.POST,
      path: `/teacher-classes/${encodeURIComponent(class_id)}/missions/preview`,
    });

    if (!res.ok) {
      throw await apiError(res, "TEACHER_CLASS_ACTIVITY_PREVIEW_FAILED");
    }

    return (await res.json()) as TeacherClassMissionResponse;
  },
  async previewStream(
    accessToken: string,
    class_id: string,
    data: CreateTeacherClassMissionRequest,
    options?: {
      signal?: AbortSignal;
      onProgress?: (progress: MissionPreviewProgressDto) => void;
    },
  ): Promise<TeacherClassMissionResponse> {
    const headers = new Headers();
    headers.set("Accept", "text/event-stream");
    const res = await httpClient.request({
      accessToken,
      data,
      headers,
      method: HttpMethod.POST,
      path: `/teacher-classes/${encodeURIComponent(class_id)}/missions/preview/stream`,
      signal: options?.signal,
    });

    if (!res.ok) {
      throw await apiError(res, "TEACHER_CLASS_ACTIVITY_PREVIEW_FAILED");
    }

    return readMissionPreviewEventStream(res, options?.onProgress);
  },
  async approve(
    accessToken: string,
    class_id: string,
    mission_id: string,
    data?: ApproveTeacherClassMissionRequest,
  ): Promise<TeacherClassMissionResponse> {
    const res = await httpClient.request({
      accessToken,
      data,
      method: HttpMethod.POST,
      path: `/teacher-classes/${encodeURIComponent(class_id)}/missions/${encodeURIComponent(mission_id)}/approve`,
    });

    if (!res.ok) {
      throw await apiError(res, "TEACHER_CLASS_ACTIVITY_APPROVE_FAILED");
    }

    return (await res.json()) as TeacherClassMissionResponse;
  },
  async remove(
    accessToken: string,
    class_id: string,
    mission_id: string,
  ): Promise<void> {
    const res = await httpClient.request({
      accessToken,
      method: HttpMethod.DELETE,
      path: `/teacher-classes/${encodeURIComponent(class_id)}/missions/${encodeURIComponent(mission_id)}`,
    });

    if (!res.ok) {
      throw await apiError(res, "TEACHER_CLASS_MISSION_DELETE_FAILED");
    }
  },
  async list(
    accessToken: string,
    class_id: string,
  ): Promise<TeacherClassMissionsResponse> {
    const items = await collectPages(async (cursor) => {
      const response = await httpClient.request({
        accessToken,
        method: HttpMethod.GET,
        path: `/teacher-classes/${encodeURIComponent(class_id)}/missions`,
        data: { limit: 100, cursor },
      });
      if (!response.ok) throw await apiError(response, "LIST_FAILED");
      const page = (await response.json()) as TeacherClassMissionsResponse;
      return { items: page.missions, nextCursor: page.next_cursor };
    });
    return { missions: items };
  },
};
