"use client";
import type {
  ClassLearningSummaryDto,
  DashboardStudentDto,
} from "@/modules/insights/adapters/types/contracts";
import { StudentsTable } from "@/modules/insights/ui/students-table";

export function StudentsTableSection({
  classLearning,
  students,
  onStudentClick,
}: {
  readonly classLearning?: ClassLearningSummaryDto;
  readonly students: DashboardStudentDto[];
  readonly onStudentClick: (student: DashboardStudentDto) => void;
}) {
  return (
    <section className="flex flex-1 flex-col gap-4">
      <div>
        <h2 className="text-xl font-semibold tracking-tight">Students</h2>
        <p className="text-sm text-muted-foreground">
          {students.length} on this page
        </p>
      </div>
      {classLearning && classLearning.students_with_evidence > 0 ? (
        <div className="grid gap-3 md:grid-cols-3">
          <div className="rounded-xl border border-[#dad2e6] bg-[#d2c2e9]/15 p-4">
            <p className="text-xs font-semibold text-[#6841c4] uppercase">
              LATEST ATTEMPTS
            </p>
            <p className="mt-2 text-2xl font-black">
              {classLearning.students_with_evidence}/{students.length}
            </p>
            <p className="mt-1 text-sm text-muted-foreground">
              students with work Mia can summarize
            </p>
            <p className="mt-2 text-xs font-medium text-muted-foreground">
              {classLearning.completed_attempts}/{classLearning.total_attempts}{" "}
              missions finished
            </p>
            {classLearning.abandoned_attempts > 0 ? (
              <p className="mt-1 text-xs font-semibold text-[#6841c4]">
                {classLearning.abandoned_attempts} left before finishing
              </p>
            ) : null}
          </div>
          <div className="rounded-xl border border-[#c9b3ec] bg-[#c9b3ec]/20 p-4">
            <p className="text-xs font-semibold text-[#6841c4] uppercase">
              A STRENGTH SHOWING UP OFTEN
            </p>
            <p className="mt-2 text-sm font-bold text-foreground">
              {classLearning.common_strengths[0]?.label ??
                "More completed evidence needed"}
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              {classLearning.common_strengths[0]?.count ?? 0} students
            </p>
          </div>
          <div className="rounded-xl border border-[#b593f2] bg-[#b593f2]/15 p-4">
            <p className="text-xs font-semibold text-[#6841c4] uppercase">
              TRY NEXT
            </p>
            <p className="mt-2 text-sm leading-6 font-semibold text-foreground">
              {classLearning.next_adaptation}
            </p>
          </div>
        </div>
      ) : null}
      <p className="text-xs text-muted-foreground">
        Select a student to see progress, recent answers, and Mia&apos;s notes.
      </p>
      <StudentsTable students={students} onStudentClick={onStudentClick} />
    </section>
  );
}
