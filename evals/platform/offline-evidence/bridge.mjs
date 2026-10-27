#!/usr/bin/env node
// Execute production TypeScript policy. Python owns validation, assertions, and reports.
import { readFileSync } from "node:fs";
import {
  newEvidenceRecord,
  planEvidenceEnqueue,
  sameEvidenceScope,
  canFinishEvidenceSync,
  evidenceRetryAt,
  evidenceBlockReason,
} from "../../../web/src/modules/learning/domain/evidence-queue.ts";


if (process.argv.length !== 2) throw new Error("The bridge accepts JSON on stdin only.");
const raw = readFileSync(0);
if (raw.length > 2 * 1024 * 1024) throw new Error("Batch exceeds 2 MiB.");
const batch = JSON.parse(raw);
if (!Array.isArray(batch.cases) || !batch.cases.length || batch.cases.length > 100) throw new Error("Invalid batch.");
const scope = { actorId: "synthetic-actor", classId: "synthetic-class", missionId: "synthetic-mission", publishedRevision: 1, attemptId: "synthetic-attempt" };
const evidence = { event_id: "synthetic-event", type: "answer", stage_id: "synthetic-stage", question_id: "synthetic-question", prompt: "What is one half?", student_response: "1/2", expected_answer: "1/2", outcome: "correct", explicit_reasoning: "", feedback: "" };
const row = (scopePatch = {}, evidencePatch = {}) => newEvidenceRecord({ ...scope, ...scopePatch }, "synthetic-local-question", { ...evidence, ...evidencePatch }, 1000);
const fields = {
  scope: ["patch"], enqueue: ["count", "distinctScopes", "responseBytes", "existing", "blocked", "scopePatch", "evidencePatch"],
  completion: ["pending", "storage_error"], retry: ["attempts", "retry_after"], block: ["status", "code"], projection: [],
};
const ids = new Set();
for (const item of batch.cases) {
  if (!item || typeof item.id !== "string" || ids.has(item.id) || item.kind !== "browser") throw new Error("Invalid case.");
  ids.add(item.id);
  const input = item.browser;
  if (!input || !Object.hasOwn(fields, input.action) || Object.keys(input).some((key) => key !== "action" && !fields[input.action].includes(key))) throw new Error("Invalid action or fields.");
  for (const key of ["count", "responseBytes", "pending", "attempts", "retry_after", "status"]) {
    if (input[key] !== undefined && (!Number.isSafeInteger(input[key]) || input[key] < 0 || input[key] > ({count: 256, responseBytes: 20000, pending: 256, attempts: 100, retry_after: 100000, status: 599}[key]))) throw new Error("Input is out of bounds.");
  }
  for (const key of ["distinctScopes", "blocked", "storage_error"]) if (input[key] !== undefined && typeof input[key] !== "boolean") throw new Error("Invalid boolean.");
  for (const key of ["patch", "scopePatch", "evidencePatch", "existing"]) if (input[key] !== undefined && (!input[key] || typeof input[key] !== "object" || Array.isArray(input[key]))) throw new Error("Invalid fixture patch.");
}
const results = batch.cases.map((item) => {
  const input = item.browser;
  let observed;
  switch (input.action) {
    case "scope":
      observed = sameEvidenceScope(scope, { ...scope, ...input.patch });
      break;
    case "enqueue": {
      const rows = Array.from({ length: input.count ?? 0 }, (_, index) => row(input.distinctScopes ? { attemptId: `synthetic-other-${index}` } : {}, { event_id: `synthetic-old-${index}`, student_response: "x".repeat(input.responseBytes ?? 3) }));
      if (input.existing) rows.push(row({}, input.existing));
      if (input.blocked) rows.push({ ...row({}, { event_id: "synthetic-blocked" }), blocked: true, blockReason: "turn-limit" });
      const candidate = row(input.scopePatch, input.evidencePatch);
      const before = JSON.stringify(rows);
      observed = { ...planEvidenceEnqueue(rows, candidate), records_unchanged: before === JSON.stringify(rows) };
      break;
    }
    case "completion":
      observed = canFinishEvidenceSync(input.pending, input.storage_error);
      break;
    case "retry":
      observed = evidenceRetryAt(input.attempts, 1000, input.retry_after);
      break;
    case "block":
      observed = evidenceBlockReason(input.status, input.code) ?? null;
      break;
    case "projection":
      observed = Object.hasOwn(row({}, { accessToken: "synthetic-never-store" }).evidence, "accessToken");
      break;
    default:
      throw new Error("Unknown case kind.");
  }
  return { id: item.id, kind: item.kind, value: observed, scripted_provider_calls: 0 };
});
console.log(JSON.stringify({ mode: "offline", real_provider_calls: 0, results }));
