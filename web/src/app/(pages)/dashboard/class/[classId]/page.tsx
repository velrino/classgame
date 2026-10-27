import { DashboardPage } from "@/modules/insights";

type DashboardClassRouteProps = {
  readonly params: Promise<{ classId: string }>;
};

export default async function DashboardClassRoute({
  params,
}: DashboardClassRouteProps) {
  const { classId } = await params;

  return <DashboardPage initialClassId={classId} />;
}
