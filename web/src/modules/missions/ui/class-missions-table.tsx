"use client";

import {
  ClipboardList,
  LoaderCircle,
  Pencil,
  Plus,
  Trash2,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatDate } from "@/lib/date";
import { cn } from "@/lib/utils";
import type { TeacherClassMissionDto } from "@/modules/missions/adapters/types/contracts";

import {
  missionHasPublishedRevision,
  missionHasUnpublishedChanges,
} from "@/modules/missions/domain/mission-lifecycle";

type ClassMissionsTableProps = {
  readonly activeMissionId?: string;
  readonly deletingMissionId?: string | null;
  readonly missions: TeacherClassMissionDto[];
  readonly loading: boolean;
  readonly onCreateMission: () => void;
  readonly onDeleteMission: (mission: TeacherClassMissionDto) => void;
  readonly onEditMission: (mission: TeacherClassMissionDto) => void;
};

export function ClassMissionsTable({
  activeMissionId,
  deletingMissionId,
  missions,
  loading,
  onCreateMission,
  onDeleteMission,
  onEditMission,
}: ClassMissionsTableProps) {
  return (
    <Card className="gap-0 overflow-hidden border-[#dad2e6] py-0">
      <CardHeader className="border-b border-[#dad2e6] bg-[#d2c2e9]/10 px-4 py-4 sm:px-6">
        <div className="flex items-start justify-between gap-3">
          <div>
            <CardTitle>Class missions</CardTitle>
            <CardDescription className="mt-1.5 max-w-2xl leading-6">
              Prepare a mission with Mia, review it, and choose when students
              can see it.
            </CardDescription>
          </div>
          <Button type="button" className="shrink-0" onClick={onCreateMission}>
            <Plus className="size-4" aria-hidden />
            <span className="hidden sm:inline">New mission</span>
            <span className="sm:hidden">New</span>
          </Button>
        </div>
      </CardHeader>

      <CardContent className="p-0">
        {loading ? (
          <div className="px-4 py-14 text-center text-sm font-medium text-muted-foreground sm:px-6">
            Loading missions...
          </div>
        ) : missions.length === 0 ? (
          <div className="grid min-h-64 place-items-center px-6 py-12 text-center">
            <div className="max-w-sm">
              <span className="mx-auto grid size-11 place-items-center rounded-xl bg-[#d2c2e9]/35 text-[#9164f9]">
                <ClipboardList className="size-5" aria-hidden />
              </span>
              <h3 className="mt-4 font-semibold text-foreground">
                No missions yet
              </h3>
              <p className="mt-1.5 text-sm leading-6 text-muted-foreground">
                Tell Mia what students should learn. You can check every
                question before sharing it.
              </p>
              <Button type="button" className="mt-5" onClick={onCreateMission}>
                <Plus className="size-4" aria-hidden />
                Create first mission
              </Button>
            </div>
          </div>
        ) : (
          <>
            <div className="divide-y divide-border md:hidden">
              {missions.map((mission) => (
                <MissionMobileRow
                  key={mission.id}
                  mission={mission}
                  active={mission.id === activeMissionId}
                  deleting={mission.id === deletingMissionId}
                  onDelete={onDeleteMission}
                  onEdit={onEditMission}
                />
              ))}
            </div>

            <div className="hidden md:block">
              <Table>
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead className="ps-6">Mission</TableHead>
                    <TableHead>For</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="text-right">Updated</TableHead>
                    <TableHead className="pe-6 text-right">Options</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {missions.map((mission) => (
                    <TableRow key={mission.id}>
                      <TableCell className="max-w-64 ps-6">
                        <div className="min-w-0">
                          <div className="flex items-center gap-2">
                            <p className="truncate font-semibold text-foreground">
                              {mission.mission_name || mission.title}
                            </p>
                            {mission.id === activeMissionId ? (
                              <span className="rounded-full bg-[#9164f9] px-2 py-0.5 text-[10px] font-bold text-white">
                                Current
                              </span>
                            ) : null}
                          </div>
                          <p className="mt-1 truncate text-xs text-muted-foreground">
                            {mission.subject || "Subject not set"} ·{" "}
                            {formatDuration(mission.suggested_duration_minutes)}
                          </p>
                          <p className="mt-1 line-clamp-1 text-xs text-muted-foreground">
                            {mission.learning_goal ||
                              "No learning goal provided"}
                          </p>
                        </div>
                      </TableCell>
                      <TableCell>
                        <AudienceLabel mission={mission} />
                      </TableCell>
                      <TableCell>
                        <MissionStatus mission={mission} />
                      </TableCell>
                      <TableCell className="text-right text-muted-foreground">
                        {formatDate(mission.updated_at, {
                          fallback: "Not updated",
                          preset: "date",
                          relative: true,
                        })}
                      </TableCell>
                      <TableCell className="pe-6">
                        <MissionActions
                          deleting={mission.id === deletingMissionId}
                          mission={mission}
                          onDelete={onDeleteMission}
                          onEdit={onEditMission}
                        />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

function MissionMobileRow({
  active,
  deleting,
  mission,
  onDelete,
  onEdit,
}: {
  readonly active: boolean;
  readonly deleting: boolean;
  readonly mission: TeacherClassMissionDto;
  readonly onDelete: (mission: TeacherClassMissionDto) => void;
  readonly onEdit: (mission: TeacherClassMissionDto) => void;
}) {
  return (
    <article className="px-4 py-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <h3 className="truncate font-semibold text-foreground">
              {mission.mission_name || mission.title}
            </h3>
            {active ? (
              <span className="rounded-full bg-[#9164f9] px-2 py-0.5 text-[10px] font-bold text-white">
                Current
              </span>
            ) : null}
          </div>
          <p className="mt-1 text-xs text-muted-foreground">
            {mission.subject || "Subject not set"}
          </p>
        </div>
        <MissionStatus mission={mission} />
      </div>

      <p className="mt-3 line-clamp-2 text-sm leading-5 text-muted-foreground">
        {mission.learning_goal || "No learning goal provided"}
      </p>
      <div className="mt-3 flex items-center justify-between gap-3 text-xs text-muted-foreground">
        <AudienceLabel mission={mission} />
        <span>
          {formatDate(mission.updated_at, {
            fallback: "Not updated",
            preset: "date",
            relative: true,
          })}
        </span>
      </div>
      <div className="mt-3 border-t border-border pt-3">
        <MissionActions
          deleting={deleting}
          mission={mission}
          onDelete={onDelete}
          onEdit={onEdit}
        />
      </div>
    </article>
  );
}

function AudienceLabel({
  mission,
}: {
  readonly mission: TeacherClassMissionDto;
}) {
  const targetCount = mission.target_student_ids?.length ?? 0;

  if (mission.audience_type === "selected" && targetCount > 0) {
    return (
      <span className="text-xs font-medium text-muted-foreground">
        {targetCount} {targetCount === 1 ? "student" : "students"}
      </span>
    );
  }

  return (
    <span className="text-xs font-medium text-muted-foreground">Everyone</span>
  );
}

function MissionStatus({
  mission,
}: {
  readonly mission: TeacherClassMissionDto;
}) {
  const hasDraftChanges =
    missionHasPublishedRevision(mission) &&
    missionHasUnpublishedChanges(mission);
  const published = mission.status === "published" && !hasDraftChanges;
  const removed = mission.status === "deleted" || Boolean(mission.deleted_at);
  const label = removed
    ? "Removed"
    : hasDraftChanges
      ? "Changes not shared"
      : published
        ? "Shared"
        : "Draft";

  return (
    <span
      className={cn(
        "inline-flex shrink-0 rounded-full px-2.5 py-1 text-[11px] font-bold",
        removed
          ? "bg-[#e2e2e2] text-foreground"
          : hasDraftChanges
            ? "bg-[#c0a3ef]/25 text-[#6841c4]"
            : published
              ? "bg-[#9164f9] text-white"
              : "bg-[#d2c2e9]/35 text-[#6841c4]",
      )}
    >
      {label}
    </span>
  );
}

function MissionActions({
  deleting,
  mission,
  onDelete,
  onEdit,
}: {
  readonly deleting: boolean;
  readonly mission: TeacherClassMissionDto;
  readonly onDelete: (mission: TeacherClassMissionDto) => void;
  readonly onEdit: (mission: TeacherClassMissionDto) => void;
}) {
  const editLabel =
    missionHasUnpublishedChanges(mission) ||
    mission.status === "draft" ||
    mission.status === "preview"
      ? "Continue editing"
      : "Edit";

  return (
    <div className="flex justify-end gap-2">
      <Button
        type="button"
        size="sm"
        variant="outline"
        disabled={deleting}
        onClick={() => onEdit(mission)}
      >
        <Pencil className="size-3.5" aria-hidden />
        {editLabel}
      </Button>
      <Button
        type="button"
        size="sm"
        variant="ghost"
        disabled={deleting}
        aria-label={`Remove ${mission.mission_name || mission.title}`}
        className="text-muted-foreground hover:bg-[#d2c2e9]/25 hover:text-foreground"
        onClick={() => onDelete(mission)}
      >
        {deleting ? (
          <LoaderCircle className="size-3.5 animate-spin" aria-hidden />
        ) : (
          <Trash2 className="size-3.5" aria-hidden />
        )}
        <span className="md:sr-only">{deleting ? "Removing" : "Remove"}</span>
      </Button>
    </div>
  );
}

function formatDuration(minutes: number) {
  return minutes > 0 ? `${minutes} min` : "Not estimated";
}
