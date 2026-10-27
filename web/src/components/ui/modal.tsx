"use client";

import { X } from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useId } from "react";
import { createPortal } from "react-dom";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type ModalProps = {
  readonly children: ReactNode;
  readonly description?: string;
  /**
   * `auto` (default): the modal grows with its content and the body scrolls.
   * `fill`: the modal takes the viewport height and the body becomes a flex
   * column that children own — use it for panels with pinned toolbars.
   */
  readonly layout?: "auto" | "fill";
  readonly open: boolean;
  readonly size?: "md" | "lg" | "xl";
  readonly title: string;
  readonly onOpenChange: (open: boolean) => void;
};

const sizeClassName = {
  md: "max-w-lg",
  lg: "max-w-3xl",
  xl: "max-w-6xl",
};

const layoutClassName = {
  auto: {
    dialog: "max-h-[calc(100svh-2rem)]",
    body: "overflow-y-auto p-4 sm:p-6",
  },
  fill: {
    dialog: "h-[calc(100svh-2rem)] max-h-[60rem]",
    body: "flex flex-col overflow-hidden",
  },
};

export function Modal({
  children,
  description,
  layout = "auto",
  open,
  onOpenChange,
  size = "lg",
  title,
}: ModalProps) {
  const titleId = useId();
  const descriptionId = useId();

  useEffect(() => {
    if (!open) {
      return;
    }

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        onOpenChange(false);
      }
    }

    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [onOpenChange, open]);

  if (!open || typeof document === "undefined") {
    return null;
  }

  return createPortal(
    <div className="fixed inset-0 z-50 grid place-items-center bg-slate-950/45 p-4 backdrop-blur-sm">
      <section
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descriptionId : undefined}
        className={cn(
          "dashboard-theme flex w-full flex-col overflow-hidden rounded-lg border border-border bg-background text-foreground shadow-2xl",
          layoutClassName[layout].dialog,
          sizeClassName[size],
        )}
      >
        <header className="flex shrink-0 items-start gap-4 border-b border-border px-4 py-4 sm:px-6">
          <div className="min-w-0 flex-1">
            <h2 id={titleId} className="text-lg font-semibold">
              {title}
            </h2>
            {description ? (
              <p
                id={descriptionId}
                className="mt-1 text-sm leading-6 text-muted-foreground"
              >
                {description}
              </p>
            ) : null}
          </div>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label="Close modal"
            onClick={() => onOpenChange(false)}
          >
            <X className="size-4" aria-hidden />
          </Button>
        </header>

        <div className={cn("min-h-0 flex-1", layoutClassName[layout].body)}>
          {children}
        </div>
      </section>
    </div>,
    document.body,
  );
}
