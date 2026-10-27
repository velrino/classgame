"use client";

import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  LabelList,
  ResponsiveContainer,
  XAxis,
  YAxis,
} from "recharts";

import type { FunnelStep } from "@/modules/insights/domain/analytics";
import { StudentChip } from "@/modules/insights/ui/overview/student-chip";

type MissionFunnelProps = {
  readonly steps: FunnelStep[];
  readonly totalStudents: number;
};

const STAGE_COLORS = [
  "#9164f9",
  "#9e74f7",
  "#aa84f4",
  "#b593f2",
  "#c0a3ef",
  "#c9b3ec",
  "#d2c2e9",
];

const MAX_VISIBLE_CHIPS = 5;

export function MissionFunnel({ steps, totalStudents }: MissionFunnelProps) {
  if (totalStudents === 0) {
    return (
      <div className="grid min-h-64 place-content-center rounded-xl border border-dashed border-[#c9b3ec] bg-[#d2c2e9]/15 px-6 text-center">
        <p className="font-semibold">No student progress yet</p>
        <p className="mt-1 max-w-md text-sm leading-6 text-muted-foreground">
          Share the class code. This chart will fill in as students move through
          the mission.
        </p>
      </div>
    );
  }

  const chartData = steps.map((step) => ({
    id: step.id,
    label: step.label,
    students: step.students.length,
  }));
  const occupiedSteps = steps.filter((step) => step.students.length > 0);

  return (
    <div className="min-w-0">
      <div
        className="h-[320px] w-full min-w-0"
        role="img"
        aria-label={`Bar chart showing the current mission stage for ${totalStudents} students`}
      >
        <ResponsiveContainer width="100%" height="100%">
          <BarChart
            accessibilityLayer
            data={chartData}
            layout="vertical"
            margin={{ top: 4, right: 30, bottom: 12, left: 8 }}
          >
            <CartesianGrid
              horizontal={false}
              stroke="#e2e2e2"
              strokeDasharray="3 3"
            />
            <XAxis
              type="number"
              allowDecimals={false}
              axisLine={false}
              tickLine={false}
              tick={{ fill: "#64748b", fontSize: 11 }}
            />
            <YAxis
              type="category"
              dataKey="label"
              axisLine={false}
              tickLine={false}
              width={116}
              tick={{ fill: "currentColor", fontSize: 11 }}
            />
            <Bar
              dataKey="students"
              radius={[0, 8, 8, 0]}
              barSize={22}
              isAnimationActive={false}
            >
              {steps.map((step, index) => (
                <Cell
                  key={step.id}
                  fill={
                    step.kind === "not_started"
                      ? "#e2e2e2"
                      : step.kind === "completed"
                        ? "#c9b3ec"
                        : STAGE_COLORS[index % STAGE_COLORS.length]
                  }
                />
              ))}
              <LabelList
                dataKey="students"
                position="right"
                className="fill-foreground text-xs font-semibold"
              />
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>

      <div className="mt-3 border-t border-[#dad2e6] pt-4">
        <p className="text-xs font-semibold text-muted-foreground">
          WHO IS WHERE
        </p>
        <div className="mt-3 grid gap-2 sm:grid-cols-2">
          {occupiedSteps.map((step) => {
            const hiddenCount = Math.max(
              0,
              step.students.length - MAX_VISIBLE_CHIPS,
            );

            return (
              <div key={step.id} className="rounded-xl bg-[#d2c2e9]/20 p-3">
                <div className="flex items-center justify-between gap-3">
                  <p className="text-sm font-semibold">{step.label}</p>
                  <span className="shrink-0 text-xs text-muted-foreground">
                    {step.students.length}
                  </span>
                </div>
                <div className="mt-2 flex flex-wrap gap-1.5">
                  {step.students.slice(0, MAX_VISIBLE_CHIPS).map((student) => (
                    <StudentChip key={student.id} name={student.name} />
                  ))}
                  {hiddenCount > 0 ? (
                    <span
                      className="inline-flex items-center rounded-md bg-card px-2 py-1 text-xs font-semibold text-muted-foreground"
                      title={step.students
                        .slice(MAX_VISIBLE_CHIPS)
                        .map((student) => student.name)
                        .join(", ")}
                    >
                      +{hiddenCount} more
                    </span>
                  ) : null}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
