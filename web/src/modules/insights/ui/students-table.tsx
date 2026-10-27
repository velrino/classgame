"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { useMemo } from "react";

import {
  DataTable,
  DataTableColumnHeader,
} from "@/components/data-table/index";
import { formatDate } from "@/lib/date";
import type { DashboardStudentDto } from "@/modules/insights/adapters/types/contracts";

type StudentsTableProps = {
  readonly students: DashboardStudentDto[];
  readonly onStudentClick?: (student: DashboardStudentDto) => void;
};

export function StudentsTable({
  students,
  onStudentClick,
}: StudentsTableProps) {
  const columns = useMemo<ColumnDef<DashboardStudentDto>[]>(
    () => [
      {
        accessorKey: "student_name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Student" />
        ),
        cell: ({ row }) => (
          <div className="min-w-0">
            <p className="truncate font-medium text-foreground">
              {row.original.student_name}
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              {formatDate(row.original.last_activity_at, {
                fallback: "No mission",
                preset: "date-time",
                relative: true,
              })}
            </p>
          </div>
        ),
        meta: {
          className: "ps-4 max-w-0 w-[24%]",
        },
      },
      {
        id: "progress",
        accessorFn: (row) => row.progress.percentage,
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Progress" />
        ),
        cell: ({ row }) => (
          <div className="w-56">
            <div className="flex items-center justify-between gap-3">
              <span className="font-semibold text-foreground">
                {row.original.progress.percentage}%
              </span>
              <span className={statusClassName(row.original.progress.status)}>
                {statusLabel(row.original.progress.status)}
              </span>
            </div>
            <div className="mt-2 h-2 rounded-full bg-muted">
              <div
                className="h-full rounded-full bg-[#9164f9]"
                style={{ width: `${row.original.progress.percentage}%` }}
              />
            </div>
            <p className="mt-2 text-xs text-muted-foreground">
              {row.original.progress.completed_questions}/
              {row.original.progress.total_questions} questions
            </p>
          </div>
        ),
        meta: {
          className: "ps-1",
          tdClassName: "ps-4",
        },
      },
      {
        id: "strengths",
        accessorFn: (row) => row.strengths.join(", "),
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Doing well" />
        ),
        cell: ({ row }) => (
          <span className="block max-w-[260px] truncate text-muted-foreground">
            {row.original.strengths.join(", ")}
          </span>
        ),
        meta: {
          className: "ps-1",
          tdClassName: "ps-4",
        },
      },
      {
        id: "improvement_areas",
        accessorFn: (row) => row.improvement_areas.join(", "),
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Needs practice" />
        ),
        cell: ({ row }) => (
          <span className="block max-w-[260px] truncate text-muted-foreground">
            {row.original.improvement_areas.join(", ")}
          </span>
        ),
        meta: {
          className: "ps-1",
          tdClassName: "ps-4",
        },
      },
      {
        id: "attempts",
        accessorFn: (row) => row.attempts.length,
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Latest attempt" />
        ),
        cell: ({ row }) => (
          <div>
            <span className="rounded-md bg-[#d2c2e9]/35 px-2 py-1 font-semibold text-[#6841c4]">
              {row.original.attempts.length ? "Available" : "No attempt"}
            </span>
            <p className="mt-2 text-xs font-medium text-[#6841c4]">
              Browse history
            </p>
          </div>
        ),
        meta: {
          className: "ps-1",
          tdClassName: "ps-4",
        },
      },
      {
        accessorKey: "notebook_notes",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Notes" />
        ),
        cell: ({ row }) => (
          <span className="rounded-md bg-muted px-2 py-1 font-semibold text-muted-foreground">
            {row.original.notebook_notes}
          </span>
        ),
        meta: {
          className: "ps-1 pe-4",
          tdClassName: "ps-4 pe-4",
        },
      },
    ],
    [],
  );

  return (
    <DataTable
      columns={columns}
      data={students}
      emptyMessage="No students have joined this class yet."
      getGlobalFilterText={(student) =>
        [
          student.student_name,
          student.class_id,
          student.progress.status,
          student.strengths.join(" "),
          student.improvement_areas.join(" "),
        ].join(" ")
      }
      minWidthClassName="min-w-[980px]"
      onRowClick={onStudentClick}
      searchPlaceholder="Search students..."
    />
  );
}

function statusLabel(status: string) {
  if (status === "completed") {
    return "Completed";
  }
  if (status === "needs_attention") {
    return "Needs help";
  }
  if (status === "abandoned") {
    return "Left early";
  }
  if (status === "not_started") {
    return "Not started";
  }

  return "In progress";
}

function statusClassName(status: string) {
  const base = "rounded-md px-2 py-1 text-xs font-bold";

  if (status === "completed") {
    return `${base} bg-[#c9b3ec]/30 text-[#6841c4]`;
  }
  if (status === "needs_attention") {
    return `${base} bg-[#9164f9] text-white`;
  }
  if (status === "abandoned") {
    return `${base} bg-[#aa84f4]/25 text-[#6841c4]`;
  }
  if (status === "not_started") {
    return `${base} bg-[#e2e2e2] text-foreground`;
  }

  return `${base} bg-[#b593f2]/25 text-[#6841c4]`;
}
