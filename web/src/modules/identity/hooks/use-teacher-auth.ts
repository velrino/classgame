"use client";
import { TeacherAuthContext } from "@/modules/identity/adapters/context/teacher-auth-context";
import { useContext } from "react";

export function useTeacherAuth() {
  const value = useContext(TeacherAuthContext);
  if (!value) {
    throw new Error("useTeacherAuth must be used inside TeacherAuthProvider.");
  }

  return value;
}
