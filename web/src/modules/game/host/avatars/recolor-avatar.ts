import * as THREE from "three";

import type { AvatarPreset } from "@/modules/game/adapters/types/avatar";

/** The preset colours a look can override. */
export type AvatarColors = Pick<
  AvatarPreset,
  "skin" | "hair" | "beard" | "eye" | "top" | "pants" | "shoes" | "accent"
>;
export type AvatarColorKey = keyof AvatarColors;

/**
 * Where each part lives on the authored body (metres, y up, soles at 0), measured
 * off the GLB. Dark preset colours (a #151515 hair next to #151619 shoes) collapse
 * together under the GLB's 8-bit colour quantisation, so a vertex is matched only
 * against the parts that can exist at its height. Skin shows everywhere.
 */
const CAP_Y = 1.22; // cap from 1.233 up; eyes end at 1.215
const EYE_Y = 1.03; // eyes 1.031–1.215; beard ends at 1.021
const BEARD_Y = 0.92; // beard 0.871–1.021; the top's collar ends at 0.915
const COLLAR_Y = 0.86;
const HEM_Y = 0.63; // pants end at 0.621
const KNEE_Y = 0.48; // top starts at 0.482
const SHOE_Y = 0.12; // shoes end at 0.109; pants reach the floor
/** Modelled hair hangs below the neck (Emily's to 0.58 m); derived hair shells never do. */
const LONG_HAIR_STYLES: ReadonlySet<AvatarPreset["hairStyle"]> = new Set([
  "long-waves",
  "center-part",
]);
/** The modelled cap is painted this fixed colour rather than the preset's accent. */
const CAP_COLOR = "#0d0d10";

const CROWN_PARTS: readonly AvatarColorKey[] = ["skin", "hair"];
const CAPPED_CROWN_PARTS: readonly AvatarColorKey[] = [
  "skin",
  "hair",
  "accent",
];
const EYE_PARTS: readonly AvatarColorKey[] = ["skin", "hair", "eye"];
const BEARD_PARTS: readonly AvatarColorKey[] = ["skin", "hair", "beard"];
const COLLAR_PARTS: readonly AvatarColorKey[] = ["skin", "top", "beard"];
const LONG_HAIR_COLLAR_PARTS: readonly AvatarColorKey[] = [
  ...COLLAR_PARTS,
  "hair",
];
const TORSO_PARTS: readonly AvatarColorKey[] = ["skin", "top"];
const LONG_HAIR_TORSO_PARTS: readonly AvatarColorKey[] = [
  ...TORSO_PARTS,
  "hair",
];
const HEM_PARTS: readonly AvatarColorKey[] = ["skin", "top", "pants"];
const LONG_HAIR_HEM_PARTS: readonly AvatarColorKey[] = [...HEM_PARTS, "hair"];
const LEG_PARTS: readonly AvatarColorKey[] = ["skin", "pants"];
const FOOT_PARTS: readonly AvatarColorKey[] = ["skin", "pants", "shoes"];

type Swatch = {
  readonly source: THREE.Color;
  readonly target: THREE.Color | null;
};

function partsAt(y: number, preset: AvatarPreset): readonly AvatarColorKey[] {
  const longHair = LONG_HAIR_STYLES.has(preset.hairStyle);

  if (y >= CAP_Y) {
    return preset.accessoryStyle === "cap" ? CAPPED_CROWN_PARTS : CROWN_PARTS;
  }
  if (y >= EYE_Y) {
    return EYE_PARTS;
  }
  if (y >= BEARD_Y) {
    return BEARD_PARTS;
  }
  if (y >= COLLAR_Y) {
    return longHair ? LONG_HAIR_COLLAR_PARTS : COLLAR_PARTS;
  }
  if (y >= HEM_Y) {
    return longHair ? LONG_HAIR_TORSO_PARTS : TORSO_PARTS;
  }
  if (y >= KNEE_Y) {
    return longHair ? LONG_HAIR_HEM_PARTS : HEM_PARTS;
  }
  return y >= SHOE_Y ? LEG_PARTS : FOOT_PARTS;
}

/** Baked colour of each part, and what to repaint it with. Colours compare in linear space, like the vertex data. */
function buildSwatches(preset: AvatarPreset, colors: Partial<AvatarColors>) {
  const swatches = {} as Record<AvatarColorKey, Swatch>;
  const keys: readonly AvatarColorKey[] = [
    "skin",
    "hair",
    "beard",
    "eye",
    "top",
    "pants",
    "shoes",
    "accent",
  ];

  for (const key of keys) {
    const baked = key === "accent" ? CAP_COLOR : preset[key];
    const wanted = colors[key];
    swatches[key] = {
      source: new THREE.Color(baked),
      target: wanted ? new THREE.Color(wanted) : null,
    };
  }

  return swatches;
}

function distanceSquared(a: THREE.Color, r: number, g: number, b: number) {
  return (a.r - r) ** 2 + (a.g - g) ** 2 + (a.b - b) ** 2;
}

/**
 * Repaint a chibi in other colours. Every mesh gets its own copy of the vertex
 * colours (the geometry otherwise stays shared across avatars), so the result
 * is one character in a different outfit, at no extra draw calls.
 */
export function recolorAvatar(
  model: THREE.Object3D,
  preset: AvatarPreset,
  colors: Partial<AvatarColors>,
) {
  const swatches = buildSwatches(preset, colors);
  const repainted = new Set<AvatarColorKey>();

  model.traverse((child) => {
    if (!(child instanceof THREE.Mesh)) {
      return;
    }

    const source = child.geometry as THREE.BufferGeometry;
    const color = source.getAttribute("color");
    const position = source.getAttribute("position");

    if (!color || !position) {
      return;
    }

    const geometry = source.clone();
    // clone() shares userData with the source, whose `shared` flag would stop this copy being freed.
    geometry.userData = {};
    const itemSize = color.itemSize;
    const painted = new Float32Array(color.count * itemSize);

    for (let index = 0; index < color.count; index += 1) {
      const r = color.getX(index);
      const g = color.getY(index);
      const b = color.getZ(index);
      let nearest: AvatarColorKey | null = null;
      let nearestDistance = Number.POSITIVE_INFINITY;

      for (const key of partsAt(position.getY(index), preset)) {
        const distance = distanceSquared(swatches[key].source, r, g, b);
        if (distance < nearestDistance) {
          nearestDistance = distance;
          nearest = key;
        }
      }

      const target = nearest ? swatches[nearest].target : null;
      if (nearest && target) {
        repainted.add(nearest);
      }

      const offset = index * itemSize;
      painted[offset] = target ? target.r : r;
      painted[offset + 1] = target ? target.g : g;
      painted[offset + 2] = target ? target.b : b;
      if (itemSize > 3) {
        painted[offset + 3] = color.getW(index);
      }
    }

    geometry.setAttribute(
      "color",
      new THREE.BufferAttribute(painted, itemSize),
    );
    child.geometry = geometry;
  });

  if (process.env.NODE_ENV !== "production") {
    for (const key of Object.keys(colors) as AvatarColorKey[]) {
      if (colors[key] && !repainted.has(key)) {
        console.warn(
          `Recolouring "${preset.id}": no vertices matched the "${key}" part.`,
        );
      }
    }
  }
}
