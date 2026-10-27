"use client";

import { Sparkles } from "lucide-react";
import { useState } from "react";

import type { StudentClassMission } from "@/modules/classrooms";
import { type MovementRef } from "@/modules/game/adapters/types/world";
import { getAvatarPreset } from "@/modules/game/host/avatars/avatar-presets";
import { AvatarPicker, AvatarPortrait } from "@/modules/game/ui/avatar-picker";
import { MissionPicker } from "@/modules/game/ui/mission-picker";
import { MovementPad } from "@/modules/game/ui/movement-pad";
import type { PodProgress } from "@/modules/learning";
import {
  POD_ORDER,
  TUTOR_NAME,
  stageLabel,
  type GameMission,
  type PodId,
  type PodStatus,
} from "@/modules/learning";

type GameHudProps = {
  readonly mission: GameMission;
  readonly missionActive: boolean;
  readonly className: string;
  readonly missions: readonly StudentClassMission[];
  readonly completedMissionKeys: ReadonlySet<string>;
  readonly pendingMissionCount: number;
  readonly missionsReady: boolean;
  readonly startingMissionID: string;
  readonly missionStartError: string | null;
  readonly movementRef: MovementRef;
  readonly progress: Record<PodId, PodProgress>;
  readonly tutorPod: PodId;
  readonly tutorLine: string;
  readonly nearSeat: boolean;
  readonly avatarPresetId: string;
  readonly musicEnabled: boolean;
  readonly musicWaitingForGesture: boolean;
  readonly onSelectAvatar: (presetId: string) => void;
  readonly onToggleMusic: (enabled: boolean) => void;
  readonly onSelectMission: (missionID: string) => Promise<boolean>;
  readonly onEndSession: () => void;
};

const SEGMENT_CLASS: Record<PodStatus, string> = {
  locked: "bg-white/12",
  ready: "bg-cyan-300 shadow-[0_0_12px_rgba(103,232,249,0.9)]",
  review: "bg-amber-300 shadow-[0_0_12px_rgba(252,211,77,0.9)]",
  complete: "bg-lime-300 shadow-[0_0_10px_rgba(190,242,100,0.7)]",
};

/** Three short bars: the mission at a glance. Also used on the chalkboard header. */
export function PodDots({
  mission,
  progress,
  className = "",
}: {
  readonly mission?: GameMission;
  readonly progress: Record<PodId, PodProgress>;
  readonly className?: string;
}) {
  return (
    <div
      className={`flex items-center gap-1 ${className}`}
      aria-label="Mission progress"
      role="img"
    >
      {POD_ORDER.map((podId) => (
        <span
          key={podId}
          title={`${mission ? stageLabel(mission, podId) : podId} · ${progress[podId].status}`}
          className={`h-1.5 w-5 rounded-full transition-colors duration-500 ${SEGMENT_CLASS[progress[podId].status]}`}
        />
      ))}
    </div>
  );
}

const pill =
  "pointer-events-auto font-display inline-flex h-12 select-none items-center justify-center gap-2 rounded-full px-5 text-center text-[0.95rem] font-semibold tracking-tight transition active:translate-y-px disabled:opacity-60";

/**
 * Roaming HUD. Four things, in the corners, and nothing in the middle:
 * where you are in the mission, the tutor's notebook, what Mia is saying,
 * and the one action you can take right now.
 */
