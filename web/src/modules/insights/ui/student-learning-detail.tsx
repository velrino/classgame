"use client";

import {
  ArrowDown,
  BrainCircuit,
  CheckCircle2,
  History,
  LoaderCircle,
  MessageCircleMore,
  Route,
  Sparkles,
  Target,
} from "lucide-react";
import Image from "next/image";
import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { formatDate } from "@/lib/date";
import {
  defaultAvatarPresetId,
  getAvatarPortraitPath,
  getAvatarPreset,
  type AvatarPresetId,
} from "@/modules/game";
import type { DashboardStudentDto } from "@/modules/insights/adapters/types/contracts";
import type {
  StudentLearningAttemptDto,
  StudentLearningEventDto,
  StudentTutorTurnDto,
} from "@/modules/learning";

import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { useStudentHistory } from "../adapters/hooks/use-student-history";
import {
  MissionFlowPreview,
  getMissionFlowKindStyle,
} from "@/modules/missions";

type StudentLearningDetailProps = {
  readonly open: boolean;
  readonly student: DashboardStudentDto | null;
  readonly onOpenChange: (open: boolean) => void;
};

export function StudentLearningDetail({
  open,
  student,
  onOpenChange,
}: StudentLearningDetailProps) {
  const history = useStudentHistory(student, open);
  const [selectedAttemptId, setSelectedAttemptId] = useState(
    safeList(student?.attempts)[0]?.id ?? "",
  );
  const [activeTab, setActiveTab] = useState("conversation");
  const attempts = history.page.attempts;
  const attempt = useMemo(
    () =>
      attempts.find((item) => item.id === selectedAttemptId) ??
      attempts[0] ??
      null,
    [attempts, selectedAttemptId],
  );

  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      title={student?.student_name ?? "Student learning path"}
      description="Observable answers, explicit explanations, feedback, and tutor adaptations. This view never invents or exposes private chain-of-thought."
      size="xl"
      layout="fill"
    >
      <section aria-label="Mission history pages" className="shrink-0 border-b px-4 py-3 text-sm sm:px-6">
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="outline" size="sm" disabled={history.loading || !history.canGoBack} onClick={history.previous}>Newer history</Button>
          <Button variant="outline" size="sm" disabled={history.loading || !history.page.next_cursor} onClick={history.next}>Older history</Button>
          <Button variant="ghost" size="sm" disabled={history.loading} onClick={history.refresh}>Refresh</Button>
          <span className="text-xs text-muted-foreground">{history.loading ? "Loading history…" : `${attempts.length} attempts on this page`}</span>
        </div>
        {history.page.read_at ? <p className="mt-1 text-xs text-muted-foreground">History snapshot read at {new Date(history.page.read_at).toLocaleTimeString()}. Refresh for new work.</p> : null}
        {history.error ? <p role="alert" className="mt-2 text-destructive">{history.error}</p> : null}
      </section>
      {attempt ? (
        <Tabs
          value={activeTab}
          onValueChange={setActiveTab}
          className="min-h-0 flex-1 gap-0"
        >
          <div className="shrink-0 border-b border-border bg-muted/20 px-4 py-3 sm:px-6">
            {attempts.length > 1 ? (
              <AttemptSwitcher
                attempts={attempts}
                selectedId={attempt.id}
                onSelect={setSelectedAttemptId}
              />
            ) : null}

            <div className="flex flex-wrap items-center justify-between gap-3">
              <TabsList className="grid w-full grid-cols-4 sm:inline-flex sm:w-fit">
                <TabsTrigger value="conversation" aria-label="Conversation">
                  <MessageCircleMore aria-hidden />
                  <span className="hidden sm:inline">Conversation</span>
                </TabsTrigger>
                <TabsTrigger value="flow" aria-label="Learning flow">
                  <Route aria-hidden />
                  <span className="hidden sm:inline">Learning flow</span>
                </TabsTrigger>
                <TabsTrigger value="memory" aria-label="Memory">
                  <Sparkles aria-hidden />
                  <span className="hidden sm:inline">Memory</span>
                </TabsTrigger>
                <TabsTrigger value="summary" aria-label="Summary">
                  <BrainCircuit aria-hidden />
                  <span className="hidden sm:inline">Summary</span>
                </TabsTrigger>
              </TabsList>

              <div className="flex flex-wrap items-center gap-2">
                <AttemptStatus attempt={attempt} />
                <LiveUpdateStatus attempt={attempt} />
              </div>
            </div>
          </div>

          <TabsContent
            value="conversation"
            className="flex min-h-0 flex-1 flex-col"
          >
            <LearningConversation
              active={activeTab === "conversation"}
              attempt={attempt}
              studentName={
                student?.student_name || attempt.student_name || "Student"
              }
            />
          </TabsContent>

          <TabsContent
            value="summary"
            className="grid min-h-0 flex-1 content-start gap-5 overflow-y-auto p-4 sm:p-6"
          >
            {attempt.status === "abandoned" || attempt.status === "paused" ? (
              <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-amber-950">
                <p className="text-sm font-bold">
                  {attempt.status === "abandoned"
                    ? "Student left before finishing"
                    : "Student paused this mission"}
                </p>
                <p className="mt-2 text-sm leading-6">
                  Last checkpoint: {checkpointLabel(attempt)}. Their next visit
                  resumes from this exact step.
                </p>
              </div>
            ) : null}

            <div className="rounded-lg border border-border bg-muted/30 p-4">
              <div className="flex items-center gap-2 text-sm font-bold">
                <BrainCircuit className="size-4 text-cyan-700" aria-hidden />
                Evidence-based summary
              </div>
              <p className="mt-3 text-sm leading-6 text-muted-foreground">
                {attempt.trace?.summary ||
                  "Mia is collecting observable evidence for this attempt."}
              </p>
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <SignalPanel
                icon={CheckCircle2}
                label="Strengths"
                tone="emerald"
                values={attempt.trace?.strengths}
              />
              <SignalPanel
                icon={Target}
                label="Areas to revisit"
                tone="amber"
                values={attempt.trace?.improvement_areas}
              />
            </div>

            <AttemptProgress attempt={attempt} />
          </TabsContent>

          <TabsContent
            value="flow"
            className="flex min-h-0 flex-1 flex-col gap-3 p-4 sm:p-6"
          >
            <LearningFlowPanel
              attempt={attempt}
              studentName={
                student?.student_name || attempt.student_name || "Student"
              }
            />
          </TabsContent>

          <TabsContent
            value="memory"
            className="grid min-h-0 flex-1 content-start gap-4 overflow-y-auto p-4 sm:p-6"
          >
            <div className="rounded-lg border border-violet-200 bg-violet-50 p-4 text-violet-950">
              <div className="flex items-center gap-2 text-sm font-bold">
                <Sparkles className="size-4" aria-hidden />
                Next adaptive turn
              </div>
              <p className="mt-3 text-sm leading-6">
                {attempt.trace?.next_adaptation ||
                  "Retrieve this attempt before the next student interaction."}
              </p>
            </div>

            <div className="grid gap-3 sm:grid-cols-3">
              <MemoryStat
                label="Recorded events"
                value={String(safeList(attempt.snapshot?.events).length)}
              />
              <MemoryStat
                label="Teacher notes"
                value={String(safeList(attempt.snapshot?.notes).length)}
              />
              <MemoryStat
                label="Tutor turns"
                value={String(safeList(attempt.tutor_turns).length)}
              />
            </div>

            <div className="rounded-lg border border-border p-4 text-sm leading-6 text-muted-foreground">
              <p className="font-semibold text-foreground">Trace provenance</p>
              <p className="mt-2">
                {attempt.trace?.generated_by_ai
                  ? `Generated by ${attempt.trace.model || "Gemini"} from ${attempt.trace.event_count} observable events.`
                  : `Built directly from ${attempt.trace?.event_count ?? 0} observable events${attempt.trace?.fallback_reason ? ` · ${attempt.trace.fallback_reason}` : ""}.`}
              </p>
              <p className="mt-1">
                Last activity:{" "}
                {formatDate(attempt.last_activity_at, {
                  fallback: "Not available",
                  preset: "date-time",
                  relative: true,
                })}
              </p>
              <p className="mt-1">
                Processing state: {attempt.trace_state || "live"}
              </p>
            </div>
          </TabsContent>
        </Tabs>
      ) : (
        <div className="grid flex-1 place-items-center p-6 text-center">
          <div className="max-w-sm">
            <span className="mx-auto grid size-12 place-items-center rounded-full bg-muted">
              <History className="size-5 text-muted-foreground" aria-hidden />
            </span>
            <p className="mt-4 font-semibold">{history.loading ? "Loading mission history…" : history.error ? "Mission history is unavailable" : "No recorded attempt yet"}</p>
            <p className="mt-2 text-sm leading-6 text-muted-foreground">
              {history.loading ? "Your saved work is being loaded." : history.error ? "Select Refresh to try again. No saved work was removed." : "The learning flow appears after this student starts answering inside the mission."}
            </p>
          </div>
        </div>
      )}
    </Modal>
  );
}

