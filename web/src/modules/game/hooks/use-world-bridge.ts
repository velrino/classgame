"use client";
import type {
  WorldDirective,
  WorldReport,
} from "@/modules/game/adapters/types/world";
import type { Session } from "@/modules/learning";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
const idleReport: WorldReport = {
  nearSeat: false,
  walking: false,
  ready: false,
};
export function useWorldBridge(session: Session, avatarPresetId: string) {
  const [report, setReport] = useState<WorldReport>(idleReport);
  const directiveRef = useRef<WorldDirective>({
    podStatus: {
      fractions: "ready",
      "simple-equations": "locked",
      "two-step-equations": "locked",
    },
    tutorPod: "fractions",
    tutorThinking: false,
    seated: false,
    walkRequest: null,
    avatarPresetId,
  });
  const directive: WorldDirective = useMemo(
    () => ({
      podStatus: {
        fractions: session.memory.progress.fractions.status,
        "simple-equations": session.memory.progress["simple-equations"].status,
        "two-step-equations":
          session.memory.progress["two-step-equations"].status,
      },
      tutorPod: session.tutorPod,
      tutorThinking: session.lesson?.phase === "thinking",
      seated: session.seatedAt !== null,
      walkRequest: session.walkRequest,
      avatarPresetId,
    }),
    [
      avatarPresetId,
      session.lesson?.phase,
      session.memory.progress,
      session.seatedAt,
      session.tutorPod,
      session.walkRequest,
    ],
  );

  useEffect(() => {
    directiveRef.current = directive;
  }, [directive]);

  const onReport = useCallback((next: WorldReport) => setReport(next), []);
  return { report, directiveRef, onReport };
}
