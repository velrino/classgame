import { httpClient } from "@/shared/http/client";
import { apiError } from "@/shared/http/errors";
import { HttpMethod } from "@/shared/http/types";

import type {
  DashboardOverviewRequest,
  DashboardOverviewResponse,
  StudentHistoryResponse,
} from "@/modules/insights/adapters/types/contracts";
export const insightsApi = {
  async studentHistory(accessToken: string, classId: string, studentId: string, cursor = ""): Promise<StudentHistoryResponse> {
    const res = await httpClient.request({accessToken, data: {class_id: classId, limit: 5, cursor}, method: HttpMethod.GET, path: `/dashboard/students/${encodeURIComponent(studentId)}/attempts`});
    if (!res.ok) throw await apiError(res, "STUDENT_HISTORY_FAILED");
    return await res.json() as StudentHistoryResponse;
  },
  async overview(
    accessToken: string,
    data?: DashboardOverviewRequest,
  ): Promise<DashboardOverviewResponse> {
    const res = await httpClient.request({
      accessToken,
      data,
      method: HttpMethod.GET,
      path: "/dashboard/overview/page",
    });

    if (!res.ok) {
      const error = await apiError(res, "DASHBOARD_OVERVIEW_FAILED");
      if (error.code !== "query_index_required") throw error;
      // A complete capped response keeps small classes usable during index rollout.
      const legacy = await httpClient.request({accessToken, data: {class_id: data?.class_id}, method: HttpMethod.GET, path: "/dashboard/overview"});
      if (!legacy.ok) {
        const legacyError = await apiError(legacy, "DASHBOARD_OVERVIEW_FAILED");
        throw legacyError.code === "pagination_required" ? error : legacyError;
      }
      return await legacy.json() as DashboardOverviewResponse;
    }

    return (await res.json()) as DashboardOverviewResponse;
  },
};
