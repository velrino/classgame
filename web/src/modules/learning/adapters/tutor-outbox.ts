import type {
  StudentAttemptScope,
  StudentTutorTurnRequest,
} from "@/modules/learning/adapters/types/contracts";

export type TutorEvidence = Omit<
  StudentTutorTurnRequest,
  keyof StudentAttemptScope
>;
export type PendingTutorTurn = {
  readonly questionId: string;
  readonly evidence: TutorEvidence;
};
const prefix = "classgame:tutor-outbox:";
const maximumPendingTurns = 64;

/** Only unacknowledged student input is retained, scoped to this tab and attempt. */
export function readTutorOutbox(recoveryId: string): PendingTutorTurn[] {
  try {
    const value: unknown = JSON.parse(
      sessionStorage.getItem(prefix + recoveryId) ?? "[]",
    );
    return Array.isArray(value)
      ? value
          .filter(
            (item): item is PendingTutorTurn =>
              typeof item === "object" &&
              item !== null &&
              typeof item.questionId === "string" &&
              typeof item.evidence?.event_id === "string" &&
              typeof item.evidence?.type === "string",
          )
          .slice(-maximumPendingTurns)
      : [];
  } catch {
    return [];
  }
}

export function queueTutorEvidence(
  recoveryId: string,
  turn: PendingTutorTurn,
): void {
  const pending = readTutorOutbox(recoveryId).filter(
    (item) => item.evidence.event_id !== turn.evidence.event_id,
  );
  try {
    sessionStorage.setItem(
      prefix + recoveryId,
      JSON.stringify([...pending, turn].slice(-maximumPendingTurns)),
    );
  } catch {
    /* The current in-flight request still works if browser storage is unavailable. */
  }
}

export function acknowledgeTutorEvidence(
  recoveryId: string,
  eventId: string,
): void {
  const pending = readTutorOutbox(recoveryId).filter(
    (item) => item.evidence.event_id !== eventId,
  );
  try {
    if (pending.length)
      sessionStorage.setItem(prefix + recoveryId, JSON.stringify(pending));
    else sessionStorage.removeItem(prefix + recoveryId);
  } catch {
    /* Server event IDs make a later replay idempotent. */
  }
}
