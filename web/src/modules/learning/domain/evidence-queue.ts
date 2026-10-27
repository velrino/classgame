import type {
  StudentAttemptScope,
  StudentTutorTurnRequest,
} from "../adapters/types/contracts";

export type TutorEvidence = Omit<
  StudentTutorTurnRequest,
  keyof StudentAttemptScope
>;
export type EvidenceScope = {
  actorId: string;
  classId: string;
  missionId: string;
  publishedRevision: number;
  attemptId: string;
};
export type EvidenceRecord = {
  version: 1;
  order?: number;
  scopeKey: string;
  eventKey: string;
  actorId: string;
  scope: EvidenceScope;
  questionId: string;
  evidence: TutorEvidence;
  createdAt: number;
  attempts: number;
  retryAt: number;
  blocked: boolean;
  blockReason?: EvidenceBlockReason;
  leaseOwner?: string;
  leaseUntil?: number;
};
export type EvidenceBlockReason =
  | "turn-limit"
  | "event-conflict"
  | "history-incomplete"
  | "scope"
  | "rejected";

export function evidenceBlockReason(
  status: number,
  code: string,
): EvidenceBlockReason | undefined {
  if (code === "tutor_turn_limit_reached") return "turn-limit";
  if (code === "tutor_event_conflict") return "event-conflict";
  if (code === "tutor_history_incomplete") return "history-incomplete";
  if (code === "mission_mismatch") return "scope";
  return status >= 400 && status < 500 && status !== 429
    ? "rejected"
    : undefined;
}

export const EVIDENCE_LIMITS = {
  perScope: 64,
  global: 256,
  bytes: 1024 * 1024,
  recordBytes: 16 * 1024,
  automaticAttempts: 5,
} as const;
export function evidenceScopeKey(scope: EvidenceScope): string {
  return JSON.stringify([
    scope.actorId,
    scope.classId,
    scope.missionId,
    scope.publishedRevision,
    scope.attemptId,
  ]);
}
export function sameEvidenceScope(
  left: EvidenceScope,
  right: EvidenceScope,
): boolean {
  return evidenceScopeKey(left) === evidenceScopeKey(right);
}
export function validEvidenceScope(scope: EvidenceScope): boolean {
  return (
    [scope.actorId, scope.classId, scope.missionId, scope.attemptId].every(
      (value) =>
        typeof value === "string" && value.length > 0 && value.length <= 160,
    ) &&
    Number.isSafeInteger(scope.publishedRevision) &&
    scope.publishedRevision > 0
  );
}
export function validEvidenceRecord(value: unknown): value is EvidenceRecord {
  if (!value || typeof value !== "object") return false;
  const row = value as EvidenceRecord;
  return (
    row.version === 1 &&
    !!row.scope &&
    validEvidenceScope(row.scope) &&
    row.actorId === row.scope.actorId &&
    row.scopeKey === evidenceScopeKey(row.scope) &&
    typeof row.questionId === "string" &&
    row.questionId.length <= 160 &&
    !!row.evidence &&
    typeof row.evidence.event_id === "string" &&
    row.evidence.event_id.length > 0 &&
    row.evidence.event_id.length <= 120 &&
    row.eventKey === row.scopeKey + ":" + row.evidence.event_id &&
    ["answer", "reasoning", "feedback", "resume"].includes(row.evidence.type) &&
    [
      "stage_id",
      "question_id",
      "prompt",
      "student_response",
      "expected_answer",
      "outcome",
      "explicit_reasoning",
      "feedback",
    ].every(
      (key) => typeof row.evidence[key as keyof TutorEvidence] === "string",
    ) &&
    Number.isFinite(row.createdAt) &&
    Number.isInteger(row.attempts) &&
    row.attempts >= 0 &&
    Number.isFinite(row.retryAt) &&
    typeof row.blocked === "boolean"
  );
}
export function evidenceFingerprint(evidence: TutorEvidence): string {
  return JSON.stringify([
    evidence.event_id,
    evidence.type,
    evidence.stage_id,
    evidence.question_id,
    evidence.prompt,
    evidence.student_response,
    evidence.expected_answer,
    evidence.outcome,
    evidence.explicit_reasoning,
    evidence.feedback,
  ]);
}
export function newEvidenceRecord(
  scope: EvidenceScope,
  questionId: string,
  evidence: TutorEvidence,
  now = Date.now(),
): EvidenceRecord {
  const scopeKey = evidenceScopeKey(scope);
  return {
    version: 1,
    scope: { ...scope },
    scopeKey,
    eventKey: scopeKey + ":" + evidence.event_id,
    actorId: scope.actorId,
    questionId,
    evidence: {
      event_id: evidence.event_id,
      type: evidence.type,
      stage_id: evidence.stage_id,
      question_id: evidence.question_id,
      prompt: evidence.prompt,
      student_response: evidence.student_response,
      expected_answer: evidence.expected_answer,
      outcome: evidence.outcome,
      explicit_reasoning: evidence.explicit_reasoning,
      feedback: evidence.feedback,
    },
    createdAt: now,
    attempts: 0,
    retryAt: 0,
    blocked: false,
  };
}
export function planEvidenceEnqueue(
  records: readonly EvidenceRecord[],
  candidate: EvidenceRecord,
): {
  allowed: boolean;
  reason: "ok" | "duplicate" | "conflict" | "invalid" | "full" | "paused";
} {
  if (
    !validEvidenceRecord(candidate) ||
    records.some((row) => !validEvidenceRecord(row))
  )
    return { allowed: false, reason: "invalid" };
  const existing = records.find((row) => row.eventKey === candidate.eventKey);
  if (existing)
    return evidenceFingerprint(existing.evidence) ===
      evidenceFingerprint(candidate.evidence) &&
      existing.questionId === candidate.questionId
      ? { allowed: true, reason: "duplicate" }
      : { allowed: false, reason: "conflict" };
  if (
    records.some(
      (row) => sameEvidenceScope(row.scope, candidate.scope) && row.blocked,
    )
  )
    return { allowed: false, reason: "paused" };
  const bytes = (row: EvidenceRecord) =>
    new TextEncoder().encode(JSON.stringify(row)).byteLength;
  if (
    records.length >= EVIDENCE_LIMITS.global ||
    records.filter((row) => sameEvidenceScope(row.scope, candidate.scope))
      .length >= EVIDENCE_LIMITS.perScope ||
    bytes(candidate) > EVIDENCE_LIMITS.recordBytes ||
    records.reduce((total, row) => total + bytes(row), 0) + bytes(candidate) >
      EVIDENCE_LIMITS.bytes
  )
    return { allowed: false, reason: "full" };
  return { allowed: true, reason: "ok" };
}
export function canFinishEvidenceSync(
  pendingCount: number,
  storageError: boolean,
): boolean {
  return pendingCount === 0 && !storageError;
}
export function evidenceRetryAt(
  attempts: number,
  now: number,
  retryAfterSeconds = 0,
): number {
  return (
    now +
    Math.max(
      Math.min(60_000, 2000 * 2 ** Math.min(attempts, 5)),
      Math.min(86400, Math.max(0, retryAfterSeconds)) * 1000,
    )
  );
}
