"use client";

import { useEffect, useRef, useState } from "react";
import * as THREE from "three";

import type {
  MovementRef,
  WorldDirectiveRef,
  WorldReport,
} from "@/modules/game/adapters/types/world";
import { getAvatarPreset } from "@/modules/game/host/avatars/avatar-presets";
import { animateChibi } from "@/modules/game/host/avatars/chibi-rig";
import {
  createChibiAvatar,
  disposeAvatar,
} from "@/modules/game/host/avatars/load-chibi-avatar";
import { createKeyboardControls } from "@/modules/game/host/controls/keyboard-controls";
import { createOrbitDrag } from "@/modules/game/host/controls/orbit-drag";
import { createCameraRig } from "@/modules/game/host/scene/camera-rig";
import { createClassmates } from "@/modules/game/host/scene/classmates";
import type {
  Aabb,
  RoomBounds,
} from "@/modules/game/host/scene/classroom-layout";
import {
  SEAT_HEIGHT,
  STUDENT_AVATAR_HEIGHT,
  createClassroom,
} from "@/modules/game/host/scene/create-classroom";
import type { PodStatus } from "@/modules/learning";

type GameCanvasProps = {
  readonly movementRef: MovementRef;
  readonly directiveRef: WorldDirectiveRef;
  readonly onReport: (report: WorldReport) => void;
  readonly showSeatMarker: boolean;
};

const TARGET_FRAME_MS = 1000 / 34;
const PLAYER_SPEED = 2.6;
const SPRINT_MULTIPLIER = 1.5;
const PLAYER_RADIUS = 0.27;
const WALL_MARGIN = 0.35;
const NEAR_DISTANCE = 1.0;
/** Auto-walk gives up after this long without progress (a desk in the way). */
const STUCK_SECONDS = 0.7;
/** How fast the camera swings back behind the avatar while walking. */
const CAMERA_RETURN_LAMBDA = 4;
/** Stick turned this much (radians) since it was pressed → re-read directions from the current camera. */
const STICK_REANCHOR = Math.PI / 6;

const SEAT_LOOK: Record<PodStatus, { color: number; intensity: number }> = {
  locked: { color: 0x475569, intensity: 0.5 },
  ready: { color: 0x22d3ee, intensity: 2.2 },
  review: { color: 0xf59e0b, intensity: 2.6 },
  complete: { color: 0x22c55e, intensity: 1.6 },
};

function normalizeRadians(radians: number) {
  return Math.atan2(Math.sin(radians), Math.cos(radians));
}

function dampRadians(
  current: number,
  target: number,
  lambda: number,
  delta: number,
) {
  return (
    current +
    normalizeRadians(target - current) * (1 - Math.exp(-lambda * delta))
  );
}

/** Push a circle (the avatar's feet) out of every box it overlaps, then keep it inside the walls. */
function resolveCollisions(
  position: THREE.Vector3,
  radius: number,
  boxes: readonly Aabb[],
  bounds: RoomBounds,
) {
  for (let pass = 0; pass < 2; pass += 1) {
    for (const box of boxes) {
      const closestX = THREE.MathUtils.clamp(position.x, box.min.x, box.max.x);
      const closestZ = THREE.MathUtils.clamp(position.z, box.min.z, box.max.z);
      const dx = position.x - closestX;
      const dz = position.z - closestZ;
      const distance = Math.hypot(dx, dz);

      if (distance >= radius) {
        continue;
      }

      if (distance > 1e-4) {
        const push = (radius - distance) / distance;
        position.x += dx * push;
        position.z += dz * push;
      } else {
        // Inside the box: leave through the nearest face.
        const exits = [
          {
            axis: "x",
            value: box.min.x - radius,
            cost: position.x - box.min.x,
          },
          {
            axis: "x",
            value: box.max.x + radius,
            cost: box.max.x - position.x,
          },
          {
            axis: "z",
            value: box.min.z - radius,
            cost: position.z - box.min.z,
          },
          {
            axis: "z",
            value: box.max.z + radius,
            cost: box.max.z - position.z,
          },
        ].sort((a, b) => a.cost - b.cost);
        const exit = exits[0];
        if (exit.axis === "x") {
          position.x = exit.value;
        } else {
          position.z = exit.value;
        }
      }
    }
  }

  position.x = THREE.MathUtils.clamp(
    position.x,
    bounds.minX + WALL_MARGIN,
    bounds.maxX - WALL_MARGIN,
  );
  position.z = THREE.MathUtils.clamp(
    position.z,
    bounds.minZ + WALL_MARGIN,
    bounds.maxZ - WALL_MARGIN,
  );
}

