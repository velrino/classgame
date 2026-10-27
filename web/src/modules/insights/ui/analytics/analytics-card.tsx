import type { ReactNode } from "react";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { cn } from "@/lib/utils";

export type AnalyticsSource = "estimated" | "live";

type AnalyticsCardProps = {
  readonly children: ReactNode;
  readonly className?: string;
  readonly contentClassName?: string;
  readonly description: string;
  readonly source?: AnalyticsSource;
  readonly title: string;
};

const sourceLabels: Record<AnalyticsSource, string> = {
  estimated: "From student work",
  live: "Updates automatically",
};

const sourceStyles: Record<AnalyticsSource, string> = {
  estimated: "bg-[#d2c2e9]/35 text-[#6841c4]",
  live: "bg-[#c0a3ef]/25 text-[#6841c4]",
};

export function AnalyticsCard({
  children,
  className,
  contentClassName,
  description,
  source,
  title,
}: AnalyticsCardProps) {
  return (
    <Card className={cn("min-w-0 border-[#dad2e6]", className)}>
      <CardHeader className="gap-2 border-b border-[#dad2e6] bg-[#d2c2e9]/10">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <CardTitle>{title}</CardTitle>
          {source ? (
            <span
              className={cn(
                "rounded-md px-2 py-1 text-[11px] font-semibold",
                sourceStyles[source],
              )}
            >
              {sourceLabels[source]}
            </span>
          ) : null}
        </div>
        <CardDescription className="max-w-3xl leading-5">
          {description}
        </CardDescription>
      </CardHeader>
      <CardContent className={cn("min-w-0", contentClassName)}>
        {children}
      </CardContent>
    </Card>
  );
}
