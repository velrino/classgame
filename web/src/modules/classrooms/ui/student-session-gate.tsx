"use client";
import { useStudentEnrollment } from "@/modules/classrooms/adapters/hooks/use-student-enrollment";

import { type ReactNode } from "react";

import { PageLoadingFallback } from "@/components/page-loading-fallback";

import { StudentSessionModal } from "@/modules/classrooms/ui/student-session-modal";

type StudentSessionGateProps = {
  readonly children: ReactNode;
  readonly initialClassCode?: string;
};

export function StudentSessionGate({
  children,
  initialClassCode = "",
}: StudentSessionGateProps) {
  const { status, error, loading, handleSubmit, ready } =
    useStudentEnrollment();
  return (
    <>
      {/*
        The classroom renders behind the modal so the student sees the game
        while entering the code. `inert` blocks focus, taps and the joystick
        until the session exists; the game's keyboard controls already ignore
        keys typed inside inputs.
      */}
      <div
        aria-hidden={!ready}
        inert={!ready}
        className={ready ? undefined : "pointer-events-none select-none"}
      >
        {children}
      </div>
      {status === "checking" ? (
        <PageLoadingFallback
          className="dashboard-theme"
          message="Checking session..."
        />
      ) : status === "needs_student_session" ? (
        <StudentSessionModal
          error={error}
          initialClassCode={initialClassCode}
          loading={loading}
          onSubmit={handleSubmit}
        />
      ) : null}
    </>
  );
}
