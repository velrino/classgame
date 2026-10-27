import * as THREE from "three";
import { GLTFLoader } from "three/addons/loaders/GLTFLoader.js";

import {
  layoutClassroom,
  type Aabb,
  type LayoutInput,
  type RoomBounds,
} from "@/modules/game/host/scene/classroom-layout";

import {
  getAvatarPreset,
  teacherAvatarPresetId,
} from "@/modules/game/host/avatars/avatar-presets";
import { createChibiAvatar } from "@/modules/game/host/avatars/load-chibi-avatar";
import {
  findClassmateSeats,
  type ClassmateSeat,
} from "@/modules/game/host/scene/desk-seats";

/** World heights of the chibi characters: the student is a teen, the teacher an adult. */
export const STUDENT_AVATAR_HEIGHT = 1.2;
export const TEACHER_AVATAR_HEIGHT = 1.32;
/** Chair seat height: where the avatar's hips go while seated. */
export const SEAT_HEIGHT = 0.3;

export const CLASSROOM_MODEL_URL =
  "/models/classroom/low-poly-classroom-open.glb?v=20260827-cctv-opposite-whiteboard";

/** How much wider the room gets and how far apart the desks move. */
const ROOM_SPREAD = 2;
const FURNITURE_SHIFT_Z = 0.5;

/** Room bounds the fallback uses until the real model has been laid out. */
const DEFAULT_BOUNDS: RoomBounds = {
  minX: -6.15,
  maxX: 6.15,
  minZ: -5.95,
  maxZ: 6.45,
  floorY: 0,
};

/** The one place the student sits: the desk in front of the teacher's, facing the whiteboard. */
export type Seat = {
  /** Avatar origin while seated. */
  readonly position: THREE.Vector3;
  readonly facingYaw: number;
  /** Where you stand to sit down, and where you're put when you get up. */
  readonly standPosition: THREE.Vector3;
  readonly cameraPosition: THREE.Vector3;
  readonly cameraTarget: THREE.Vector3;
  /** Floor marker; recoloured by the current stage. */
  readonly ring: THREE.Mesh<THREE.RingGeometry, THREE.MeshStandardMaterial>;
  /** Bigger invisible target so a thumb can tap the desk. */
  readonly hitArea: THREE.Mesh;
};

export type Collision = {
  boxes: readonly Aabb[];
  bounds: RoomBounds;
};

export type Classroom = {
  readonly room: THREE.Group;
  /** Transform node for the student; its single child is the current character model. */
  readonly player: THREE.Group;
  readonly seat: Seat;
  /** The teacher (Emily), standing at the whiteboard, facing the student's desk. */
  readonly tutor: THREE.Group;
  /** Rigged character models once loaded, for animation. Null while the placeholders show. */
  readonly models: {
    player: THREE.Object3D | null;
    tutor: THREE.Object3D | null;
  };
  readonly tutorRing: THREE.Mesh<
    THREE.TorusGeometry,
    THREE.MeshStandardMaterial
  >;
  readonly tutorLight: THREE.PointLight;
  /** Updated in place once the model is laid out. */
  readonly collision: Collision;
  /** One chair per other desk group, filled in once the model is laid out. Reading order: front row first. */
  readonly classmateSeats: ClassmateSeat[];
  readonly spawn: THREE.Vector3;
  readonly spawnYaw: number;
  readonly dispose: () => void;
};

/** Where things are in the laid-out room (see classroom-layout.ts for how the numbers come about). */
const SEAT_POSITION = new THREE.Vector3(-0.88, 0, 0.95);
const SEAT_STAND_POSITION = new THREE.Vector3(-1.8, 0, 0.95);
/** Everyone at a desk faces the whiteboard on the -x wall. */
const FACE_BOARD_YAW = Math.PI / 2;
/** The teacher stands between her desk and the whiteboard on the left wall. */
const TUTOR_ANCHOR = new THREE.Vector3(-4.9, 0, 0.5);

function createMat(
  color: number,
  options: Partial<THREE.MeshStandardMaterialParameters> = {},
) {
  return new THREE.MeshStandardMaterial({
    color,
    metalness: 0.18,
    roughness: 0.72,
    ...options,
  });
}

function createBox(
  name: string,
  size: [number, number, number],
  position: [number, number, number],
  material: THREE.Material,
) {
  const mesh = new THREE.Mesh(new THREE.BoxGeometry(...size), material);
  mesh.name = name;
  mesh.position.set(...position);
  return mesh;
}

