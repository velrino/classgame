import { httpClient } from "@/shared/http/client";
import { apiError } from "@/shared/http/errors";
import { HttpMethod } from "@/shared/http/types";

import type {
  DashboardOverviewRequest,
  DashboardOverviewResponse,
} from "@/modules/insights/adapters/types/contracts";
export const insightsApi = {
  async overview(
    accessToken: string,
    data?: DashboardOverviewRequest,
  ): Promise<DashboardOverviewResponse> {
    const res = await httpClient.request({
      accessToken,
      data,
      method: HttpMethod.GET,
      path: "/dashboard/overview",
    });

    if (!res.ok) {
      throw await apiError(res, "DASHBOARD_OVERVIEW_FAILED");
    }

    return (await res.json()) as DashboardOverviewResponse;
  },
};
