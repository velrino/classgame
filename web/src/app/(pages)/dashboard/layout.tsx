import type { ReactNode } from "react";

import { TeacherDashboardGate } from "@/modules/identity";

export default function DashboardLayout({
  children,
}: {
  readonly children: ReactNode;
}) {
  return <TeacherDashboardGate>{children}</TeacherDashboardGate>;
}