function AttemptSwitcher({
  attempts,
  selectedId,
  onSelect,
}: {
  readonly attempts: readonly StudentLearningAttemptDto[];
  readonly selectedId: string;
  readonly onSelect: (id: string) => void;
}) {
  return (
    <div className="mb-3 flex items-center gap-2 overflow-x-auto pb-1">
      <span className="shrink-0 text-xs font-semibold text-muted-foreground">
        Attempts
      </span>
      {attempts.map((item, index) => {
        const selected = item.id === selectedId;

        return (
          <button
            key={item.id}
            type="button"
            aria-pressed={selected}
            onClick={() => onSelect(item.id)}
            className={`shrink-0 rounded-full border px-3 py-1 text-xs transition ${
              selected
                ? "border-cyan-300 bg-cyan-50 font-bold text-cyan-950"
                : "border-border bg-background text-muted-foreground hover:bg-muted"
            }`}
          >
            #{attempts.length - index} · {item.subject || "Mission"} · rev.{" "}
            {item.published_revision}
          </button>
        );
      })}
    </div>
  );
}

/**
 * The backend always ships a flow, but until the student answers it is just the fixed
 * start/memory bookends. Only draw it once at least one node is backed by a recorded event.
 */
function LearningFlowPanel({
  attempt,
  studentName,
}: {
  readonly attempt: StudentLearningAttemptDto;
  readonly studentName: string;
}) {
  const flow = attempt.trace?.flow;
  const nodes = safeList(flow?.nodes);
  const evidenceNodeCount = nodes.filter(
    (node) => safeList(node.evidence_event_ids).length > 0,
  ).length;
  const kinds = [...new Set(nodes.map((node) => node.kind))];

  if (evidenceNodeCount === 0) {
    // Only the start bookend is real so far; the memory bookend would just be filler.
    const startNode = nodes.find((node) => node.kind === "start") ?? {
      id: "attempt-start",
      kind: "start",
      label: "Mission attempt starts",
      description: `Mia begins collecting ${studentName}'s observable answers, feedback, and progress.`,
      evidence_event_ids: [],
      position: { x: 0, y: 0 },
    };

    return (
      <>
        <FlowProvenance kinds={[startNode.kind]}>
          Waiting for {studentName}&apos;s first recorded answer. Only captured
          evidence, nothing inferred.
        </FlowProvenance>
        <MissionFlowPreview
          className="h-auto min-h-0 flex-1"
          orientation="horizontal"
          flow={{ nodes: [startNode], edges: [] }}
          pending={{
            label: "Listening for the first answer",
            description: `The path fills in as ${studentName} answers inside the mission.`,
          }}
        />
      </>
    );
  }

  const provenance = attempt.trace?.generated_by_ai
    ? `${attempt.trace.model || "Gemini"} organized ${attempt.trace.event_count} recorded interactions into this path.`
    : `Drawn live from ${evidenceNodeCount} recorded ${evidenceNodeCount === 1 ? "interaction" : "interactions"}. Only captured evidence, nothing inferred.`;

  return (
    <>
      <FlowProvenance kinds={kinds}>{provenance}</FlowProvenance>
      <MissionFlowPreview
        className="h-auto min-h-0 flex-1"
        orientation="horizontal"
        flow={flow}
      />
    </>
  );
}

