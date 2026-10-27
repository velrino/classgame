import * as THREE from "three";
import { DRACOLoader } from "three/addons/loaders/DRACOLoader.js";
import { GLTFLoader } from "three/addons/loaders/GLTFLoader.js";
import { clone as cloneWithSkeleton } from "three/addons/utils/SkeletonUtils.js";

import type { AvatarPreset } from "@/modules/game/adapters/types/avatar";
import { bindChibiRig } from "@/modules/game/host/avatars/chibi-rig";
import {
  recolorAvatar,
  type AvatarColors,
} from "@/modules/game/host/avatars/recolor-avatar";

const ASSET_PATH = "/assets/avatars/chibi-band/chibi-band-avatar.glb";
// Synced from the three package by `npm run sync:draco-decoder`.
const DRACO_DECODER_PATH = "/vendor/draco/";
/** Authored body height in the GLB (sole to crown, hair excluded); shared by every preset. */
const ASSET_BODY_HEIGHT = 1.699;

let assetPromise: Promise<THREE.Group> | null = null;

function markShared(object: THREE.Object3D) {
  // Every avatar instance reuses the asset's buffers, so disposing one avatar must not free them.
  object.traverse((child) => {
    if (!(child instanceof THREE.Mesh)) {
      return;
    }

    child.geometry.userData.shared = true;
    const materials = Array.isArray(child.material)
      ? child.material
      : [child.material];
    materials.forEach((material) => {
      material.userData.shared = true;
    });
  });
}

function loadAsset() {
  assetPromise ??= (() => {
    const dracoLoader = new DRACOLoader().setDecoderPath(DRACO_DECODER_PATH);
    const loader = new GLTFLoader().setDRACOLoader(dracoLoader);

    return loader
      .loadAsync(ASSET_PATH)
      .then((gltf) => {
        markShared(gltf.scene);
        return gltf.scene;
      })
      .finally(() => dracoLoader.dispose());
  })();

  return assetPromise;
}

function isVisualMesh(child: THREE.Object3D) {
  return (
    child instanceof THREE.Mesh &&
    (child.userData.variantRole === "avatar-visual" ||
      child.name.startsWith("Mesh_"))
  );
}

function keepOnlyPreset(model: THREE.Object3D, preset: AvatarPreset) {
  const removable: THREE.Object3D[] = [];
  let kept = 0;

  model.traverse((child) => {
    if (!isVisualMesh(child)) {
      return;
    }

    if (
      child.userData.presetId === preset.id ||
      child.name.startsWith(`Mesh_${preset.id}_`)
    ) {
      child.visible = true;
      kept += 1;
    } else {
      removable.push(child);
    }
  });

  removable.forEach((child) => child.parent?.remove(child));
  return kept;
}

/**
 * One chibi character, ready to drop into the scene. Forward is -Z at yaw 0,
 * like the rest of the game; the outer group owns position and yaw. Pass
 * `colors` to repaint parts (hair, top, ...) for a look the preset doesn't ship.
 */
export async function createChibiAvatar(
  preset: AvatarPreset,
  worldHeight: number,
  colors?: Partial<AvatarColors>,
) {
  const asset = await loadAsset();
  const source =
    asset.children.find((child) => child.name === "AvatarRig") ?? asset;

  // Object3D.clone() would leave every skinned mesh bound to the original skeleton.
  const model = cloneWithSkeleton(source);

  if (keepOnlyPreset(model, preset) === 0) {
    throw new Error(`Missing chibi avatar preset: ${preset.id}`);
  }

  if (colors && Object.keys(colors).length > 0) {
    recolorAvatar(model, preset, colors);
  }

  model.name = `avatar-${preset.id}-model`;
  model.position.set(0, 0, 0);
  // The base mesh faces +Z; the game walks along -Z at yaw 0.
  model.rotation.set(0, Math.PI, 0);

  const avatar = new THREE.Group();
  avatar.name = `avatar-${preset.id}`;
  avatar.scale.setScalar(worldHeight / ASSET_BODY_HEIGHT);
  avatar.add(model);

  if (!bindChibiRig(avatar)) {
    console.warn(`Chibi rig bones missing for preset: ${preset.id}`);
  }

  return avatar;
}

/** Free what an avatar owns, leaving the shared asset buffers alone. */
export function disposeAvatar(object: THREE.Object3D) {
  object.traverse((child) => {
    if (!(child instanceof THREE.Mesh)) {
      return;
    }

    if (!child.geometry.userData.shared) {
      child.geometry.dispose();
    }

    const materials = Array.isArray(child.material)
      ? child.material
      : [child.material];
    materials.forEach((material) => {
      if (!material.userData.shared) {
        material.dispose();
      }
    });
  });
}
