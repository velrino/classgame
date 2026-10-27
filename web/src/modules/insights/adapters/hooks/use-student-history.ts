"use client";
import { useTeacherAuth } from "@/modules/identity";
import { useCallback, useEffect, useRef, useState } from "react";
import { insightsApi } from "../http";
import type { DashboardStudentDto, StudentHistoryResponse } from "../types/contracts";

export function useStudentHistory(student: DashboardStudentDto | null, open: boolean) {
  const auth = useTeacherAuth();
  const generation = useRef(0);
  const busy = useRef(false);
  const [page, setPage] = useState<StudentHistoryResponse>({attempts: [], next_cursor: "", read_at: ""});
  const [trail, setTrail] = useState<string[]>([]);
  const [cursor, setCursor] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const classId = student?.class_id, studentId = student?.id;
  const user = auth.user;
  const load = useCallback(async (next: string, previous: string[]) => {
    if (!user || !classId || !studentId || busy.current) return;
    const ticket = generation.current;
    busy.current = true;
    setLoading(true); setError(null);
    try {
      const token = await user.getIdToken();
      if (ticket !== generation.current) return;
      const response = await insightsApi.studentHistory(token, classId, studentId, next);
      if (ticket !== generation.current) return;
      setPage(response); setCursor(next); setTrail(previous);
    } catch (caught) {
      if (ticket === generation.current) setError(caught instanceof Error ? caught.message : "Could not load mission history.");
    } finally {
      if (ticket === generation.current) { busy.current = false; setLoading(false); }
    }
  }, [user, classId, studentId]);
  useEffect(() => {
    const ticket = ++generation.current; busy.current = false;
    if (open) void Promise.resolve().then(() => { if (ticket === generation.current) void load("", []); });
    return () => { generation.current += 1; busy.current = false; };
  }, [load, open]);
  return {page, loading, error, canGoBack: trail.length > 0,
    previous: () => { if (trail.length) void load(trail.at(-1)!, trail.slice(0,-1)); },
    next: () => { if (page.next_cursor) void load(page.next_cursor, [...trail, cursor]); },
    refresh: () => { void load(cursor, trail); },
  };
}
