"use client";
import { classroomsApi } from "@/modules/classrooms/adapters/http";
import {
  observeStudentCompletions,
  observeStudentMissions,
} from "@/modules/classrooms/adapters/student-catalog";
import type { StudentClassMission } from "@/modules/classrooms/adapters/types/student-catalog";
import {
  currentStudentIdentity,
  observeStudentIdentity,
} from "@/modules/identity";
import { ApiError } from "@/shared/http/errors";

import { useEffect, useRef, useState } from "react";

import {
  readStudentClassSession,
  studentClassSessionChangedEvent,
  type StudentClassSession,
} from "@/modules/classrooms/adapters/session-storage";

type StudentClassMissionsOptions = {
  readonly activeMissionID?: string;
  readonly onClassUnavailable?: () => void;
  readonly onMissionUnavailable?: () => void;
};

const INITIAL_MISSION_RETRY_MS = 1_000;
const MAX_MISSION_RETRY_MS = 15_000;

export function useStudentClassMissions(
  options: StudentClassMissionsOptions = {},
) {
  const [classSession, setClassSession] = useState<StudentClassSession | null>(
    null,
  );
  const [studentID, setStudentID] = useState("");
  const [missions, setMissions] = useState<StudentClassMission[]>([]);
  const [completedMissionKeys, setCompletedMissionKeys] = useState<Set<string>>(
    () => new Set(),
  );
  const [missionsReady, setMissionsReady] = useState(false);
  const [missionScope, setMissionScope] = useState("");
  const [completionScope, setCompletionScope] = useState("");
  const activeMissionIDRef = useRef(options.activeMissionID ?? "");
  const onClassUnavailableRef = useRef(options.onClassUnavailable);
  const onMissionUnavailableRef = useRef(options.onMissionUnavailable);

  useEffect(() => {
    activeMissionIDRef.current = options.activeMissionID ?? "";
    onClassUnavailableRef.current = options.onClassUnavailable;
    onMissionUnavailableRef.current = options.onMissionUnavailable;
  }, [
    options.activeMissionID,
    options.onClassUnavailable,
    options.onMissionUnavailable,
  ]);

  useEffect(() => {
    const refreshClassSession = () => {
      setClassSession(readStudentClassSession());
    };
    refreshClassSession();
    window.addEventListener(
      studentClassSessionChangedEvent,
      refreshClassSession,
    );
    window.addEventListener("storage", refreshClassSession);
    return () => {
      window.removeEventListener(
        studentClassSessionChangedEvent,
        refreshClassSession,
      );
      window.removeEventListener("storage", refreshClassSession);
    };
  }, []);

  useEffect(
    () =>
      observeStudentIdentity((user) => {
        setStudentID(user?.isAnonymous ? user.uid : "");
      }),
    [],
  );

  useEffect(() => {
    if (!classSession || !studentID) {
      return;
    }

    const scope = `${classSession.classId}:${studentID}`;
    let checkingClass = false;
    let disposed = false;
    const refreshClass = async () => {
      if (checkingClass || disposed || document.visibilityState !== "visible")
        return;
      const user = currentStudentIdentity();
      if (!user?.isAnonymous || user.uid !== studentID) return;
      checkingClass = true;
      try {
        const response = await classroomsApi.currentStudentClass(
          await user.getIdToken(),
        );
        if (
          !disposed &&
          readStudentClassSession()?.classId === classSession.classId &&
          (response.class.id !== classSession.classId ||
            response.class.status !== "active")
        )
          onClassUnavailableRef.current?.();
      } catch (error) {
        if (
          !disposed &&
          error instanceof ApiError &&
          ["class_not_found", "student_not_found"].includes(error.code) &&
          readStudentClassSession()?.classId === classSession.classId
        )
          onClassUnavailableRef.current?.();
      } finally {
        checkingClass = false;
      }
    };
    void refreshClass();
    const classTimer = window.setInterval(() => void refreshClass(), 30_000);
    const onVisible = () => void refreshClass();
    document.addEventListener("visibilitychange", onVisible);
    let missionRetryDelay = INITIAL_MISSION_RETRY_MS;
    let missionRetryTimer: number | undefined;
    let missionListenerStopped = false;
    let unsubscribeMissions: () => void = () => undefined;
    const subscribeToMissions = () => {
      unsubscribeMissions();
      unsubscribeMissions = observeStudentMissions(
        classSession.classId,
        (next) => {
          missionRetryDelay = INITIAL_MISSION_RETRY_MS;
          setMissionScope(scope);
          setMissions(next);
          setMissionsReady(true);
          const activeMissionID = activeMissionIDRef.current;
          if (
            activeMissionID &&
            readStudentClassSession()?.classId === classSession.classId &&
            !next.some((mission) => mission.id === activeMissionID)
          ) {
            onMissionUnavailableRef.current?.();
          }
        },
        (error) => {
          if (missionListenerStopped) {
            return;
          }

          setMissionsReady(false);
          console.warn(
            "Student mission updates disconnected; retrying.",
            error,
          );
          missionRetryTimer = window.setTimeout(
            subscribeToMissions,
            missionRetryDelay,
          );
          missionRetryDelay = Math.min(
            missionRetryDelay * 2,
            MAX_MISSION_RETRY_MS,
          );
        },
      );
    };
    subscribeToMissions();
    const unsubscribeCompletions = observeStudentCompletions(
      classSession.classId,
      studentID,
      (completed) => {
        setCompletionScope(scope);
        setCompletedMissionKeys(completed);
      },
    );

    return () => {
      missionListenerStopped = true;
      window.clearTimeout(missionRetryTimer);
      disposed = true;
      window.clearInterval(classTimer);
      document.removeEventListener("visibilitychange", onVisible);
      unsubscribeMissions();
      unsubscribeCompletions();
    };
  }, [classSession, studentID]);

  const currentScope =
    classSession && studentID ? `${classSession.classId}:${studentID}` : "";

  return {
    classSession,
    completedMissionKeys:
      currentScope && completionScope === currentScope
        ? completedMissionKeys
        : new Set<string>(),
    missions: currentScope && missionScope === currentScope ? missions : [],
    missionsReady:
      Boolean(currentScope) && missionScope === currentScope && missionsReady,
  };
}
