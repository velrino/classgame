"use client";
import { useTeacherAuth } from "@/modules/identity";
import { useCallback, useEffect, useRef, useState } from "react";
import { observeDashboardChanges } from "../realtime";
import { insightsApi } from "../http";
import type { DashboardStudentDto, StudentHistoryResponse } from "../types/contracts";

export type HistoryStudent = Pick<DashboardStudentDto, "id" | "class_id"> & Partial<Pick<DashboardStudentDto, "student_name" | "attempts">>;

export function useStudentHistory(student: HistoryStudent | null, open: boolean, completeHistory = false) {
  const auth = useTeacherAuth();
  const generation = useRef(0);
  const busy = useRef(false);
  const request = useRef(0);
  const [page, setPage] = useState<StudentHistoryResponse>({attempts: [], next_cursor: "", read_at: ""});
  const [trail, setTrail] = useState<string[]>([]);
  const [cursor, setCursor] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const classId = student?.class_id, studentId = student?.id;
  const user = auth.user;
  const load = useCallback(async (next: string, previous: string[], background = false) => {
    if (!user || !classId || !studentId || (background && busy.current)) return;
    const ticket = ++request.current;
    busy.current = true;
    if (!background) setLoading(true);
    try {
      const token = await user.getIdToken();
      if (ticket !== request.current) return;
      const response = await insightsApi.studentHistory(token, classId, studentId, next);
      if (ticket !== request.current) return;
      setError(null); setPage(response); setCursor(next); setTrail(previous);
    } catch (caught) {
      if (ticket === request.current) setError(caught instanceof Error ? caught.message : "Could not load mission history.");
    } finally {
      if (ticket === request.current) { busy.current = false; setLoading(false); }
    }
  }, [user, classId, studentId]);
  useEffect(() => {
    const ticket = ++generation.current; busy.current = false;
    if (open && !completeHistory) void Promise.resolve().then(() => { if (ticket === generation.current) void load("", []); });
    return () => { generation.current += 1; request.current += 1; busy.current = false; };
  }, [load, open, completeHistory]);
  useEffect(() => {
    if (!open || completeHistory || cursor || !classId || !user) return;
    const refreshLatest = () => {
      if (document.visibilityState === "visible") void load("", [], true);
    };
    const unsubscribe = observeDashboardChanges(classId, refreshLatest);
    // Individual answers can arrive before the class summary changes.
    const timer = window.setInterval(refreshLatest, 5000);
    document.addEventListener("visibilitychange", refreshLatest);
    return () => {
      unsubscribe();
      window.clearInterval(timer);
      document.removeEventListener("visibilitychange", refreshLatest);
    };
  }, [open, completeHistory, cursor, classId, user, load]);
  return {page: completeHistory ? {attempts: student?.attempts ?? [], next_cursor: "", read_at: ""} : page, loading: completeHistory ? false : loading, error: completeHistory ? null : error, canGoBack: !completeHistory && trail.length > 0,
    previous: () => { if (trail.length) void load(trail.at(-1)!, trail.slice(0,-1)); },
    next: () => { if (page.next_cursor) void load(page.next_cursor, [...trail, cursor]); },
    refresh: () => { void load(cursor, trail); },
  };
}
