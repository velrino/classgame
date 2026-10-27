import * as THREE from "three";

import { getAvatarPreset } from "@/modules/game/host/avatars/avatar-presets";
import { animateChibi } from "@/modules/game/host/avatars/chibi-rig";
import {
  classmateLooks,
  type AvatarLook,
} from "@/modules/game/host/avatars/classmate-looks";
import {
  createChibiAvatar,
  disposeAvatar,
} from "@/modules/game/host/avatars/load-chibi-avatar";
import type { ClassmateSeat } from "@/modules/game/host/scene/desk-seats";

type Classmate = {
  /** Transform node at the chair; its single child is the character model. */
  readonly anchor: THREE.Group;
  /** Offsets the idle sway so the class doesn't nod in unison. */
  readonly phase: number;
  look: AvatarLook | null;
  model: THREE.Object3D | null;
};

/**
 * The rest of the class, one per seat: every look whose character the player
 * isn't playing as, cycling round if there are more seats than looks. When
 * the player picks a new character, the classmate wearing it swaps out.
 */
export function createClassmates(room: THREE.Group, avatarHeight: number) {
  const classmates: Classmate[] = [];
  let disposed = false;
  let lastSeatCount = -1;
  let lastPlayerPresetId = "";

  const clearModel = (classmate: Classmate) => {
    if (classmate.model) {
      classmate.anchor.remove(classmate.model);
      disposeAvatar(classmate.model);
      classmate.model = null;
    }
  };

  const load = (classmate: Classmate, look: AvatarLook) => {
    classmate.look = look;

    void createChibiAvatar(
      getAvatarPreset(look.presetId),
      avatarHeight,
      look.colors,
    )
      .then((model) => {
        if (disposed || classmate.look !== look) {
          disposeAvatar(model);
          return;
        }

        clearModel(classmate);
        classmate.anchor.add(model);
        classmate.model = model;
      })
      .catch((error: unknown) => {
        console.warn(
          `Classmate "${look.presetId}" failed to load; leaving the chair empty.`,
          error,
        );
      });
  };

  const remove = (classmate: Classmate) => {
    clearModel(classmate);
    room.remove(classmate.anchor);
  };

  return {
    /** Seat the class. Safe to call every frame: returns early unless the desks or the player's character changed. */
    sync(seats: readonly ClassmateSeat[], playerPresetId: string) {
      const resolvedPlayerId = getAvatarPreset(playerPresetId).id;

      if (
        seats.length === lastSeatCount &&
        resolvedPlayerId === lastPlayerPresetId
      ) {
        return;
      }

      lastSeatCount = seats.length;
      lastPlayerPresetId = resolvedPlayerId;

      const pool = classmateLooks.filter(
        (look) => look.presetId !== resolvedPlayerId,
      );
      const wanted =
        pool.length > 0
          ? seats.map((_, index) => pool[index % pool.length])
          : [];

      while (classmates.length > wanted.length) {
        remove(classmates.pop()!);
      }

      wanted.forEach((look, index) => {
        let classmate = classmates[index];

        if (!classmate) {
          const seat = seats[index];
          const anchor = new THREE.Group();
          anchor.name = `classmate-${index}`;
          anchor.position.copy(seat.position);
          anchor.rotation.y = seat.facingYaw;
          room.add(anchor);
          classmate = { anchor, phase: index * 1.7, look: null, model: null };
          classmates.push(classmate);
        }

        if (classmate.look !== look) {
          load(classmate, look);
        }
      });
    },

    animate(elapsedTime: number, delta: number) {
      for (const classmate of classmates) {
        if (classmate.model) {
          animateChibi(classmate.model, {
            delta,
            elapsedTime: elapsedTime + classmate.phase,
            isMoving: false,
            isRunning: false,
            isSeated: true,
          });
        }
      }
    },

    dispose() {
      disposed = true;
      classmates.splice(0).forEach(remove);
    },
  };
}
