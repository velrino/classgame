export type AvatarAccessoryStyle =
  | "camo"
  | "cap"
  | "hoodie"
  | "jacket"
  | "none"
  | "striped";
export type AvatarBeardStyle = "full" | "goatee" | "none" | "short";
export type AvatarHairStyle =
  | "buzz"
  | "cap"
  | "center-part"
  | "curly"
  | "full-curly"
  | "long-waves";
export type AvatarSleeveStyle = "long" | "short" | "sleeveless";
export type AvatarTattooStyle = "both-arms" | "left-arm" | "none";

/** One of the chibi-band characters. Colours are informational here: the GLB is pre-coloured per preset. */
export type AvatarPreset = {
  readonly id: string;
  readonly label: string;
  readonly accessoryStyle: AvatarAccessoryStyle;
  readonly accent: string;
  readonly beard: string;
  readonly beardStyle: AvatarBeardStyle;
  readonly eye: string;
  readonly hair: string;
  readonly hairStyle: AvatarHairStyle;
  readonly skin: string;
  readonly sleeveStyle: AvatarSleeveStyle;
  readonly tattooStyle: AvatarTattooStyle;
  readonly top: string;
  readonly pants: string;
  readonly shoes: string;
};
