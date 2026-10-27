"use client";

import { Sparkles, Star, X } from "lucide-react";
import { useEffect, useRef, type CSSProperties } from "react";

import type { GameMission } from "@/modules/learning";

type MissionRewardProps = {
  readonly open: boolean;
  readonly mission: GameMission;
  readonly onDismiss: () => void;
};

const SPARKLES = [
  { top: "14%", left: "18%", delay: "0ms", size: "size-4" },
  { top: "20%", left: "74%", delay: "140ms", size: "size-5" },
  { top: "36%", left: "10%", delay: "260ms", size: "size-3.5" },
  { top: "56%", left: "82%", delay: "80ms", size: "size-4" },
  { top: "75%", left: "22%", delay: "220ms", size: "size-5" },
  { top: "80%", left: "68%", delay: "340ms", size: "size-3.5" },
] as const;

const REWARD_SOUND_SRC = "/assets/audio/collect-gem.mp3";
const REWARD_SOUND_VOLUME = 0.68;

export function MissionReward({
  open,
  mission,
  onDismiss,
}: MissionRewardProps) {
  const buttonRef = useRef<HTMLButtonElement>(null);
  const playedForOpenRef = useRef(false);

  useEffect(() => {
    if (!open) {
      playedForOpenRef.current = false;
      return;
    }

    buttonRef.current?.focus();
    if (!playedForOpenRef.current && typeof Audio !== "undefined") {
      playedForOpenRef.current = true;
      const rewardSound = new Audio(REWARD_SOUND_SRC);
      rewardSound.volume = REWARD_SOUND_VOLUME;
      void rewardSound.play().catch(() => undefined);
    }

    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onDismiss();
      }
    };

    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [onDismiss, open]);

  if (!open) {
    return null;
  }

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="mission-reward-title"
      className="pointer-events-auto absolute inset-0 z-40 grid place-items-center overflow-hidden bg-[#05070d]/75 px-4 py-6 text-white backdrop-blur-[3px]"
    >
      <div className="pointer-events-none absolute inset-0" aria-hidden>
        <div className="classgame-reward-rays absolute top-1/2 left-1/2" />
        {SPARKLES.map((sparkle) => (
          <Sparkles
            key={`${sparkle.left}-${sparkle.top}`}
            className={`classgame-reward-sparkle absolute ${sparkle.size} text-yellow-200`}
            style={{
              animationDelay: sparkle.delay,
              left: sparkle.left,
              top: sparkle.top,
            }}
            strokeWidth={2.4}
          />
        ))}
        {Array.from({ length: 18 }, (_, index) => (
          <span
            key={index}
            className="classgame-reward-confetti absolute top-1/2 left-1/2 h-2 w-1 rounded-full"
            style={
              {
                "--reward-rotate": `${index * 20}deg`,
                "--reward-distance": `${7 + (index % 5) * 1.35}rem`,
                animationDelay: `${index * 22}ms`,
              } as CSSProperties
            }
          />
        ))}
      </div>

      <section className="glass classgame-reward-card relative w-full max-w-md overflow-hidden rounded-2xl border-yellow-200/35 px-5 pt-8 pb-5 text-center shadow-[0_0_70px_rgba(250,204,21,0.22)] sm:px-8 sm:pt-9 sm:pb-7">
        <button
          type="button"
          onClick={onDismiss}
          aria-label="Close reward"
          className="absolute top-3 right-3 grid size-10 place-items-center rounded-full border border-white/10 bg-white/5 text-slate-200 transition hover:bg-white/10 active:translate-y-px"
        >
          <X className="size-5" aria-hidden />
        </button>

        <div className="relative mx-auto grid size-32 place-items-center sm:size-36">
          <div className="classgame-reward-ring absolute inset-0 rounded-full" />
          <div className="classgame-reward-medal relative grid size-24 place-items-center rounded-full text-yellow-950 sm:size-28">
            <Star
              className="size-14 fill-current sm:size-16"
              strokeWidth={1.8}
            />
          </div>
        </div>

        <p className="font-display mt-5 text-xs font-semibold tracking-[0.18em] text-yellow-200 uppercase">
          Reward unlocked
        </p>
        <h2
          id="mission-reward-title"
          className="font-display mt-2 text-3xl leading-tight font-black text-white sm:text-4xl"
        >
          Golden Scholar Star
        </h2>
        <p className="mx-auto mt-3 line-clamp-2 max-w-sm text-sm leading-6 text-yellow-50/85 sm:text-base">
          {mission.title}
        </p>
        <p className="mx-auto mt-2 max-w-sm text-sm leading-6 text-slate-300">
          Your progress is saved for your teacher.
        </p>

        <button
          ref={buttonRef}
          type="button"
          onClick={onDismiss}
          className="classgame-reward-button font-display mt-6 inline-flex h-12 w-full items-center justify-center rounded-full px-5 text-sm font-bold text-yellow-950 transition hover:brightness-105 active:translate-y-px sm:w-auto sm:min-w-44"
        >
          Keep exploring
        </button>
      </section>
    </div>
  );
}