function createPlaceholder(name: string) {
  const group = new THREE.Group();
  group.name = name;

  const shadow = new THREE.Mesh(
    new THREE.CircleGeometry(0.42, 32),
    createMat(0x020617, { opacity: 0.34, transparent: true }),
  );
  shadow.position.y = 0.025;
  shadow.rotation.x = -Math.PI / 2;

  const body = new THREE.Mesh(
    new THREE.CapsuleGeometry(0.2, 0.52, 8, 18),
    createMat(0x38bdf8, {
      emissive: new THREE.Color(0x0369a1),
      emissiveIntensity: 0.65,
    }),
  );
  body.position.y = 0.58;

  const head = new THREE.Mesh(
    new THREE.SphereGeometry(0.18, 24, 16),
    createMat(0xe0f2fe, {
      emissive: new THREE.Color(0x7dd3fc),
      emissiveIntensity: 0.55,
    }),
  );
  head.position.y = 1.06;

  const visor = createBox(
    "avatar-visor",
    [0.22, 0.045, 0.035],
    [0, 1.08, -0.16],
    createMat(0x020617, {
      emissive: new THREE.Color(0x22d3ee),
      emissiveIntensity: 1.6,
    }),
  );
  const pack = createBox(
    "avatar-pack",
    [0.24, 0.28, 0.08],
    [0, 0.63, 0.18],
    createMat(0x0f172a),
  );

  group.add(shadow, body, head, visor, pack);
  return group;
}

function createFallbackRoom(bounds: RoomBounds) {
  const group = new THREE.Group();
  group.name = "fallback-room";

  const width = bounds.maxX - bounds.minX;
  const depth = bounds.maxZ - bounds.minZ;
  const cx = (bounds.minX + bounds.maxX) / 2;
  const cz = (bounds.minZ + bounds.maxZ) / 2;

  group.add(
    createBox(
      "fallback-floor",
      [width, 0.12, depth],
      [cx, -0.06, cz],
      createMat(0x223049),
    ),
    createBox(
      "fallback-back-wall",
      [width, 2.5, 0.14],
      [cx, 1.25, bounds.minZ],
      createMat(0x12305a),
    ),
    createBox(
      "fallback-left-wall",
      [0.14, 2.5, depth],
      [bounds.minX, 1.25, cz],
      createMat(0x12305a),
    ),
    createBox(
      "fallback-right-wall",
      [0.14, 2.5, depth],
      [bounds.maxX, 1.25, cz],
      createMat(0x12305a),
    ),
    createBox(
      "fallback-board",
      [4.6, 1.3, 0.08],
      [0.5, 1.4, bounds.minZ + 0.12],
      createMat(0x1e463a),
    ),
  );

  return group;
}

/** Same look tweaks the earlier build applied to the low-poly textures. */
function tuneClassroomMaterial(material: THREE.Material) {
  if (!(material instanceof THREE.MeshStandardMaterial)) {
    return material;
  }

  if (material.map) {
    material.color.setScalar(1.25);
    material.emissive = new THREE.Color(0xffffff);
    material.emissiveIntensity = 0.16;
    material.emissiveMap = material.map;
    material.map.colorSpace = THREE.SRGBColorSpace;
    material.map.magFilter = THREE.NearestFilter;
    material.map.minFilter = THREE.NearestMipmapNearestFilter;
  } else {
    material.color.multiplyScalar(0.74);
  }

  material.roughness = Math.max(material.roughness, 0.78);
  material.metalness = Math.min(material.metalness, 0.08);
  // Front faces only: from just outside a wall you look straight into the room.
  material.side = THREE.FrontSide;
  return material;
}

/** Bake the GLB into world-space geometries at the same placement the earlier build used. */
function collectLayoutInputs(gltfScene: THREE.Object3D): {
  inputs: LayoutInput[];
  materials: THREE.Material[];
} {
  const bounds = new THREE.Box3().setFromObject(gltfScene);
  const size = bounds.getSize(new THREE.Vector3());
  const center = bounds.getCenter(new THREE.Vector3());
  const scale = 6.2 / Math.max(size.x, size.y, size.z);

  gltfScene.scale.setScalar(scale);
  gltfScene.position.set(
    -center.x * scale,
    -bounds.min.y * scale - 0.08,
    -center.z * scale + 0.25,
  );
  gltfScene.updateMatrixWorld(true);

  const inputs: LayoutInput[] = [];
  const materials: THREE.Material[] = [];

  gltfScene.traverse((object) => {
    if (!(object instanceof THREE.Mesh)) {
      return;
    }

    const geometry = (object.geometry as THREE.BufferGeometry)
      .clone()
      .applyMatrix4(object.matrixWorld);
    const material = Array.isArray(object.material)
      ? object.material[0]
      : object.material;
    let materialIndex = materials.indexOf(material);

    if (materialIndex === -1) {
      materialIndex = materials.push(tuneClassroomMaterial(material)) - 1;
    }

    inputs.push({ geometry, materialIndex });
  });

  return { inputs, materials };
}

