import {
  defaultAvatarPresetId,
  getAvatarPreset,
  teacherAvatarPresetId,
} from "@/modules/game/host/avatars/avatar-presets";

const STORAGE_KEY = "classgame.avatar.v1";
const CHANGE_EVENT = "classgame-avatar-change";

let fallbackPresetId: string = defaultAvatarPresetId;

function sanitize(id: string | null | undefined) {
  const preset = getAvatarPreset(id);
  return preset.id === teacherAvatarPresetId
    ? defaultAvatarPresetId
    : preset.id;
}

export function readStoredAvatarPresetId() {
  if (typeof window === "undefined") {
    return defaultAvatarPresetId;
  }

  try {
    return sanitize(window.localStorage.getItem(STORAGE_KEY));
  } catch {
    return fallbackPresetId;
  }
}

export function storeAvatarPresetId(presetId: string) {
  const stored = sanitize(presetId);
  fallbackPresetId = stored;

  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(STORAGE_KEY, stored);
  } catch {
    // Storage is a convenience; the game keeps working without it.
  }

  window.dispatchEvent(new Event(CHANGE_EVENT));
}

/** For useSyncExternalStore: re-read when this tab or another one changes the avatar. */
export function subscribeStoredAvatarPresetId(onStoreChange: () => void) {
  if (typeof window === "undefined") {
    return () => {};
  }

  const handleStorage = (event: StorageEvent) => {
    if (event.key === STORAGE_KEY) {
      onStoreChange();
    }
  };

  window.addEventListener("storage", handleStorage);
  window.addEventListener(CHANGE_EVENT, onStoreChange);

  return () => {
    window.removeEventListener("storage", handleStorage);
    window.removeEventListener(CHANGE_EVENT, onStoreChange);
  };
}
