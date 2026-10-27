"use client";
import { useCallback } from "react";
import { useAvatarPreference } from "@/modules/game/adapters/hooks/use-avatar-preference";
import { useWorldBridge } from "@/modules/game/hooks/use-world-bridge";
import { useLearningSession } from "@/modules/learning";
export function useGameSession() {
  const {
    session,
    handlers,
    recoveryError,
    evidenceState,
    completionAcknowledged,
    retryEvidence,
    deleteUnsentEvidence,
  } = useLearningSession();
  const { avatarPresetId, selectAvatar } = useAvatarPreference();
  const { report, directiveRef, onReport, resetReport } = useWorldBridge(
    session,
    avatarPresetId,
  );
  const leaveMission = useCallback(() => {
    handlers.leaveMission();
    resetReport();
  }, [handlers, resetReport]);
  return {
    session,
    report,
    directiveRef,
    avatarPresetId,
    recoveryError,
    evidenceState,
    completionAcknowledged,
    retryEvidence,
    deleteUnsentEvidence,
    handlers: { ...handlers, leaveMission, selectAvatar, onReport },
  };
}