function createBoardFrame(board: Aabb) {
  const frameMat = createMat(0x6f4a2c, { roughness: 0.9 });
  const group = new THREE.Group();
  group.name = "whiteboard-frame";

  const thickness = 0.07;
  const depth = 0.09;
  const x = board.max.x + depth / 2 - 0.02;
  const width = board.max.z - board.min.z + thickness * 2;
  const height = board.max.y - board.min.y + thickness * 2;
  const cy = (board.min.y + board.max.y) / 2;
  const cz = (board.min.z + board.max.z) / 2;

  group.add(
    createBox(
      "frame-top",
      [depth, thickness, width],
      [x, board.max.y + thickness / 2, cz],
      frameMat,
    ),
    createBox(
      "frame-bottom",
      [depth, thickness, width],
      [x, board.min.y - thickness / 2, cz],
      frameMat,
    ),
    createBox(
      "frame-left",
      [depth, height, thickness],
      [x, cy, board.min.z - thickness / 2],
      frameMat,
    ),
    createBox(
      "frame-right",
      [depth, height, thickness],
      [x, cy, board.max.z + thickness / 2],
      frameMat,
    ),
    // Chalk tray.
    createBox(
      "frame-tray",
      [0.16, 0.03, width * 0.5],
      [board.max.x + 0.08, board.min.y - thickness, cz],
      frameMat,
    ),
  );

  return group;
}

function createSeat(room: THREE.Group): Seat {
  const ring = new THREE.Mesh(
    new THREE.RingGeometry(0.42, 0.56, 48),
    createMat(0x67e8f9, {
      emissive: new THREE.Color(0x22d3ee),
      emissiveIntensity: 2,
      transparent: true,
      opacity: 0.9,
      side: THREE.DoubleSide,
    }),
  );
  ring.name = "seat-marker";
  ring.position.set(SEAT_STAND_POSITION.x, 0.02, SEAT_STAND_POSITION.z);
  ring.rotation.x = -Math.PI / 2;

  const hitArea = new THREE.Mesh(
    new THREE.CylinderGeometry(1.1, 1.1, 1.4, 12),
    new THREE.MeshBasicMaterial({ visible: false }),
  );
  hitArea.name = "seat-hit";
  hitArea.position.set(
    (SEAT_POSITION.x + SEAT_STAND_POSITION.x) / 2,
    0.7,
    SEAT_POSITION.z,
  );

  room.add(ring, hitArea);

  return {
    position: SEAT_POSITION.clone(),
    facingYaw: FACE_BOARD_YAW,
    standPosition: SEAT_STAND_POSITION.clone(),
    cameraPosition: new THREE.Vector3(
      SEAT_POSITION.x + 1.15,
      1.4,
      SEAT_POSITION.z + 0.6,
    ),
    cameraTarget: new THREE.Vector3(TUTOR_ANCHOR.x - 0.6, 1.1, TUTOR_ANCHOR.z),
    ring,
    hitArea,
  };
}

