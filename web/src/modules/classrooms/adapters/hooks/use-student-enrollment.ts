"use client";
import { miaAnalytics } from "@/lib/mia-analytics";
import {
  studentClassUnavailableEvent,
  studentSessionReadyEvent,
} from "@/modules/classrooms/adapters/hooks/use-student-presence";
import { classroomsApi } from "@/modules/classrooms/adapters/http";
import {
  clearStudentSession,
  readStudentClassSession,
  saveStudentClassSession,
  studentClassSessionStorageKey,
} from "@/modules/classrooms/adapters/session-storage";
import {
  describeStudentEntryError,
  studentEntryCopy,
  studentSessionErrorScope,
} from "@/modules/classrooms/domain/student-entry-messages";
import { type StudentSessionFormValue } from "@/modules/classrooms/ui/student-session-modal";
import {
  currentStudentIdentity,
  ensureAnonymousStudent,
  isIdentityConfigured,
  observeStudentIdentity,
  signOutStudentIdentity,
  type StudentIdentity,
} from "@/modules/identity";
import { useEffect, useRef, useState } from "react";
type GateStatus = "checking" | "needs_student_session" | "ready";
export function useStudentEnrollment() {
  const firebaseConfigured = isIdentityConfigured();
  const registeringRef = useRef(false);
  const [status, setStatus] = useState<GateStatus>(() =>
    firebaseConfigured ? "checking" : "needs_student_session",
  );
  const [error, setError] = useState<string | null>(() =>
    firebaseConfigured
      ? null
      : "Firebase is not configured for this environment.",
  );
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!firebaseConfigured) {
      return;
    }

    let cancelled = false;
    const showClassUnavailable = () => {
      setError(studentEntryCopy.errors.classUnavailable);
    };
    window.addEventListener(studentClassUnavailableEvent, showClassUnavailable);

    async function restoreStoredSession(user: StudentIdentity | null) {
      if (registeringRef.current) {
        return;
      }

      const classSession = readStudentClassSession();
      if (user?.isAnonymous && classSession) {
        setStatus("checking");
        try {
          await classroomsApi.validateStudentSession({
            class_code: classSession.classCode,
          });
        } catch (caughtError) {
          if (cancelled) {
            return;
          }
          if (studentSessionErrorScope(caughtError) === "class") {
            clearStudentSession();
            setStatus("needs_student_session");
            setError(studentEntryCopy.errors.classUnavailable);
            await signOutStudentIdentity().catch(() => undefined);
            return;
          }
          // Keep a previously opened mission playable during a network outage.
        }

        if (cancelled) {
          return;
        }
        setStatus("ready");
        setError(null);
        window.dispatchEvent(new Event(studentSessionReadyEvent));
        return;
      }

      clearStudentSession();
      setStatus("needs_student_session");
    }

    const unsubscribe = observeStudentIdentity((user) => {
      void restoreStoredSession(user);
    });
    const restoreSessionFromAnotherTab = (event: StorageEvent) => {
      if (event.key !== null && event.key !== studentClassSessionStorageKey) {
        return;
      }
      void restoreStoredSession(currentStudentIdentity());
    };
    window.addEventListener("storage", restoreSessionFromAnotherTab);
    return () => {
      cancelled = true;
      unsubscribe();
      window.removeEventListener("storage", restoreSessionFromAnotherTab);
      window.removeEventListener(
        studentClassUnavailableEvent,
        showClassUnavailable,
      );
    };
  }, [firebaseConfigured]);

  async function handleSubmit(value: StudentSessionFormValue) {
    setError(null);
    setLoading(true);
    registeringRef.current = true;
    let newAnonymousUser: StudentIdentity | null = null;
    let joinStage: Parameters<typeof miaAnalytics.gameJoinFailed>[1]["stage"] =
      "class_validation";

    try {
      await classroomsApi.validateStudentSession({
        class_code: value.classCode,
      });

      joinStage = "authentication";

      const { isNew, user } = await ensureAnonymousStudent();
      if (isNew) {
        newAnonymousUser = user;
      }
      const idToken = await user.getIdToken(true);

      joinStage = "session_creation";
      const response = await classroomsApi.studentSession(idToken, {
        student_name: value.studentName,
        class_code: value.classCode,
      });

      if (!response.class) {
        joinStage = "response_validation";
        throw new Error("The class is unavailable.");
      }

      saveStudentClassSession(response.class, value.studentName);
      miaAnalytics.gameJoined({ class_id: response.class.id });

      setStatus("ready");
      window.dispatchEvent(new Event(studentSessionReadyEvent));
    } catch (caughtError) {
      miaAnalytics.gameJoinFailed(caughtError, { stage: joinStage });
      if (newAnonymousUser) {
        await newAnonymousUser.delete().catch(() => undefined);
      }

      setStatus("needs_student_session");
      setError(describeStudentEntryError(caughtError));
    } finally {
      registeringRef.current = false;
      setLoading(false);
    }
  }

  const ready = status === "ready";

  return { status, error, loading, handleSubmit, ready };
}
