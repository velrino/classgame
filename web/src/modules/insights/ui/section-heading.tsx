"use client";

export function SectionHeading({
  detail,
  eyebrow,
  source,
  title,
}: {
  readonly detail: string;
  readonly eyebrow: string;
  readonly source?: "estimated" | "live";
  readonly title: string;
}) {
  return (
    <div className="flex flex-col gap-1 pt-3 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <p className="text-[11px] font-bold tracking-wide text-muted-foreground uppercase">
          {eyebrow}
        </p>
        <h2 className="mt-1 text-xl font-semibold tracking-tight">{title}</h2>
        <p className="mt-1 max-w-3xl text-sm leading-6 text-muted-foreground">
          {detail}
        </p>
      </div>
      {source ? <SourceBadge source={source} /> : null}
    </div>
  );
}

export function SourceBadge({
  source,
}: {
  readonly source: "estimated" | "live";
}) {
  const label = {
    estimated: "From student work",
    live: "Updates automatically",
  }[source];
  const className = {
    estimated: "bg-[#d2c2e9]/35 text-[#6841c4]",
    live: "bg-[#c0a3ef]/25 text-[#6841c4]",
  }[source];

  return (
    <span
      className={`w-fit shrink-0 rounded-md px-2 py-1 text-[11px] font-semibold ${className}`}
    >
      {label}
    </span>
  );
}

export function InfoPanel({
  label,
  value,
}: {
  readonly label: string;
  readonly value: string;
}) {
  return (
    <div className="rounded-xl border border-[#dad2e6] bg-[#d2c2e9]/10 p-3">
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <p className="mt-1 text-sm font-semibold">{value}</p>
    </div>
  );
}
