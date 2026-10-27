import type { AvatarPresetId } from "@/modules/game/host/avatars/avatar-presets";
import type { AvatarColors } from "@/modules/game/host/avatars/recolor-avatar";

/** A character to put in the room: one of the band, optionally in other colours. */
export type AvatarLook = {
  readonly presetId: AvatarPresetId;
  readonly colors?: Partial<AvatarColors>;
};

/**
 * The rest of the class. Each band member in colours the picker never shows, so
 * the room reads as other kids rather than copies of the menu — including Emily's
 * look-alike, a brunette in blue, so nobody mistakes her for the teacher.
 */
export const classmateLooks: readonly AvatarLook[] = [
  {
    presetId: "emily",
    colors: { hair: "#3f2a1d", top: "#2563eb", pants: "#1e293b" },
  },
  { presetId: "brad", colors: { top: "#c2410c", pants: "#1f2937" } },
  { presetId: "colin", colors: { top: "#15803d", pants: "#374151" } },
  { presetId: "dave", colors: { top: "#eab308", accent: "#b91c1c" } },
  { presetId: "joe", colors: { hair: "#8b5a2b", top: "#6d28d9" } },
  {
    presetId: "mike",
    colors: { top: "#0e7490", pants: "#3f3f46", shoes: "#18181b" },
  },
];