export function GameHud({
  mission,
  missionActive,
  className,
  missions,
  completedMissionKeys,
  pendingMissionCount,
  missionsReady,
  startingMissionID,
  missionStartError,
  movementRef,
  progress,
  tutorPod,
  tutorLine,
  nearSeat,
  avatarPresetId,
  musicEnabled,
  musicWaitingForGesture,
  onSelectAvatar,
  onToggleMusic,
  onSelectMission,
  onEndSession,
}: GameHudProps) {
  const [pickerOpen, setPickerOpen] = useState(false);
  const [missionPickerOpen, setMissionPickerOpen] = useState(false);
  const avatar = getAvatarPreset(avatarPresetId);
  const canChooseMission = nearSeat && pendingMissionCount > 0;
  const modalOpen = pickerOpen || missionPickerOpen;

  return (
    <div className="pointer-events-none absolute inset-0 z-10 select-none text-white">
      {/* Mission chip */}
      <div className="glass animate-rise absolute top-[max(env(safe-area-inset-top),0.75rem)] left-3 flex h-12 items-center gap-2.5 rounded-2xl px-3 sm:h-auto sm:gap-3 sm:px-3.5 sm:py-2.5 sm:left-5">
        {missionActive ? (
          <PodDots mission={mission} progress={progress} />
        ) : null}
        <div className="font-display leading-none">
          <p className="text-[0.56rem] font-semibold tracking-[0.12em] text-cyan-200/90 uppercase sm:text-[0.62rem] sm:tracking-[0.2em]">
            {missionActive ? mission.title : className || "Your class"}
          </p>
          <p className="mt-1 hidden text-xs font-medium text-slate-300 sm:block">
            {missionActive ? (
              <>
                Next up ·{" "}
                <span className="text-white">
                  {stageLabel(mission, tutorPod)}
                </span>
              </>
            ) : missionsReady ? (
              <span className="text-white">
                {pendingMissionCount === 0
                  ? "Waiting for a new mission"
                  : `${pendingMissionCount} mission${pendingMissionCount === 1 ? "" : "s"} ready`}
              </span>
            ) : (
              "Loading missions…"
            )}
          </p>
        </div>
      </div>

      {/* Top-right: the student's character. Teacher-only memory stays off this screen. */}
      <div className="absolute top-[max(env(safe-area-inset-top),0.75rem)] right-3 flex items-center gap-2 sm:right-5">
        <button
          type="button"
          onClick={() => setPickerOpen((open) => !open)}
          aria-expanded={pickerOpen}
          aria-label={`Open player menu (currently ${avatar.label})`}
          className="glass animate-rise pointer-events-auto flex h-12 items-center gap-2 rounded-full px-1.5 text-slate-100 transition hover:bg-white/10 active:translate-y-px sm:pr-3.5"
        >
          <AvatarPortrait preset={avatar} className="size-9" />
          <span className="font-display hidden text-sm font-semibold sm:inline">
            {avatar.label}
          </span>
          <svg
            viewBox="0 0 24 24"
            className={`hidden size-4 text-slate-400 transition sm:block ${pickerOpen ? "rotate-180" : ""}`}
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden
          >
            <path d="m6 9 6 6 6-6" />
          </svg>
        </button>
      </div>

      {pickerOpen ? (
        <AvatarPicker
          selectedAvatarId={avatarPresetId}
          musicEnabled={musicEnabled}
          musicWaitingForGesture={musicWaitingForGesture}
          onSelect={(presetId) => {
            onSelectAvatar(presetId);
            setPickerOpen(false);
          }}
          onToggleMusic={onToggleMusic}
          onClose={() => setPickerOpen(false)}
          onLogout={onEndSession}
        />
      ) : null}

      {/* Mia speaks */}
      {missionActive && tutorLine ? (
        <div className="absolute bottom-[calc(190px+4rem+env(safe-area-inset-bottom))] left-3 right-3 sm:right-auto sm:bottom-7 sm:left-6 sm:max-w-md">
          <div className="glass animate-rise flex items-start gap-3 rounded-2xl px-4 py-3">
            <span className="mt-0.5 grid size-10 shrink-0 place-items-center rounded-full border border-cyan-200/40 bg-cyan-300/15 text-cyan-100 shadow-[0_0_18px_rgba(103,232,249,0.45)]">
              <Sparkles className="size-5" aria-hidden />
            </span>
            <div className="min-w-0">
              <p className="font-display text-[0.62rem] font-semibold tracking-[0.2em] text-cyan-200/90 uppercase">
                {TUTOR_NAME}
              </p>
              <p className="mt-0.5 text-[0.95rem] leading-snug text-slate-100">
                {tutorLine}
              </p>
            </div>
          </div>
        </div>
      ) : null}

      {/* The one action */}
      <div className="absolute inset-x-3 bottom-[calc(5.25rem+env(safe-area-inset-bottom))] z-20 flex justify-center sm:inset-x-auto sm:right-6 sm:bottom-7 sm:items-end">
        {!modalOpen && nearSeat && pendingMissionCount > 0 ? (
          <button
            type="button"
            disabled={!canChooseMission}
            onClick={() => canChooseMission && setMissionPickerOpen(true)}
            className={`${pill} w-full max-w-[18rem] animate-rise sm:w-auto sm:max-w-none ${canChooseMission ? "cta hover:brightness-105" : "glass text-slate-200"}`}
          >
            START
          </button>
        ) : null}
      </div>

      {/* Controls hint, fades on its own */}
      {!modalOpen ? (
        <>
          <p className="font-display animate-fade-out absolute inset-x-0 bottom-[calc(4.5rem+env(safe-area-inset-bottom))] text-center text-[0.7rem] font-medium tracking-wide text-white/60 sm:hidden">
            Drag here to walk · swipe the room to look around
          </p>
          <p className="font-display animate-fade-out absolute inset-x-0 top-[max(env(safe-area-inset-top),1.25rem)] hidden text-center text-xs font-medium tracking-wide text-white/50 sm:block">
            WASD to walk · drag to look around · click the glowing desk to go
            there
          </p>
        </>
      ) : null}

      <div className="pointer-events-auto absolute inset-x-0 bottom-0 z-10 h-[190px] sm:hidden">
        <MovementPad movementRef={movementRef} />
      </div>

      <MissionPicker
        open={missionPickerOpen}
        className={className}
        missions={missions}
        completedMissionKeys={completedMissionKeys}
        missionsReady={missionsReady}
        startingMissionID={startingMissionID}
        error={missionStartError}
        onOpenChange={setMissionPickerOpen}
        onSelect={onSelectMission}
      />
    </div>
  );
}
