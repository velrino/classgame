"use client";
import { cn } from "@/lib/utils";
import type { MissionPreviewProgressDto } from "@/modules/missions/adapters/types/contracts";
import { LoaderCircle } from "lucide-react";

export function PreviewLoadingState({
  progress,
}: {
  readonly progress?: MissionPreviewProgressDto | null;
}) {
  return (
    <div className="grid min-h-72 place-items-center rounded-md border border-dashed border-border bg-muted/20 px-6 py-10 text-center">
      <div className="grid w-full max-w-xs justify-items-center gap-3">
        <LoaderCircle
          className="size-7 animate-spin text-primary"
          aria-hidden
        />
        <div>
          <p className="text-sm font-semibold text-foreground">
            Generating preview...
          </p>
          <MissionGenerationProgress progress={progress} />
        </div>
      </div>
    </div>
  );
}

export function PreviewLoadingOverlay({
  progress,
}: {
  readonly progress?: MissionPreviewProgressDto | null;
}) {
  return (
    <div className="absolute inset-0 z-10 grid place-items-center rounded-lg bg-background/80 backdrop-blur-sm">
      <div className="grid w-[min(20rem,calc(100%-2rem))] justify-items-center gap-3 rounded-md border border-border bg-background px-5 py-4 shadow-lg">
        <LoaderCircle
          className="size-6 animate-spin text-primary"
          aria-hidden
        />
        <p className="text-sm font-semibold text-foreground">
          Updating preview...
        </p>
        <MissionGenerationProgress progress={progress} compact />
      </div>
    </div>
  );
}

export function MissionGenerationProgress({
  compact = false,
  progress,
}: {
  readonly compact?: boolean;
  readonly progress?: MissionPreviewProgressDto | null;
}) {
  const percent =
    typeof progress?.percent === "number"
      ? Math.min(100, Math.max(0, progress.percent))
      : null;
  const detail =
    progress?.attachment ||
    (progress?.current && progress.total
      ? `${progress.current}/${progress.total} files`
      : "");
  const message =
    progress?.message || "Mia is reading the prompt and attachments.";

  return (
    <div className={cn("mt-1 w-full", compact ? "text-center" : "")}>
      <p className="text-xs leading-5 text-muted-foreground">{message}</p>
      {detail ? (
        <p className="mt-0.5 truncate text-xs font-medium text-muted-foreground">
          {detail}
        </p>
      ) : null}
      <div
        className={cn(
          "mt-3 h-1.5 w-full overflow-hidden rounded-full bg-muted",
          compact ? "max-w-64" : "",
        )}
        aria-hidden
      >
        <div
          className="h-full rounded-full bg-primary transition-[width] duration-300"
          style={{ width: `${percent ?? 12}%` }}
        />
      </div>
      {percent !== null ? (
        <p className="mt-1 text-xs font-medium tabular-nums text-muted-foreground">
          {percent}%
        </p>
      ) : null}
    </div>
  );
}
