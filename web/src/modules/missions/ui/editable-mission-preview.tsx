"use client";
import { cn } from "@/lib/utils";
import type {
  MissionPreviewQuestionDto,
  TeacherClassMissionDto,
} from "@/modules/missions/adapters/types/contracts";
import { normalizeMissionPreviewQuestions } from "@/modules/missions/domain/mission-preview";
import { INPUT_LIMITS, limitInput } from "@/shared/input-limits";
import { ChevronDown } from "lucide-react";
import { useState } from "react";

export type EditableMissionPreviewProps = {
  readonly disabled: boolean;
  readonly mission: TeacherClassMissionDto;
  readonly onMissionChange: (
    updater: (mission: TeacherClassMissionDto) => TeacherClassMissionDto,
  ) => void;
};

export function EditableMissionPreview({
  disabled,
  mission,
  onMissionChange,
}: EditableMissionPreviewProps) {
  const previewQuestions = normalizeMissionPreviewQuestions(
    mission.preview_questions,
  );
  const [openQuestionIds, setOpenQuestionIds] = useState<ReadonlySet<string>>(
    () => new Set(),
  );

  function updateQuestion(
    questionId: string,
    updater: (question: MissionPreviewQuestionDto) => MissionPreviewQuestionDto,
  ) {
    onMissionChange((current) => ({
      ...current,
      preview_questions: normalizeMissionPreviewQuestions(
        current.preview_questions,
      ).map((question) =>
        question.id === questionId ? updater(question) : question,
      ),
    }));
  }

  function toggleQuestion(questionId: string) {
    setOpenQuestionIds((current) => {
      const next = new Set(current);
      if (next.has(questionId)) {
        next.delete(questionId);
      } else {
        next.add(questionId);
      }

      return next;
    });
  }

  return (
    <div className="grid gap-4">
      <p className="text-xs font-medium text-muted-foreground">
        Click any text field to adjust the preview before approving.
      </p>

      <label className="grid gap-2 text-sm font-medium text-foreground">
        Summary
        <EditablePreviewField
          ariaLabel="Mission summary"
          disabled={disabled}
          showCharacterCounter={false}
          maxLength={INPUT_LIMITS.missionSummary}
          minRows={4}
          value={mission.summary}
          onChange={(summary) =>
            onMissionChange((current) => ({
              ...current,
              summary,
            }))
          }
          className="text-sm leading-6 text-muted-foreground"
        />
      </label>

      <div className="grid gap-2">
        {previewQuestions.map((question, index) => {
          const open = openQuestionIds.has(question.id);
          const options = question.options ?? [];
          const panelId = `mission-preview-question-${index + 1}`;

          return (
            <article
              key={question.id}
              className="overflow-hidden rounded-md border border-border"
            >
              <div className="px-3 py-3 transition hover:bg-muted/40">
                <div className="mb-2 flex items-center gap-3">
                  <button
                    type="button"
                    aria-label={`${open ? "Collapse" : "Expand"} question ${index + 1}`}
                    aria-expanded={open}
                    aria-controls={panelId}
                    onClick={() => toggleQuestion(question.id)}
                    className="flex-1 grid size-7 shrink-0 place-items-center rounded-md bg-primary/10 text-xs font-bold text-primary transition hover:bg-primary/15 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    Question {index + 1}
                  </button>

                  <button
                    type="button"
                    aria-label={`${open ? "Collapse" : "Expand"} question ${index + 1}`}
                    aria-expanded={open}
                    aria-controls={panelId}
                    onClick={() => toggleQuestion(question.id)}
                    className="grid size-8 shrink-0 place-items-center rounded-md text-muted-foreground transition hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    <ChevronDown
                      className={cn(
                        "size-4 transition-transform",
                        open ? "rotate-180" : "",
                      )}
                      aria-hidden
                    />
                  </button>
                </div>
                <EditablePreviewField
                  ariaLabel={`Question ${index + 1} prompt`}
                  disabled={disabled}
                  showCharacterCounter={false}
                  maxLength={INPUT_LIMITS.missionQuestionPrompt}
                  minRows={2}
                  value={question.prompt}
                  onChange={(prompt) =>
                    updateQuestion(question.id, (current) => ({
                      ...current,
                      prompt,
                    }))
                  }
                  className="text-sm font-semibold leading-5 text-foreground"
                />
              </div>

              {open ? (
                <div
                  id={panelId}
                  className="border-t border-border px-3 pb-3 pt-3"
                >
                  {options.length > 0 ? (
                    <div className="grid gap-2">
                      {options.map((option, optionIndex) => (
                        <div
                          key={`${question.id}:option:${optionIndex}`}
                          className="grid gap-2"
                        >
                          <input
                            aria-label={`Question ${index + 1} option ${optionIndex + 1}`}
                            disabled={disabled}
                            value={option}
                            onChange={(event) =>
                              updateQuestion(question.id, (current) => ({
                                ...current,
                                options: (current.options ?? []).map(
                                  (currentOption, index) =>
                                    index === optionIndex
                                      ? limitInput(
                                          event.target.value,
                                          INPUT_LIMITS.missionQuestionOption,
                                        )
                                      : currentOption,
                                ),
                              }))
                            }
                            maxLength={INPUT_LIMITS.missionQuestionOption}
                            className="rounded-md border border-transparent bg-muted px-3 py-2 text-sm text-muted-foreground outline-none transition focus:border-ring focus:bg-background focus:ring-2 focus:ring-ring/20 disabled:cursor-not-allowed disabled:opacity-70"
                          />
                        </div>
                      ))}
                    </div>
                  ) : null}

                  <label className="mt-3 grid gap-2 text-sm font-medium text-emerald-700">
                    Expected answer
                    <EditablePreviewField
                      ariaLabel={`Question ${index + 1} expected answer`}
                      disabled={disabled}
                      showCharacterCounter={false}
                      maxLength={INPUT_LIMITS.missionExpectedAnswer}
                      minRows={2}
                      value={question.expected_answer}
                      onChange={(expectedAnswer) =>
                        updateQuestion(question.id, (current) => ({
                          ...current,
                          expected_answer: expectedAnswer,
                        }))
                      }
                      className="font-semibold text-emerald-700"
                    />
                  </label>
                </div>
              ) : null}
            </article>
          );
        })}
      </div>
    </div>
  );
}

