import {
  EVIDENCE_LIMITS,
  evidenceScopeKey,
  newEvidenceRecord,
  planEvidenceEnqueue,
  validEvidenceRecord,
  evidenceRetryAt,
  type EvidenceScope,
  type EvidenceRecord,
  type TutorEvidence,
  type EvidenceBlockReason,
} from "../domain/evidence-queue";
export type {
  EvidenceScope,
  EvidenceRecord,
  TutorEvidence,
} from "../domain/evidence-queue";
const databaseName = "classgame-unsent-evidence-v1";
const changes = "classgame:unsent-evidence-changed";
let database: Promise<IDBDatabase> | undefined;
function openStore(): Promise<IDBDatabase> {
  if (!database)
    database = new Promise<IDBDatabase>((resolve, reject) => {
      if (typeof indexedDB === "undefined") {
        reject(new Error("Browser storage is not available."));
        return;
      }
      const request = indexedDB.open(databaseName, 1);
      const timeout = setTimeout(
        () =>
          reject(
            new Error(
              "Browser storage is busy. Close other game tabs and retry.",
            ),
          ),
        5000,
      );
      request.onupgradeneeded = () => {
        const store = request.result.createObjectStore("turns", {
          keyPath: "order",
          autoIncrement: true,
        });
        store.createIndex("scope", "scopeKey");
        store.createIndex("actor", "actorId");
        store.createIndex("event", "eventKey", { unique: true });
      };
      request.onerror = () => {
        clearTimeout(timeout);
        reject(request.error ?? new Error("Browser storage is blocked."));
      };
      request.onblocked = () => {
        clearTimeout(timeout);
        reject(new Error("Close other game tabs and retry."));
      };
      request.onsuccess = () => {
        clearTimeout(timeout);
        const db = request.result;
        db.onversionchange = () => {
          db.close();
          database = undefined;
        };
        resolve(db);
      };
    }).catch((error) => {
      database = undefined;
      throw error;
    });
  return database;
}
async function transaction<T>(
  mode: IDBTransactionMode,
  run: (store: IDBObjectStore, result: (value: T) => void) => void,
): Promise<T> {
  const db = await openStore();
  return new Promise<T>((resolve, reject) => {
    let tx: IDBTransaction;
    try {
      tx = db.transaction("turns", mode, {
        durability: mode === "readwrite" ? "strict" : "default",
      });
    } catch {
      tx = db.transaction("turns", mode);
    }
    let value: T;
    const timeout = setTimeout(() => {
      tx.abort();
    }, 10_000);
    tx.oncomplete = () => {
      clearTimeout(timeout);
      resolve(value);
      if (mode === "readwrite") notifyEvidenceChanged();
    };
    const failure = () => {
      clearTimeout(timeout);
      reject(tx.error ?? new Error("Could not save unsent work."));
    };
    tx.onerror = failure;
    tx.onabort = failure;
    try {
      run(tx.objectStore("turns"), (result) => {
        value = result;
      });
    } catch (error) {
      tx.abort();
      reject(error);
    }
  });
}
function checked(rows: unknown[]): EvidenceRecord[] {
  if (rows.some((row) => !validEvidenceRecord(row)))
    throw new Error(
      "Saved work needs review. Do not clear browser data until it is sent.",
    );
  return rows as EvidenceRecord[];
}
export function observeEvidenceChanges(listener: () => void): () => void {
  window.addEventListener(changes, listener);
  let channel: BroadcastChannel | null = null;
  try {
    if (typeof BroadcastChannel !== "undefined") {
      channel = new BroadcastChannel(changes);
      channel.onmessage = listener;
    }
  } catch {
    /* Notifications are optional; the storage transaction is authoritative. */
  }
  return () => {
    window.removeEventListener(changes, listener);
    channel?.close();
  };
}
function notifyEvidenceChanged() {
  window.dispatchEvent(new Event(changes));
  try {
    if (typeof BroadcastChannel !== "undefined") {
      const channel = new BroadcastChannel(changes);
      channel.postMessage("changed");
      channel.close();
    }
  } catch {
    /* A blocked notification channel must not fail a durable write. */
  }
}
export function readTutorOutbox(
  scope: EvidenceScope,
): Promise<EvidenceRecord[]> {
  return transaction("readonly", (store, result) => {
    const request = store.index("scope").getAll(evidenceScopeKey(scope));
    request.onsuccess = () => {
      try {
        result(checked(request.result));
      } catch {
        request.transaction?.abort();
      }
    };
  });
}
export function countActorEvidence(actorId: string): Promise<number> {
  return transaction("readonly", (store, result) => {
    const request = store.index("actor").count(actorId);
    request.onsuccess = () => result(request.result);
  });
}
export function queueTutorEvidence(
  scope: EvidenceScope,
  questionId: string,
  evidence: TutorEvidence,
): Promise<void> {
  const candidate = newEvidenceRecord(scope, questionId, evidence);
  return transaction("readwrite", (store, result) => {
    const request = store.getAll(undefined, EVIDENCE_LIMITS.global + 1);
    request.onsuccess = () => {
      try {
        const plan = planEvidenceEnqueue(checked(request.result), candidate);
        if (!plan.allowed)
          throw new Error(
            plan.reason === "full"
              ? "Unsent work is full. Reconnect and retry before answering more."
              : "Could not save this response. Retry before answering more.",
          );
        if (plan.reason === "duplicate") {
          result(undefined);
          return;
        }
        const write = store.add(candidate);
        write.onsuccess = () => result(undefined);
      } catch {
        store.transaction.abort();
      }
    };
  });
}
export type EvidenceClaim =
  | { kind: "empty" }
  | { kind: "wait"; until: number }
  | { kind: "paused" }
  | { kind: "record"; record: EvidenceRecord };
