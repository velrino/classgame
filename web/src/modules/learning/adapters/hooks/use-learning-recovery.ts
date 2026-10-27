"use client";
import { currentStudentIdentity } from "@/modules/identity";

import { learningApi } from "@/modules/learning/adapters/http";
import { useCallback, useEffect, useRef, useState } from "react";

import type { StudentLearningStatus } from "@/modules/learning/adapters/types/contracts";

import {
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
  acknowledgeTutorEvidence,
  queueTutorEvidence,
  readTutorOutbox,
  type TutorEvidence,
} from "@/modules/learning/adapters/tutor-outbox";
import type { SessionAction } from "@/modules/learning/domain/session-reducer";
import type { Dispatch } from "react";
const snapshotDebounceMilliseconds = 450;
const activeHeartbeatMilliseconds = 30_000;
export function useLearningRecovery(
  session: Session,
  dispatch: Dispatch<SessionAction>,
) {
  const [learningSyncReady, setLearningSyncReady] = useState(false);
  const sessionRef = useRef(session);
  const revisionRef = useRef(0);
  const tokenRef = useRef("");
  const attemptReadyRef = useRef(false);
  const hydrationRef = useRef(0);
  const skipLocalSaveRef = useRef(false);
  const tutorRequestRef = useRef(0);
  const lastPauseAtRef = useRef(0);
  const pendingTurnsRef = useRef(new Map<string, Promise<void>>());
  const flushEvidenceRef = useRef<() => Promise<void>>(async () => undefined);
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
      setLearningSyncReady(false);

      const user = currentStudentIdentity();
      if (!user?.isAnonymous) {
        return;
      }

      try {
        const idToken = await user.getIdToken();
        tokenRef.current = idToken;
        const response = await learningApi.studentLearning(
          idToken,
          missionSession.missionId,
        );
        if (cancelled || generation !== hydrationRef.current) {
          return;
        }

        attemptReadyRef.current = response.attempt !== null;
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
      } catch {
        // The local recovery remains playable while the API is unavailable.
      } finally {
        if (!cancelled && generation === hydrationRef.current) {
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
      window.removeEventListener(studentSessionReadyEvent, hydrateWhenReady);
    };
  }, [dispatch]);

  const syncSnapshot = useCallback(
    async (
      statusOverride?: StudentLearningStatus,
      keepalive = false,
      flushEvidence = true,
    ) => {
      const activeSession = sessionRef.current;
      const missionSession = readStudentMissionSession();
      const user = currentStudentIdentity();
      if (
        !activeSession.hydrated ||
        !missionSession ||
        !user?.isAnonymous ||
        missionSession.missionId !== activeSession.mission.id
      ) {
        return false;
      }

      const generation = hydrationRef.current;
      if (!keepalive && flushEvidence) {
        await flushEvidenceRef.current();
        await Promise.all([...pendingTurnsRef.current.values()]);
        if (
          generation !== hydrationRef.current ||
          readStudentMissionSession()?.missionId !== missionSession.missionId
        )
          return false;
      }

      let idToken = tokenRef.current;
      if (!keepalive) {
        try {
          idToken = await user.getIdToken();
          tokenRef.current = idToken;
        } catch {
          return false;
        }
      }
      if (!idToken) {
        return false;
      }

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
            mission_id: missionSession.missionId,
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
        attemptReadyRef.current = response.attempt !== null;
        revisionRef.current = Math.max(
          revisionRef.current,
          response.attempt?.snapshot.revision ?? 0,
        );
        return true;
      } catch {
        // Keep local resume state; the API remains authoritative for learning evidence.
        return false;
      }
    },
    [],
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
    if (!missionSession || missionSession.missionId !== session.mission.id) {
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
  }, [learningSyncReady, session, syncSnapshot]);

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

  const requestTutorTurn = useCallback(
    (renderedQuestionID: string, evidence: TutorEvidence): Promise<void> => {
      const missionSession = readStudentMissionSession();
      const user = currentStudentIdentity();
      if (!missionSession || !user?.isAnonymous) return Promise.resolve();
      const recoveryId = studentRecoveryID(missionSession);
      const existing = pendingTurnsRef.current.get(evidence.event_id);
      if (existing) return existing;
      queueTutorEvidence(recoveryId, {
        questionId: renderedQuestionID,
        evidence,
      });
      const requestVersion = ++tutorRequestRef.current;
      const generation = hydrationRef.current;
      const pending = (async () => {
        if (
          !attemptReadyRef.current &&
          !(await syncSnapshot("learning", false, false))
        )
          return;
        try {
          const idToken = await user.getIdToken();
          tokenRef.current = idToken;
          const response = await learningApi.studentTutorTurn(idToken, {
            mission_id: missionSession.missionId,
            ...evidence,
          });
          acknowledgeTutorEvidence(recoveryId, evidence.event_id);
          if (
            requestVersion !== tutorRequestRef.current ||
            generation !== hydrationRef.current
          )
            return;
          dispatch({
            type: "apply-tutor-turn",
            questionId: renderedQuestionID,
            message: response.turn.message,
            whyPrompt: response.turn.why_prompt,
            explanationSteps: response.turn.explanation_steps,
            explanationStyle: response.turn.explanation_style,
            difficultyBias: response.turn.difficulty_bias,
          });
        } catch {
          /* Keep student evidence for the next heartbeat; deterministic tutor copy remains playable. */
        }
      })().finally(() => pendingTurnsRef.current.delete(evidence.event_id));
      pendingTurnsRef.current.set(evidence.event_id, pending);
      return pending;
    },
    [dispatch, syncSnapshot],
  );

  useEffect(() => {
    flushEvidenceRef.current = async () => {
      const selected = readStudentMissionSession();
      if (!selected) return;
      const turns = readTutorOutbox(studentRecoveryID(selected));
      // Replay sequentially to preserve the conversation order and respect per-student quotas.
      for (const turn of turns.slice(0, 6))
        await requestTutorTurn(turn.questionId, turn.evidence);
    };
  }, [requestTutorTurn]);

  const resetRecovery = useCallback(() => {
    hydrationRef.current += 1;
    tutorRequestRef.current += 1;
    revisionRef.current = 0;
    tokenRef.current = "";
    attemptReadyRef.current = false;
    skipLocalSaveRef.current = false;
    setLearningSyncReady(false);
  }, []);
  const clearCurrentRecovery = useCallback(() => {
    const selected = readStudentMissionSession();
    clearMemory(
      selected ? studentRecoveryID(selected) : sessionRef.current.mission.id,
    );
    revisionRef.current = 0;
  }, []);
  return { requestTutorTurn, resetRecovery, clearCurrentRecovery };
}

function studentRecoveryID(session: StudentMissionSession) {
  const anonymousUID = currentStudentIdentity()?.uid ?? "student";
  return session.attemptId || `${anonymousUID}:${session.missionId}`;
}