export function createClassroom(scene: THREE.Scene): Classroom {
  const room = new THREE.Group();
  room.name = "classgame-classroom";

  const collision: Collision = { boxes: [], bounds: DEFAULT_BOUNDS };
  const classmateSeats: ClassmateSeat[] = [];
  const fallback = createFallbackRoom(DEFAULT_BOUNDS);
  room.add(fallback);

  const seat = createSeat(room);

  // Teacher: a placeholder until Emily's model is in, plus the glowing "thinking" aura at her feet.
  const tutor = new THREE.Group();
  tutor.name = "teacher";
  tutor.position.copy(TUTOR_ANCHOR);
  tutor.rotation.y = -Math.PI / 2; // face +x: toward the student's desk
  tutor.add(createPlaceholder("teacher-placeholder"));

  const tutorRing = new THREE.Mesh(
    new THREE.TorusGeometry(0.42, 0.018, 8, 40),
    createMat(0x67e8f9, {
      emissive: new THREE.Color(0x22d3ee),
      emissiveIntensity: 2.6,
    }),
  );
  tutorRing.name = "teacher-aura";
  tutorRing.position.set(TUTOR_ANCHOR.x, 0.03, TUTOR_ANCHOR.z);
  tutorRing.rotation.x = Math.PI / 2;

  const tutorLight = new THREE.PointLight(0x22d3ee, 2.5, 2.8);
  tutorLight.position.set(TUTOR_ANCHOR.x, 1.1, TUTOR_ANCHOR.z);

  // You come in through the door at the front-left corner, facing the desk. A step away
  // from the corner so the camera has its 2.5 m behind you and doesn't need to swing.
  const spawn = new THREE.Vector3(-3.6, 0, -3.6);
  const spawnYaw = Math.atan2(
    -(SEAT_STAND_POSITION.x - spawn.x),
    -(SEAT_STAND_POSITION.z - spawn.z),
  );
  const player = new THREE.Group();
  player.name = "student";
  player.position.copy(spawn);
  player.rotation.y = spawnYaw;
  player.add(createPlaceholder("student-placeholder"));

  const models: Classroom["models"] = { player: null, tutor: null };

  room.add(tutor, tutorRing, tutorLight, player);
  scene.add(room);

  const ambient = new THREE.HemisphereLight(0x9ee7ff, 0x111827, 2.4);
  const key = new THREE.DirectionalLight(0xffe5ad, 2.8);
  key.position.set(-5, 8, 6);
  const warmFill = new THREE.DirectionalLight(0xffd48a, 2.2);
  warmFill.position.set(6, 7, 8);
  const frontLight = new THREE.PointLight(0xfff0bd, 6, 12);
  frontLight.position.set(0.5, 2.3, -3.5);
  const backLight = new THREE.PointLight(0xfff0bd, 6, 12);
  backLight.position.set(0.5, 2.3, 3.5);

  scene.add(ambient, key, warmFill, frontLight, backLight);

  let disposed = false;
  let layoutMesh: THREE.Mesh | null = null;

  void createChibiAvatar(
    getAvatarPreset(teacherAvatarPresetId),
    TEACHER_AVATAR_HEIGHT,
  )
    .then((model) => {
      if (disposed) {
        return;
      }

      tutor.clear();
      tutor.add(model);
      models.tutor = model;
    })
    .catch((error: unknown) => {
      console.warn(
        "Teacher avatar failed to load; keeping the placeholder.",
        error,
      );
    });

  new GLTFLoader()
    .loadAsync(CLASSROOM_MODEL_URL)
    .then((gltf) => {
      if (disposed) {
        return;
      }

      const { inputs, materials } = collectLayoutInputs(gltf.scene);
      const layout = layoutClassroom(inputs, {
        spread: ROOM_SPREAD,
        furnitureShiftZ: FURNITURE_SHIFT_Z,
      });

      // Slot 2 is the whiteboard, repainted to match the lesson board.
      const boardMaterial = createMat(0x1e463a, {
        roughness: 0.95,
        metalness: 0,
        emissive: new THREE.Color(0x0b2a20),
        emissiveIntensity: 0.35,
        side: THREE.DoubleSide,
      });
      const slots: THREE.Material[] = [];
      for (let index = 0; index < layout.materialCount; index += 1) {
        slots[index] =
          materials[index] ?? (index === 2 ? boardMaterial : materials[0]);
      }
      slots[2] = boardMaterial;

      layoutMesh = new THREE.Mesh(layout.geometry, slots);
      layoutMesh.name = "low-poly-classroom-layout";
      fallback.visible = false;
      room.add(layoutMesh);

      if (layout.board) {
        room.add(createBoardFrame(layout.board));
      }

      collision.boxes = layout.colliders;
      collision.bounds = layout.bounds;
      classmateSeats.push(
        ...findClassmateSeats(
          layout,
          SEAT_POSITION,
          STUDENT_AVATAR_HEIGHT,
          FACE_BOARD_YAW,
        ),
      );
    })
    .catch((error: unknown) => {
      console.warn(
        "ClassGame classroom model failed to load; keeping the procedural room.",
        error,
      );
    });

  return {
    room,
    player,
    seat,
    tutor,
    models,
    tutorRing,
    tutorLight,
    collision,
    classmateSeats,
    spawn,
    spawnYaw,
    dispose() {
      disposed = true;
      layoutMesh?.geometry.dispose();

      scene.traverse((object) => {
        if (!(object instanceof THREE.Mesh)) {
          return;
        }

        object.geometry.dispose();

        const materials = Array.isArray(object.material)
          ? object.material
          : [object.material];
        materials.forEach((material) => material.dispose());
      });
    },
  };
}