function FlowProvenance({
  children,
  kinds,
}: {
  readonly children: ReactNode;
  readonly kinds: readonly string[];
}) {
  return (
    <div className="flex shrink-0 flex-wrap items-center justify-between gap-3 rounded-md border border-border bg-muted/30 px-3 py-2.5 text-sm">
      <p className="flex items-center gap-2 leading-6 text-muted-foreground">
        <Route className="size-4 shrink-0 text-cyan-700" aria-hidden />
        <span>{children}</span>
      </p>
      <ul className="flex flex-wrap gap-1.5" aria-label="Node types">
        {kinds.map((kind) => (
          <li
            key={kind}
            style={getMissionFlowKindStyle(kind)}
            className="rounded-full px-2 py-0.5 text-[0.62rem] font-black tracking-[0.12em] text-slate-700 uppercase"
          >
            {kind}
          </li>
        ))}
      </ul>
    </div>
  );
}

function AttemptStatus({
  attempt,
}: {
  readonly attempt: StudentLearningAttemptDto;
}) {
  const completed = attempt.status === "completed";
  const abandoned = attempt.status === "abandoned";
  const paused = attempt.status === "paused";

  return (
    <span
      className={`rounded-md px-2.5 py-1.5 text-xs font-bold ${
        completed
          ? "bg-emerald-100 text-emerald-800"
          : abandoned
            ? "bg-rose-100 text-rose-800"
            : paused
              ? "bg-amber-100 text-amber-800"
              : "bg-primary/10 text-primary"
      }`}
    >
      {completed
        ? "Completed"
        : abandoned
          ? "Left early"
          : paused
            ? "Paused"
            : "In progress"}
    </span>
  );
}

