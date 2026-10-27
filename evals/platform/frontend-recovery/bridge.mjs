// Read-only offline evaluation bridge. Python owns expectations and reporting.
import { registerHooks } from "node:module";
import { readFileSync, existsSync } from "node:fs";
import { fileURLToPath, pathToFileURL } from "node:url";
import { dirname, resolve } from "node:path";
const root = resolve(dirname(fileURLToPath(import.meta.url)), "../../..");
registerHooks({ resolve(specifier, context, nextResolve) {
  if (specifier.startsWith("@/modules/learning/")) return nextResolve(pathToFileURL(resolve(root, "web/src", specifier.slice(2) + ".ts")).href, context);
  if (specifier.startsWith(".") && context.parentURL?.startsWith(pathToFileURL(root + "/web/src/").href)) {
    const path = resolve(dirname(fileURLToPath(context.parentURL)), specifier + ".ts");
    if (existsSync(path)) return nextResolve(pathToFileURL(path).href, context);
  }
  return nextResolve(specifier, context);
} });
const { FALLBACK_GAME_MISSION } = await import("../../../web/src/modules/learning/domain/published-mission.ts");
const { createInitialSession, sessionReducer } = await import("../../../web/src/modules/learning/domain/session-reducer.ts");
const { createInitialMemory, repairUnansweredCompletion, normalizeStudentMemory } = await import("../../../web/src/modules/learning/domain/memory.ts");
const { decideAfterPodComplete } = await import("../../../web/src/modules/learning/domain/tutor-policy.ts");
const { memoryFromLearningAttempt, checkpointFromLearningAttempt } = await import("../../../web/src/modules/learning/adapters/checkpoint-mapper.ts");
if (process.argv.length !== 2) throw new Error("JSON batch on stdin only.");
const raw = readFileSync(0);
if (raw.length > 2 * 1024 * 1024) throw new Error("Batch exceeds 2 MiB.");
const batch = JSON.parse(raw);
if (!Array.isArray(batch.cases) || !batch.cases.length || batch.cases.length > 100) throw new Error("Invalid batch.");
const actions = new Set(["wrong-feedback-retry", "retry-correct", "repair-old-remediation", "unfinished-before-completion", "resume-correct", "resume-source-ids", "historical-complete", "duplicate-retry-pool", "nullable-checkpoint", "source-checkpoint"]);
for (const item of batch.cases) {
  const input = item.recovery;
  if (item.kind !== "recovery" || !input || typeof input !== "object" || Array.isArray(input) ||
      Object.keys(input).some((key) => !["action", "feedback"].includes(key)) ||
      !actions.has(input.action) ||
      (input.feedback !== undefined && !["helpful", "confusing", "too-easy", "too-hard"].includes(input.feedback))) {
    throw new Error("Invalid recovery input.");
  }
}
const mission = { ...FALLBACK_GAME_MISSION, id: "synthetic-recovery", publishedRevision: 1 };
let clock = 1000;
const act = (session, action) => sessionReducer(session, { ...action, now: ++clock });
function initial(selectedMission = mission, memory = null) {
  return act(createInitialSession(selectedMission), { type: "hydrate", mission: selectedMission, memory });
}
function wrongFeedback(feedback, duplicatePool = false) {
  let session = initial();
  if (duplicatePool) session = { ...session, memory: { ...session.memory, progress: { ...session.memory.progress, fractions: { ...session.memory.progress.fractions, pool: [session.memory.progress.fractions.queue[0]] } } } };
  session = act(session, { type: "sit" });
  const id = session.lesson.question.id;
  const wrong = session.lesson.question.options.findIndex((option) => !option.correct);
  session = act(session, { type: "choose", index: wrong });
  session = act(session, { type: "why", reasoning: { text: "Synthetic reasoning." } });
  for (let i = 0; i < 20 && session.lesson.phase === "explain"; i++) session = act(session, { type: "explain-next" });
  session = act(session, { type: "feedback", value: feedback });
  session = act(session, { type: "reveal-decision" });
  const decision = session.lesson.decision;
  session = act(session, { type: "continue" });
  return { session, id, decision };
}
const results = batch.cases.map((item) => {
  let observed;
  switch (item.recovery.action) {
    case "wrong-feedback-retry":
    case "duplicate-retry-pool": {
      const { session, id, decision } = wrongFeedback(item.recovery.feedback ?? "helpful", item.recovery.action === "duplicate-retry-pool");
      observed = { same_question: session.lesson?.question.id === id, pod: session.lesson?.pod, phase: session.lesson?.phase, correct: session.memory.progress.fractions.correct, stage_complete: session.memory.progress.fractions.status === "complete", clean_retry: session.lesson?.clean, queue_size: session.memory.progress.fractions.queue.length, feedback_support: session.memory.events.find((event) => event.type === "feedback")?.outcome === "support", decision: decision?.action };
      break;
    }
    case "retry-correct": {
      let { session, id } = wrongFeedback("helpful");
      const same = session.lesson?.question.id === id;
      session = act(session, { type: "choose", index: session.lesson.question.options.findIndex((option) => option.correct) });
      session = act(session, { type: "continue" });
      session = act(session, { type: "reveal-decision" });
      session = act(session, { type: "continue" });
      observed = { retried_question: same, correct: session.memory.progress.fractions.correct, stage_complete: session.memory.progress.fractions.status === "complete", next_pod: session.lesson?.pod, incorrect_kept: session.memory.events.some((event) => event.type === "answer" && event.outcome === "incorrect") };
      break;
    }
    case "repair-old-remediation": {
      const memory = createInitialMemory(mission); memory.progress.fractions = { ...memory.progress.fractions, status: "complete", queue: [], correct: 0 };
      const repaired = repairUnansweredCompletion(memory, mission);
      observed = { status: repaired.progress.fractions.status, question_restored: repaired.progress.fractions.queue[0] === mission.stages.fractions.primary[0], correct: repaired.progress.fractions.correct, original_unchanged: memory.progress.fractions.status === "complete" };
      break;
    }
    case "unfinished-before-completion": {
      const memory = createInitialMemory(mission); memory.progress["simple-equations"] = { ...memory.progress["simple-equations"], status: "complete", correct: 1 }; memory.progress["two-step-equations"] = { ...memory.progress["two-step-equations"], correct: 1 };
      observed = decideAfterPodComplete(mission, memory, "two-step-equations");
      break;
    }
    case "resume-correct": {
      const extra = { ...mission.questions[1], id: "synthetic-next-question", skill: "fractions" };
      const selected = { ...mission, questions: [...mission.questions, extra], stages: { ...mission.stages, fractions: { ...mission.stages.fractions, primary: [mission.questions[0].id, extra.id], required: 2 } } };
      const memory = createInitialMemory(selected); memory.progress.fractions = { ...memory.progress.fractions, correct: 1, queue: [extra.id] };
      let session = initial(selected, memory); session = { ...session, seatedAt: "fractions", lesson: { pod: "fractions", question: mission.questions[0], phase: "correct", clean: true, lines: [], stepIndex: 0, explainIndex: 0, wrong: null, decision: null, tutorMessage: "", whyPrompt: "" } };
      session = act(session, { type: "continue" });
      observed = { correct: session.memory.progress.fractions.correct, next_question: session.lesson?.question.id, stage_complete: session.memory.progress.fractions.status === "complete" };
      break;
    }
    case "resume-source-ids": {
      const selected = { ...mission, sourceQuestionIds: { ...mission.sourceQuestionIds, [mission.questions[0].id]: "published_question_1" } };
      const attempt = { status: "learning", snapshot: { progress: [{ stage_id: "fractions", status: "ready", queue: ["published_question_1"], pool: ["published_question_1"], correct: 0, required: 1 }], preferences: { explanation_style: "steps", difficulty_bias: 0 }, redirected: false, return_to: "", last_pod: "fractions" }, last_event_sequence: 0 };
      const memory = memoryFromLearningAttempt(attempt, selected);
      observed = { queue: memory.progress.fractions.queue, pool: memory.progress.fractions.pool };
      break;
    }
    case "nullable-checkpoint": {
      const checkpoint = { stageId: "fractions", questionId: mission.questions[0].id, phase: "question", seated: true, lines: null, stepIndex: null, explainIndex: null, tutorMessage: null, whyPrompt: null, decisionAction: null, decisionTarget: null, wrongChoice: null, wrongMisconceptionId: null, clean: true };
      let session = initial();
      session = act(session, { type: "hydrate", mission, memory: session.memory, checkpoint });
      observed = { phase: session.lesson?.phase, lines: session.lesson?.lines, step_index: session.lesson?.stepIndex, tutor_message: session.lesson?.tutorMessage, wrong: session.lesson?.wrong, decision: session.lesson?.decision };
      break;
    }
    case "source-checkpoint": {
      const selected = { ...mission, sourceQuestionIds: { ...mission.sourceQuestionIds, [mission.questions[0].id]: "published_question_1" } };
      const attempt = { snapshot: { checkpoint: { stage_id: "fractions", question_id: "published_question_1", phase: "question", seated: true, clean: true, lines: null } } };
      const checkpoint = checkpointFromLearningAttempt(attempt, selected);
      observed = { question_id: checkpoint?.questionId, lines: checkpoint?.lines, step_index: checkpoint?.stepIndex, tutor_message: checkpoint?.tutorMessage };
      break;
    }
    case "historical-complete": {
      const memory = createInitialMemory(mission); for (const pod of Object.keys(memory.progress)) memory.progress[pod] = { ...memory.progress[pod], status: "complete", correct: 0, queue: [] };
      const unchanged = normalizeStudentMemory(memory, mission);
      observed = { statuses: Object.values(unchanged.progress).map((p) => p.status), counts: Object.values(unchanged.progress).map((p) => p.correct) };
      break;
    }
  }
  return { id: item.id, kind: item.kind, ...observed, scripted_provider_calls: 0 };
});
console.log(JSON.stringify({ mode: "offline", real_provider_calls: 0, results }));
