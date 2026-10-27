"use client";
import { useCallback, useEffect, useRef, useState } from "react";
import { currentStudentIdentity } from "@/modules/identity";
import {
  readStudentClassSession,
  readStudentMissionSession,
} from "@/modules/classrooms";
import { ApiError } from "@/shared/http/errors";
import { learningApi } from "../http";
import {
  acknowledgeTutorEvidence,
  claimEvidence,
  countActorEvidence,
  deleteActorEvidence,
  failEvidence,
  observeEvidenceChanges,
  queueTutorEvidence,
  readTutorOutbox,
  retryTutorEvidence,
  type EvidenceScope,
  type EvidenceRecord,
  type TutorEvidence,
} from "../tutor-outbox";
import {
  EVIDENCE_LIMITS,
  evidenceScopeKey,
  sameEvidenceScope,
  canFinishEvidenceSync,
  evidenceBlockReason,
} from "../../domain/evidence-queue";
import { buildGameMission } from "../../domain/published-mission";
import { createInitialSession } from "../../domain/session-reducer";
import { createInitialMemory } from "../../domain/memory";
import {
  checkpointFromSession,
  toLearningSnapshot,
} from "../checkpoint-mapper";
import type { StudentTutorTurnResponse } from "../types/contracts";
export type EvidenceSyncState = {
  pending: number;
  actorPending: number;
  accepting: boolean;
  syncing: boolean;
  paused: boolean;
  terminal: boolean;
  blocked: boolean;
  message: string | null;
};
export type EvidenceFlushResult = { complete: boolean; pending: number };
export function currentEvidenceScope(): EvidenceScope | null {
  const user = currentStudentIdentity(),
    classSession = readStudentClassSession(),
    selected = readStudentMissionSession();
  if (
    !user?.isAnonymous ||
    !classSession ||
    !selected ||
    !selected.attemptId ||
    selected.publishedRevision < 1
  )
    return null;
  return {
    actorId: user.uid,
    classId: classSession.classId,
    missionId: selected.missionId,
    publishedRevision: selected.publishedRevision,
    attemptId: selected.attemptId,
  };
}
export function useEvidenceSync(
  scope: EvidenceScope | null,
  onConflict: () => void,
  onTurn: (row: EvidenceRecord, response: StudentTutorTurnResponse) => void,
) {
  const [state, setState] = useState<EvidenceSyncState>({
    pending: 0,
    actorPending: 0,
    accepting: false,
    syncing: false,
    paused: false,
    terminal: false,
    blocked: true,
    message: null,
  });
  const scopeRef = useRef(scope),
    callbacks = useRef({ onConflict, onTurn }),
    busy = useRef<Promise<EvidenceFlushResult> | null>(null),
    accepting = useRef<object | null>(null),
    controllerRef = useRef<AbortController | null>(null),
    problemRef = useRef(false),
    nextRetry = useRef(0);
  const activeRun = useRef<object | null>(null);
  const owner = useRef("");
  const key = scope ? evidenceScopeKey(scope) : "";
  useEffect(() => {
    scopeRef.current = scope;
    callbacks.current = { onConflict, onTurn };
  }, [scope, onConflict, onTurn]);
  const matches = useCallback((expected: EvidenceScope) => {
    const active = currentEvidenceScope();
    return (
      !!active &&
      sameEvidenceScope(active, expected) &&
      scopeRef.current !== null &&
      sameEvidenceScope(scopeRef.current, expected)
    );
  }, []);
  const refresh = useCallback(async () => {
    const current = scopeRef.current;
    if (!current) return;
    try {
      const [rows, total] = await Promise.all([
        readTutorOutbox(current),
        countActorEvidence(current.actorId),
      ]);
      if (!matches(current)) return;
      const first = rows[0];
      nextRetry.current = Math.max(
        first?.retryAt ?? 0,
        first?.leaseOwner && first.leaseOwner !== owner.current
          ? (first.leaseUntil ?? 0)
          : 0,
      );
      setState((previous) => ({
        ...previous,
        pending: rows.length,
        actorPending: total,
        blocked:
          problemRef.current ||
          rows.length >= EVIDENCE_LIMITS.perScope ||
          !!first?.blocked,
        terminal: !!first?.blocked,
        paused: Boolean(
          first?.blocked ||
            first?.attempts >= EVIDENCE_LIMITS.automaticAttempts,
        ),
        message: problemRef.current
          ? previous.message
          : first?.blocked
            ? first.blockReason === "turn-limit"
              ? "This attempt has reached its response limit. Your unsent work is kept. Ask your teacher to start a new attempt."
              : "This work could not be accepted for this attempt. It is kept in this browser. Ask your teacher for help before deleting it."
            : first?.attempts >= EVIDENCE_LIMITS.automaticAttempts
              ? "Unsent work is paused. Check your connection and select Retry."
              : rows.length
                ? `${rows.length} responses are waiting to send. Keep this browser data until they are sent.`
                : null,
      }));
    } catch {
      if (!matches(current)) return;
      problemRef.current = true;
      setState((previous) => ({
        ...previous,
        blocked: true,
        message:
          "This browser cannot save work. Allow site data, then select Retry. Your response has not been accepted.",
      }));
    }
  }, [matches]);
  const flush = useCallback(async (): Promise<EvidenceFlushResult> => {
    if (busy.current) return busy.current;
    const current = scopeRef.current;
    if (!current || !matches(current)) return { complete: false, pending: 0 };
    if (!owner.current)
      owner.current =
        globalThis.crypto?.randomUUID?.() ??
        `tab-${Date.now()}-${Math.random()}`;
    const controller = new AbortController();
    controllerRef.current = controller;
    const timer = setTimeout(() => controller.abort(), 60_000);
    const ticket = {};
    activeRun.current = ticket;
    const work = (async () => {
      setState((previous) => ({ ...previous, syncing: true }));
      let claimed: EvidenceRecord | undefined;
      try {
        if (typeof navigator !== "undefined" && !navigator.onLine)
          return {
            complete: false,
            pending: (await readTutorOutbox(current)).length,
          };
        for (let sent = 0; sent < 6; sent++) {
          if (controller.signal.aborted || !matches(current))
            return { complete: false, pending: 0 };
          const claim = await claimEvidence(current, owner.current);
          if (controller.signal.aborted || !matches(current))
            return { complete: false, pending: 0 };
          if (claim.kind === "empty")
            return { complete: !problemRef.current, pending: 0 };
          if (claim.kind === "wait") {
            nextRetry.current = claim.until;
            break;
          }
          if (claim.kind === "paused") break;
          claimed = claim.record;
          const user = currentStudentIdentity(),
            selected = readStudentMissionSession();
          if (!user || !selected || !matches(current)) break;
          const token = await abortable(user.getIdToken(), controller.signal);
          if (controller.signal.aborted || !matches(current)) break;
          const wire = {
            mission_id: current.missionId,
            published_revision: current.publishedRevision,
            attempt_id: current.attemptId,
          };
          const remote = await learningApi.studentLearning(
            token,
            current.missionId,
            wire,
            { signal: controller.signal },
          );
          if (controller.signal.aborted || !matches(current)) break;
          if (
            remote.attempt &&
            (remote.attempt.id !== current.attemptId ||
              remote.attempt.published_revision !== current.publishedRevision)
          )
            throw new ApiError(
              409,
              "mission_mismatch",
              "Choose the mission again.",
            );
          if (!remote.attempt) {
            const mission = buildGameMission(selected.mission),
              initial = {
                ...createInitialSession(),
                mission,
                hydrated: true,
                memory: createInitialMemory(mission),
              };
            await learningApi.saveStudentLearning(
              token,
              {
                ...wire,
                status: "exploring",
                snapshot: toLearningSnapshot(
                  initial,
                  checkpointFromSession(initial),
                  Date.now(),
                ),
              },
              { signal: controller.signal },
            );
          }
          if (controller.signal.aborted || !matches(current)) break;
          const perTurn = new AbortController(),
            abortTurn = () => perTurn.abort(),
            turnTimer = setTimeout(abortTurn, 45_000);
          controller.signal.addEventListener("abort", abortTurn, {
            once: true,
          });
          let response: StudentTutorTurnResponse;
          try {
            response = await learningApi.studentTutorTurn(
              token,
              { ...claimed.evidence, ...wire },
              { signal: perTurn.signal },
            );
          } finally {
            clearTimeout(turnTimer);
            controller.signal.removeEventListener("abort", abortTurn);
          }
          await acknowledgeTutorEvidence(current, claimed.evidence.event_id);
          if (matches(current) && !controller.signal.aborted)
            callbacks.current.onTurn(claimed, response);
          claimed = undefined;
        }
        const rows = await readTutorOutbox(current);
        return {
          complete: canFinishEvidenceSync(rows.length, problemRef.current),
          pending: rows.length,
        };
      } catch (error) {
        if (claimed) {
          try {
            await failEvidence(
              claimed,
              owner.current,
              error instanceof ApiError
                ? evidenceBlockReason(error.status, error.code)
                : undefined,
              error instanceof ApiError ? error.retryAfterSeconds : undefined,
            );
          } catch {
            if (matches(current)) problemRef.current = true;
          }
        }
        if (
          matches(current) &&
          error instanceof ApiError &&
          error.status === 409 &&
          error.code === "mission_mismatch"
        )
          callbacks.current.onConflict();
        if (matches(current))
          setState((previous) => ({
            ...previous,
            message: problemRef.current
              ? "Unsent work is kept, but this browser cannot update it. Allow site data and select Retry."
              : "Work is still waiting to send. Check your connection and select Retry.",
            blocked: problemRef.current || previous.blocked,
          }));
        return { complete: false, pending: claimed ? 1 : 0 };
      } finally {
        clearTimeout(timer);
        controller.abort();
        if (controllerRef.current === controller) controllerRef.current = null;
        if (activeRun.current === ticket) {
          busy.current = null;
          activeRun.current = null;
        }
        if (matches(current)) {
          setState((previous) => ({ ...previous, syncing: false }));
          await refresh();
        }
      }
    })();
    busy.current = work;
    return work;
  }, [matches, refresh]);
  const record = useCallback(
    async (
      questionId: string,
      evidence: TutorEvidence,
      commit: () => void,
    ): Promise<boolean> => {
      const current = scopeRef.current;
      if (
        !current ||
        !matches(current) ||
        accepting.current ||
        problemRef.current
      ) {
        setState((previous) => ({
          ...previous,
          message:
            "Wait until this browser can save your response. Select Retry.",
        }));
        return false;
      }
      const ticket = {};
      accepting.current = ticket;
      setState((previous) => ({ ...previous, accepting: true }));
      try {
        await queueTutorEvidence(current, questionId, evidence);
        if (!matches(current)) return false;
        commit();
        await refresh();
        if (matches(current)) void flush();
        return true;
      } catch {
        if (!matches(current)) return false;
        problemRef.current = true;
        setState((previous) => ({
          ...previous,
          blocked: true,
          message:
            "Your response was not saved. Reconnect and select Retry. If storage is full, send or delete unsent work first.",
        }));
        return false;
      } finally {
        if (accepting.current === ticket) {
          accepting.current = null;
          if (matches(current))
            setState((previous) => ({ ...previous, accepting: false }));
        }
      }
    },
    [flush, matches, refresh],
  );
  const retry = useCallback(async () => {
    const current = scopeRef.current;
    if (!current) return;
    try {
      await retryTutorEvidence(current);
      if (!matches(current)) return;
      problemRef.current = false;
      nextRetry.current = 0;
      await refresh();
      if (matches(current)) await flush();
    } catch {
      if (!matches(current)) return;
      problemRef.current = true;
      setState((previous) => ({
        ...previous,
        blocked: true,
        message:
          "This browser cannot save work. Allow site data or use another browser.",
      }));
    }
  }, [flush, refresh, matches]);
  const deletePending = useCallback(async (): Promise<boolean> => {
    const current = scopeRef.current;
    if (!current || currentStudentIdentity()?.uid !== current.actorId)
      return false;
    const total = await countActorEvidence(current.actorId);
    if (!matches(current)) return false;
    if (
      !window.confirm(
        `Delete ${total} unsent responses from this browser? Work already sent stays with your teacher. You may need to answer again.`,
      )
    )
      return false;
    if (!matches(current)) return false;
    controllerRef.current?.abort();
    await deleteActorEvidence(current.actorId);
    if (!matches(current)) return false;
    problemRef.current = false;
    await refresh();
    return true;
  }, [refresh, matches]);
  useEffect(() => {
    problemRef.current = false;
    accepting.current = null;
    controllerRef.current?.abort();
    busy.current = null;
    activeRun.current = null;
    let disposed = false;
    queueMicrotask(() => {
      if (disposed) return;
      setState({
        pending: 0,
        actorPending: 0,
        accepting: false,
        syncing: false,
        paused: false,
        terminal: false,
        blocked: true,
        message: null,
      });
      void refresh();
    });
    const stop = observeEvidenceChanges(() => void refresh());
    return () => {
      disposed = true;
      controllerRef.current?.abort();
      stop();
    };
  }, [key, refresh]);
  useEffect(() => {
    if (
      !key ||
      state.syncing ||
      state.pending === 0 ||
      state.paused ||
      problemRef.current
    )
      return;
    const due = Math.max(1000, nextRetry.current - Date.now());
    const timer = setTimeout(
      () => {
        if (document.visibilityState === "visible") void flush();
      },
      Math.min(due, 86400_000),
    );
    return () => clearTimeout(timer);
  }, [key, state.pending, state.syncing, state.paused, flush]);
  useEffect(() => {
    const retryOnline = () => {
      if (document.visibilityState === "visible") void flush();
    };
    window.addEventListener("online", retryOnline);
    document.addEventListener("visibilitychange", retryOnline);
    return () => {
      window.removeEventListener("online", retryOnline);
      document.removeEventListener("visibilitychange", retryOnline);
    };
  }, [flush]);
  const hasPending = useCallback(async () => {
    const current = scopeRef.current;
    if (!current) return true;
    try {
      return (await readTutorOutbox(current)).length > 0;
    } catch {
      return true;
    }
  }, []);
  return { state, record, flush, refresh, retry, deletePending, hasPending };
}

function abortable<T>(promise: Promise<T>, signal: AbortSignal): Promise<T> {
  return new Promise((resolve, reject) => {
    const abort = () =>
      reject(new DOMException("Request stopped.", "AbortError"));
    if (signal.aborted) {
      abort();
      return;
    }
    signal.addEventListener("abort", abort, { once: true });
    promise
      .then(resolve, reject)
      .finally(() => signal.removeEventListener("abort", abort));
  });
}
