import {
  BookOpenCheck,
  LifeBuoy,
  Sparkles,
  Users,
  type LucideIcon,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import type {
  PlannedSuggestion,
  SuggestionTier,
} from "@/modules/insights/domain/analytics";

type NextLessonPlanProps = {
  readonly canBuild: boolean;
  readonly suggestions: PlannedSuggestion[];
  readonly onBuildNextMission: () => void;
};

const tierMeta: Record<
  SuggestionTier,
  {
    readonly detail: string;
    readonly icon: LucideIcon;
    readonly iconClassName: string;
    readonly label: string;
  }
> = {
  reteaching: {
    label: "Review together",
    detail: "A quick activity for students who need help",
    icon: LifeBuoy,
    iconClassName: "bg-[#9164f9]/12 text-[#9164f9]",
  },
  whole_class: {
    label: "Try with everyone",
    detail: "A simple next step for the whole class",
    icon: Users,
    iconClassName: "bg-[#b593f2]/20 text-[#9164f9]",
  },
  enrichment: {
    label: "Add a challenge",
    detail: "For students who are ready to go further",
    icon: Sparkles,
    iconClassName: "bg-[#d2c2e9]/35 text-[#9164f9]",
  },
};

export function NextLessonPlan({
  canBuild,
  onBuildNextMission,
  suggestions,
}: NextLessonPlanProps) {
  return (
    <div className="space-y-4">
      {suggestions.length > 0 ? (
        <div className="grid gap-3 lg:grid-cols-3">
          {suggestions.map((suggestion) => {
            const meta = tierMeta[suggestion.tier];
            const Icon = meta.icon;

            return (
              <article
                key={suggestion.id}
                className="flex min-w-0 flex-col rounded-xl border border-[#dad2e6] bg-card p-4"
              >
                <div className="flex items-start gap-3">
                  <span
                    className={cn(
                      "grid size-9 shrink-0 place-items-center rounded-md",
                      meta.iconClassName,
                    )}
                  >
                    <Icon className="size-4" aria-hidden />
                  </span>
                  <div className="min-w-0">
                    <h3 className="text-sm font-semibold">{meta.label}</h3>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      {meta.detail}
                    </p>
                  </div>
                </div>
                <p className="mt-4 text-sm font-semibold leading-5">
                  {suggestion.title}
                </p>
                <p className="mt-2 text-sm leading-6 text-muted-foreground">
                  {suggestion.activity}
                </p>
                {suggestion.reason ? (
                  <p className="mt-2 text-xs leading-5 text-muted-foreground">
                    <span className="font-semibold text-foreground">Why: </span>
                    {suggestion.reason}
                  </p>
                ) : null}
                {suggestion.questions.length > 0 ? (
                  <ul className="mt-3 space-y-1.5 border-t border-[#dad2e6] pt-3">
                    {suggestion.questions.slice(0, 2).map((question) => (
                      <li
                        key={question}
                        className="flex gap-2 text-xs leading-5 text-muted-foreground"
                      >
                        <span aria-hidden>“</span>
                        <span>{question}</span>
                      </li>
                    ))}
                  </ul>
                ) : null}
                {suggestion.focus_skills.length > 0 ? (
                  <div className="mt-auto flex flex-wrap gap-1.5 pt-3">
                    {suggestion.focus_skills.map((skill) => (
                      <span
                        key={skill}
                        className="rounded-md bg-[#d2c2e9]/30 px-2 py-0.5 text-[11px] font-semibold text-muted-foreground"
                      >
                        {skill}
                      </span>
                    ))}
                  </div>
                ) : null}
              </article>
            );
          })}
        </div>
      ) : (
        <p className="rounded-md border border-dashed border-border bg-muted/40 p-4 text-sm leading-6 text-muted-foreground">
          Publish a mission and Mia will suggest what to reteach, what to do
          with the whole class, and how to stretch the students who are ahead.
        </p>
      )}

      <div className="flex flex-col gap-3 rounded-xl border border-[#c9b3ec] bg-[#d2c2e9]/20 p-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <p className="text-sm font-semibold">
            Turn this into tomorrow&apos;s mission
          </p>
          <p className="mt-1 text-xs leading-5 text-muted-foreground">
            Mia starts a draft using what the class found easy and difficult.
            You can review everything before sharing it.
          </p>
        </div>
        <Button
          type="button"
          disabled={!canBuild}
          onClick={onBuildNextMission}
          className="shrink-0"
        >
          <BookOpenCheck className="size-4" aria-hidden />
          Draft the next mission
        </Button>
      </div>
    </div>
  );
}
