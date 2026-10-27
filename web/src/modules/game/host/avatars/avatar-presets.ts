import type { AvatarPreset } from "@/modules/game/adapters/types/avatar";

// Shared chibi avatar asset presets.
export const avatarPresets = [
  {
    id: "brad",
    label: "Brad",
    accessoryStyle: "hoodie",
    skin: "#be7d58",
    hair: "#151515",
    beard: "#171717",
    beardStyle: "full",
    eye: "#0b0b0d",
    top: "#1d1e21",
    pants: "#242529",
    shoes: "#151619",
    accent: "#3b3c42",
    sleeveStyle: "long",
    tattooStyle: "none",
    hairStyle: "full-curly",
  },
  {
    id: "colin",
    label: "Colin",
    accessoryStyle: "none",
    skin: "#d69a72",
    hair: "#261711",
    beard: "#24160f",
    beardStyle: "short",
    eye: "#0b0b0d",
    top: "#161719",
    pants: "#1f2225",
    shoes: "#101114",
    accent: "#2c2f34",
    sleeveStyle: "sleeveless",
    tattooStyle: "left-arm",
    hairStyle: "curly",
  },
  {
    id: "dave",
    label: "Dave",
    accessoryStyle: "cap",
    skin: "#edc09b",
    hair: "#141414",
    beard: "#8a4f28",
    beardStyle: "full",
    eye: "#0b0b0d",
    top: "#e7e2dc",
    pants: "#18191b",
    shoes: "#f6f5f0",
    accent: "#0f1012",
    sleeveStyle: "sleeveless",
    tattooStyle: "both-arms",
    hairStyle: "cap",
  },
  {
    id: "emily",
    label: "Emily",
    accessoryStyle: "striped",
    skin: "#edbd95",
    hair: "#f0c981",
    beard: "#f0c981",
    beardStyle: "none",
    eye: "#0b0b0d",
    top: "#a62832",
    pants: "#28292d",
    shoes: "#111214",
    accent: "#111214",
    sleeveStyle: "long",
    tattooStyle: "none",
    hairStyle: "long-waves",
  },
  {
    id: "joe",
    label: "Joe",
    accessoryStyle: "jacket",
    skin: "#e3b083",
    hair: "#2a1713",
    beard: "#2a1713",
    beardStyle: "none",
    eye: "#0b0b0d",
    top: "#556a47",
    pants: "#202124",
    shoes: "#17181b",
    accent: "#b8c279",
    sleeveStyle: "long",
    tattooStyle: "none",
    hairStyle: "center-part",
  },
  {
    id: "mike",
    label: "Mike",
    accessoryStyle: "camo",
    skin: "#d4a178",
    hair: "#141112",
    beard: "#181211",
    beardStyle: "goatee",
    eye: "#0b0b0d",
    top: "#17181c",
    pants: "#68613e",
    shoes: "#f3f1e8",
    accent: "#d23d34",
    sleeveStyle: "long",
    tattooStyle: "none",
    hairStyle: "buzz",
  },
] as const satisfies readonly AvatarPreset[];

export type AvatarPresetId = (typeof avatarPresets)[number]["id"];

/** Emily is the teacher, so she isn't on the student's menu. */
export const teacherAvatarPresetId = "emily" satisfies AvatarPresetId;

/** Pinned to a preset rather than a position, so reordering the list never changes who a new player spawns as. */
export const defaultAvatarPresetId = "joe" satisfies AvatarPresetId;

export const studentAvatarPresets = avatarPresets.filter(
  (preset) => preset.id !== teacherAvatarPresetId,
);

export function getAvatarPreset(id: string | null | undefined): AvatarPreset {
  return (
    avatarPresets.find((preset) => preset.id === id) ??
    avatarPresets.find((preset) => preset.id === defaultAvatarPresetId)!
  );
}

/** Head-and-shoulders render generated alongside the GLB, so the picker shows the face that actually spawns. */
export function getAvatarPortraitPath(preset: AvatarPreset) {
  return `/assets/avatars/chibi-band/portraits/${preset.id}.png`;
}
