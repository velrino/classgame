"use client";
import { Button } from "@/components/ui/button";
import { type ClassGlance } from "@/modules/insights/domain/analytics";
import type { TeacherClassMissionDto } from "@/modules/missions";
import {
  ArrowRight,
  BookOpenCheck,
  CircleAlert,
  ClipboardCopy,
  Gamepad2,
  PartyPopper,
  Sparkles,
} from "lucide-react";

export function TodayGuide({
  activeMission,
  classCode,
  glance,
  hasSelectedClass,
  onCreateMission,
  onOpenStudents,
  onPlanFollowUp,
  onShareCode,
  teacherName,
}: {
  readonly activeMission: TeacherClassMissionDto | null;
  readonly classCode: string;
  readonly glance: ClassGlance;
  readonly hasSelectedClass: boolean;
  readonly onCreateMission: () => void;
  readonly onOpenStudents: () => void;
  readonly onPlanFollowUp: () => void;
  readonly onShareCode: () => void;
  readonly teacherName: string;
}) {
  const firstName = teacherName.trim().split(/\s+/)[0] || "there";
  const everyoneFinished =
    glance.total > 0 && glance.completed.length === glance.total;

  let guide = {
    detail:
      "Create a class, then Mia will help you prepare a mission for your students.",
    icon: BookOpenCheck,
    label: "Create a class",
    onClick: onCreateMission,
    title: "Start with your first class",
  };

  if (hasSelectedClass && !activeMission) {
    guide = {
      detail:
        "Tell Mia the subject and what you want students to learn. You will review the mission before anyone sees it.",
      icon: Sparkles,
      label: "Create today's mission",
      onClick: onCreateMission,
      title: "Let's prepare something for today",
    };
  } else if (activeMission && glance.total === 0) {
    guide = {
      detail: `Share code ${classCode || "from your class"}. Student progress will appear here automatically when they join.`,
      icon: ClipboardCopy,
      label: "Copy class code",
      onClick: onShareCode,
      title: "Your mission is ready — invite your students",
    };
  } else if (glance.needsAttention.length > 0) {
    const count = glance.needsAttention.length;
    guide = {
      detail:
        "Open the student list to see names, recent work, and where a quick conversation may help.",
      icon: CircleAlert,
      label: "See who needs help",
      onClick: onOpenStudents,
      title: `${count} ${count === 1 ? "student may" : "students may"} need a check-in`,
    };
  } else if (everyoneFinished) {
    guide = {
      detail:
        "Use what Mia noticed to choose a review, a whole-class activity, or an extra challenge.",
      icon: PartyPopper,
      label: "Plan the next lesson",
      onClick: onPlanFollowUp,
      title: "Everyone finished — nice work!",
    };
  } else if (glance.playingNow.length > 0) {
    guide = {
      detail:
        "The charts below update as students work. You can keep teaching and check back whenever you need to.",
      icon: Gamepad2,
      label: "View student details",
      onClick: onOpenStudents,
      title: "Your class is working right now",
    };
  } else if (activeMission) {
    guide = {
      detail:
        "The mission is published. Share the code when class starts, then progress will update here automatically.",
      icon: ClipboardCopy,
      label: "Share class code",
      onClick: onShareCode,
      title: "Everything is ready for your students",
    };
  }

  const GuideIcon = guide.icon;

  return (
    <section className="relative overflow-hidden rounded-2xl border border-[#c0a3ef] bg-gradient-to-br from-[#9164f9] via-[#9e74f7] to-[#b593f2] p-5 text-white shadow-sm sm:p-6">
      <div
        className="pointer-events-none absolute -top-20 -right-12 size-56 rounded-full border-[32px] border-white/10"
        aria-hidden
      />
      <div className="relative flex flex-col gap-5 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex max-w-3xl items-start gap-4">
          <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-white/18 ring-1 ring-white/25">
            <GuideIcon className="size-5" aria-hidden />
          </span>
          <div>
            <p className="text-xs font-semibold text-white/75">
              HI {firstName.toLocaleUpperCase("en")} · HERE&apos;S WHAT MATTERS
              TODAY
            </p>
            <h2 className="mt-1 text-xl font-bold tracking-tight sm:text-2xl">
              {guide.title}
            </h2>
            <p className="mt-2 max-w-2xl text-sm leading-6 text-white/85">
              {guide.detail}
            </p>
          </div>
        </div>
        <Button
          type="button"
          onClick={guide.onClick}
          className="shrink-0 bg-white text-[#6841c4] hover:bg-[#e2e2e2]"
        >
          {guide.label}
          <ArrowRight className="size-4" aria-hidden />
        </Button>
      </div>
    </section>
  );
}

export function DashboardLoadingState() {
  return (
    <div
      className="space-y-5"
      aria-label="Loading class dashboard"
      aria-busy="true"
    >
      <div className="h-40 animate-pulse rounded-2xl bg-[#d2c2e9]/45" />
      <div className="grid gap-5 xl:grid-cols-2">
        <div className="h-96 animate-pulse rounded-xl bg-[#e2e2e2]" />
        <div className="h-96 animate-pulse rounded-xl bg-[#e2e2e2]" />
      </div>
    </div>
  );
}
