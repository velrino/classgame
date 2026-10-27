"use client";
import { defaultAvatarPresetId } from "@/modules/game/host/avatars/avatar-presets";
import {
  readStoredAvatarPresetId,
  storeAvatarPresetId,
  subscribeStoredAvatarPresetId,
} from "@/modules/game/host/avatars/avatar-storage";
import { useSyncExternalStore } from "react";
export function useAvatarPreference() {
  const avatarPresetId = useSyncExternalStore(
    subscribeStoredAvatarPresetId,
    readStoredAvatarPresetId,
    () => defaultAvatarPresetId,
  );
  return { avatarPresetId, selectAvatar: storeAvatarPresetId };
}
