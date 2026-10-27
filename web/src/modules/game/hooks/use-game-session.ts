"use client";
import { useAvatarPreference } from "@/modules/game/adapters/hooks/use-avatar-preference";
import { useWorldBridge } from "@/modules/game/hooks/use-world-bridge";
import { useLearningSession } from "@/modules/learning";
export function useGameSession() {
  const { session, handlers } = useLearningSession();
  const { avatarPresetId, selectAvatar } = useAvatarPreference();
  const { report, directiveRef, onReport } = useWorldBridge(
    session,
    avatarPresetId,
  );
  return {
    session,
    report,
    directiveRef,
    avatarPresetId,
    handlers: { ...handlers, selectAvatar, onReport },
  };
}
