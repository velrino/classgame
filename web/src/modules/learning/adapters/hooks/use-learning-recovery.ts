"use client";
import { ApiError } from "@/shared/http/errors";
import { currentStudentIdentity } from "@/modules/identity";

import { learningApi } from "@/modules/learning/adapters/http";
import { useCallback, useEffect, useRef, useState } from "react";

import type {
  StudentAttemptScope,
  StudentLearningStatus,
} from "@/modules/learning/adapters/types/contracts";

import {
  clearStudentMissionSession,
  readStudentMissionSession,
  studentSessionReadyEvent,
  type StudentMissionSession,
} from "@/modules/classrooms";
import {
  clearMemory,
  loadStudentRecovery,
  saveMemory,
} from "@/modules/learning/adapters/recovery-storage";
import { createInitialMemory } from "@/modules/learning/domain/memory";
import { buildGameMission } from "@/modules/learning/domain/published-mission";
import { type Session } from "@/modules/learning/domain/session-reducer";

import {
  checkpointFromLearningAttempt,
  checkpointFromSession,
  learningStatus,
  memoryFromLearningAttempt,
  nextRevision,
  toLearningSnapshot,
} from "@/modules/learning/adapters/checkpoint-mapper";
import {
  migrateLegacyEvidence,
  readTutorOutbox,
  type EvidenceScope,
  type EvidenceRecord,
  type TutorEvidence,
} from "../tutor-outbox";
import { currentEvidenceScope, useEvidenceSync } from "./use-evidence-sync";
import { sameEvidenceScope } from "../../domain/evidence-queue";
import type { StudentTutorTurnResponse } from "../types/contracts";
import type { SessionAction } from "@/modules/learning/domain/session-reducer";
import type { Dispatch } from "react";
const snapshotDebounceMilliseconds = 450;
const activeHeartbeatMilliseconds = 30_000;
export function useLearningRecovery(
  session: Session,
  dispatch: Dispatch<SessionAction>,
) {
  const [learningSyncReady, setLearningSyncReady] = useState(false);
  const [recoveryError, setRecoveryError] = useState<string | null>(null);
  const sessionRef = useRef(session);
  const revisionRef = useRef(0);
  const tokenRef = useRef<{ actorId: string; token: string } | null>(null);
  const [evidenceScope, setEvidenceScope] = useState<EvidenceScope | null>(
    null,
  );
  const [completionAcknowledged, setCompletionAcknowledged] = useState(false);
  const hydrationRef = useRef(0);
  const skipLocalSaveRef = useRef(false);
  const latestEvidenceRef = useRef<string | null>(null);
  const lastPauseAtRef = useRef(0);

  const resetRecovery = useCallback(() => {
    hydrationRef.current += 1;
    latestEvidenceRef.current = null;
    revisionRef.current = 0;
    tokenRef.current = null;
    setEvidenceScope(null);
    setCompletionAcknowledged(false);
    skipLocalSaveRef.current = false;
    setLearningSyncReady(false);
    setRecoveryError(null);
  }, []);

  const isCurrentRecovery = useCallback(
    (generation: number, expected: EvidenceScope) => {
      const current = currentEvidenceScope();
      return (
        generation === hydrationRef.current &&
        current !== null &&
        sameEvidenceScope(current, expected)
      );
    },
    [],
  );

  const rejectChangedAttempt = useCallback(() => {
    resetRecovery();
    clearStudentMissionSession();
    dispatch({ type: "leave-mission" });
    setRecoveryError(
      "This mission changed in another tab. Choose it again at the glowing desk to resume the correct version.",
    );
  }, [dispatch, resetRecovery]);

  const onEvidenceTurn = useCallback(
    (row: EvidenceRecord, response: StudentTutorTurnResponse) => {
      if (
        row.evidence.type === "feedback" ||
        (latestEvidenceRef.current &&
          latestEvidenceRef.current !== row.evidence.event_id)
      )
        return;
      dispatch({
        type: "apply-tutor-turn",
        questionId: row.questionId,
        message: response.turn.message,
        whyPrompt: response.turn.why_prompt,
        explanationSteps: response.turn.explanation_steps,
        explanationStyle: response.turn.explanation_style,
        difficultyBias: response.turn.difficulty_bias,
      });
    },
    [dispatch],
  );
  const {
    state: evidenceState,
    record: recordEvidence,
    flush: flushEvidence,
    hasPending,
    retry: retryStoredEvidence,
    deletePending,
  } = useEvidenceSync(evidenceScope, rejectChangedAttempt, onEvidenceTurn);
  const requestTutorTurn = useCallback(
    (questionId: string, evidence: TutorEvidence, commit: () => void) =>
      recordEvidence(questionId, evidence, () => {
        latestEvidenceRef.current = evidence.event_id;
        setCompletionAcknowledged(false);
        commit();
      }),
    [recordEvidence],
  );

  useEffect(() => {
    sessionRef.current = session;
  }, [session]);

  // The class gate and mission picker dispatch this after anonymous auth state
  // changes. Hydration runs only when a selected published mission exists. The
  // newer of Firestore and the local checkpoint wins, so an offline click made
  // just before refresh is not lost.
  useEffect(() => {
    let cancelled = false;

    async function hydrate() {
      const missionSession = readStudentMissionSession();
      if (!missionSession) {
        return;
      }

      const generation = ++hydrationRef.current;
      const candidateScope = currentEvidenceScope();
      const user = currentStudentIdentity();
      const isCurrent = () =>
        !cancelled &&
        candidateScope !== null &&
        isCurrentRecovery(generation, candidateScope);
      setRecoveryError(null);
      setEvidenceScope(null);
      setCompletionAcknowledged(false);
      setLearningSyncReady(false);
      tokenRef.current = null;
      if (!user?.isAnonymous) return;
      if (!candidateScope) {
        setRecoveryError(
          "Open this mission again to save responses in this browser.",
        );
        return;
      }
      const mission = buildGameMission(missionSession.mission);
      const local = loadStudentRecovery(
        mission,
        studentRecoveryID(missionSession),
      );
      revisionRef.current = local?.revision ?? 0;
      skipLocalSaveRef.current = true;
      dispatch({
        type: "hydrate",
        mission,
        memory: local?.memory ?? null,
        checkpoint: local?.checkpoint,
        now: Date.now(),
      });
      try {
        const idToken = await user.getIdToken();
        if (!isCurrent()) return;
        tokenRef.current = { actorId: candidateScope.actorId, token: idToken };
        const response = await learningApi.studentLearning(
          idToken,
          missionSession.missionId,
          pinnedAttemptScope(missionSession),
        );
        if (!isCurrent()) {
          return;
        }

        if (
          response.attempt &&
          (response.attempt.mission_id !== missionSession.missionId ||
            response.attempt.published_revision !==
              missionSession.publishedRevision ||
            (missionSession.attemptId &&
              response.attempt.id !== missionSession.attemptId))
        ) {
          rejectChangedAttempt();
          return;
        }

        if (
          !response.attempt &&
          sessionStorage.getItem(
            "classgame:tutor-outbox:" + candidateScope.attemptId,
          )
        )
          throw new Error(
            "Older work needs a confirmed attempt before it can be restored.",
          );
        await migrateLegacyEvidence(
          candidateScope,
          new Set(
            missionSession.mission.preview_questions.map(
              (question) => question.id,
            ),
          ),
        );
        if (!isCurrent()) return;
        const unsent = await readTutorOutbox(candidateScope);
        if (!isCurrent()) return;
        setEvidenceScope(candidateScope);
        latestEvidenceRef.current = unsent.at(-1)?.evidence.event_id ?? null;
        setCompletionAcknowledged(
          response.attempt?.status === "completed" && unsent.length === 0,
        );
        const latestLocal =
          loadStudentRecovery(mission, studentRecoveryID(missionSession)) ??
          local;
        const remoteRevision = response.attempt?.snapshot.revision ?? 0;
        const localWins =
          latestLocal !== null &&
          latestLocal.revision > remoteRevision &&
          response.attempt?.status !== "completed";
        const memory = localWins
          ? latestLocal.memory
          : response.attempt
            ? memoryFromLearningAttempt(response.attempt, mission)
            : (latestLocal?.memory ?? createInitialMemory(mission));
        const checkpoint = localWins
          ? latestLocal.checkpoint
          : response.attempt
            ? checkpointFromLearningAttempt(response.attempt, mission)
            : (latestLocal?.checkpoint ?? null);

        revisionRef.current = Math.max(
          latestLocal?.revision ?? 0,
          remoteRevision,
        );
        skipLocalSaveRef.current = true;
        dispatch({
          type: "hydrate",
          mission,
          memory,
          checkpoint,
          now: Date.now(),
        });
      } catch (error) {
        if (isCurrent() && isAttemptScopeConflict(error)) {
          rejectChangedAttempt();
        }
        if (isCurrent()) {
          let hasLegacy = true;
          try {
            hasLegacy = Boolean(
              sessionStorage.getItem(
                "classgame:tutor-outbox:" + candidateScope.attemptId,
              ),
            );
          } catch {}
          if (!hasLegacy) setEvidenceScope(candidateScope);
          else
            setRecoveryError(
              "Older unsent work could not be restored. Keep this tab open. Allow site data, reconnect, and select Retry.",
            );
        }
        // Offline responses may advance only after the durable queue accepted them.
      } finally {
        if (isCurrent()) {
          setLearningSyncReady(true);
        }
      }
    }

    const hydrateWhenReady = () => void hydrate();
    window.addEventListener(studentSessionReadyEvent, hydrateWhenReady);
    if (readStudentMissionSession()) {
      void hydrate();
    }

    return () => {
      cancelled = true;
      hydrationRef.current += 1;
      tokenRef.current = null;
      window.removeEventListener(studentSessionReadyEvent, hydrateWhenReady);
    };
  }, [dispatch, rejectChangedAttempt, isCurrentRecovery]);

  const syncSnapshot = useCallback(
    async (statusOverride?: StudentLearningStatus, keepalive = false) => {
      let activeSession = sessionRef.current;
      const missionSession = readStudentMissionSession();
      const user = currentStudentIdentity();
      const capturedScope = currentEvidenceScope();
      if (
        !activeSession.hydrated ||
        !missionSession ||
        !user?.isAnonymous ||
        !capturedScope ||
        missionSession.missionId !== activeSession.mission.id ||
        capturedScope.publishedRevision !== activeSession.mission.publishedRevision
      ) {
        return false;
      }

      const generation = hydrationRef.current;
      const isCurrent = () => isCurrentRecovery(generation, capturedScope);
      if (keepalive) {
        if (await hasPending()) return false;
      } else {
        const result = await flushEvidence();
        if (!result.complete) return false;
      }
      if (!isCurrent()) return false;

      let idToken =
        tokenRef.current?.actorId === capturedScope.actorId
          ? tokenRef.current.token
          : "";
      if (!keepalive) {
        try {
          idToken = await user.getIdToken();
          if (!isCurrent()) return false;
          tokenRef.current = { actorId: capturedScope.actorId, token: idToken };
        } catch {
          return false;
        }
      }
      if (!idToken) {
        return false;
      }

      // An answer may have completed while evidence was in flight. Persist the
      // latest reducer commit so a delayed write cannot overwrite a newer board.
      if (await hasPending()) return false;
      if (!isCurrent()) return false;
      activeSession = sessionRef.current;
      if (
        !activeSession.hydrated ||
        activeSession.mission.id !== missionSession.missionId ||
        activeSession.mission.publishedRevision !== capturedScope.publishedRevision ||
        !isCurrent()
      )
        return false;
      const revision = nextRevision(revisionRef);
      const checkpoint = checkpointFromSession(activeSession);
      saveMemory(
        activeSession.memory,
        studentRecoveryID(missionSession),
        checkpoint,
        revision,
      );

      try {
        const response = await learningApi.saveStudentLearning(
          idToken,
          {
            ...pinnedAttemptScope(missionSession),
            status:
              statusOverride ??
              learningStatus(activeSession.memory, activeSession.lesson?.phase),
            snapshot: toLearningSnapshot(
              activeSession,
              checkpoint,
              revision,
              keepalive,
            ),
          },
          { keepalive },
        );
        if (!isCurrent()) return false;
        if (
          response.attempt &&
          (response.attempt.id !== capturedScope.attemptId ||
            response.attempt.mission_id !== capturedScope.missionId ||
            response.attempt.published_revision !== capturedScope.publishedRevision)
        ) {
          rejectChangedAttempt();
          return false;
        }
        const pending = await hasPending();
        if (!isCurrent()) return false;
        setCompletionAcknowledged(
          response.attempt?.status === "completed" && !pending,
        );
        revisionRef.current = Math.max(
          revisionRef.current,
          response.attempt?.snapshot.revision ?? 0,
        );
        return true;
      } catch (error) {
        if (isCurrent() && isAttemptScopeConflict(error))
          rejectChangedAttempt();
        // Keep local resume state during transient failures; scope conflicts require reselection.
        return false;
      }
    },
    [rejectChangedAttempt, flushEvidence, hasPending, isCurrentRecovery],
  );

  // Save the exact board phase synchronously on every reducer commit. Notes,
  // model observations and event history are deliberately stripped locally.
  useEffect(() => {
    if (!session.hydrated) {
      return;
    }
    if (skipLocalSaveRef.current) {
      skipLocalSaveRef.current = false;
      return;
    }
    const missionSession = readStudentMissionSession();
    if (
      !missionSession ||
      missionSession.missionId !== session.mission.id ||
      missionSession.publishedRevision !== session.mission.publishedRevision
    ) {
      return;
    }

    const revision = nextRevision(revisionRef);
    saveMemory(
      session.memory,
      studentRecoveryID(missionSession),
      checkpointFromSession(session),
      revision,
    );
  }, [session]);

  // Debounced writes capture each meaningful step. A heartbeat also proves the
  // tab is still active while a student spends time reading an explanation.
  useEffect(() => {
    if (!session.hydrated || !learningSyncReady) {
      return;
    }

    const timer = window.setTimeout(
      () => void syncSnapshot(),
      snapshotDebounceMilliseconds,
    );
    return () => window.clearTimeout(timer);
  }, [learningSyncReady, session, syncSnapshot, evidenceState.pending]);

  useEffect(() => {
    if (!learningSyncReady) {
      return;
    }

    const heartbeat = window.setInterval(() => {
      if (document.visibilityState === "visible") {
        void syncSnapshot();
      }
    }, activeHeartbeatMilliseconds);

    return () => window.clearInterval(heartbeat);
  }, [learningSyncReady, syncSnapshot]);

  // A final keepalive checkpoint marks the attempt paused immediately. The
  // queued watchdog turns it into abandoned only if no newer heartbeat arrives.
  useEffect(() => {
    const pause = () => {
      const now = Date.now();
      if (now - lastPauseAtRef.current < 500) {
        return;
      }
      lastPauseAtRef.current = now;
      void syncSnapshot("paused", true);
    };
    const visibilityChanged = () => {
      if (document.visibilityState === "hidden") {
        pause();
      } else {
        void syncSnapshot();
      }
    };

    window.addEventListener("pagehide", pause);
    document.addEventListener("visibilitychange", visibilityChanged);
    return () => {
      window.removeEventListener("pagehide", pause);
      document.removeEventListener("visibilitychange", visibilityChanged);
    };
  }, [syncSnapshot]);

  const clearCurrentRecovery = useCallback(() => {
    const selected = readStudentMissionSession();
    clearMemory(
      selected ? studentRecoveryID(selected) : sessionRef.current.mission.id,
    );
    revisionRef.current = 0;
  }, []);
  const retryEvidence = useCallback(async () => {
    if (!evidenceScope)
      window.dispatchEvent(new Event(studentSessionReadyEvent));
    else {
      const scope = currentEvidenceScope();
      const generation = hydrationRef.current;
      if (!scope) return;
      await retryStoredEvidence();
      if (!isCurrentRecovery(generation, scope)) return;
      await syncSnapshot();
    }
  }, [evidenceScope, retryStoredEvidence, syncSnapshot, isCurrentRecovery]);
  const deleteUnsentEvidence = useCallback(async () => {
    const scope = currentEvidenceScope();
    const generation = hydrationRef.current;
    if (!scope) return;
    try {
      if (await deletePending()) {
        if (!isCurrentRecovery(generation, scope)) return;
        const selected = readStudentMissionSession();
        if (selected) clearMemory(studentRecoveryID(selected));
        resetRecovery();
        clearStudentMissionSession();
        dispatch({ type: "leave-mission" });
        setRecoveryError(
          "Unsent work was deleted from this browser. Open the mission again to continue.",
        );
      }
    } catch {
      if (!isCurrentRecovery(generation, scope)) return;
      setRecoveryError(
        "Could not delete unsent work. Check browser storage and retry.",
      );
    }
  }, [deletePending, dispatch, resetRecovery, isCurrentRecovery]);
  return {
    requestTutorTurn,
    evidenceState,
    completionAcknowledged,
    retryEvidence,
    deleteUnsentEvidence,
    resetRecovery,
    clearCurrentRecovery,
    recoveryError,
  };
}

function studentRecoveryID(session: StudentMissionSession) {
  const anonymousUID = currentStudentIdentity()?.uid ?? "student";
  return session.attemptId || `${anonymousUID}:${session.missionId}`;
}

function pinnedAttemptScope(
  session: StudentMissionSession,
): StudentAttemptScope {
  return {
    mission_id: session.missionId,
    published_revision: session.publishedRevision,
    attempt_id: session.attemptId || undefined,
  };
}

function isAttemptScopeConflict(error: unknown): boolean {
  return (
    error instanceof ApiError &&
    error.status === 409 &&
    error.code === "mission_mismatch"
  );
}
