import * as THREE from "three";

import type { Aabb } from "@/modules/game/host/scene/classroom-layout";

/** A chair a classmate sits in: avatar origin (soles) and which way they face. */
export type ClassmateSeat = {
  readonly position: THREE.Vector3;
  readonly facingYaw: number;
};

type Layout = {
  readonly colliders: readonly Aabb[];
  readonly props: readonly Aabb[];
};

/**
 * Where a seated chibi's thighs rest, as a fraction of its height: the leg pivots sit
 * at 0.50 of the 1.699 m authored body, minus half a thigh. Put the origin this far
 * below the seat and the character sits on it instead of hovering.
 */
const SEATED_THIGH_RATIO = 0.25;
/** The seat slab of the low-poly chair is about this thick; the frame's bottom is its underside. */
const SEAT_SLAB = 0.02;
/** Hips go this far back from the chair's centre, toward the backrest. */
const HIP_SETBACK = 0.08;
/** A chair counts as facing the board when it's within this much of the board yaw. */
const FACING_TOLERANCE = Math.PI / 4;

function span(box: Aabb, axis: "x" | "y" | "z") {
  return box.max[axis] - box.min[axis];
}

function within(value: number, min: number, max: number) {
  return value >= min && value <= max;
}

/** The chair's seat-and-back frame: ~0.4 × 0.3 footprint, seat starting ~0.28 m up, back reaching ~0.57 m. */
function isChairFrame(box: Aabb) {
  const long = Math.max(span(box, "x"), span(box, "z"));
  const short = Math.min(span(box, "x"), span(box, "z"));
  return (
    within(long, 0.36, 0.5) &&
    within(short, 0.24, 0.37) &&
    within(box.min.y, 0.24, 0.34) &&
    within(box.max.y, 0.5, 0.64)
  );
}

/** A desk top with its shelf: ~0.4 × 0.7, a 0.12 m thick slab topping out at ~0.56 m. */
function isDeskTop(box: Aabb) {
  const long = Math.max(span(box, "x"), span(box, "z"));
  const short = Math.min(span(box, "x"), span(box, "z"));
  return (
    within(long, 0.6, 0.85) &&
    within(short, 0.3, 0.5) &&
    within(span(box, "y"), 0.06, 0.18) &&
    within(box.max.y, 0.5, 0.62)
  );
}

function center(box: Aabb) {
  return new THREE.Vector3(
    (box.min.x + box.max.x) / 2,
    0,
    (box.min.z + box.max.z) / 2,
  );
}

function containsXZ(box: Aabb, point: THREE.Vector3, margin = 0) {
  return (
    point.x >= box.min.x - margin &&
    point.x <= box.max.x + margin &&
    point.z >= box.min.z - margin &&
    point.z <= box.max.z + margin
  );
}

/** Yaw that faces along the dominant axis of `direction`, in the game's convention (forward = -z at yaw 0). */
function axisAlignedYaw(direction: THREE.Vector3) {
  return Math.abs(direction.x) > Math.abs(direction.z)
    ? Math.atan2(-Math.sign(direction.x), 0)
    : Math.atan2(0, -Math.sign(direction.z));
}

function angleBetween(a: number, b: number) {
  return Math.abs(Math.atan2(Math.sin(a - b), Math.cos(a - b)));
}

/**
 * One chair per desk group for the rest of the class, skipping the group the
 * player sits at. Chairs face their nearest desk; per group we prefer the one
 * that ends up looking at the board. Reading order: front row first.
 */
export function findClassmateSeats(
  layout: Layout,
  ownSeat: THREE.Vector3,
  avatarHeight: number,
  boardYaw: number,
): ClassmateSeat[] {
  const deskTops = layout.props.filter(isDeskTop).map(center);

  if (deskTops.length === 0) {
    return [];
  }

  // Student desk groups are the ones with student desk tops in them; the teacher's desk isn't.
  const groups = layout.colliders
    .filter(
      (box) =>
        !containsXZ(box, ownSeat, 0.15) &&
        deskTops.some((desk) => containsXZ(box, desk)),
    )
    .sort((a, b) => a.min.z - b.min.z || a.min.x - b.min.x);

  const chairs = layout.props.filter(isChairFrame).map((box) => {
    const chairCenter = center(box);
    const nearestDesk = deskTops.reduce((best, desk) =>
      desk.distanceToSquared(chairCenter) < best.distanceToSquared(chairCenter)
        ? desk
        : best,
    );
    const facingYaw = axisAlignedYaw(nearestDesk.clone().sub(chairCenter));
    const forward = new THREE.Vector3(
      -Math.sin(facingYaw),
      0,
      -Math.cos(facingYaw),
    );
    const position = chairCenter.addScaledVector(forward, -HIP_SETBACK);
    position.y = box.min.y + SEAT_SLAB - avatarHeight * SEATED_THIGH_RATIO;
    return { position, facingYaw };
  });

  return groups.flatMap((group) => {
    const inGroup = chairs.filter((chair) =>
      containsXZ(group, chair.position, 0.05),
    );
    const pick =
      inGroup.find(
        (chair) => angleBetween(chair.facingYaw, boardYaw) < FACING_TOLERANCE,
      ) ?? inGroup[0];
    return pick ? [pick] : [];
  });
}
