"use client";

import { CheckCircle2, Clock3, LoaderCircle, Play } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

import type { StudentClassMission } from "@/modules/classrooms";
import { studentMissionRevisionKey } from "@/modules/classrooms";

type MissionPickerProps = {
  readonly open: boolean;
  readonly className: string;
  readonly missions: readonly StudentClassMission[];
  readonly completedMissionKeys: ReadonlySet<string>;
  readonly missionsReady: boolean;
  readonly startingMissionID: string;
  readonly error: string | null;
  readonly onOpenChange: (open: boolean) => void;
  readonly onSelect: (missionID: string) => Promise<boolean>;
};

export function MissionPicker({
  open,
  className,
  missions,
  completedMissionKeys,
  missionsReady,
  startingMissionID,
  error,
  onOpenChange,
  onSelect,
}: MissionPickerProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="glass max-h-[calc(100dvh-1.5rem)] w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] grid-rows-[auto_minmax(0,1fr)] gap-0 overflow-hidden border-white/10 bg-[#090c18] p-0 text-white sm:max-h-[calc(100dvh-2rem)] sm:max-w-2xl">
        <DialogHeader className="border-b border-white/10 px-4 py-4 pr-12 text-start sm:px-5">
          <DialogTitle className="font-display text-xl">
            Choose a mission
          </DialogTitle>
          <DialogDescription className="leading-5 text-slate-400">
            {className || "Your class"} · Pick a mission you have not completed.
          </DialogDescription>
        </DialogHeader>

        <div className="min-h-0 overflow-y-auto p-3 sm:p-5">
          {!missionsReady ? (
            <div className="grid min-h-40 place-items-center text-center text-sm text-slate-400">
              <div>
                <LoaderCircle className="mx-auto mb-3 size-6 animate-spin text-cyan-300" />
                Loading missions…
              </div>
            </div>
          ) : missions.length === 0 ? (
            <div className="grid min-h-40 place-items-center rounded-2xl border border-dashed border-white/15 bg-white/[0.03] px-6 text-center">
              <div>
                <p className="font-display font-semibold text-white">
                  No missions yet
                </p>
                <p className="mt-1 text-sm leading-6 text-slate-400">
                  This list updates automatically when your teacher publishes
                  one.
                </p>
              </div>
            </div>
          ) : (
            <div className="grid min-w-0 gap-3 sm:grid-cols-2">
              {missions.map((mission) => {
                const completed = completedMissionKeys.has(
                  studentMissionRevisionKey(
                    mission.id,
                    mission.publishedRevision,
                  ),
                );
                const starting = startingMissionID === mission.id;

                return (
                  <button
                    key={`${mission.id}:${mission.publishedRevision}`}
                    type="button"
                    disabled={completed || startingMissionID !== ""}
                    onClick={() => {
                      void onSelect(mission.id).then((started) => {
                        if (started) {
                          onOpenChange(false);
                        }
                      });
                    }}
                    className={`group min-h-36 w-full min-w-0 overflow-hidden rounded-xl border p-4 text-start transition active:translate-y-px disabled:cursor-default sm:min-h-40 ${
                      completed
                        ? "border-lime-300/15 bg-lime-300/[0.04] text-slate-400"
                        : "border-white/10 bg-white/[0.04] hover:border-cyan-300/45 hover:bg-cyan-300/[0.08] hover:shadow-[0_0_24px_rgba(34,211,238,0.12)]"
                    }`}
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="font-display line-clamp-2 break-words text-[0.95rem] leading-5 font-semibold text-white sm:text-base">
                          {mission.title}
                        </p>
                        <p className="mt-1 break-words text-xs font-medium tracking-wide text-cyan-200/80 uppercase">
                          {[mission.subject, mission.gradeLevel]
                            .filter(Boolean)
                            .join(" · ") || "Class mission"}
                        </p>
                      </div>
                      {completed ? (
                        <CheckCircle2
                          className="size-5 shrink-0 text-lime-300"
                          aria-label="Completed"
                        />
                      ) : starting ? (
                        <LoaderCircle className="size-5 shrink-0 animate-spin text-cyan-300" />
                      ) : (
                        <Play className="size-5 shrink-0 text-cyan-300 transition group-hover:translate-x-0.5" />
                      )}
                    </div>

                    <p className="mt-3 line-clamp-3 break-words text-sm leading-5 text-slate-300 sm:line-clamp-2">
                      {mission.summary ||
                        mission.learningGoal ||
                        "A mission prepared by your teacher."}
                    </p>

                    <div className="mt-4 flex flex-wrap items-center justify-between gap-2 text-xs">
                      <span className="inline-flex min-w-0 items-center gap-1.5 text-slate-400">
                        <Clock3 className="size-3.5" />
                        {mission.suggestedDurationMinutes > 0
                          ? `${mission.suggestedDurationMinutes} min`
                          : "Short mission"}
                      </span>
                      <span
                        className={
                          completed
                            ? "shrink-0 font-semibold text-lime-300"
                            : "shrink-0 font-semibold text-cyan-200"
                        }
                      >
                        {completed
                          ? "Completed"
                          : starting
                            ? "Starting…"
                            : "Start"}
                      </span>
                    </div>
                  </button>
                );
              })}
            </div>
          )}

          {error ? (
            <p
              role="alert"
              className="mt-4 rounded-xl border border-red-300/20 bg-red-400/10 px-3 py-2.5 text-sm text-red-100"
            >
              {error}
            </p>
          ) : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}
