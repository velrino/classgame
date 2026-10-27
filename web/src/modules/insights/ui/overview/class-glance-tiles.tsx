"use client";

import { ArrowRight, Gamepad2, RotateCcw, UsersRound } from "lucide-react";
import { Cell, Pie, PieChart, ResponsiveContainer } from "recharts";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

import type { ClassGlance } from "@/modules/insights/domain/analytics";

type ClassGlanceTilesProps = {
  readonly glance: ClassGlance;
  readonly shareCodeLabel: string;
  readonly onPlanFollowUp: () => void;
  readonly onShareCode: () => void;
};

const STATUS_COLORS = ["#e2e2e2", "#b593f2", "#9164f9", "#c9b3ec"];

export function ClassGlanceTiles({
  glance,
  onPlanFollowUp,
  onShareCode,
  shareCodeLabel,
}: ClassGlanceTilesProps) {
  const statusData = [
    { label: "Not started", value: glance.notStarted.length },
    { label: "Working on it", value: glance.inProgress.length },
    { label: "May need support", value: glance.needsAttention.length },
    { label: "Finished", value: glance.completed.length },
  ];

  return (
    <Card className="min-w-0 gap-0 overflow-hidden py-0">
      <CardHeader className="border-b border-[#dad2e6] bg-[#d2c2e9]/20 px-5 py-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p className="text-xs font-semibold text-[#9164f9]">RIGHT NOW</p>
            <CardTitle className="mt-1">How the class is doing</CardTitle>
          </div>
          <div className="flex items-center gap-2 rounded-full bg-card px-3 py-1.5 text-xs font-semibold text-foreground shadow-sm">
            <span className="size-2 rounded-full bg-[#9164f9]" aria-hidden />
            {glance.playingNow.length} playing now
          </div>
        </div>
      </CardHeader>

      <CardContent className="grid min-w-0 gap-5 p-5 md:grid-cols-[190px_minmax(0,1fr)] md:items-center">
        {glance.total > 0 ? (
          <div
            className="relative mx-auto size-44"
            aria-label={`${glance.total} students in this class`}
          >
            <ResponsiveContainer width="100%" height="100%">
              <PieChart accessibilityLayer>
                <Pie
                  data={statusData}
                  dataKey="value"
                  innerRadius={58}
                  outerRadius={82}
                  paddingAngle={2}
                  stroke="none"
                  isAnimationActive={false}
                >
                  {statusData.map((item, index) => (
                    <Cell key={item.label} fill={STATUS_COLORS[index]} />
                  ))}
                </Pie>
              </PieChart>
            </ResponsiveContainer>
            <div className="pointer-events-none absolute inset-0 grid place-content-center text-center">
              <strong className="text-3xl leading-none">{glance.total}</strong>
              <span className="mt-1 text-xs text-muted-foreground">
                {glance.total === 1 ? "student" : "students"}
              </span>
            </div>
          </div>
        ) : (
          <div className="mx-auto grid size-40 place-content-center rounded-full border-8 border-[#e2e2e2] text-center">
            <UsersRound className="mx-auto size-6 text-[#9164f9]" aria-hidden />
            <span className="mt-2 text-sm font-semibold">No students yet</span>
          </div>
        )}

        <div className="min-w-0">
          <div className="grid gap-2 sm:grid-cols-2">
            {statusData.map((item, index) => (
              <div
                key={item.label}
                className="flex items-center gap-3 rounded-xl border border-[#dad2e6] bg-card p-3"
              >
                <span
                  className="size-3 shrink-0 rounded-full ring-4 ring-[#d2c2e9]/25"
                  style={{ backgroundColor: STATUS_COLORS[index] }}
                  aria-hidden
                />
                <div className="min-w-0">
                  <p className="text-xl font-bold leading-none">{item.value}</p>
                  <p className="mt-1 truncate text-xs text-muted-foreground">
                    {item.label}
                  </p>
                </div>
              </div>
            ))}
          </div>

          <div className="mt-4 flex flex-wrap gap-2 text-xs text-muted-foreground">
            <span className="inline-flex items-center gap-1.5 rounded-full bg-[#d2c2e9]/30 px-3 py-1.5">
              <Gamepad2 className="size-3.5 text-[#9164f9]" aria-hidden />
              {glance.atTheDesk.length} learning with Mia
            </span>
            <span className="inline-flex items-center gap-1.5 rounded-full bg-[#d2c2e9]/30 px-3 py-1.5">
              <RotateCcw className="size-3.5 text-[#9164f9]" aria-hidden />
              {glance.inReview.length} reviewing a topic
            </span>
          </div>

          <div className="mt-4 flex flex-wrap gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={onShareCode}
            >
              {shareCodeLabel}
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={onPlanFollowUp}
            >
              Plan the next lesson
              <ArrowRight className="size-3.5" aria-hidden />
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
