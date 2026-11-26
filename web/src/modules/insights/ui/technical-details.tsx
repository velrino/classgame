import type { StudentLearningAttemptDto, StudentTutorTurnDto } from "@/modules/learning";
import { formatDate } from "@/lib/date";

function duration(value?: number) {
  return value === undefined ? undefined : `${value.toLocaleString()} ms`;
}

function Fields({ rows }: { rows: [string, string | number | undefined][] }) {
  return <dl className="grid grid-cols-2 gap-3 sm:grid-cols-3">
    {rows.map(([label, value]) => <div key={label} className="min-w-0">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="break-all text-sm font-medium tabular-nums">{value ?? "—"}</dd>
    </div>)}
  </dl>;
}

function Usage({ usage }: { usage?: StudentTutorTurnDto["tutor"]["usage"] }) {
  if (!usage?.total_token_count) return <p className="text-xs text-muted-foreground">No token usage recorded.</p>;
  return <Fields rows={[
    ["Input tokens", usage.prompt_token_count], ["Output tokens", usage.candidates_token_count],
    ["Thinking tokens", usage.thoughts_token_count], ["Total tokens", usage.total_token_count],
    ["Provider time", duration(usage.provider_duration_ms)], ["Estimated cost (USD)", usage.estimated_cost_usd],
  ]} />;
}

export function TechnicalDetails({ attempt }: { attempt: StudentLearningAttemptDto }) {
  const turns = attempt.tutor_turns ?? [];
  const measuredTurns = turns.filter(turn => (turn.tutor?.usage?.total_token_count ?? 0) > 0);
  const tutorTokens = measuredTurns.reduce((sum, turn) => sum + (turn.tutor.usage?.total_token_count ?? 0), 0);
  return <section className="grid gap-4">
    <Fields rows={[
      ["Attempt ID", attempt.id], ["Mission ID", attempt.mission_id], ["Published revision", attempt.published_revision],
      ["Started", formatDate(attempt.started_at, { preset: "date-time" })],
      ["Last activity", formatDate(attempt.last_activity_at, { preset: "date-time" })], ["Trace status", attempt.trace_state],
    ]} />
    <div>
      <h3 className="font-semibold">Tutor turns</h3>
      {measuredTurns.length > 0 && <p className="text-sm text-muted-foreground">{tutorTokens.toLocaleString()} reported tokens across {measuredTurns.length} of {turns.length} turns.</p>}
    </div>
    {!attempt.tutor_turns?.length && <p className="text-sm text-muted-foreground">No tutor turns recorded yet.</p>}
    {attempt.tutor_turns?.map(turn => <article key={turn.id} className="grid gap-4 rounded-lg border p-4">
      <h4 className="font-semibold">Turn {turn.sequence} · {turn.type}</h4>
      <Fields rows={[
        ["Reply source", turn.tutor?.generated_by_ai ? "AI" : "Observed / fallback"], ["Turn ID", turn.id], ["Question ID", turn.question_id || undefined], ["Recorded", formatDate(turn.created_at, { preset: "date-time" })],
        ["Provider", turn.tutor?.provider], ["Model", turn.tutor?.model_version || turn.tutor?.model || undefined],
        ["Request ID", turn.timing?.request_id || undefined], ["Trace ID", turn.timing?.trace_id || undefined],
        ["Passage retrieval", duration(turn.timing?.retrieval_ms)], ["Tutor workflow", duration(turn.timing?.workflow_ms)],
        ["Service time before save", duration(turn.timing?.before_save_ms)],
      ]} />
      <Usage usage={turn.tutor?.usage} />
    </article>)}
    {attempt.trace?.finalized && <article className="grid gap-3 rounded-lg border p-4">
      <h3 className="font-semibold">Learning trace</h3>
      <Fields rows={[["Provider", attempt.trace.provider], ["Model", attempt.trace.model || undefined], ["Evidence events", attempt.trace.event_count]]} />
      <Usage usage={attempt.trace.usage} />
    </article>}
  </section>;
}
