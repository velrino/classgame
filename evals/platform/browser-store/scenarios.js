import { queueTutorEvidence, readTutorOutbox, countActorEvidence, claimEvidence, acknowledgeTutorEvidence, failEvidence, retryTutorEvidence, deleteActorEvidence } from "/source/outbox.js";
import { canFinishEvidenceSync } from "/domain/evidence-queue";
const status = document.querySelector("#status"), list = document.querySelector("#results"), report = document.querySelector("#report");
const runButton = document.querySelector("#run"), reloadButton = document.querySelector("#reload");
const key = "classgame:local-browser-eval";
const peerRun = new URLSearchParams(location.search).get("peer");
const pause = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
const assert = (condition, detail) => { if (!condition) throw new Error(detail); };
const evidence = (id, patch = {}) => ({ event_id: id, type: "answer", stage_id: "synthetic-stage", question_id: "synthetic-question", prompt: "What is one half?", student_response: "1/2", expected_answer: "1/2", outcome: "correct", explicit_reasoning: "", feedback: "", ...patch });
const scope = (run, patch = {}) => ({ actorId: "eval-browser-" + run, classId: "synthetic-class", missionId: "synthetic-mission", publishedRevision: 1, attemptId: "synthetic-attempt", ...patch });
const put = (s, id, patch) => queueTutorEvidence(s, "synthetic-local-question", evidence(id, patch));
let channel, peerWindow, run, results = [];
const pending = new Map();
function connect(id, peer) {
  channel?.close(); channel = new BroadcastChannel("classgame-browser-eval:" + id);
  channel.onmessage = async ({ data }) => {
    if (data.response && pending.has(data.id)) { const request = pending.get(data.id); pending.delete(data.id); data.error ? request.reject(new Error(data.error)) : request.resolve(data.result); return; }
    if (!peer || !data.command) return;
    try {
      let result;
      switch (data.command) {
        case "ping": result = true; break;
        case "append": await Promise.all(data.ids.map((id) => put(data.scope, id))); result = true; break;
        case "claim": result = await claimEvidence(data.scope, data.owner, data.now); break;
        case "read": result = await readTutorOutbox(data.scope); break;
        case "close": channel.close(); window.close(); return;
        default: throw new Error("Unknown local command.");
      }
      channel.postMessage({ response: true, id: data.id, result });
    } catch (error) { channel.postMessage({ response: true, id: data.id, error: error.message }); }
  };
}
function peer(command, values = {}) {
  const id = crypto.randomUUID();
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => { pending.delete(id); reject(new Error("Peer tab did not answer: " + command)); }, 10000);
    pending.set(id, { resolve: (value) => { clearTimeout(timer); resolve(value); }, reject: (error) => { clearTimeout(timer); reject(error); } });
    channel.postMessage({ id, command, ...values });
  });
}
function render() {
  list.replaceChildren(...results.map((item) => { const li = document.createElement("li"); li.className = item.passed ? "pass" : "fail"; li.textContent = (item.passed ? "PASS " : "FAIL ") + item.id + (item.detail ? ": " + item.detail : ""); return li; }));
}
async function check(id, action) {
  try { await action(); results.push({ id, passed: true }); }
  catch (error) { results.push({ id, passed: false, detail: error.message }); }
  render();
}
async function clearRun() { if (run) await deleteActorEvidence(scope(run).actorId); }
if (peerRun) {
  assert(/^[a-f0-9-]{36}$/.test(peerRun), "Invalid synthetic run ID.");
  connect(peerRun, true); runButton.hidden = true; reloadButton.hidden = true; document.querySelector("#cleanup").hidden = true; status.textContent = "Peer ready. This tab handles only synthetic storage commands.";
} else {
  runButton.onclick = async () => {
    runButton.disabled = true; run = crypto.randomUUID(); results = []; connect(run, false);
    peerWindow = window.open("/?peer=" + run, "classgame-browser-eval-peer");
    status.textContent = "Checking native browser storage and the peer tab…";
    try {
      for (let n = 0; n < 20; n++) { await pause(100); try { if (await peer("ping")) break; } catch { if (n === 19) throw new Error("Open the peer tab and retry."); } }
      await clearRun(); const s = scope(run);
      await check("native-write-read-v1", async () => { await put(s, "first"); assert((await readTutorOutbox(s)).length === 1, "Missing persisted row."); });
      await check("cross-tab-exact-duplicate-v1", async () => { await Promise.all([put(s, "first"), peer("append", { scope: s, ids: ["first"] })]); assert((await readTutorOutbox(s)).length === 1, "Duplicate row was accepted."); });
      await check("changed-intent-refused-v1", async () => { let failed = false; try { await put(s, "first", { student_response: "1/3" }); } catch { failed = true; } assert(failed && (await readTutorOutbox(s))[0].evidence.student_response === "1/2", "Changed intent replaced evidence."); });
      await check("scope-isolation-v1", async () => { for (const patch of [{ actorId: s.actorId + "-other" }, { classId: "other" }, { missionId: "other" }, { publishedRevision: 2 }, { attemptId: "other" }]) assert((await readTutorOutbox({ ...s, ...patch })).length === 0, "Another scope read the row."); });
      await check("cross-tab-no-lost-writes-v1", async () => { await clearRun(); const a = Array.from({ length: 32 }, (_, n) => "left-" + n), b = Array.from({ length: 32 }, (_, n) => "right-" + n); await Promise.all([Promise.all(a.map((id) => put(s, id))), peer("append", { scope: s, ids: b })]); const rows = await readTutorOutbox(s); assert(rows.length === 64 && new Set(rows.map((r) => r.evidence.event_id)).size === 64, "Concurrent writes were lost."); });
      await check("scope-capacity-keeps-evidence-v1", async () => { let failed = false; try { await put(s, "overflow"); } catch { failed = true; } assert(failed && (await readTutorOutbox(s)).length === 64, "Capacity silently changed old records."); });
      await check("cross-tab-single-lease-v1", async () => { await clearRun(); await put(s, "leased"); const now = Date.now(), claims = await Promise.all([claimEvidence(s, "left", now), peer("claim", { scope: s, owner: "right", now })]); assert(claims.filter((c) => c.kind === "record").length === 1 && claims.filter((c) => c.kind === "wait").length === 1, "Two tabs own the same live lease."); });
      await check("old-lease-failure-fenced-v1", async () => { const old = (await readTutorOutbox(s))[0]; const next = await peer("claim", { scope: s, owner: "successor", now: old.leaseUntil + 1 }); assert(next.kind === "record", "Expired lease did not transfer."); await failEvidence(old, old.leaseOwner, "rejected"); const current = (await readTutorOutbox(s))[0]; assert(current.leaseOwner === "successor" && !current.blocked && current.attempts === 0, "Old failure overwrote a successor."); });
      await check("ack-only-one-scope-v1", async () => { await acknowledgeTutorEvidence({ ...s, attemptId: "other" }, "leased"); assert((await readTutorOutbox(s)).length === 1, "Wrong scope deleted an event."); await put(s, "second"); await acknowledgeTutorEvidence(s, "leased"); const rows = await readTutorOutbox(s); assert(rows.length === 1 && rows[0].evidence.event_id === "second", "ACK deleted another row."); });
      await check("transient-retry-keeps-id-v1", async () => { const claim = await claimEvidence(s, "retry-owner"); assert(claim.kind === "record", "Missing row."); await failEvidence(claim.record, "retry-owner", undefined, 2); let row = (await readTutorOutbox(s))[0]; assert(row.attempts === 1 && row.retryAt > Date.now() && row.evidence.event_id === "second", "Retry identity or backoff changed."); await retryTutorEvidence(s); row = (await readTutorOutbox(s))[0]; assert(row.attempts === 0 && row.retryAt === 0, "Explicit retry did not reset transient work."); });
      await check("permanent-rejection-keeps-work-v1", async () => { const claim = await claimEvidence(s, "terminal-owner"); assert(claim.kind === "record", "Missing row."); await failEvidence(claim.record, "terminal-owner", "event-conflict"); await retryTutorEvidence(s); let failed = false; try { await put(s, "after-rejection"); } catch { failed = true; } const rows = await readTutorOutbox(s); assert(failed && rows.length === 1 && rows[0].blocked, "Terminal evidence was dropped or retried."); });
      await check("credentials-not-stored-v1", async () => { await clearRun(); await put(s, "no-token", { accessToken: "synthetic-do-not-store", teacherNote: "synthetic-private" }); const json = JSON.stringify(await readTutorOutbox(s)); assert(!json.includes("synthetic-do-not-store") && !json.includes("synthetic-private"), "Unexpected fields entered storage."); });
      await check("large-record-refused-v1", async () => { let failed = false; try { await put(s, "large", { student_response: "x".repeat(20000) }); } catch { failed = true; } assert(failed && (await readTutorOutbox(s)).length === 1, "Oversize record changed saved work."); });
      await clearRun(); await put(s, "reload-check");
      localStorage.setItem(key, JSON.stringify({ run, results })); reloadButton.disabled = false;
      status.textContent = "Storage checks complete. Select Check reload recovery to reload this page and verify the saved event.";
    } catch (error) { status.textContent = "Evaluation stopped: " + error.message; runButton.disabled = false; }
  };
  reloadButton.onclick = () => location.reload();
  document.querySelector("#cleanup").onclick = async () => { const marker = JSON.parse(localStorage.getItem(key) ?? "null"); run ??= marker?.run; await clearRun(); localStorage.removeItem(key); if (channel) channel.postMessage({ command: "close", id: crypto.randomUUID() }); status.textContent = "This run's synthetic records are removed."; };
  const marker = JSON.parse(localStorage.getItem(key) ?? "null");
  if (marker?.run && /^[a-f0-9-]{36}$/.test(marker.run)) {
    run = marker.run; results = marker.results; connect(run, false); runButton.disabled = true;
    await check("native-reload-recovery-v1", async () => { const rows = await readTutorOutbox(scope(run)); assert(rows.length === 1 && rows[0].evidence.event_id === "reload-check", "Reload lost the original event."); assert(!canFinishEvidenceSync(rows.length, false), "Pending work confirmed completion."); await acknowledgeTutorEvidence(scope(run), "reload-check"); assert((await readTutorOutbox(scope(run))).length === 0, "Acknowledgment did not remove its event."); });
    await clearRun(); localStorage.removeItem(key); channel.postMessage({ command: "close", id: crypto.randomUUID() });
    const response = await fetch("/report", { method: "POST", headers: { "Content-Type": "application/json", "X-ClassGame-Local-Eval": window.localEval.token }, body: JSON.stringify({ results, cleanup_complete: (await countActorEvidence(scope(run).actorId)) === 0, browser_user_agent: navigator.userAgent }) });
    const saved = await response.json(); report.textContent = JSON.stringify(saved, null, 2); status.textContent = `${results.filter((r) => r.passed).length}/${results.length} real browser storage cases passed. Synthetic cleanup complete. Report saved.`;
  }
}
