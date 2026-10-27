"use client";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { TeacherClassDto } from "@/modules/classrooms";
import { InfoPanel } from "@/modules/insights/ui/section-heading";
import type { TeacherClassMissionDto } from "@/modules/missions";

export function ActiveMissionCard({
  activeMission,
  classCode,
  generatedByAi,
  teacherClass,
}: {
  readonly activeMission: TeacherClassMissionDto | null;
  readonly classCode: string;
  readonly generatedByAi: boolean;
  readonly teacherClass: TeacherClassDto | null;
}) {
  return (
    <Card className="gap-0 border-[#dad2e6] py-0">
      <CardHeader className="border-b border-[#dad2e6] bg-[#d2c2e9]/10 px-4 py-4 sm:px-6">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <p className="text-xs font-bold uppercase text-[#9164f9]">
              TODAY&apos;S PLAN
            </p>
            <CardTitle className="mt-2 text-2xl">
              {activeMission?.title ?? "No mission shared yet"}
            </CardTitle>
            <CardDescription className="mt-2 max-w-2xl leading-6">
              {teacherClass
                ? (activeMission?.summary ??
                  "Tell Mia what students should practice. You can review every question before sharing it with the class.")
                : "Create a class first. It will have its own code, missions, students, and notes."}
            </CardDescription>
          </div>
        </div>
      </CardHeader>

      <CardContent className="grid gap-4 px-4 py-5 sm:px-6">
        <div className="flex flex-wrap gap-2">
          <StatusBadge
            value={activeMission?.status ?? teacherClass?.status ?? "draft"}
          />
          {generatedByAi ? (
            <BadgeTone tone="violet">Built with Mia</BadgeTone>
          ) : null}
          <BadgeTone tone="soft">
            {activeMission?.subject ??
              teacherClass?.subject ??
              "No subject yet"}
          </BadgeTone>
        </div>

        {activeMission ? (
          <div className="grid gap-4 md:grid-cols-3">
            <InfoPanel
              label="Class"
              value={
                activeMission.class_name ||
                teacherClass?.class_name ||
                classCode
              }
            />
            <InfoPanel
              label="Grade"
              value={
                activeMission.grade_level ||
                teacherClass?.grade_level ||
                "Not set"
              }
            />
            <InfoPanel
              label="Suggested time"
              value={`${activeMission.suggested_duration_minutes} minutes`}
            />
          </div>
        ) : (
          <div className="rounded-xl border border-dashed border-[#c9b3ec] bg-[#d2c2e9]/15 p-4 text-sm leading-6 text-muted-foreground">
            {teacherClass
              ? "No mission has been shared yet. Create one, review it, and then share it with students."
              : "No class selected. Create a class, then prepare its first mission."}
          </div>
        )}

        {activeMission ? (
          <div className="grid gap-3">
            <h3 className="text-sm font-semibold">What students will do</h3>
            <div className="grid gap-2">
              {activeMission.stages.map((stage, index) => (
                <div
                  key={stage.id}
                  className="flex gap-3 rounded-xl border border-[#dad2e6] bg-[#d2c2e9]/10 p-3"
                >
                  <span className="grid size-7 shrink-0 place-items-center rounded-full bg-[#9164f9] text-xs font-bold text-white">
                    {index + 1}
                  </span>
                  <div>
                    <p className="font-semibold">{stage.title}</p>
                    <p className="mt-1 text-sm leading-6 text-muted-foreground">
                      {stage.goal}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}

export function ClassCodeBadge({
  compact = false,
  value,
}: {
  readonly compact?: boolean;
  readonly value: string;
}) {
  if (compact) {
    return (
      <div className="flex items-center gap-2 rounded-xl border border-[#c9b3ec] bg-[#d2c2e9]/20 px-3 py-1.5 text-sm">
        <span className="text-[11px] font-bold uppercase text-primary">
          Code
        </span>
        <span className="font-mono text-base font-black text-[#6841c4]">
          {value}
        </span>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-[#c9b3ec] bg-[#d2c2e9]/20 px-3 py-2 text-sm">
      <span className="block text-xs font-bold uppercase text-primary">
        Class code
      </span>
      <span className="mt-1 block font-mono text-lg font-black text-[#6841c4]">
        {value}
      </span>
    </div>
  );
}

export function StatusBadge({ value }: { readonly value: string }) {
  return (
    <BadgeTone tone={value === "published" ? "violet" : "muted"}>
      {statusLabel(value)}
    </BadgeTone>
  );
}

export function BadgeTone({
  children,
  tone,
}: {
  readonly children: string;
  readonly tone: "muted" | "soft" | "violet";
}) {
  const classNameByTone = {
    muted: "bg-[#e2e2e2] text-foreground",
    soft: "bg-[#d2c2e9]/35 text-[#6841c4]",
    violet: "bg-[#9164f9] text-white",
  };

  return (
    <span
      className={`w-fit rounded-md px-2.5 py-1 text-xs font-bold uppercase ${classNameByTone[tone]}`}
    >
      {children}
    </span>
  );
}

export function statusLabel(value: string) {
  if (value === "published") {
    return "Shared with students";
  }

  if (value === "preview") {
    return "Ready to review";
  }

  if (value === "draft") {
    return "Draft";
  }

  return value;
}
