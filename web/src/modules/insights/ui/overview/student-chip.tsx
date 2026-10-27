import { cn } from "@/lib/utils";

type StudentChipProps = {
  readonly className?: string;
  readonly name: string;
};

export function StudentChip({ className, name }: StudentChipProps) {
  return (
    <span
      className={cn(
        "inline-flex max-w-full items-center gap-1.5 rounded-md border border-border bg-card py-1 pe-2.5 ps-1 text-xs font-medium text-card-foreground shadow-xs",
        className,
      )}
      title={name}
    >
      <span className="grid size-5 shrink-0 place-items-center rounded-sm bg-muted text-[10px] font-bold text-muted-foreground">
        {studentInitials(name)}
      </span>
      <span className="truncate">{name}</span>
    </span>
  );
}

export function studentInitials(value: string) {
  const parts = value.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return "?";
  }
  if (parts.length === 1) {
    return parts[0].slice(0, 2).toUpperCase();
  }

  return `${parts[0][0] ?? ""}${parts[parts.length - 1]?.[0] ?? ""}`.toUpperCase();
}
