"use client";
import { useTeacherAuthState } from "@/modules/identity/adapters/hooks/use-teacher-auth-state";
import { createContext, createElement, type ReactNode } from "react";
type TeacherAuthContextValue = ReturnType<typeof useTeacherAuthState>;

export const TeacherAuthContext = createContext<TeacherAuthContextValue | null>(
  null,
);

export function TeacherAuthProvider({
  children,
}: {
  readonly children: ReactNode;
}) {
  const value = useTeacherAuthState();

  return createElement(TeacherAuthContext.Provider, { value }, children);
}