export function GameCanvas({
  movementRef,
  directiveRef,
  onReport,
  showSeatMarker,
}: GameCanvasProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const onReportRef = useRef(onReport);
  const showSeatMarkerRef = useRef(showSeatMarker);
  const [webglUnavailable, setWebglUnavailable] = useState(false);

  useEffect(() => {
    onReportRef.current = onReport;
  }, [onReport]);

  useEffect(() => {
    showSeatMarkerRef.current = showSeatMarker;
  }, [showSeatMarker]);

  useEffect(() => {
    const container = containerRef.current;

    if (!container) {
      return;
    }

    let renderer: THREE.WebGLRenderer;
    let disposed = false;

    try {
      renderer = new THREE.WebGLRenderer({
        antialias: true,
        powerPreference: "low-power",
      });
    } catch {
      const frame = window.requestAnimationFrame(() =>
        setWebglUnavailable(true),
      );
      return () => window.cancelAnimationFrame(frame);
    }

    const scene = new THREE.Scene();
    scene.background = new THREE.Color(0x162634);

    const camera = new THREE.PerspectiveCamera(60, 1, 0.1, 100);
    const rig = createCameraRig(camera);

    renderer.setPixelRatio(1);
    renderer.outputColorSpace = THREE.SRGBColorSpace;
    renderer.toneMapping = THREE.ACESFilmicToneMapping;
    renderer.toneMappingExposure = 1.35;
    renderer.domElement.setAttribute("aria-label", "ClassGame classroom");
    renderer.domElement.style.cssText =
      "display:block;width:100%;height:100%;touch-action:none;";
    container.appendChild(renderer.domElement);

    const classroom = createClassroom(scene);
    const classmates = createClassmates(classroom.room, STUDENT_AVATAR_HEIGHT);
    const keyboard = createKeyboardControls(window);
    const raycaster = new THREE.Raycaster();
    const pointer = new THREE.Vector2();
    const moveDirection = new THREE.Vector3();
    const nextPosition = new THREE.Vector3();
    const scratch = new THREE.Vector3();

    let animationFrame = 0;
    let avatarYaw = classroom.spawnYaw;
    let cameraYaw = classroom.spawnYaw;
    let lastRenderTime = 0;
    let lastTime = performance.now();
    let isVisible = !document.hidden;
    let walkingToSeat = false;
    let walkStuckFor = 0;
    let lastWalkToken = 0;
    let lastReportKey = "";
    let wasSeated = false;
    let wasMoving = false;
    // Stick directions are read against the camera as it was when you pressed, not as it
    // swings back behind you — otherwise "forward" would rotate under your thumb and spin.
    let inputYaw = classroom.spawnYaw;
    let inputStickAngle = 0;

    const orbit = createOrbitDrag(renderer.domElement, (clientX, clientY) => {
      if (directiveRef.current.seated) {
        return;
      }

      const rect = renderer.domElement.getBoundingClientRect();
      pointer.set(
        ((clientX - rect.left) / rect.width) * 2 - 1,
        -((clientY - rect.top) / rect.height) * 2 + 1,
      );
      raycaster.setFromCamera(pointer, camera);

      if (
        showSeatMarkerRef.current &&
        raycaster.intersectObject(classroom.seat.hitArea, false).length > 0
      ) {
        walkingToSeat = true;
        walkStuckFor = 0;
      }
    });

    // Dev-only peek at the camera/avatar state, for debugging controls from the console.
    if (process.env.NODE_ENV !== "production") {
      (
        window as unknown as {
          __classgameDebug?: () => Record<string, unknown>;
        }
      ).__classgameDebug = () => ({
        cameraYaw: Number(cameraYaw.toFixed(3)),
        avatarYaw: Number(avatarYaw.toFixed(3)),
        inputYaw: Number(inputYaw.toFixed(3)),
        dragging: orbit.isDragging(),
        x: Number(classroom.player.position.x.toFixed(2)),
        z: Number(classroom.player.position.z.toFixed(2)),
        boxes: classroom.collision.boxes.map((box) =>
          [
            box.min.x,
            box.min.y,
            box.min.z,
            box.max.x,
            box.max.y,
            box.max.z,
          ].map((n) => Number(n.toFixed(3))),
        ),
        bounds: classroom.collision.bounds,
      });
    }

    // The student's character follows the picker: load the chosen preset and swap it in.
    let wantedAvatarId = "";
    let loadingAvatarId = "";

    const swapPlayerAvatar = (presetId: string) => {
      loadingAvatarId = presetId;
      void createChibiAvatar(getAvatarPreset(presetId), STUDENT_AVATAR_HEIGHT)
        .then((model) => {
          if (disposed || presetId !== wantedAvatarId) {
            disposeAvatar(model);
            return;
          }

          for (const old of [...classroom.player.children]) {
            classroom.player.remove(old);
            disposeAvatar(old);
          }

          classroom.player.add(model);
          classroom.models.player = model;
        })
        .catch((error: unknown) => {
          console.warn(
            `Avatar "${presetId}" failed to load; keeping the current one.`,
            error,
          );
        })
        .finally(() => {
          if (loadingAvatarId === presetId) {
            loadingAvatarId = "";
          }
        });
    };

    const publish = (report: WorldReport) => {
      const key = `${report.nearSeat}|${report.walking}|${report.ready}`;

      if (key !== lastReportKey) {
        lastReportKey = key;
        onReportRef.current(report);
      }
    };

    const resize = () => {
      const rect = container.getBoundingClientRect();
      const width = Math.max(1, Math.floor(rect.width));
      const height = Math.max(1, Math.floor(rect.height));
      camera.aspect = width / height;
      camera.fov = width < 720 ? 66 : 58;
      camera.updateProjectionMatrix();
      renderer.setPixelRatio(
        width < 720 ? 1 : Math.min(window.devicePixelRatio, 1.25),
      );
      renderer.setSize(width, height, false);
    };

    const onVisibilityChange = () => {
      isVisible = !document.hidden;
      lastTime = performance.now();
    };

    window.addEventListener("resize", resize);
    document.addEventListener("visibilitychange", onVisibilityChange);
    resize();

    const render = () => {
      animationFrame = requestAnimationFrame(render);
      const now = performance.now();

      if (!isVisible || now - lastRenderTime < TARGET_FRAME_MS) {
        return;
      }

      lastRenderTime = now;
      const elapsed = now / 1000;
      const delta = Math.min(0.05, (now - lastTime) / 1000);
      lastTime = now;

      const directive = directiveRef.current;
      const { player, seat, tutor, tutorRing, tutorLight, collision, models } =
        classroom;

      if (directive.avatarPresetId !== wantedAvatarId) {
        wantedAvatarId = directive.avatarPresetId;
        if (loadingAvatarId !== wantedAvatarId) {
          swapPlayerAvatar(wantedAvatarId);
        }
      }

      let movedThisFrame = false;
      let sprinting = false;

      // React asked us to walk to the desk ("Go to my desk").
      if (
        showSeatMarkerRef.current &&
        directive.walkRequest &&
        directive.walkRequest.token !== lastWalkToken
      ) {
        lastWalkToken = directive.walkRequest.token;
        walkingToSeat = true;
        walkStuckFor = 0;
      }

      cameraYaw += orbit.consumeYawDelta();

      if (directive.seated) {
        walkingToSeat = false;
        wasSeated = true;
        player.position.x = THREE.MathUtils.lerp(
          player.position.x,
          seat.position.x,
          1 - Math.exp(-8 * delta),
        );
        player.position.y = THREE.MathUtils.lerp(
          player.position.y,
          SEAT_HEIGHT,
          1 - Math.exp(-8 * delta),
        );
        player.position.z = THREE.MathUtils.lerp(
          player.position.z,
          seat.position.z,
          1 - Math.exp(-8 * delta),
        );
        avatarYaw = dampRadians(avatarYaw, seat.facingYaw, 10, delta);
        cameraYaw = avatarYaw;
      } else {
        if (wasSeated) {
          // Getting up: step out of the chair to the spot in front of the desk.
          wasSeated = false;
          player.position.copy(seat.standPosition);
          player.position.y = 0;
          avatarYaw = seat.facingYaw;
          cameraYaw = avatarYaw;
        }

        const manual = movementRef.current;
        const keys = keyboard.getMovement();
        const forward = THREE.MathUtils.clamp(
          keys.forward + manual.forward,
          -1,
          1,
        );
        const right = THREE.MathUtils.clamp(keys.right + manual.right, -1, 1);
        const wantsToMove = forward !== 0 || right !== 0;

        if (wantsToMove) {
          walkingToSeat = false;
          const stickAngle = Math.atan2(right, forward);

          if (
            !wasMoving ||
            Math.abs(normalizeRadians(stickAngle - inputStickAngle)) >
              STICK_REANCHOR
          ) {
            inputYaw = cameraYaw;
            inputStickAngle = stickAngle;
          }

          moveDirection.set(
            -Math.sin(inputYaw) * forward + Math.cos(inputYaw) * right,
            0,
            -Math.cos(inputYaw) * forward - Math.sin(inputYaw) * right,
          );
        } else if (walkingToSeat) {
          moveDirection.subVectors(seat.standPosition, player.position).setY(0);

          if (moveDirection.length() < 0.06) {
            walkingToSeat = false;
            moveDirection.set(0, 0, 0);
          }
        } else {
          moveDirection.set(0, 0, 0);
        }

        if (moveDirection.lengthSq() > 0) {
          const strength = wantsToMove
            ? Math.min(1, Math.hypot(forward, right))
            : 1;
          const speed =
            PLAYER_SPEED *
            (keyboard.isSprinting() ? SPRINT_MULTIPLIER : 1) *
            strength;
          const stepLength = Math.min(
            delta * speed,
            wantsToMove ? Number.POSITIVE_INFINITY : moveDirection.length(),
          );

          moveDirection.normalize();
          nextPosition
            .copy(player.position)
            .addScaledVector(moveDirection, stepLength);
          resolveCollisions(
            nextPosition,
            PLAYER_RADIUS,
            collision.boxes,
            collision.bounds,
          );

          const progressed = nextPosition.distanceTo(player.position);
          movedThisFrame = progressed > 0.002;
          sprinting = keyboard.isSprinting();
          player.position.copy(nextPosition);
          avatarYaw = dampRadians(
            avatarYaw,
            Math.atan2(-moveDirection.x, -moveDirection.z),
            14,
            delta,
          );

          if (walkingToSeat) {
            walkStuckFor =
              progressed < stepLength * 0.2 ? walkStuckFor + delta : 0;
            if (walkStuckFor > STUCK_SECONDS) {
              walkingToSeat = false;
            }
          }

          // Walking always brings the camera back behind the avatar (unless you're holding it).
          if (!orbit.isDragging()) {
            cameraYaw = dampRadians(
              cameraYaw,
              avatarYaw,
              CAMERA_RETURN_LAMBDA,
              delta,
            );
          }
        }

        wasMoving = wantsToMove;
      }

      player.rotation.y = avatarYaw;

      // Seat marker: coloured by the current stage, brighter when you're close.
      scratch.set(
        seat.standPosition.x,
        player.position.y,
        seat.standPosition.z,
      );
      const markerVisible = showSeatMarkerRef.current && !directive.seated;
      const nearSeat =
        markerVisible && player.position.distanceTo(scratch) < NEAR_DISTANCE;
      const look = SEAT_LOOK[directive.podStatus[directive.tutorPod]];
      const pulse = 1 + Math.sin(elapsed * 4.2) * 0.05;
      seat.ring.visible = markerVisible;
      seat.ring.material.emissive.setHex(look.color);
      seat.ring.material.color.setHex(look.color);
      seat.ring.material.emissiveIntensity =
        look.intensity * (nearSeat ? 1.7 : 1);
      seat.ring.material.opacity = directive.seated ? 0.25 : 0.9;
      seat.ring.scale.setScalar(pulse * (nearSeat ? 1.12 : 1));

      // The teacher, Emily, stays at the board while Mia is thinking.
      const think = directive.tutorThinking
        ? 1 + Math.sin(elapsed * 9) * 0.18
        : 1 + Math.sin(elapsed * 2.2) * 0.03;
      tutorRing.scale.setScalar(think);
      tutorRing.rotation.z = elapsed * 0.7;
      tutorRing.material.emissiveIntensity = directive.tutorThinking
        ? 4.5
        : 2.6;
      tutorLight.intensity = directive.tutorThinking ? 4.5 : 2.5;

      if (models.tutor) {
        animateChibi(models.tutor, {
          delta,
          elapsedTime: elapsed,
          isMoving: false,
          isRunning: false,
        });
      }

      // The rest of the class fills the other desks with whoever the student isn't playing as.
      classmates.sync(classroom.classmateSeats, directive.avatarPresetId);
      classmates.animate(elapsed, delta);

      if (models.player) {
        animateChibi(models.player, {
          delta,
          elapsedTime: elapsed,
          isMoving: movedThisFrame,
          isRunning: movedThisFrame && sprinting,
          isSeated: directive.seated,
        });
      }

      void tutor;

      const shot = rig.update(
        {
          playerPosition: player.position,
          cameraYaw,
          seat: directive.seated ? seat : null,
          bounds: collision.bounds,
        },
        delta,
      );

      if (shot.yawNudge !== 0 && !orbit.isDragging()) {
        cameraYaw += shot.yawNudge * (1 - Math.exp(-3 * delta));
      }

      publish({ nearSeat, walking: walkingToSeat, ready: true });
      renderer.render(scene, camera);
    };

    render();

    return () => {
      disposed = true;
      cancelAnimationFrame(animationFrame);
      window.removeEventListener("resize", resize);
      document.removeEventListener("visibilitychange", onVisibilityChange);
      orbit.dispose();
      keyboard.dispose();
      classmates.dispose();
      classroom.dispose();
      renderer.dispose();
      renderer.domElement.remove();
    };
  }, [directiveRef, movementRef]);

  if (webglUnavailable) {
    return (
      <div className="absolute inset-0 grid place-items-center bg-[#05070d] px-6 text-center text-slate-200">
        <div className="max-w-sm rounded-lg border border-cyan-200/20 bg-slate-950/70 p-5 shadow-2xl">
          <p className="text-xs font-semibold tracking-[0.18em] text-cyan-200 uppercase">
            3D renderer unavailable
          </p>
          <p className="mt-3 text-sm leading-6 text-slate-300">
            ClassGame needs WebGL to render the classroom.
          </p>
        </div>
      </div>
    );
  }

  return <div ref={containerRef} className="absolute inset-0" />;
}
