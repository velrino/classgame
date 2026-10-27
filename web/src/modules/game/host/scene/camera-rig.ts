import * as THREE from "three";

import type { RoomBounds } from "@/modules/game/host/scene/classroom-layout";
import type { Seat } from "@/modules/game/host/scene/create-classroom";

const FOLLOW_DISTANCE = 1.5;
const FOLLOW_HEIGHT = 2;
const FOLLOW_TARGET_HEIGHT = 1.25;
/** Never closer than this fraction of FOLLOW_DISTANCE. */
const MIN_PULL = 0.45;
/** Extra height gained as the camera is pulled in, so it looks over the avatar's head. */
const PULL_LIFT = 0.5;
const WALL_MARGIN = 0.2;
/** The camera may swing all the way round the avatar to find room, in these steps. */
const ESCAPE_STEP = Math.PI / 18;
const ESCAPE_MAX = Math.PI;
/** Only bother swinging if it buys at least this much more room. */
const ESCAPE_GAIN = 0.15;

export type CameraRigInput = {
  readonly playerPosition: THREE.Vector3;
  /** Yaw the camera orbits around: the avatar's facing plus whatever the player dragged. */
  readonly cameraYaw: number;
  readonly seat: Seat | null;
  readonly bounds: RoomBounds;
};

export type CameraRigOutput = {
  /** Yaw the caller should add over time to slide the camera out of a wall. 0 when there's room. */
  readonly yawNudge: number;
};

function smooth(lambda: number, delta: number) {
  return 1 - Math.exp(-lambda * delta);
}

/** Largest t in [0,1] such that target + offset·t stays inside the walls. */
function insideFraction(
  target: THREE.Vector3,
  offsetX: number,
  offsetZ: number,
  bounds: RoomBounds,
) {
  let t = 1;
  const axes: Array<[number, number, number, number]> = [
    [target.x, offsetX, bounds.minX + WALL_MARGIN, bounds.maxX - WALL_MARGIN],
    [target.z, offsetZ, bounds.minZ + WALL_MARGIN, bounds.maxZ - WALL_MARGIN],
  ];

  for (const [origin, direction, min, max] of axes) {
    if (direction > 1e-6) {
      t = Math.min(t, (max - origin) / direction);
    } else if (direction < -1e-6) {
      t = Math.min(t, (min - origin) / direction);
    }
  }

  return THREE.MathUtils.clamp(t, 0, 1);
}

function offsetFor(yaw: number, out: THREE.Vector3) {
  return out.set(
    Math.sin(yaw) * FOLLOW_DISTANCE,
    0,
    Math.cos(yaw) * FOLLOW_DISTANCE,
  );
}

/**
 * Third person behind the avatar while roaming, an over-the-shoulder shot of
 * the board and the tutor once seated. With a wall at the avatar's back the
 * camera first pulls in a little, then asks to swing around to whichever side
 * has room — it never leaves the room and never sits on the avatar's head.
 */
export function createCameraRig(camera: THREE.PerspectiveCamera) {
  const position = new THREE.Vector3();
  const target = new THREE.Vector3();
  const wantedPosition = new THREE.Vector3();
  const wantedTarget = new THREE.Vector3();
  const offset = new THREE.Vector3();
  const probe = new THREE.Vector3();
  let pull = 1;
  let initialised = false;

  return {
    update(input: CameraRigInput, delta: number): CameraRigOutput {
      const { playerPosition, cameraYaw, seat, bounds } = input;
      let yawNudge = 0;

      if (seat) {
        wantedPosition.copy(seat.cameraPosition);
        wantedTarget.copy(seat.cameraTarget);
        pull = 1;
      } else {
        wantedTarget.set(
          playerPosition.x,
          playerPosition.y + FOLLOW_TARGET_HEIGHT,
          playerPosition.z,
        );
        offsetFor(cameraYaw, offset);

        const room = insideFraction(wantedTarget, offset.x, offset.z, bounds);

        if (room < 0.85) {
          // Cornered: sweep both ways round the avatar and head for the yaw with the most room,
          // preferring the smaller turn when two are equally good.
          let bestRoom = room;
          for (
            let step = ESCAPE_STEP;
            step <= ESCAPE_MAX + 1e-6;
            step += ESCAPE_STEP
          ) {
            for (const sign of [1, -1]) {
              offsetFor(cameraYaw + sign * step, probe);
              const candidate = insideFraction(
                wantedTarget,
                probe.x,
                probe.z,
                bounds,
              );
              if (candidate > bestRoom + ESCAPE_GAIN) {
                bestRoom = candidate;
                yawNudge = sign * step;
              }
            }
            if (bestRoom >= 0.95) {
              break;
            }
          }
        }

        const wantedPull = Math.max(MIN_PULL, room);
        // Pull in fast, let out slowly, so the camera doesn't pump against the wall.
        pull = THREE.MathUtils.lerp(
          pull,
          wantedPull,
          smooth(wantedPull < pull ? 14 : 4, delta),
        );

        const lift = (1 - pull) * PULL_LIFT;
        wantedPosition.copy(wantedTarget).addScaledVector(offset, pull);
        wantedPosition.y = playerPosition.y + FOLLOW_HEIGHT + lift;
        wantedTarget.y += lift * 0.25;
      }

      if (!initialised) {
        initialised = true;
        position.copy(wantedPosition);
        target.copy(wantedTarget);
      }

      position.lerp(wantedPosition, smooth(seat ? 4.5 : 10, delta));
      target.lerp(wantedTarget, smooth(seat ? 4.5 : 12, delta));
      camera.position.copy(position);
      camera.lookAt(target);

      return { yawNudge };
    },
  };
}
