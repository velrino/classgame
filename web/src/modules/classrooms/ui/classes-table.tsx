"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { ClipboardList, Pencil } from "lucide-react";
import { useMemo } from "react";

import {
  DataTable,
  DataTableColumnHeader,
} from "@/components/data-table/index";
import { Button } from "@/components/ui/button";
import { formatDate } from "@/lib/date";
import type { TeacherClassDto } from "@/modules/classrooms/adapters/types/contracts";

type ClassesTableProps = {
  readonly teacherClasses: TeacherClassDto[];
  readonly onEditClass: (teacherClass: TeacherClassDto) => void;
  readonly onOpenClass: (teacherClass: TeacherClassDto) => void;
};

export function ClassesTable({
  onEditClass,
  onOpenClass,
  teacherClasses,
}: ClassesTableProps) {
  const columns = useMemo<ColumnDef<TeacherClassDto>[]>(
    () => [
      {
        accessorKey: "class_name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Class" />
        ),
        cell: ({ row }) => (
          <div className="min-w-0">
            <p className="truncate font-medium text-foreground">
              {row.original.class_name || row.original.class_code}
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              {statusLabel(row.original.status)}
            </p>
          </div>
        ),
        meta: {
          className: "ps-4 max-w-0 w-[32%]",
        },
      },
      {
        accessorKey: "class_code",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Code" />
        ),
        cell: ({ row }) => (
          <span className="rounded-md bg-emerald-50 px-2.5 py-1 font-mono text-xs font-bold text-emerald-800">
            {row.original.class_code}
          </span>
        ),
        meta: {
          className: "ps-1",
          tdClassName: "ps-3",
        },
      },
      {
        accessorKey: "subject",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Subject" />
        ),
        cell: ({ row }) => (
          <span className="text-muted-foreground">
            {row.original.subject || "Not set"}
          </span>
        ),
        meta: {
          className: "ps-1",
          tdClassName: "ps-4",
        },
      },
      {
        accessorKey: "grade_level",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Grade" />
        ),
        cell: ({ row }) => (
          <span className="text-muted-foreground">
            {row.original.grade_level || "Not set"}
          </span>
        ),
        meta: {
          className: "ps-1",
          tdClassName: "ps-4",
        },
      },
      {
        accessorKey: "mission_count",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Missions" />
        ),
        cell: ({ row }) => (
          <span className="rounded-md bg-muted px-2 py-1 text-xs font-semibold text-muted-foreground">
            {row.original.mission_count}
          </span>
        ),
        meta: {
          className: "ps-1",
          tdClassName: "ps-4",
        },
      },
      {
        accessorKey: "updated_at",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Updated" />
        ),
        cell: ({ row }) => (
          <span className="text-muted-foreground">
            {formatDate(row.original.updated_at, {
              fallback: "Never",
              preset: "date-time",
              relative: true,
            })}
          </span>
        ),
        meta: {
          className: "ps-1",
          tdClassName: "ps-4",
        },
      },
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => (
          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              aria-label={`Edit ${row.original.class_name || row.original.class_code}`}
              onClick={(event) => {
                event.stopPropagation();
                onEditClass(row.original);
              }}
            >
              <Pencil className="size-4" aria-hidden />
              Edit
            </Button>
            <Button
              type="button"
              size="sm"
              aria-label={`Open ${row.original.class_name || row.original.class_code}`}
              onClick={(event) => {
                event.stopPropagation();
                onOpenClass(row.original);
              }}
            >
              <ClipboardList className="size-4" aria-hidden />
              Open
            </Button>
          </div>
        ),
        meta: {
          className: "pe-4 text-right",
        },
      },
    ],
    [onEditClass, onOpenClass],
  );

  return (
    <DataTable
      columns={columns}
      data={teacherClasses}
      emptyMessage="No classes yet."
      getGlobalFilterText={(teacherClass) =>
        [
          teacherClass.class_name,
          teacherClass.class_code,
          teacherClass.subject,
          teacherClass.grade_level,
          teacherClass.status,
        ].join(" ")
      }
      minWidthClassName="min-w-[900px]"
      searchPlaceholder="Filter classes..."
    />
  );
}

function statusLabel(status: string) {
  if (status === "published") {
    return "Published";
  }
  if (status === "draft") {
    return "Draft";
  }
  if (status === "archived") {
    return "Archived";
  }

  return status || "Active";
}
