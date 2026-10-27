import { DashboardTab } from "@/modules/insights/adapters/types/dashboard";

export const defaultDashboardTab: DashboardTab = "overview";

export const hashByTab: Record<DashboardTab, string> = {
  mission: "#mission-builder",
  students: "#students",
  notebook: "#notebook",
  overview: "#overview",
};

export const tabByHash: Record<string, DashboardTab> = {
  "#mission-builder": "mission",
  "#students": "students",
  "#notebook": "notebook",
  "#overview": "overview",
};

export function toDashboardTab(value: string): DashboardTab {
  if (
    value === "mission" ||
    value === "students" ||
    value === "notebook" ||
    value === "overview"
  ) {
    return value;
  }

  return defaultDashboardTab;
}
