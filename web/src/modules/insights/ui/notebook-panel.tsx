"use client";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type {
  DashboardStudentDto,
  SuggestionDto,
} from "@/modules/insights/adapters/types/contracts";
import type { TeacherClassMissionDto } from "@/modules/missions";
import type { LucideIcon } from "lucide-react";
import {
  AlertTriangle,
  BookOpenCheck,
  CheckCircle2,
  NotebookTabs,
} from "lucide-react";

export function NotebookPanel({
  activeMission,
  students,
  suggestions,
}: {
  readonly activeMission: TeacherClassMissionDto | null;
  readonly students: DashboardStudentDto[];
  readonly suggestions: SuggestionDto[];
}) {
  const attentionStudents = students.filter(
    (student) =>
      student.progress.status === "needs_attention" ||
      student.progress.status === "abandoned",
  );
  const completedStudents = students.filter(
    (student) => student.progress.status === "completed",
  );

  return (
    <section className="grid gap-4 lg:grid-cols-3">
      <Card className="gap-0 border-[#dad2e6] py-0 lg:col-span-2">
        <CardHeader className="border-b border-[#dad2e6] bg-[#d2c2e9]/10 px-4 py-4 sm:px-6">
          <div className="flex items-start gap-3">
            <div className="grid size-10 place-items-center rounded-lg bg-[#9164f9] text-white">
              <NotebookTabs className="size-5" aria-hidden />
            </div>
            <div>
              <CardTitle>Teacher notes</CardTitle>
              <CardDescription className="mt-2 leading-6">
                A short, readable summary of what students are learning and what
                may help next.
              </CardDescription>
            </div>
          </div>
        </CardHeader>

        <CardContent className="grid gap-3 px-4 py-5 sm:px-6">
          <NotebookRow
            icon={BookOpenCheck}
            title="Current focus"
            value={
              activeMission?.learning_goal ?? "No active learning goal yet."
            }
          />
          <NotebookRow
            icon={CheckCircle2}
            title="A positive sign"
            value={
              completedStudents.length > 0
                ? `${completedStudents.length} students completed the mission.`
                : "Completed missions will appear after students play."
            }
          />
          <NotebookRow
            icon={AlertTriangle}
            title="Needs follow-up"
            value={
              attentionStudents.length > 0
                ? attentionStudents
                    .map((student) => student.student_name)
                    .join(", ")
                : "No students marked as needing attention yet."
            }
          />
        </CardContent>
      </Card>

      <Card className="gap-0 border-[#dad2e6] py-0">
        <CardHeader className="border-b border-[#dad2e6] bg-[#d2c2e9]/10 px-4 py-4 sm:px-6">
          <CardTitle>Next lesson notes</CardTitle>
          <CardDescription className="mt-2 leading-6">
            Ideas you can use or adapt.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 px-4 py-5 sm:px-6">
          {(suggestions.length > 0 ? suggestions.slice(0, 3) : []).map(
            (suggestion) => (
              <div
                key={suggestion.id}
                className="rounded-xl border border-[#dad2e6] p-3"
              >
                <p className="text-sm font-semibold">{suggestion.title}</p>
                <p className="mt-1 text-sm leading-6 text-muted-foreground">
                  {suggestion.activity}
                </p>
              </div>
            ),
          )}
          {suggestions.length === 0 ? (
            <p className="text-sm leading-6 text-muted-foreground">
              Share a mission and collect student work to unlock lesson ideas.
            </p>
          ) : null}
        </CardContent>
      </Card>
    </section>
  );
}

export function NotebookRow({
  icon: Icon,
  title,
  value,
}: {
  readonly icon: LucideIcon;
  readonly title: string;
  readonly value: string;
}) {
  return (
    <div className="flex gap-3 rounded-xl border border-[#dad2e6] p-3">
      <Icon className="mt-0.5 size-4 shrink-0 text-[#9164f9]" aria-hidden />
      <div>
        <p className="text-sm font-semibold">{title}</p>
        <p className="mt-1 text-sm leading-6 text-muted-foreground">{value}</p>
      </div>
    </div>
  );
}
