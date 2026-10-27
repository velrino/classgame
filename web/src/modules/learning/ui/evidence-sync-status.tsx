"use client";
import type { EvidenceSyncState } from "../adapters/hooks/use-evidence-sync";
export function EvidenceSyncStatus({
  state,
  completionPending,
  onRetry,
  onDelete,
}: {
  readonly state: EvidenceSyncState;
  readonly completionPending: boolean;
  readonly onRetry: () => void;
  readonly onDelete: () => void;
}) {
  if (
    !state.actorPending &&
    !state.pending &&
    !state.message &&
    !state.blocked &&
    !state.accepting &&
    !completionPending
  )
    return null;
  const message = state.accepting
    ? "Saving your response in this browser…"
    : (state.message ??
      (completionPending
        ? "Your mission is waiting for confirmation. Keep your connection on and select Retry if needed."
        : state.actorPending
          ? "Your current mission has no unsent responses."
          : "Getting ready to save your work…"));
  return (
    <section
      aria-label="Unsent work"
      className="rounded-lg border border-amber-300/30 bg-slate-950/95 px-3 py-2 text-sm text-amber-100"
    >
      <p role={state.blocked ? "alert" : "status"}>{message}</p>
      {state.actorPending > state.pending ? (
        <p className="mt-1 text-xs">
          {state.actorPending - state.pending} unsent responses from other
          attempts are kept in this browser. They are not sent to this mission.
        </p>
      ) : null}
      <div className="mt-2 flex flex-wrap items-center gap-3">
        <button
          type="button"
          disabled={state.syncing || state.accepting || state.terminal}
          onClick={onRetry}
          className="rounded border border-amber-200/50 px-3 py-1 font-semibold disabled:opacity-50"
        >
          {state.syncing ? "Sending…" : "Retry"}
        </button>
        {state.actorPending > 0 ? (
          <button
            type="button"
            disabled={state.accepting}
            onClick={onDelete}
            className="text-xs underline disabled:opacity-50"
          >
            Delete unsent work
          </button>
        ) : null}
      </div>
    </section>
  );
}
