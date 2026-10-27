import { cn } from "@/lib/utils";

import type {
  PatternGroup,
  PatternLevel,
} from "@/modules/insights/domain/analytics";
import { StudentChip } from "@/modules/insights/ui/overview/student-chip";

export type PatternVariant = "growth" | "strength";

type PatternGroupsProps = {
  readonly emptyMessage: string;
  readonly groups: PatternGroup[];
  readonly limit?: number;
  readonly totalStudents: number;
  readonly variant: PatternVariant;
};

type LevelStyle = {
  readonly dot: string;
  readonly label: string;
  readonly row: string;
};

const levelOrder: PatternLevel[] = ["high", "medium", "low"];

const levelStyles: Record<PatternVariant, Record<PatternLevel, LevelStyle>> = {
  growth: {
    high: {
      label: "Check in soon",
      dot: "bg-[#9164f9]",
      row: "bg-[#9164f9]/12 text-foreground",
    },
    medium: {
      label: "Keep an eye on",
      dot: "bg-[#aa84f4]",
      row: "bg-[#aa84f4]/15 text-foreground",
    },
    low: {
      label: "Small reminder",
      dot: "bg-[#c9b3ec]",
      row: "bg-[#c9b3ec]/20 text-foreground",
    },
  },
  strength: {
    high: {
      label: "Very confident",
      dot: "bg-[#9164f9]",
      row: "bg-[#9164f9]/12 text-foreground",
    },
    medium: {
      label: "Confident",
      dot: "bg-[#b593f2]",
      row: "bg-[#b593f2]/15 text-foreground",
    },
    low: {
      label: "Getting there",
      dot: "bg-[#d2c2e9]",
      row: "bg-[#d2c2e9]/25 text-foreground",
    },
  },
};

export function PatternGroups({
  emptyMessage,
  groups,
  limit = 6,
  totalStudents,
  variant,
}: PatternGroupsProps) {
  const visibleGroups = groups.slice(0, limit);
  const styles = levelStyles[variant];
  const countVerb =
    variant === "growth" ? "showed this pattern" : "showed this strength";

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-end gap-x-4 gap-y-1 text-xs text-muted-foreground">
        {levelOrder.map((level) => (
          <span key={level} className="flex items-center gap-1.5">
            <span
              className={cn("size-2 rounded-full", styles[level].dot)}
              aria-hidden
            />
            {styles[level].label}
          </span>
        ))}
      </div>

      {visibleGroups.length === 0 ? (
        <p className="rounded-md border border-dashed border-border bg-muted/40 p-4 text-sm leading-6 text-muted-foreground">
          {emptyMessage}
        </p>
      ) : (
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          {visibleGroups.map((group) => (
            <article
              key={group.pattern}
              className="flex min-w-0 flex-col rounded-xl border border-[#dad2e6] bg-card"
            >
              <header className="border-b border-[#dad2e6] px-4 py-3">
                <h3 className="text-sm font-semibold leading-5">
                  {group.pattern}
                </h3>
                <p className="mt-1 text-xs text-muted-foreground">
                  {group.students.length} of {totalStudents} {countVerb}
                </p>
              </header>
              <div className="grid gap-2 p-3">
                {levelOrder.map((level) => {
                  const students = group.students.filter(
                    (student) => student.level === level,
                  );
                  if (students.length === 0) {
                    return null;
                  }
                  const style = styles[level];

                  return (
                    <div key={level} className="grid gap-2">
                      <p
                        className={cn(
                          "flex items-center gap-2 rounded-md px-2.5 py-1.5 text-xs font-semibold",
                          style.row,
                        )}
                      >
                        <span
                          className={cn("size-2 rounded-full", style.dot)}
                          aria-hidden
                        />
                        {style.label}
                        <span className="opacity-70">· {students.length}</span>
                      </p>
                      <div className="flex flex-wrap gap-1.5 px-1">
                        {students.map((student) => (
                          <StudentChip key={student.id} name={student.name} />
                        ))}
                      </div>
                    </div>
                  );
                })}
              </div>
            </article>
          ))}
        </div>
      )}
    </div>
  );
}
