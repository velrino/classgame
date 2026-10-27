import { httpClient } from "@/shared/http/client";
import { apiError } from "@/shared/http/errors";
import { HttpMethod } from "@/shared/http/types";

import type {
  StudentAttemptScope,
  StudentLearningResponse,
  StudentLearningSnapshotRequest,
  StudentTutorTurnRequest,
  StudentTutorTurnResponse,
} from "@/modules/learning/adapters/types/contracts";
export const learningApi = {
  async studentLearning(
    accessToken: string,
    mission_id: string,
    scope?: Omit<StudentAttemptScope, "mission_id">,
    options?: { signal?: AbortSignal },
  ): Promise<StudentLearningResponse> {
    const res = await httpClient.request({
      accessToken,
      signal: options?.signal,
      data: { ...scope, mission_id },
      method: HttpMethod.GET,
      path: "/users/student-learning",
    });

    if (!res.ok) {
      throw await apiError(res, "STUDENT_LEARNING_GET_FAILED");
    }

    return (await res.json()) as StudentLearningResponse;
  },
  async saveStudentLearning(
    accessToken: string,
    data: StudentLearningSnapshotRequest,
    options?: { keepalive?: boolean; signal?: AbortSignal },
  ): Promise<StudentLearningResponse> {
    const res = await httpClient.request({
      accessToken,
      data,
      keepalive: options?.keepalive,
      signal: options?.signal,
      method: HttpMethod.POST,
      path: "/users/student-learning",
    });

    if (!res.ok) {
      throw await apiError(res, "STUDENT_LEARNING_SAVE_FAILED");
    }

    return (await res.json()) as StudentLearningResponse;
  },
  async studentTutorTurn(
    accessToken: string,
    data: StudentTutorTurnRequest,
    options?: { signal?: AbortSignal },
  ): Promise<StudentTutorTurnResponse> {
    const res = await httpClient.request({
      accessToken,
      data,
      method: HttpMethod.POST,
      path: "/users/student-tutor-turn",
      signal: options?.signal,
    });

    if (!res.ok) {
      throw await apiError(res, "STUDENT_TUTOR_TURN_FAILED");
    }

    return (await res.json()) as StudentTutorTurnResponse;
  },
};
