"use client";

import { useLayoutEffect, useRef, useState } from "react";

import { INPUT_LIMITS, limitInput } from "@/shared/input-limits";

import { PodDots } from "@/modules/game/ui/game-hud";
import type { PodProgress } from "@/modules/learning";
import {
  TUTOR_NAME,
  buildWhyOptions,
  shuffledMoves,
  stageLabel,
  type Explanation,
  type ExplanationStyle,
  type FeedbackValue,
  type GameMission,
  type Lesson,
  type PodId,
  type Question,
  type Visual,
} from "@/modules/learning";

type ChalkboardProps = {
  readonly mission: GameMission;
  readonly lesson: Lesson;
  readonly progress: Record<PodId, PodProgress>;
  readonly explanationStyle: ExplanationStyle;
  readonly onChoose: (index: number) => void;
  readonly onWriteAnswer: (text: string) => void;
  readonly onStep: (index: number) => void;
  readonly onWhy: (
    reasoning: { misconceptionId: string } | { text: string },
  ) => void;
  readonly onExplainNext: () => void;
  readonly onFeedback: (value: FeedbackValue) => void;
  readonly onContinue: () => void;
  readonly onStand: () => void;
  readonly onEndSession: () => void;
};

/** Slight, deterministic tilt per option so the board reads as handwriting, not a form. */
const TILT = ["-rotate-1", "rotate-1", "rotate-[0.5deg]", "-rotate-[1.5deg]"];

/*
 * Everything on the board is sized in `em` against the board body's font-size.
 * `useFitScale` shrinks that font-size until the content fits the board with no
 * scrolling — an option hidden below the fold is an answer nobody chose on purpose.
 */
const BASE_FONT_PX = 16;
const MIN_SCALE = 0.55;
/** Tilted option buttons poke a couple of px past the layout box; that is not overflow. */
const OVERFLOW_TOLERANCE_PX = 6;

