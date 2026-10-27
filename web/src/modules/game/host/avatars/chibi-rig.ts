import * as THREE from "three";

/**
 * Procedural animation for the chibi rig exported from Blender: six pivots
 * (body, head, two arms, two legs) driven relative to their rest pose.
 */

export type ChibiAnimationState = {
  readonly delta: number;
  readonly elapsedTime: number;
  readonly isMoving: boolean;
  readonly isRunning: boolean;
  /** Sitting: legs out, arms resting, nothing swings. */
  readonly isSeated?: boolean;
};

type Gait = {
  readonly armSwing: number;
  readonly frequency: number;
  readonly leanForward: number;
  readonly legSwing: number;
};

const walkGait: Gait = {
  armSwing: 0.42,
  frequency: 8.6,
  leanForward: 0.05,
  legSwing: 0.34,
};
const runGait: Gait = {
  armSwing: 0.98,
  frequency: 13.4,
  leanForward: 0.26,
  legSwing: 0.82,
};

/**
 * Bones from Blender carry a rest rotation (leg bones point down, roughly a half
 * turn), so every angle is applied relative to `rest` — writing rotation.x
 * absolutely would fold the limbs into the torso.
 */
type Joint = {
  angleX: number;
  angleY: number;
  readonly node: THREE.Object3D;
  readonly rest: THREE.Quaternion;
};

type Rig = {
  readonly body: Joint;
  readonly head: Joint;
  readonly leftArm: Joint;
  readonly leftLeg: Joint;
  readonly rightArm: Joint;
  readonly rightLeg: Joint;
};

const RIG_KEY = "classgameChibiRig";
const euler = new THREE.Euler();
const rotation = new THREE.Quaternion();

function createJoint(node: THREE.Object3D): Joint {
  return { angleX: 0, angleY: 0, node, rest: node.quaternion.clone() };
}

/** Blender exports "left-arm-pivot" as "left-arm-pivot", "left_arm_pivot" or "leftarmpivot" depending on the exporter. */
function normalizeName(name: string) {
  return name
    .toLowerCase()
    .replace(/[\s_.]+/g, "-")
    .replace(/-+/g, "-");
}

function findRigNode(avatar: THREE.Object3D, rigNodeName: string) {
  const wanted = normalizeName(rigNodeName);
  let found: THREE.Object3D | null = null;

  avatar.traverse((child) => {
    if (found) {
      return;
    }

    const name = normalizeName(child.name);
    if (
      name === wanted ||
      name.replace(/-/g, "") === wanted.replace(/-/g, "")
    ) {
      found = child;
    }
  });

  return found;
}

export function bindChibiRig(avatar: THREE.Object3D) {
  const body = findRigNode(avatar, "body");
  const head = findRigNode(avatar, "head-pivot");
  const leftArm = findRigNode(avatar, "left-arm-pivot");
  const leftLeg = findRigNode(avatar, "left-leg-pivot");
  const rightArm = findRigNode(avatar, "right-arm-pivot");
  const rightLeg = findRigNode(avatar, "right-leg-pivot");

  if (!body || !head || !leftArm || !leftLeg || !rightArm || !rightLeg) {
    return false;
  }

  avatar.userData[RIG_KEY] = {
    body: createJoint(body),
    head: createJoint(head),
    leftArm: createJoint(leftArm),
    leftLeg: createJoint(leftLeg),
    rightArm: createJoint(rightArm),
    rightLeg: createJoint(rightLeg),
  } satisfies Rig;

  return true;
}

function driveJoint(
  joint: Joint,
  targetX: number,
  targetY: number,
  delta: number,
  rate = 16,
) {
  const blend = 1 - Math.exp(-rate * delta);
  joint.angleX += (targetX - joint.angleX) * blend;
  joint.angleY += (targetY - joint.angleY) * blend;
  euler.set(joint.angleX, joint.angleY, 0);
  rotation.setFromEuler(euler);
  joint.node.quaternion.copy(joint.rest).multiply(rotation);
}

export function animateChibi(
  avatar: THREE.Object3D,
  state: ChibiAnimationState,
) {
  const rig = avatar.userData[RIG_KEY] as Rig | undefined;

  if (!rig) {
    return;
  }

  if (state.isSeated) {
    // Thighs level, forearms resting, a little slouch. Negative X on a hanging limb swings it forward.
    driveJoint(rig.leftArm, -0.55, 0, state.delta);
    driveJoint(rig.rightArm, -0.5, 0, state.delta);
    driveJoint(rig.leftLeg, -1.45, 0, state.delta);
    driveJoint(rig.rightLeg, -1.4, 0, state.delta);
    driveJoint(rig.body, 0.04, 0, state.delta);
    driveJoint(
      rig.head,
      -0.03,
      Math.sin(state.elapsedTime * 1.4) * 0.04,
      state.delta,
      8,
    );
    return;
  }

  const gait = state.isRunning ? runGait : walkGait;
  const movement = state.isMoving ? 1 : 0;
  const swing = Math.sin(state.elapsedTime * gait.frequency) * movement;
  const idle = Math.sin(state.elapsedTime * 2.2) * (1 - movement);
  const breathe = Math.sin(state.elapsedTime * 1.6) * (1 - movement);

  driveJoint(rig.leftArm, swing * gait.armSwing, 0, state.delta);
  driveJoint(rig.rightArm, -swing * gait.armSwing, 0, state.delta);
  driveJoint(rig.leftLeg, -swing * gait.legSwing, 0, state.delta);
  driveJoint(rig.rightLeg, swing * gait.legSwing, 0, state.delta);
  driveJoint(
    rig.body,
    movement * gait.leanForward + breathe * 0.012,
    0,
    state.delta,
  );
  driveJoint(
    rig.head,
    -movement * gait.leanForward,
    idle * 0.05,
    state.delta,
    8,
  );
}
