import { httpClient } from "@/shared/http/client";
import { apiError } from "@/shared/http/errors";
import { HttpMethod } from "@/shared/http/types";

export const identityApi = {
  async recordTeacherLogin(accessToken: string): Promise<void> {
    const res = await httpClient.request({ method: HttpMethod.POST, path: "/dashboard/teacher-login", accessToken });
    if (!res.ok) throw await apiError(res, "TEACHER_LOGIN_RECORD_FAILED");
  },
  async get(signal?: AbortSignal): Promise<void> {
    const res = await httpClient.request({
      method: HttpMethod.GET,
      path: "/",
      signal,
    });

    if (!res.ok) {
      throw await apiError(res, "API_HEALTH_CHECK_FAILED");
    }
  },
};
