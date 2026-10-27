"use client";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatDate } from "@/lib/date";
import type { DashboardStudentDto } from "@/modules/insights/adapters/types/contracts";

type RecentPlayersTableProps = {
  readonly fallbackSubject?: string;
  readonly limit?: number;
  readonly students: DashboardStudentDto[];
};

export function RecentPlayersTable({
  fallbackSubject = "Not set",
  limit = 10,
  students,
}: RecentPlayersTableProps) {
  const recentStudents = [...students]
    .sort(
      (left, right) =>
        timestamp(right.last_activity_at) - timestamp(left.last_activity_at),
    )
    .slice(0, limit);

  return (
    <div className="overflow-hidden border border-border">
      <Table className="min-w-[720px]">
        <TableHeader className="bg-muted/40">
          <TableRow>
            <TableHead className="ps-4">Student</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Subject</TableHead>
            <TableHead>Current stage</TableHead>
            <TableHead className="pe-4 text-right">Last active</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {recentStudents.length > 0 ? (
            recentStudents.map((student) => (
              <TableRow key={student.id}>
                <TableCell className="ps-4">
                  <div className="flex items-center gap-3">
                    <span className="grid size-8 shrink-0 place-items-center rounded-full bg-muted text-xs font-semibold">
                      {studentInitials(student.student_name)}
                    </span>
                    <span className="max-w-52 truncate font-medium">
                      {student.student_name}
                    </span>
                  </div>
                </TableCell>
                <TableCell>
                  <PresenceBadge online={student.session?.is_online ?? false} />
                </TableCell>
                <TableCell className="text-muted-foreground">
                  {student.session?.subject || fallbackSubject}
                </TableCell>
                <TableCell>
                  <div>
                    <p className="font-medium">
                      {student.session?.stage_name || "Entered the game"}
                    </p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {sessionStatusLabel(student.session?.status ?? "")}
                    </p>
                  </div>
                </TableCell>
                <TableCell className="pe-4 text-right text-muted-foreground">
                  {student.session?.is_online
                    ? "Now"
                    : formatDate(student.last_activity_at, {
                        fallback: "Never",
                        preset: "date-time",
                        relative: true,
                      })}
                </TableCell>
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell
                colSpan={5}
                className="h-28 text-center text-muted-foreground"
              >
                Recent players will appear when students enter the game.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
}

function PresenceBadge({ online }: { readonly online: boolean }) {
  return (
    <span
      className={
        online
          ? "inline-flex items-center gap-2 rounded-md bg-emerald-50 px-2.5 py-1 text-xs font-semibold text-emerald-800 dark:bg-emerald-950/50 dark:text-emerald-200"
          : "inline-flex items-center gap-2 rounded-md bg-muted px-2.5 py-1 text-xs font-semibold text-muted-foreground"
      }
    >
      <span
        className={
          online
            ? "size-2 rounded-full bg-emerald-500"
            : "size-2 rounded-full bg-slate-400"
        }
      />
      {online ? "Online" : "Offline"}
    </span>
  );
}

function studentInitials(value: string) {
  const parts = value.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return "?";
  }
  if (parts.length === 1) {
    return parts[0].slice(0, 2).toUpperCase();
  }

  return `${parts[0][0] ?? ""}${parts[parts.length - 1]?.[0] ?? ""}`.toUpperCase();
}

function sessionStatusLabel(status: string) {
  const labels: Record<string, string> = {
    completed: "Mission completed",
    abandoned: "Left before finishing",
    exploring: "Exploring the classroom",
    learning: "Working with Mia",
    reviewing: "Reviewing a skill",
    paused: "Paused — progress saved",
    thinking: "Mia is adapting",
  };

  return labels[status] ?? "Waiting to begin";
}

function timestamp(value: string) {
  const parsed = new Date(value).getTime();
  return Number.isNaN(parsed) ? 0 : parsed;
}