export function EditablePreviewField({
  ariaLabel,
  className,
  disabled,
  maxLength,
  minRows = 1,
  onChange,
  showCharacterCounter = true,
  value,
}: {
  readonly ariaLabel: string;
  readonly className?: string;
  readonly disabled: boolean;
  readonly maxLength?: number;
  readonly minRows?: number;
  readonly onChange: (value: string) => void;
  readonly showCharacterCounter?: boolean;
  readonly value: string;
}) {
  return (
    <div className="grid gap-2">
      <textarea
        aria-label={ariaLabel}
        disabled={disabled}
        rows={minRows}
        value={value}
        onChange={(event) =>
          onChange(
            maxLength
              ? limitInput(event.target.value, maxLength)
              : event.target.value,
          )
        }
        maxLength={maxLength}
        className={cn(
          "w-full resize-none rounded-md border border-transparent bg-transparent p-0 outline-none transition focus:border-ring focus:bg-background focus:px-3 focus:py-2 focus:ring-2 focus:ring-ring/20 disabled:cursor-not-allowed disabled:opacity-70",
          className,
        )}
      />
      {maxLength && showCharacterCounter ? (
        <CharacterCounter value={value} maxLength={maxLength} />
      ) : null}
    </div>
  );
}

export function CharacterCounter({
  maxLength,
  value,
}: {
  readonly maxLength: number;
  readonly value: string;
}) {
  const currentLength = [...value].length;

  return (
    <output
      aria-label={`${currentLength} of ${maxLength} characters used`}
      className="-mt-1 block text-right text-xs font-normal tabular-nums text-muted-foreground"
    >
      {currentLength} of {maxLength}
    </output>
  );
}
