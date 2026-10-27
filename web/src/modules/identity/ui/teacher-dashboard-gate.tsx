"use client";

import { usePathname, useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";

import { PageLoadingFallback } from "@/components/page-loading-fallback";

import { useTeacherAuth } from "@/modules/identity/hooks/use-teacher-auth";

export function TeacherDashboardGate({
  children,
}: {
  readonly children: ReactNode;
}) {
  const auth = useTeacherAuth();
  const pathname = usePathname();
  const router = useRouter();

  useEffect(() => {
    if (auth.status === "signed_out") {
      const nextPath = pathname.startsWith("/dashboard")
        ? pathname
        : "/dashboard";
      router.replace(`/login?next=${encodeURIComponent(nextPath)}`);
    }
  }, [auth.status, pathname, router]);

  if (auth.status === "signed_in" && auth.user) {
    return children;
  }

  return (
    <PageLoadingFallback
      className="dashboard-theme"
      message="Checking session..."
    />
  );
}
