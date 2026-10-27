"use client";

import { useId, useState } from "react";

import { INPUT_LIMITS, limitInput } from "@/shared/input-limits";

import { studentEntryCopy } from "@/modules/classrooms/domain/student-entry-messages";

export type StudentSessionFormValue = {
  readonly classCode: string;
  readonly studentName: string;
};

type StudentSessionModalProps = {
  readonly error: string | null;
  readonly initialClassCode?: string;
  readonly loading: boolean;
  readonly onSubmit: (value: StudentSessionFormValue) => void;
};

const CLASS_CODE_LENGTH = 6;
const CLASS_CODE_MASKED_LENGTH = 7;
const classCodeNonDigits = /\D/g;
const classCodeBlockedCharacters = /[^\d\s]/;

export function StudentSessionModal({
  error,
  initialClassCode = "",
  loading,
  onSubmit,
}: StudentSessionModalProps) {
  const classCodeInputId = useId();
  const classCodeHintId = useId();
  const studentNameInputId = useId();
  const studentNameHintId = useId();
  const [classCode, setClassCode] = useState(initialClassCode);
  const [studentName, setStudentName] = useState("");
  const [typedBlockedCharacter, setTypedBlockedCharacter] = useState(false);

  const trimmedName = studentName.trim();
  const codeComplete = classCode.length === CLASS_CODE_LENGTH;
  const canSubmit = codeComplete && trimmedName.length >= 2 && !loading;

  const codeHint = typedBlockedCharacter
    ? studentEntryCopy.codeBlockedHint
    : codeComplete
      ? "Got it!"
      : `${classCode.length}/${CLASS_CODE_LENGTH} · ${studentEntryCopy.codeHint}`;
  const nameHint =
    trimmedName.length === 1
      ? studentEntryCopy.nameTooShort
      : studentEntryCopy.nameHint;

  function handleClassCodeChange(rawValue: string) {
    const cleaned = rawValue
      .replace(classCodeNonDigits, "")
      .slice(0, CLASS_CODE_LENGTH);

    setTypedBlockedCharacter(
      classCodeBlockedCharacters.test(rawValue) &&
        cleaned.length < CLASS_CODE_LENGTH,
    );
    setClassCode(cleaned);
  }

  function handleStudentNameChange(rawValue: string) {
    setStudentName(limitInput(rawValue, INPUT_LIMITS.studentName));
  }

  return (
    <div className="pointer-events-auto fixed inset-0 z-50 grid place-items-center bg-[radial-gradient(ellipse_at_center,rgba(5,7,13,0.5)_0%,rgba(5,7,13,0.25)_55%,rgba(5,7,13,0.6)_100%)] px-4 py-[max(env(safe-area-inset-top),1rem)] backdrop-blur-[3px]">
      <section className="w-full max-w-sm rounded-2xl border border-slate-700 bg-[#090c18] p-5 text-white shadow-2xl shadow-black/40">
        <div className="text-center">
          <p className="font-display text-[0.68rem] font-semibold tracking-[0.22em] text-cyan-200 uppercase">
            ClassGame
          </p>
          <h1 className="font-display mt-2 text-2xl font-bold tracking-tight">
            Join your class
          </h1>
          <p className="mt-2 text-sm leading-6 text-slate-300">
            Type the code your teacher gave you. Mia remembers where you stop,
            so you can come back anytime.
          </p>
        </div>

        <form
          className="mt-5 space-y-4"
          onSubmit={(event) => {
            event.preventDefault();

            if (canSubmit) {
              onSubmit({ classCode, studentName: trimmedName });
            }
          }}
        >
          <div>
            <label
              htmlFor={classCodeInputId}
              className="font-display text-xs font-semibold tracking-[0.16em] text-slate-300 uppercase"
            >
              Class code
            </label>
            <input
              id={classCodeInputId}
              value={formatClassCode(classCode)}
              onChange={(event) => handleClassCodeChange(event.target.value)}
              disabled={loading}
              autoFocus={!codeComplete}
              autoComplete="off"
              inputMode="numeric"
              maxLength={CLASS_CODE_MASKED_LENGTH}
              placeholder="123 456"
              aria-describedby={classCodeHintId}
              className="mt-2 h-12 w-full rounded-xl border border-white/12 bg-white/8 px-4 text-center font-display text-base font-semibold tracking-[0.08em] text-white outline-none transition placeholder:text-slate-500 focus:border-cyan-200/70 focus:bg-white/12 disabled:opacity-60"
            />
            <p
              id={classCodeHintId}
              className={`mt-1.5 text-xs leading-5 ${typedBlockedCharacter ? "text-amber-200" : codeComplete ? "text-emerald-200" : "text-slate-400"}`}
              aria-live="polite"
            >
              {codeHint}
            </p>
          </div>

          <div>
            <label
              htmlFor={studentNameInputId}
              className="font-display text-xs font-semibold tracking-[0.16em] text-slate-300 uppercase"
            >
              Your name
            </label>
            <input
              id={studentNameInputId}
              value={studentName}
              onChange={(event) => handleStudentNameChange(event.target.value)}
              disabled={loading}
              autoFocus={codeComplete}
              autoComplete="off"
              maxLength={INPUT_LIMITS.studentName}
              placeholder="What should Mia call you?"
              aria-describedby={studentNameHintId}
              className="mt-2 h-12 w-full rounded-xl border border-white/12 bg-white/8 px-4 text-center text-base font-semibold text-white outline-none transition placeholder:text-slate-500 focus:border-cyan-200/70 focus:bg-white/12 disabled:opacity-60"
            />
            <p
              id={studentNameHintId}
              className="mt-1.5 text-xs leading-5 text-slate-400"
            >
              {nameHint}
            </p>
          </div>

          {error ? (
            <p
              role="alert"
              className="rounded-xl border border-red-300/20 bg-red-500/10 px-3 py-2 text-sm leading-5 text-red-100"
            >
              {error}
            </p>
          ) : null}

          <button
            type="submit"
            disabled={!canSubmit}
            className="cta font-display flex h-12 w-full items-center justify-center rounded-full px-5 text-sm font-bold tracking-tight transition hover:brightness-105 active:translate-y-px disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? studentEntryCopy.opening : "Enter class"}
          </button>
        </form>
      </section>
    </div>
  );
}

function formatClassCode(value: string) {
  const firstGroup = value.slice(0, 3);
  const secondGroup = value.slice(3);

  return secondGroup ? `${firstGroup} ${secondGroup}` : firstGroup;
}