function LiveUpdateStatus({
  attempt,
}: {
  readonly attempt: StudentLearningAttemptDto;
}) {
  const processing = isInsightProcessing(attempt.trace_state);
  const failed = attempt.trace_state === "failed";
  const active = !["completed", "abandoned", "paused"].includes(attempt.status);
  const ready = attempt.trace_state === "ready";
  const label = processing
    ? "Updating insights"
    : failed
      ? "Insight update failed"
      : active
        ? "Live updates"
        : ready
          ? "Insights ready"
          : "Watching updates";
  const tone = failed
    ? "border-rose-200 bg-rose-50 text-rose-700"
    : ready && !active
      ? "border-emerald-200 bg-emerald-50 text-emerald-700"
      : "border-primary/20 bg-primary/10 text-primary";

  return (
    <span
      className={`inline-flex items-center gap-2 rounded-md border px-2.5 py-1.5 text-xs font-bold ${tone}`}
      aria-live="polite"
    >
      {processing ? (
        <LoaderCircle className="size-3.5 animate-spin" aria-hidden />
      ) : ready && !active ? (
        <CheckCircle2 className="size-3.5" aria-hidden />
      ) : (
        <span
          className={`size-2 rounded-full ${failed ? "bg-rose-500" : "animate-pulse bg-primary"}`}
          aria-hidden
        />
      )}
      {label}
    </span>
  );
}