function useFitScale(
  ref: React.RefObject<HTMLElement | null>,
  deps: readonly unknown[],
) {
  useLayoutEffect(() => {
    const element = ref.current;

    if (!element) {
      return;
    }

    const fit = () => {
      let scale = 1;
      element.style.fontSize = `${BASE_FONT_PX}px`;

      while (
        element.scrollHeight > element.clientHeight + OVERFLOW_TOLERANCE_PX &&
        scale > MIN_SCALE
      ) {
        scale = Math.round((scale - 0.05) * 100) / 100;
        element.style.fontSize = `${BASE_FONT_PX * scale}px`;
      }
    };

    fit();

    const observer = new ResizeObserver(fit);
    observer.observe(element);

    return () => observer.disconnect();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
}

const chalkButton =
  "chalk-line chalk pointer-events-auto rounded-sm border-2 px-[0.8em] py-[0.5em] leading-tight transition hover:bg-white/10 active:bg-white/15 active:translate-y-px disabled:opacity-40";
const chalkPrimary = `${chalkButton} border-chalk-accent text-chalk-accent`;
const chalkHeaderButton =
  "chalk-line min-h-10 whitespace-nowrap rounded-sm border-2 px-3 py-2 text-sm font-semibold transition hover:bg-white/10 active:translate-y-px active:bg-white/15 sm:px-4 sm:text-base";

const FEEDBACK: readonly {
  value: FeedbackValue;
  label: string;
  emoji: string;
}[] = [
  { value: "helpful", label: "Helpful", emoji: "👍" },
  { value: "confusing", label: "Confusing", emoji: "😵‍💫" },
  { value: "too-easy", label: "Too easy", emoji: "🥱" },
  { value: "too-hard", label: "Too hard", emoji: "🔥" },
];

// ── visuals ─────────────────────────────────────────────────────────────────

function FractionBar({
  num,
  den,
  label,
}: {
  readonly num: number;
  readonly den: number;
  readonly label: string;
}) {
  return (
    <div className="flex items-center gap-[0.6em]">
      <span className="chalk w-[5.5em] text-right text-[1.05em]">{label}</span>
      <div className="chalk-line flex h-[1.6em] flex-1 gap-[3px] rounded-sm border-2 p-[3px]">
        {Array.from({ length: den }, (_, index) => (
          <div
            key={index}
            className={`flex-1 rounded-[2px] ${index < num ? "bg-chalk-ink/85" : "bg-white/5"}`}
          />
        ))}
      </div>
    </div>
  );
}

function ChainVisual({
  lines,
  upTo,
}: {
  readonly lines: readonly string[];
  readonly upTo: number;
}) {
  return (
    <ol className="chalk space-y-[0.2em]">
      {lines.slice(0, upTo).map((line, index) => {
        const isOp = index % 2 === 1;

        return (
          <li
            key={`${line}-${index}`}
            className={`animate-pop ${isOp ? "pl-[1.5em] text-[1em] text-chalk-accent" : "text-[1.6em]"}`}
          >
            {isOp ? `↓ ${line}` : line}
          </li>
        );
      })}
    </ol>
  );
}

function VisualExplanation({
  visual,
  upTo,
}: {
  readonly visual: Visual;
  readonly upTo: number;
}) {
  if (visual.kind === "fraction-bars") {
    return (
      <div className="space-y-[0.6em]">
        {visual.bars.slice(0, upTo).map((bar) => (
          <div key={bar.label} className="animate-pop">
            <FractionBar {...bar} />
          </div>
        ))}
      </div>
    );
  }

  // Each explanation step reveals one more "equation + move" pair.
  return (
    <ChainVisual
      lines={visual.lines}
      upTo={Math.min(visual.lines.length, upTo * 2 + 1)}
    />
  );
}

// ── phases ──────────────────────────────────────────────────────────────────

function QuestionPhase({
  lesson,
  onChoose,
  onWriteAnswer,
  onStep,
}: Pick<ChalkboardProps, "lesson" | "onChoose" | "onWriteAnswer" | "onStep">) {
  const [writing, setWriting] = useState(false);
  const [text, setText] = useState("");
  const { question } = lesson;

  function handleAnswerTextChange(rawValue: string) {
    setText(limitInput(rawValue, INPUT_LIMITS.studentAnswer));
  }

  if (question.kind === "choice") {
    return (
      <>
        <p className="chalk text-[1.9em] leading-tight">{question.prompt}</p>
        {writing ? (
          <form
            className="mt-auto flex flex-col gap-[0.6em] pt-[1em]"
            onSubmit={(event) => {
              event.preventDefault();
              if (text.trim().length > 0) {
                onWriteAnswer(text);
              }
            }}
          >
            <textarea
              autoFocus
              value={text}
              onChange={(event) => handleAnswerTextChange(event.target.value)}
              maxLength={INPUT_LIMITS.studentAnswer}
              rows={3}
              placeholder="Write your answer…"
              className="chalk chalk-line pointer-events-auto w-full resize-none rounded-sm border-2 bg-transparent px-[0.6em] py-[0.4em] text-[1.1em] outline-none placeholder:text-chalk-ink/40"
            />
            <div className="grid grid-cols-2 gap-[0.6em]">
              <button
                type="button"
                onClick={() => setWriting(false)}
                className={`${chalkButton} text-[1.05em]`}
              >
                Back
              </button>
              <button
                type="submit"
                disabled={text.trim().length === 0}
                className={`${chalkPrimary} text-[1.05em]`}
              >
                Submit answer
              </button>
            </div>
          </form>
        ) : (
          <div className="mt-auto grid gap-[0.6em] pt-[1em] pb-[3px]">
            <div className="grid grid-cols-2 gap-[0.6em]">
              {question.options.map((option, index) => (
                <button
                  key={option.text}
                  type="button"
                  onClick={() => onChoose(index)}
                  className={`${chalkButton} text-[1.4em] ${TILT[index % TILT.length]}`}
                >
                  {option.text}
                </button>
              ))}
            </div>
            <button
              type="button"
              onClick={() => setWriting(true)}
              className={`${chalkButton} text-[0.95em] text-chalk-ink/80`}
            >
              ✎ Write my answer
            </button>
          </div>
        )}
      </>
    );
  }

  const moves = shuffledMoves(
    question.id,
    lesson.stepIndex,
    question.steps[lesson.stepIndex],
  );

  return (
    <>
      <p className="chalk text-[0.85em] text-chalk-ink/70">
        Get x alone. Pick the next move.
      </p>
      <ol className="chalk mt-[0.4em] space-y-[0.15em]">
        {lesson.lines.map((line, index) => (
          <li
            key={`${line}-${index}`}
            className={`text-[1.9em] leading-tight ${index === 0 ? "" : "animate-pop"}`}
          >
            {line}
          </li>
        ))}
      </ol>
      <div className="mt-auto grid grid-cols-2 gap-[0.6em] pt-[1em] pb-[3px]">
        {moves.map((move, index) => (
          <button
            key={move.op}
            type="button"
            onClick={() => onStep(index)}
            className={`${chalkButton} text-[1.05em] ${TILT[index % TILT.length]}`}
          >
            {move.op}
          </button>
        ))}
      </div>
    </>
  );
}

function CorrectPhase({
  lesson,
  onContinue,
}: Pick<ChalkboardProps, "lesson" | "onContinue">) {
  const lastLine =
    lesson.question.kind === "steps"
      ? lesson.lines[lesson.lines.length - 1]
      : lesson.question.options.find((o) => o.correct)?.text;

  return (
    <>
      <p className="chalk animate-pop text-[3em] leading-none text-chalk-right">
        ✓
      </p>
      <p className="chalk mt-[0.3em] text-[1.9em]">{lastLine}</p>
      <p className="chalk mt-[0.6em] text-[1.05em] text-chalk-ink/80">
        {lesson.tutorMessage ||
          (lesson.clean
            ? "Clean. That one counts."
            : "Right — and now you know why.")}
      </p>
      <div className="mt-auto pt-[1em]">
        <button
          type="button"
          onClick={onContinue}
          className={`${chalkPrimary} w-full text-[1.3em]`}
        >
          Next →
        </button>
      </div>
    </>
  );
}

function WhyPhase({
  mission,
  lesson,
  onWhy,
}: Pick<ChalkboardProps, "mission" | "lesson" | "onWhy">) {
  const [writing, setWriting] = useState(false);
  const [text, setText] = useState("");
  const wrong = lesson.wrong;

  function handleReasoningTextChange(rawValue: string) {
    setText(limitInput(rawValue, INPUT_LIMITS.studentReasoning));
  }

  if (!wrong) {
    return null;
  }

  const options = buildWhyOptions(
    mission,
    lesson.question,
    wrong.misconception,
  );
  const isSteps = lesson.question.kind === "steps";

  return (
    <>
      {isSteps ? (
        // Show what the wrong move actually did to the equation: the +1 didn't disappear, it doubled.
        <ol className="chalk mb-[0.5em] space-y-[0.1em]">
          {lesson.lines.map((line, index) => {
            const isLast = index === lesson.lines.length - 1;

            return (
              <li
                key={`${line}-${index}`}
                className={`text-[1.4em] leading-tight ${isLast ? "animate-pop text-chalk-wrong" : "text-chalk-ink/60"}`}
              >
                {isLast ? `${wrong.chosen} → ${line}` : line}
              </li>
            );
          })}
        </ol>
      ) : null}
      <p className="chalk text-[1.05em] text-chalk-wrong">
        Not yet — and I&apos;m not telling you the answer.
      </p>
      <p className="chalk mt-[0.35em] text-[1.7em] leading-tight">
        {lesson.whyPrompt || (
          <>
            Why did you pick{" "}
            <span className="underline decoration-wavy decoration-chalk-wrong/70">
              {wrong.chosen}
            </span>
            ?
          </>
        )}
      </p>

      {writing ? (
        <form
          className="mt-auto flex flex-col gap-[0.6em] pt-[1em]"
          onSubmit={(event) => {
            event.preventDefault();

            if (text.trim().length > 1) {
              onWhy({ text });
            }
          }}
        >
          <textarea
            autoFocus
            value={text}
            onChange={(event) => handleReasoningTextChange(event.target.value)}
            maxLength={INPUT_LIMITS.studentReasoning}
            rows={3}
            placeholder="Tell me how you were thinking…"
            className="chalk chalk-line pointer-events-auto w-full resize-none rounded-sm border-2 bg-transparent px-[0.6em] py-[0.4em] text-[1.1em] outline-none placeholder:text-chalk-ink/40"
          />
          <div className="grid grid-cols-2 gap-[0.6em]">
            <button
              type="button"
              onClick={() => setWriting(false)}
              className={`${chalkButton} text-[1.05em]`}
            >
              Back
            </button>
            <button
              type="submit"
              disabled={text.trim().length < 2}
              className={`${chalkPrimary} text-[1.05em]`}
            >
              That&apos;s why
            </button>
          </div>
        </form>
      ) : (
        <div className="mt-auto grid gap-[0.5em] pt-[1em] pb-[3px]">
          {options.map((option, index) => (
            <button
              key={option.id}
              type="button"
              onClick={() => onWhy({ misconceptionId: option.id })}
              className={`${chalkButton} text-left text-[1.05em] ${TILT[index % TILT.length]}`}
            >
              “{option.label}”
            </button>
          ))}
          <button
            type="button"
            onClick={() => setWriting(true)}
            className={`${chalkButton} text-[0.95em] text-chalk-ink/80`}
          >
            ✎ Write it in my own words
          </button>
        </div>
      )}
    </>
  );
}

function ExplainPhase({
  lesson,
  explanationStyle,
  onExplainNext,
}: Pick<ChalkboardProps, "lesson" | "explanationStyle" | "onExplainNext">) {
  const explanation: Explanation = lesson.question.explanation;
  const shown = lesson.explainIndex + 1;
  const isLast = shown >= explanation.steps.length;

  return (
    <>
      <p className="chalk text-[0.85em] text-chalk-ink/70">
        {explanationStyle === "visual" ? "Let me draw it." : "Step by step."} ·{" "}
        {shown}/{explanation.steps.length}
      </p>

      {explanationStyle === "visual" ? (
        <div className="mt-[0.6em]">
          <VisualExplanation visual={explanation.visual} upTo={shown} />
          <p
            key={shown}
            className="chalk animate-pop mt-[0.8em] text-[1.25em] leading-snug"
          >
            {explanation.steps[lesson.explainIndex]}
          </p>
        </div>
      ) : (
        <ol className="chalk mt-[0.6em] space-y-[0.6em]">
          {explanation.steps.slice(0, shown).map((step, index) => (
            <li
              key={step}
              className={`flex gap-[0.6em] text-[1.25em] leading-snug ${index === lesson.explainIndex ? "animate-pop" : "text-chalk-ink/70"}`}
            >
              <span className="text-chalk-accent">{index + 1}.</span>
              <span>{step}</span>
            </li>
          ))}
        </ol>
      )}

      <div className="mt-auto pt-[1em]">
        <button
          type="button"
          onClick={onExplainNext}
          className={`${chalkPrimary} w-full text-[1.3em]`}
        >
          {isLast ? "Got it" : "Next step →"}
        </button>
      </div>
    </>
  );
}

function FeedbackPhase() {
  return (
    <div className="flex flex-1 flex-col justify-center">
      <p className="chalk text-[1.9em] leading-tight">
        How was that explanation?
      </p>
      <p className="chalk mt-[0.6em] text-[1.05em] text-chalk-ink/70">
        Be honest — I change how I teach based on this.
      </p>
    </div>
  );
}

function ThinkingPhase() {
  return (
    <div className="chalk flex flex-1 flex-col items-center justify-center gap-[0.6em] text-[1.25em] text-chalk-ink/80">
      <span
        className="animate-think size-[0.9em] rounded-full bg-cyan-300 shadow-[0_0_18px_rgba(103,232,249,0.9)]"
        aria-hidden
      />
      {TUTOR_NAME} is thinking…
    </div>
  );
}

function DecisionPhase({
  lesson,
  onContinue,
}: Pick<ChalkboardProps, "lesson" | "onContinue">) {
  const decision = lesson.decision;

  if (!decision) {
    return null;
  }

  const leaves = decision.action === "mission-complete";

  return (
    <>
      <p className="chalk text-[0.85em] text-chalk-ink/70">
        {TUTOR_NAME} decided
      </p>
      <p className="chalk animate-pop mt-[0.3em] text-[1.6em] leading-tight">
        {decision.message}
      </p>
      <div className="chalk-line mt-[0.8em] rounded-sm border-2 border-dashed border-chalk-accent/60 p-[0.6em]">
        <p className="chalk text-[0.75em] tracking-[0.14em] text-chalk-accent uppercase">
          Why
        </p>
        <p className="chalk mt-[0.2em] text-[1.1em] leading-snug">
          {decision.why}
        </p>
      </div>
      <div className="mt-auto pt-[1em]">
        <button
          type="button"
          onClick={onContinue}
          className={`${chalkPrimary} w-full text-[1.3em]`}
        >
          {leaves ? "Stand up →" : "Continue →"}
        </button>
      </div>
    </>
  );
}

// ── board ───────────────────────────────────────────────────────────────────

function questionLabel(question: Question) {
  return question.kind === "choice"
    ? "Multiple choice"
    : `${question.steps.length}-move equation`;
}

/**
 * The whole lesson lives on this board — Bully's math class, minus the timer.
 * The board fills the screen; text shrinks to fit rather than scroll.
 */
export function Chalkboard(props: ChalkboardProps) {
  const { lesson, mission, progress, onEndSession, onFeedback, onStand } =
    props;
  const bodyRef = useRef<HTMLDivElement>(null);

  useFitScale(bodyRef, [
    lesson.phase,
    lesson.question.id,
    lesson.stepIndex,
    lesson.explainIndex,
    lesson.lines.length,
    lesson.decision,
  ]);

  return (
    <div className="absolute inset-0 z-20 flex flex-col bg-black/45 backdrop-blur-[2px]">
      <div className="mx-auto flex w-full max-w-3xl flex-1 flex-col px-2 pt-[max(env(safe-area-inset-top),0.5rem)] pb-1 sm:px-6 sm:pt-4 sm:pb-3">
        <section className="chalk-board flex min-h-0 flex-1 flex-col rounded-md border-[8px] border-chalk-frame p-3 sm:border-[10px] sm:p-6">
          <header className="flex flex-col gap-3">
            <div className="pointer-events-auto flex w-full flex-row gap-2">
              <button
                type="button"
                onClick={onStand}
                className={`${chalkHeaderButton} flex-1 text-chalk-ink/80`}
              >
                Leave board
              </button>
              <button
                type="button"
                onClick={onEndSession}
                className={`${chalkHeaderButton} flex-1 text-chalk-wrong`}
              >
                New session
              </button>
            </div>
            <div className="flex items-start justify-between gap-3">
              <div className="chalk min-w-0 leading-tight">
                <p className="text-base text-chalk-ink/80 sm:text-lg">
                  {mission.title} · {stageLabel(mission, lesson.pod)}
                </p>
                <p className="text-base text-chalk-ink/60 sm:text-base">
                  {TUTOR_NAME} · {questionLabel(lesson.question)}
                </p>
              </div>
              <PodDots progress={progress} className="mt-1 shrink-0" />
            </div>
          </header>

          {/* Body: sized in em; the font-size is set imperatively by useFitScale. */}
          <div
            ref={bodyRef}
            className="mt-3 flex min-h-0 flex-1 flex-col overflow-hidden sm:mt-4"
          >
            {lesson.phase === "question" ? <QuestionPhase {...props} /> : null}
            {lesson.phase === "correct" ? <CorrectPhase {...props} /> : null}
            {lesson.phase === "why" ? (
              <WhyPhase
                key={lesson.question.id + lesson.stepIndex}
                {...props}
              />
            ) : null}
            {lesson.phase === "explain" ? <ExplainPhase {...props} /> : null}
            {lesson.phase === "feedback" ? <FeedbackPhase /> : null}
            {lesson.phase === "thinking" ? <ThinkingPhase /> : null}
            {lesson.phase === "decision" ? <DecisionPhase {...props} /> : null}
          </div>
        </section>

        {lesson.phase === "feedback" ? (
          <div className="mt-1 pb-[max(env(safe-area-inset-bottom),0.25rem)]">
            <p className="chalk mb-1 text-center text-[0.75rem] text-chalk-ink/55">
              ↓ pick one below
            </p>
            <div className="grid grid-cols-4 gap-2">
              {FEEDBACK.map((item) => (
                <button
                  key={item.value}
                  type="button"
                  onClick={() => onFeedback(item.value)}
                  className="glass font-display animate-rise flex h-14 flex-col items-center justify-center gap-0.5 rounded-2xl px-1 text-[0.7rem] font-semibold text-white transition hover:bg-white/10 active:translate-y-px sm:text-xs"
                >
                  <span className="text-base leading-none" aria-hidden>
                    {item.emoji}
                  </span>
                  {item.label}
                </button>
              ))}
            </div>
          </div>
        ) : (
          <div className="pb-[env(safe-area-inset-bottom)]" />
        )}
      </div>
    </div>
  );
}
