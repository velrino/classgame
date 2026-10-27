"use client";
import type { TeacherClassDto } from "@/modules/classrooms";
import {
  classroomsApi,
  rememberSelectedClass,
  resolveSelectedClassId,
  upsertTeacherClass,
} from "@/modules/classrooms";
import { useTeacherAuth } from "@/modules/identity";
import { insightsApi } from "@/modules/insights/adapters/http";
import { observeDashboardChanges } from "@/modules/insights/adapters/realtime";
import type {
  DashboardOverviewResponse,
  DashboardStudentDto,
} from "@/modules/insights/adapters/types/contracts";
import type { TeacherClassMissionDto } from "@/modules/missions";
import { missionsApi } from "@/modules/missions";
import { useEffect, useMemo, useRef, useState } from "react";
export function useDashboardSnapshots(
  auth: Pick<ReturnType<typeof useTeacherAuth>, "status" | "user">,
  initialClassId?: string,
) {
  const [overview, setOverview] = useState<DashboardOverviewResponse | null>(
    null,
  );

  const [teacherClasses, setTeacherClasses] = useState<TeacherClassDto[]>([]);

  const [selectedClassId, setSelectedClassId] = useState<string | null>(null);

  const [rosterPage, setRosterPage] = useState<{classId: string; cursor: string; trail: string[]}>({classId: "", cursor: "", trail: []});
  const rosterCursor = rosterPage.classId === selectedClassId ? rosterPage.cursor : "";
  const queryRef = useRef({actorId: auth.user?.uid, classId: selectedClassId as string | null, cursor: rosterCursor});
  useEffect(() => { queryRef.current = {actorId: auth.user?.uid, classId: selectedClassId, cursor: rosterCursor}; }, [auth.user?.uid, selectedClassId, rosterCursor]);

  const [classesLoading, setClassesLoading] = useState(true);

  const [overviewLoading, setOverviewLoading] = useState(false);

  const [missions, setMissions] = useState<TeacherClassMissionDto[]>([]);

  const [missionsLoading, setMissionsLoading] = useState(false);

  const [error, setError] = useState<string | null>(null);

  const [selectedStudent, setSelectedStudent] =
    useState<DashboardStudentDto | null>(null);

  useEffect(() => {
    if (auth.status !== "signed_in" || !auth.user) {
      return;
    }

    let cancelled = false;
    const user = auth.user;

    Promise.resolve()
      .then(async () => {
        setClassesLoading(true);
        setError(null);

        const token = await user.getIdToken();
        return classroomsApi.list(token);
      })
      .then((response) => {
        if (cancelled) {
          return;
        }

        setTeacherClasses(response.teacher_classes);
        setSelectedClassId((current) =>
          resolveSelectedClassId(
            initialClassId ?? current,
            response.teacher_classes,
          ),
        );
      })
      .catch((caughtError: unknown) => {
        if (!cancelled) {
          setError(
            caughtError instanceof Error
              ? caughtError.message
              : "Could not load classes.",
          );
        }
      })
      .finally(() => {
        if (!cancelled) {
          setClassesLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [auth.status, auth.user, initialClassId]);

  useEffect(() => {
    if (auth.status !== "signed_in" || !auth.user) {
      return;
    }

    if (!selectedClassId) {
      return;
    }

    let cancelled = false;
    const user = auth.user;

    Promise.resolve()
      .then(async () => {
        setOverviewLoading(true);
        setOverview(null);
        setSelectedStudent(null);
        setError(null);

        const token = await user.getIdToken();
        return insightsApi.overview(token, {
          class_id: selectedClassId,
          limit: 25, cursor: rosterCursor,
        });
      })
      .then((response) => {
        if (cancelled) {
          return;
        }

        setOverview(response);
        const teacherClass = response.teacher_class;
        if (teacherClass) {
          setTeacherClasses((current) =>
            upsertTeacherClass(current, teacherClass),
          );
        }
        rememberSelectedClass(response.class_id);
      })
      .catch((caughtError: unknown) => {
        if (!cancelled) {
          setError(
            caughtError instanceof Error
              ? caughtError.message
              : "Could not load dashboard.",
          );
        }
      })
      .finally(() => {
        if (!cancelled) {
          setOverviewLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [auth.status, auth.user, selectedClassId, rosterCursor]);

  useEffect(() => {
    if (auth.status !== "signed_in" || !auth.user || !selectedClassId) {
      return;
    }

    let cancelled = false;
    const user = auth.user;
    const classId = selectedClassId;

    Promise.resolve()
      .then(async () => {
        setMissionsLoading(true);
        const token = await user.getIdToken();
        return missionsApi.list(token, classId);
      })
      .then((response) => {
        if (!cancelled) {
          setMissions(response.missions);
        }
      })
      .catch((caughtError: unknown) => {
        if (!cancelled) {
          setError(
            caughtError instanceof Error
              ? caughtError.message
              : "Could not load class missions.",
          );
        }
      })
      .finally(() => {
        if (!cancelled) {
          setMissionsLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [auth.status, auth.user, selectedClassId]);

  useEffect(() => {
    if (auth.status !== "signed_in" || !auth.user || !selectedClassId) {
      return;
    }

    let cancelled = false;
    let refreshing = false;
    const user = auth.user;
    const classId = selectedClassId;

    async function refreshOverviewInBackground() {
      if (cancelled || refreshing || document.visibilityState !== "visible") {
        return;
      }

      refreshing = true;
      try {
        const token = await user.getIdToken();
        const response = await insightsApi.overview(token, {
          class_id: classId,
          limit: 25, cursor: rosterCursor,
        });
        if (cancelled) {
          return;
        }

        setOverview(response);
        setSelectedStudent((current) =>
          current
            ? (response.students.find((student) => student.id === current.id) ??
              null)
            : null,
        );
        const teacherClass = response.teacher_class;
        if (teacherClass) {
          setTeacherClasses((current) =>
            upsertTeacherClass(current, teacherClass),
          );
        }
      } catch {
        // Keep the last successful dashboard snapshot during transient refresh failures.
      } finally {
        refreshing = false;
      }
    }

    const unsubscribeRealtime = observeDashboardChanges(
      classId,
      () => void refreshOverviewInBackground(),
    );
    const interval = window.setInterval(refreshOverviewInBackground, 15_000);
    const refreshWhenVisible = () => {
      if (document.visibilityState === "visible") {
        void refreshOverviewInBackground();
      }
    };
    document.addEventListener("visibilitychange", refreshWhenVisible);

    return () => {
      cancelled = true;
      unsubscribeRealtime();
      window.clearInterval(interval);
      document.removeEventListener("visibilitychange", refreshWhenVisible);
    };
  }, [auth.status, auth.user, selectedClassId, rosterCursor]);

  const selectedClass = useMemo(() => {
    if (
      overview?.teacher_class &&
      overview.teacher_class.class_id === selectedClassId
    ) {
      return overview.teacher_class;
    }

    return (
      teacherClasses.find(
        (teacherClass) => teacherClass.class_id === selectedClassId,
      ) ?? null
    );
  }, [overview?.teacher_class, selectedClassId, teacherClasses]);

  async function loadOverview(token: string, classId: string) {
    const captured = queryRef.current;
    const response = await insightsApi.overview(token, {class_id: classId, limit: 25, cursor: captured.classId === classId ? captured.cursor : ""});
    const current = queryRef.current;
    if (current.actorId !== captured.actorId || current.classId !== captured.classId || current.cursor !== captured.cursor || current.classId !== classId) return;
    setOverview(response);
    if (response.teacher_class) setTeacherClasses((rows) => upsertTeacherClass(rows, response.teacher_class!));
    rememberSelectedClass(response.class_id);
  }

  return {
    overview,
    setOverview,
    teacherClasses,
    setTeacherClasses,
    selectedClassId,
    setSelectedClassId,
    classesLoading,
    setClassesLoading,
    overviewLoading,
    setOverviewLoading,
    missions,
    setMissions,
    missionsLoading,
    setMissionsLoading,
    error,
    setError,
    selectedStudent,
    setSelectedStudent,
    selectedClass,
    loadOverview,
    canGoBack: rosterPage.classId === selectedClassId && rosterPage.trail.length > 0,
    nextStudentPage: () => {
      const next = overview?.page?.next_cursor;
      if (!next || !selectedClassId) return;
      setRosterPage({classId: selectedClassId, cursor: next, trail: [...(rosterPage.classId === selectedClassId ? rosterPage.trail : []), rosterCursor]});
    },
    previousStudentPage: () => {
      if (!selectedClassId || rosterPage.classId !== selectedClassId || !rosterPage.trail.length) return;
      setRosterPage({classId: selectedClassId, cursor: rosterPage.trail.at(-1)!, trail: rosterPage.trail.slice(0, -1)});
    },
  };
}