function SignalPanel({
  icon: Icon,
  label,
  tone,
  values,
}: {
  readonly icon: typeof CheckCircle2;
  readonly label: string;
  readonly tone: "amber" | "emerald";
  readonly values: readonly string[] | null | undefined;
}) {
  const toneClass =
    tone === "emerald"
      ? "border-emerald-200 bg-emerald-50/60 text-emerald-950"
      : "border-amber-200 bg-amber-50/60 text-amber-950";
  const safeValues = safeList(values);

  return (
    <div className={`rounded-lg border p-4 ${toneClass}`}>
      <div className="flex items-center gap-2 text-sm font-bold">
        <Icon className="size-4" aria-hidden />
        {label}
      </div>
      <ul className="mt-3 grid gap-2 text-sm leading-6">
        {(safeValues.length > 0
          ? safeValues
          : ["More evidence is needed."]
        ).map((value) => (
          <li key={value} className="flex gap-2">
            <span aria-hidden>•</span>
            <span>{value}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function AttemptProgress({
  attempt,
}: {
  readonly attempt: StudentLearningAttemptDto;
}) {
  return (
    <div className="grid gap-3 sm:grid-cols-3">
      {safeList(attempt.snapshot?.progress).map((stage) => {
        const percentage =
          stage.required > 0
            ? Math.min(100, Math.round((stage.correct / stage.required) * 100))
            : stage.status === "complete"
              ? 100
              : 0;

        return (
          <div
            key={stage.stage_id}
            className="rounded-lg border border-border p-4"
          >
            <p className="text-xs font-bold tracking-wide text-muted-foreground uppercase">
              {stage.stage_id.replaceAll("-", " ")}
            </p>
            <div className="mt-3 flex items-end justify-between gap-3">
              <span className="text-2xl font-black">{percentage}%</span>
              <span className="text-xs text-muted-foreground">
                {stage.correct}/{stage.required}
              </span>
            </div>
            <div className="mt-3 h-2 overflow-hidden rounded-full bg-muted">
              <div
                className="h-full rounded-full bg-cyan-500"
                style={{ width: `${percentage}%` }}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
}

function LearningConversation({
  active,
  attempt,
  studentName,
}: {
  readonly active: boolean;
  readonly attempt: StudentLearningAttemptDto;
  readonly studentName: string;
}) {
  const viewportRef = useRef<HTMLDivElement>(null);
  const previousVersionRef = useRef("");
  const pinnedToBottomRef = useRef(true);
  const [hasUnseenMessages, setHasUnseenMessages] = useState(false);
  const turns = [...safeList(attempt.tutor_turns)].sort(
    (left, right) => left.sequence - right.sequence,
  );
  const canHavePendingTurn =
    (attempt.trace_state || "live") === "live" &&
    !["completed", "abandoned", "paused"].includes(attempt.status);
  const pendingEvent = canHavePendingTurn
    ? latestPendingEvent(attempt.snapshot?.events, turns)
    : null;
  const processingConclusion = isInsightProcessing(attempt.trace_state);
  const conclusionReady = attempt.trace_state === "ready";
  const conclusionFailed = attempt.trace_state === "failed";
  const version = `${attempt.id}:${attempt.updated_at}:${turns.length}:${pendingEvent?.id ?? "none"}:${attempt.trace_state}`;
  const hasMessages =
    turns.length > 0 ||
    pendingEvent !== null ||
    processingConclusion ||
    conclusionReady ||
    conclusionFailed;

  useEffect(() => {
    if (!active) {
      return;
    }

    const previousVersion = previousVersionRef.current;
    const isFreshAttempt = !previousVersion.startsWith(`${attempt.id}:`);
    previousVersionRef.current = version;

    if (isFreshAttempt) {
      pinnedToBottomRef.current = true;
    }

    // The teacher scrolled up to read: don't yank them down, just flag it.
    if (!pinnedToBottomRef.current) {
      setHasUnseenMessages(true);
      return;
    }

    const frame = window.requestAnimationFrame(() => {
      viewportRef.current?.scrollTo({
        top: viewportRef.current.scrollHeight,
        behavior: isFreshAttempt ? "auto" : "smooth",
      });
    });

    return () => window.cancelAnimationFrame(frame);
  }, [active, attempt.id, version]);

  function handleViewportScroll() {
    const viewport = viewportRef.current;
    if (!viewport) {
      return;
    }

    pinnedToBottomRef.current = isNearBottom(viewport);
    if (pinnedToBottomRef.current) {
      setHasUnseenMessages(false);
    }
  }

  function jumpToLatest() {
    pinnedToBottomRef.current = true;
    setHasUnseenMessages(false);
    viewportRef.current?.scrollTo({
      top: viewportRef.current.scrollHeight,
      behavior: "smooth",
    });
  }

  return (
    <section className="flex min-h-0 flex-1 flex-col bg-muted/30">
      <header className="flex shrink-0 items-center gap-2 border-b border-border bg-background px-4 py-3 sm:px-6">
        <span className="grid size-8 shrink-0 place-items-center rounded-full bg-violet-100 text-violet-700">
          <MessageCircleMore className="size-4" aria-hidden />
        </span>
        <div className="min-w-0">
          <p className="mt-0.5 text-xs text-muted-foreground">
            New messages appear here automatically.
          </p>
        </div>
      </header>

      <div className="relative flex min-h-0 flex-1 flex-col">
        <div
          ref={viewportRef}
          onScroll={handleViewportScroll}
          className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 py-5 sm:px-6"
          aria-live="polite"
        >
          {hasMessages ? (
            <ol className="grid gap-6">
              {turns.map((turn) => (
                <TutorConversationTurn
                  key={turn.id}
                  studentName={studentName}
                  turn={turn}
                />
              ))}

              {pendingEvent ? (
                <li className="grid gap-3">
                  <MiaMessage
                    label="Mia asked"
                    text={pendingEvent.title || "Current activity"}
                  />
                  <StudentMessage
                    label={studentName}
                    outcome={pendingEvent.outcome}
                    text={pendingEvent.detail || "Answer submitted."}
                    timestamp={pendingEvent.occurred_at}
                  />
                  <MiaMessage
                    conclusionLoading
                    label="Mia is thinking"
                    loading
                    text="Reviewing the latest answer"
                  />
                </li>
              ) : null}

              {processingConclusion ? (
                <li>
                  <MiaMessage
                    conclusionLoading
                    label="Mia · Mission report"
                    loading
                    text="Reviewing all answers"
                  />
                </li>
              ) : conclusionReady ? (
                <li>
                  <MiaMessage
                    conclusion={
                      attempt.trace?.summary ||
                      "The observable learning report is ready."
                    }
                    label="Mia · Mission report"
                    text="I finished the learning report."
                  />
                </li>
              ) : conclusionFailed ? (
                <li>
                  <MiaMessage
                    conclusion="The live answers above are still available for review."
                    label="Mia · Mission report"
                    text="I could not finish the AI conclusion."
                  />
                </li>
              ) : null}
            </ol>
          ) : (
            <div className="grid h-full min-h-64 place-items-center text-center">
              <div>
                <MessageCircleMore
                  className="mx-auto size-6 text-muted-foreground"
                  aria-hidden
                />
                <p className="mt-3 text-sm font-semibold">No messages yet</p>
                <p className="mt-1 text-xs leading-5 text-muted-foreground">
                  The first answer will appear here while {studentName} plays.
                </p>
              </div>
            </div>
          )}
        </div>

        {hasUnseenMessages ? (
          <button
            type="button"
            onClick={jumpToLatest}
            className="absolute bottom-4 left-1/2 inline-flex -translate-x-1/2 items-center gap-1.5 rounded-full bg-violet-600 px-3.5 py-1.5 text-xs font-semibold text-white shadow-lg transition hover:bg-violet-700 focus-visible:ring-[3px] focus-visible:ring-violet-300 focus-visible:outline-none"
          >
            New messages
            <ArrowDown className="size-3.5" aria-hidden />
          </button>
        ) : null}
      </div>
    </section>
  );
}

function isNearBottom(viewport: HTMLElement, threshold = 48) {
  return (
    viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight <=
    threshold
  );
}

function TutorConversationTurn({
  studentName,
  turn,
}: {
  readonly studentName: string;
  readonly turn: StudentTutorTurnDto;
}) {
  const response =
    turn.explicit_reasoning ||
    turn.student_response ||
    turn.feedback ||
    "No written response.";

  return (
    <li className="grid gap-3">
      {turn.prompt ? <MiaMessage label="Mia asked" text={turn.prompt} /> : null}
      <StudentMessage
        label={studentName}
        outcome={turn.outcome}
        text={response}
        timestamp={turn.created_at}
      />
      <MiaMessage
        conclusion={turn.tutor?.teacher_observation}
        explanationSteps={turn.tutor?.explanation_steps}
        label="Mia replied"
        retrievedChunkIDs={turn.tutor?.retrieved_chunk_ids}
        text={turn.tutor?.message || "No AI reply was recorded."}
      />
    </li>
  );
}

function MiaMessage({
  conclusion,
  conclusionLoading = false,
  explanationSteps,
  label,
  loading = false,
  retrievedChunkIDs,
  text,
}: {
  readonly conclusion?: string | null;
  readonly conclusionLoading?: boolean;
  readonly explanationSteps?: readonly string[] | null;
  readonly label: string;
  readonly loading?: boolean;
  readonly retrievedChunkIDs?: readonly string[] | null;
  readonly text: string;
}) {
  return (
    <div className="flex max-w-[92%] items-end gap-2 sm:max-w-[82%]">
      <span className="grid size-8 shrink-0 place-items-center rounded-full border border-violet-200 bg-violet-100 text-violet-700">
        <Sparkles className="size-4" aria-hidden />
      </span>
      <div className="min-w-0">
        <p className="mb-1 text-xs font-semibold text-muted-foreground">
          {label}
        </p>
        <div className="rounded-e-2xl rounded-tl-2xl border border-violet-200 bg-white px-4 py-3 shadow-sm">
          {loading ? (
            <div className="flex items-center gap-2 text-sm font-medium text-violet-800">
              <LoaderCircle className="size-4 animate-spin" aria-hidden />
              <span>{text}</span>
              <span className="tracking-[0.18em]" aria-label="processing">
                ...
              </span>
            </div>
          ) : (
            <p className="text-sm leading-6 text-foreground">{text}</p>
          )}

          {safeList(explanationSteps).length > 0 ? (
            <ol className="mt-3 grid gap-1.5 border-t border-violet-100 pt-3 text-xs leading-5 text-muted-foreground">
              {safeList(explanationSteps).map((step, index) => (
                <li key={`${index}:${step}`} className="flex gap-2">
                  <span className="font-bold text-violet-600">
                    {index + 1}.
                  </span>
                  <span>{step}</span>
                </li>
              ))}
            </ol>
          ) : null}

          {conclusion || conclusionLoading ? (
            <div className="mt-3 rounded-lg border border-violet-200 bg-violet-50 px-3 py-2.5">
              <p className="text-[10px] font-bold uppercase tracking-wide text-violet-700">
                AI conclusion for the teacher
              </p>
              {conclusionLoading ? (
                <div className="mt-1.5 flex items-center gap-2 text-sm text-violet-950">
                  <LoaderCircle className="size-3.5 animate-spin" aria-hidden />
                  <span className="tracking-[0.2em]" aria-label="processing">
                    ...
                  </span>
                </div>
              ) : (
                <p className="mt-1.5 text-sm leading-6 text-violet-950">
                  {conclusion}
                </p>
              )}
            </div>
          ) : null}

          {safeList(retrievedChunkIDs).length > 0 ? (
            <div className="mt-3 flex flex-wrap gap-1.5">
              {safeList(retrievedChunkIDs).map((chunkID) => (
                <span
                  key={chunkID}
                  className="rounded bg-violet-50 px-2 py-1 text-[10px] font-semibold text-violet-700"
                >
                  RAG · {chunkID}
                </span>
              ))}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}

function StudentMessage({
  label,
  outcome,
  text,
  timestamp,
}: {
  readonly label: string;
  readonly outcome: string;
  readonly text: string;
  readonly timestamp: number | string;
}) {
  return (
    <div className="ml-auto flex max-w-[92%] flex-row-reverse items-end gap-2 sm:max-w-[82%]">
      <ChatAvatar
        presetId={defaultAvatarPresetId}
        className="border-primary/20 bg-primary/10"
      />
      <div className="min-w-0">
        <p className="mb-1 text-end text-xs font-semibold text-muted-foreground">
          {label}
        </p>
        <div className="min-w-50 rounded-s-2xl rounded-tr-2xl bg-primary px-4 py-3 text-primary-foreground shadow-sm">
          <p className="text-sm leading-6">{text}</p>
          {outcome ? (
            <span className="mt-2 inline-flex rounded-full bg-white/15 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide">
              {outcome}
            </span>
          ) : null}
          <time
            className="mt-1.5 block text-end text-[10px] whitespace-nowrap text-primary-foreground/70"
            title={formatDate(timestamp, { preset: "date-time" })}
          >
            {formatDate(timestamp, { fallback: "now", relative: true })}
          </time>
        </div>
      </div>
    </div>
  );
}

/** Same head-and-shoulders render the game spawns, so the chat matches who the student actually sees. */
function ChatAvatar({
  presetId,
  className,
}: {
  readonly presetId: AvatarPresetId;
  readonly className: string;
}) {
  return (
    <Image
      src={getAvatarPortraitPath(getAvatarPreset(presetId))}
      alt=""
      aria-hidden
      width={192}
      height={192}
      className={`size-8 shrink-0 rounded-full border object-cover ${className}`}
    />
  );
}

function latestPendingEvent(
  events: readonly StudentLearningEventDto[] | null | undefined,
  turns: readonly StudentTutorTurnDto[],
): StudentLearningEventDto | null {
  const latestEvent = [...safeList(events)]
    .filter((event) => ["answer", "reasoning", "feedback"].includes(event.type))
    .sort((left, right) => left.sequence - right.sequence)
    .at(-1);
  if (!latestEvent) {
    return null;
  }

  const coveredEventIDs = new Set(
    turns.flatMap((turn) => safeList(turn.tutor?.evidence_event_ids)),
  );
  if (coveredEventIDs.has(latestEvent.id)) {
    return null;
  }

  const latestTurnTimestamp = turns.reduce(
    (latest, turn) => Math.max(latest, dateValue(turn.created_at)),
    0,
  );
  return latestTurnTimestamp >= latestEvent.occurred_at ? null : latestEvent;
}

function MemoryStat({
  label,
  value,
}: {
  readonly label: string;
  readonly value: string;
}) {
  return (
    <div className="rounded-lg border border-border p-4">
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <p className="mt-2 font-bold capitalize">{value}</p>
    </div>
  );
}

function isInsightProcessing(traceState: string | null | undefined) {
  return traceState === "pending" || traceState === "processing";
}

function dateValue(value: number | string) {
  if (typeof value === "number") {
    return Number.isFinite(value) ? value : 0;
  }

  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

function checkpointLabel(attempt: StudentLearningAttemptDto) {
  const checkpoint = attempt.snapshot?.checkpoint;
  const stage = checkpoint?.stage_id?.replaceAll("-", " ") || "mission start";
  const phase = checkpoint?.phase?.replaceAll("-", " ") || "between questions";
  return `${stage}, ${phase}${checkpoint?.question_id ? ` (${checkpoint.question_id})` : ""}`;
}

function safeList<Value>(
  values: readonly Value[] | null | undefined,
): readonly Value[] {
  return Array.isArray(values) ? values : [];
}