export function claimEvidence(
  scope: EvidenceScope,
  owner: string,
  now = Date.now(),
): Promise<EvidenceClaim> {
  return transaction("readwrite", (store, result) => {
    const request = store.index("scope").getAll(evidenceScopeKey(scope));
    request.onsuccess = () => {
      try {
        const row = checked(request.result)[0];
        if (!row) {
          result({ kind: "empty" });
          return;
        }
        if (row.blocked || row.attempts >= EVIDENCE_LIMITS.automaticAttempts) {
          result({ kind: "paused" });
          return;
        }
        const until = Math.max(
          row.retryAt,
          row.leaseOwner && row.leaseOwner !== owner
            ? (row.leaseUntil ?? 0)
            : 0,
        );
        if (until > now) {
          result({ kind: "wait", until });
          return;
        }
        const leased = { ...row, leaseOwner: owner, leaseUntil: now + 65_000 };
        store.put(leased);
        result({ kind: "record", record: leased });
      } catch {
        store.transaction.abort();
      }
    };
  });
}
export function acknowledgeTutorEvidence(
  scope: EvidenceScope,
  eventId: string,
): Promise<void> {
  return transaction("readwrite", (store, result) => {
    const request = store
      .index("event")
      .get(evidenceScopeKey(scope) + ":" + eventId);
    request.onsuccess = () => {
      const row = request.result as EvidenceRecord | undefined;
      if (row) store.delete(row.order!);
      result(undefined);
    };
  });
}
export function failEvidence(
  record: EvidenceRecord,
  owner: string,
  blockReason: EvidenceBlockReason | undefined,
  retryAfter = 0,
): Promise<void> {
  return transaction("readwrite", (store, result) => {
    const request = store.get(record.order!);
    request.onsuccess = () => {
      const current = request.result as EvidenceRecord | undefined;
      if (current?.leaseOwner === owner) {
        const attempts = current.attempts + 1;
        store.put({
          ...current,
          leaseOwner: undefined,
          leaseUntil: 0,
          attempts,
          retryAt: evidenceRetryAt(attempts, Date.now(), retryAfter),
          blocked: !!blockReason,
          blockReason,
        });
      }
      result(undefined);
    };
  });
}
export function retryTutorEvidence(scope: EvidenceScope): Promise<void> {
  return transaction("readwrite", (store, result) => {
    const request = store.index("scope").getAll(evidenceScopeKey(scope));
    request.onsuccess = () => {
      try {
        for (const row of checked(request.result))
          if (!row.blocked && (!row.leaseUntil || row.leaseUntil < Date.now()))
            store.put({
              ...row,
              attempts: 0,
              retryAt: 0,
              blocked: false,
              leaseOwner: undefined,
              leaseUntil: 0,
            });
        result(undefined);
      } catch {
        store.transaction.abort();
      }
    };
  });
}
export function deleteActorEvidence(actorId: string): Promise<void> {
  return transaction("readwrite", (store, result) => {
    const request = store.index("actor").openCursor(actorId);
    request.onsuccess = () => {
      const cursor = request.result;
      if (cursor) {
        cursor.delete();
        cursor.continue();
      } else result(undefined);
    };
  });
}
// Legacy records lack full scope. Import only after the API confirmed this exact attempt.
export async function migrateLegacyEvidence(
  scope: EvidenceScope,
  knownQuestionIds: ReadonlySet<string>,
): Promise<void> {
  const key = "classgame:tutor-outbox:" + scope.attemptId;
  const raw = sessionStorage.getItem(key);
  if (!raw) return;
  const rows: unknown = JSON.parse(raw);
  if (!Array.isArray(rows)) throw new Error("Older unsent work needs review.");
  for (const item of rows) {
    if (!item || typeof item !== "object")
      throw new Error("Older unsent work needs review.");
    const row = item as { questionId: string; evidence: TutorEvidence };
    if (!row.evidence || !knownQuestionIds.has(row.evidence.question_id))
      throw new Error(
        "Older unsent work belongs to another mission version. Keep this tab open for review.",
      );
    await queueTutorEvidence(scope, row.questionId, row.evidence);
  }
  sessionStorage.removeItem(key);
}
