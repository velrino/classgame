import * as THREE from "three";

/**
 * The classroom GLB is one merged mesh, so "move the desks apart" has to be
 * done on the geometry itself. We split the mesh into connected islands,
 * sort them into the room shell (floor, walls, ceiling) and props (furniture,
 * boards, door), then stretch the shell and spread the props without scaling
 * them. The same pass also finds the whiteboard and the walkable colliders.
 */

export type Aabb = { readonly min: THREE.Vector3; readonly max: THREE.Vector3 };

export type RoomBounds = {
  readonly minX: number;
  readonly maxX: number;
  readonly minZ: number;
  readonly maxZ: number;
  readonly floorY: number;
};

export type LayoutInput = {
  /** Geometry already in world space (transform baked in). */
  readonly geometry: THREE.BufferGeometry;
  /** Which output material this geometry renders with (0 textured, 1 glass). */
  readonly materialIndex: number;
};

export type LayoutOptions = {
  /** Horizontal stretch of the room and spread of prop positions. 2 = twice the room. */
  readonly spread: number;
  /** Extra shift of floor-standing furniture toward +z, to widen the front aisle. */
  readonly furnitureShiftZ: number;
};

export type LayoutResult = {
  readonly geometry: THREE.BufferGeometry;
  /** Output material slots: 0 textured, 1 glass, 2 board (chalk green). */
  readonly materialCount: number;
  readonly colliders: readonly Aabb[];
  readonly bounds: RoomBounds;
  /** World bbox of the recoloured whiteboard, for the procedural frame. */
  readonly board: Aabb | null;
  /** World bboxes of the individual prop pieces (a chair, a desk top, a leg), for picking out furniture. */
  readonly props: readonly Aabb[];
};

const QUANT = 500; // 2 mm vertex weld
const WALL_TOUCH = 0.12;
const GROUP_GAP_XZ = 0.05;
const GROUP_GAP_Y = 0.16;
/** Prop pieces smaller than this (screws, feet) aren't worth reporting. */
const PROP_MIN_TRIS = 100;

type Island = {
  readonly id: number;
  triCount: number;
  min: THREE.Vector3;
  max: THREE.Vector3;
  shell: boolean;
  group: number;
};

class UnionFind {
  private readonly parent: Int32Array;

  constructor(size: number) {
    this.parent = new Int32Array(size);
    for (let index = 0; index < size; index += 1) {
      this.parent[index] = index;
    }
  }

  find(index: number) {
    let root = index;
    while (this.parent[root] !== root) {
      root = this.parent[root];
    }
    while (this.parent[index] !== root) {
      const next = this.parent[index];
      this.parent[index] = root;
      index = next;
    }
    return root;
  }

  union(a: number, b: number) {
    const rootA = this.find(a);
    const rootB = this.find(b);
    if (rootA !== rootB) {
      this.parent[rootA] = rootB;
    }
  }
}

function footprintsTouch(a: Island, b: Island) {
  const xz =
    a.min.x - GROUP_GAP_XZ <= b.max.x &&
    b.min.x - GROUP_GAP_XZ <= a.max.x &&
    a.min.z - GROUP_GAP_XZ <= b.max.z &&
    b.min.z - GROUP_GAP_XZ <= a.max.z;
  const y =
    a.min.y - GROUP_GAP_Y <= b.max.y && b.min.y - GROUP_GAP_Y <= a.max.y;
  return xz && y;
}

export function layoutClassroom(
  inputs: readonly LayoutInput[],
  options: LayoutOptions,
): LayoutResult {
  // ── 1. Merge inputs into flat arrays ──────────────────────────────────────
  let vertexCount = 0;
  let triCount = 0;
  for (const input of inputs) {
    vertexCount += input.geometry.getAttribute("position").count;
    triCount +=
      (input.geometry.index?.count ??
        input.geometry.getAttribute("position").count) / 3;
  }

  const positions = new Float32Array(vertexCount * 3);
  const normals = new Float32Array(vertexCount * 3);
  const uvs = new Float32Array(vertexCount * 2);
  const triVertices = new Uint32Array(triCount * 3);
  const triMaterial = new Uint8Array(triCount);

  let vertexOffset = 0;
  let triOffset = 0;
  for (const input of inputs) {
    const position = input.geometry.getAttribute("position");
    const normal = input.geometry.getAttribute("normal");
    const uv = input.geometry.getAttribute("uv");
    const count = position.count;

    for (let index = 0; index < count; index += 1) {
      const target = vertexOffset + index;
      positions[target * 3] = position.getX(index);
      positions[target * 3 + 1] = position.getY(index);
      positions[target * 3 + 2] = position.getZ(index);
      if (normal) {
        normals[target * 3] = normal.getX(index);
        normals[target * 3 + 1] = normal.getY(index);
        normals[target * 3 + 2] = normal.getZ(index);
      }
      if (uv) {
        uvs[target * 2] = uv.getX(index);
        uvs[target * 2 + 1] = uv.getY(index);
      }
    }

    const indexAttr = input.geometry.index;
    const localTriCount = (indexAttr?.count ?? count) / 3;
    for (let tri = 0; tri < localTriCount; tri += 1) {
      for (let corner = 0; corner < 3; corner += 1) {
        const local = indexAttr
          ? indexAttr.getX(tri * 3 + corner)
          : tri * 3 + corner;
        triVertices[(triOffset + tri) * 3 + corner] = vertexOffset + local;
      }
      triMaterial[triOffset + tri] = input.materialIndex;
    }

    vertexOffset += count;
    triOffset += localTriCount;
  }

  // ── 2. Connected islands: weld coincident vertices, then follow triangles ──
  const vertices = new UnionFind(vertexCount);
  const welded = new Map<string, number>();
  for (let index = 0; index < vertexCount; index += 1) {
    const key = `${Math.round(positions[index * 3] * QUANT)},${Math.round(positions[index * 3 + 1] * QUANT)},${Math.round(positions[index * 3 + 2] * QUANT)}`;
    const existing = welded.get(key);
    if (existing === undefined) {
      welded.set(key, index);
    } else {
      vertices.union(index, existing);
    }
  }
  for (let tri = 0; tri < triCount; tri += 1) {
    vertices.union(triVertices[tri * 3], triVertices[tri * 3 + 1]);
    vertices.union(triVertices[tri * 3 + 1], triVertices[tri * 3 + 2]);
  }

  const islandByRoot = new Map<number, Island>();
  const islands: Island[] = [];
  const triIsland = new Int32Array(triCount);
  for (let tri = 0; tri < triCount; tri += 1) {
    const root = vertices.find(triVertices[tri * 3]);
    let island = islandByRoot.get(root);
    if (!island) {
      island = {
        id: islands.length,
        triCount: 0,
        min: new THREE.Vector3(Infinity, Infinity, Infinity),
        max: new THREE.Vector3(-Infinity, -Infinity, -Infinity),
        shell: false,
        group: -1,
      };
      islandByRoot.set(root, island);
      islands.push(island);
    }
    island.triCount += 1;
    triIsland[tri] = island.id;
    for (let corner = 0; corner < 3; corner += 1) {
      const vertex = triVertices[tri * 3 + corner];
      const x = positions[vertex * 3];
      const y = positions[vertex * 3 + 1];
      const z = positions[vertex * 3 + 2];
      island.min.x = Math.min(island.min.x, x);
      island.min.y = Math.min(island.min.y, y);
      island.min.z = Math.min(island.min.z, z);
      island.max.x = Math.max(island.max.x, x);
      island.max.y = Math.max(island.max.y, y);
      island.max.z = Math.max(island.max.z, z);
    }
  }

  // ── 3. Shell vs props ─────────────────────────────────────────────────────
  const room = {
    min: new THREE.Vector3(Infinity, Infinity, Infinity),
    max: new THREE.Vector3(-Infinity, -Infinity, -Infinity),
  };
  for (const island of islands) {
    room.min.min(island.min);
    room.max.max(island.max);
  }
  const roomSize = new THREE.Vector3().subVectors(room.max, room.min);
  const center = new THREE.Vector3()
    .addVectors(room.min, room.max)
    .multiplyScalar(0.5);

  let floorTop = room.min.y;
  for (const island of islands) {
    const spanX = island.max.x - island.min.x;
    const spanZ = island.max.z - island.min.z;
    island.shell = spanX > roomSize.x * 0.5 || spanZ > roomSize.z * 0.5;
    // The floor slab: wide, flat, at the bottom.
    if (
      island.shell &&
      spanX > roomSize.x * 0.9 &&
      spanZ > roomSize.z * 0.9 &&
      island.min.y < room.min.y + 0.05
    ) {
      floorTop = Math.max(floorTop, island.max.y);
    }
  }

  // ── 4. Group props that touch (desk + legs + chair + books on it) ─────────
  const props = islands.filter((island) => !island.shell);
  const groups = new UnionFind(props.length);
  for (let a = 0; a < props.length; a += 1) {
    for (let b = a + 1; b < props.length; b += 1) {
      if (footprintsTouch(props[a], props[b])) {
        groups.union(a, b);
      }
    }
  }

  type Group = {
    min: THREE.Vector3;
    max: THREE.Vector3;
    delta: THREE.Vector3;
    members: Island[];
  };
  const groupByRoot = new Map<number, Group>();
  props.forEach((island, index) => {
    const root = groups.find(index);
    let group = groupByRoot.get(root);
    if (!group) {
      group = {
        min: island.min.clone(),
        max: island.max.clone(),
        delta: new THREE.Vector3(),
        members: [],
      };
      groupByRoot.set(root, group);
    }
    group.min.min(island.min);
    group.max.max(island.max);
    group.members.push(island);
  });

  const stretch = options.spread - 1;
  const groupList = [...groupByRoot.values()];
  for (const group of groupList) {
    const gx = (group.min.x + group.max.x) / 2;
    const gz = (group.min.z + group.max.z) / 2;
    const floorStanding = group.min.y < floorTop + 0.3;

    let dx = (gx - center.x) * stretch;
    let dz =
      (gz - center.z) * stretch + (floorStanding ? options.furnitureShiftZ : 0);

    // Things hanging on a wall stay flush with it: the face that touches the wall
    // moves exactly where the stretched wall surface moves (the wall gets thicker too).
    if (group.min.x - room.min.x < WALL_TOUCH) {
      dx = (group.min.x - center.x) * stretch;
    } else if (room.max.x - group.max.x < WALL_TOUCH) {
      dx = (group.max.x - center.x) * stretch;
    }
    if (group.min.z - room.min.z < WALL_TOUCH) {
      dz = (group.min.z - center.z) * stretch;
    } else if (room.max.z - group.max.z < WALL_TOUCH) {
      dz = (group.max.z - center.z) * stretch;
    }

    group.delta.set(dx, 0, dz);
    for (const member of group.members) {
      member.group = groupList.indexOf(group);
    }
  }

  // ── 5. The whiteboard: a tall thin panel on a side wall, not the front ───
  let boardIsland: Island | null = null;
  for (const island of props) {
    const spanX = island.max.x - island.min.x;
    const spanY = island.max.y - island.min.y;
    const spanZ = island.max.z - island.min.z;
    const onSideWall =
      island.min.x - room.min.x < WALL_TOUCH ||
      room.max.x - island.max.x < WALL_TOUCH;
    if (
      onSideWall &&
      spanX < 0.15 &&
      spanZ >= 2 &&
      spanY >= 0.9 &&
      island.min.y > floorTop + 0.4
    ) {
      if (!boardIsland || island.triCount > boardIsland.triCount) {
        boardIsland = island;
      }
    }
  }

  // ── 6. Move vertices ──────────────────────────────────────────────────────
  const moved = new Uint8Array(vertexCount);
  const yShift = -floorTop; // put the floor top at y = 0
  for (let tri = 0; tri < triCount; tri += 1) {
    const island = islands[triIsland[tri]];
    for (let corner = 0; corner < 3; corner += 1) {
      const vertex = triVertices[tri * 3 + corner];
      if (moved[vertex]) {
        continue;
      }
      moved[vertex] = 1;
      const x = positions[vertex * 3];
      const z = positions[vertex * 3 + 2];
      if (island.shell) {
        positions[vertex * 3] = center.x + (x - center.x) * options.spread;
        positions[vertex * 3 + 2] = center.z + (z - center.z) * options.spread;
      } else {
        const delta = groupList[island.group].delta;
        positions[vertex * 3] = x + delta.x;
        positions[vertex * 3 + 2] = z + delta.z;
      }
      positions[vertex * 3 + 1] += yShift;
    }
  }

  // ── 7. Rebuild geometry sorted by material, with groups ──────────────────
  const order = Array.from({ length: triCount }, (_, tri) => tri);
  const materialOf = (tri: number) =>
    boardIsland && triIsland[tri] === boardIsland.id ? 2 : triMaterial[tri];
  order.sort((a, b) => materialOf(a) - materialOf(b));

  const index = new Uint32Array(triCount * 3);
  const geometry = new THREE.BufferGeometry();
  let cursor = 0;
  let groupStart = 0;
  let currentMaterial = order.length ? materialOf(order[0]) : 0;
  for (const tri of order) {
    const material = materialOf(tri);
    if (material !== currentMaterial) {
      geometry.addGroup(groupStart, cursor - groupStart, currentMaterial);
      groupStart = cursor;
      currentMaterial = material;
    }
    index[cursor] = triVertices[tri * 3];
    index[cursor + 1] = triVertices[tri * 3 + 1];
    index[cursor + 2] = triVertices[tri * 3 + 2];
    cursor += 3;
  }
  geometry.addGroup(groupStart, cursor - groupStart, currentMaterial);
  geometry.setAttribute("position", new THREE.BufferAttribute(positions, 3));
  geometry.setAttribute("normal", new THREE.BufferAttribute(normals, 3));
  geometry.setAttribute("uv", new THREE.BufferAttribute(uvs, 2));
  geometry.setIndex(new THREE.BufferAttribute(index, 1));
  geometry.computeBoundingBox();
  geometry.computeBoundingSphere();

  // ── 8. Colliders and bounds in the new layout ────────────────────────────
  const colliders: Aabb[] = [];
  for (const group of groupList) {
    const footprint = Math.max(
      group.max.x - group.min.x,
      group.max.z - group.min.z,
    );
    const floorStanding = group.min.y < floorTop + 0.3;
    if (!floorStanding || footprint < 0.15) {
      continue;
    }
    colliders.push({
      min: new THREE.Vector3(
        group.min.x + group.delta.x,
        group.min.y + yShift,
        group.min.z + group.delta.z,
      ),
      max: new THREE.Vector3(
        group.max.x + group.delta.x,
        group.max.y + yShift,
        group.max.z + group.delta.z,
      ),
    });
  }

  const bounds: RoomBounds = {
    minX: center.x + (room.min.x - center.x) * options.spread,
    maxX: center.x + (room.max.x - center.x) * options.spread,
    minZ: center.z + (room.min.z - center.z) * options.spread,
    maxZ: center.z + (room.max.z - center.z) * options.spread,
    floorY: 0,
  };

  const propBoxes: Aabb[] = [];
  for (const island of props) {
    if (island.triCount < PROP_MIN_TRIS) {
      continue;
    }
    const delta = groupList[island.group].delta;
    propBoxes.push({
      min: new THREE.Vector3(
        island.min.x + delta.x,
        island.min.y + yShift,
        island.min.z + delta.z,
      ),
      max: new THREE.Vector3(
        island.max.x + delta.x,
        island.max.y + yShift,
        island.max.z + delta.z,
      ),
    });
  }

  const board = boardIsland
    ? (() => {
        const delta = groupList[boardIsland.group].delta;
        return {
          min: new THREE.Vector3(
            boardIsland.min.x + delta.x,
            boardIsland.min.y + yShift,
            boardIsland.min.z + delta.z,
          ),
          max: new THREE.Vector3(
            boardIsland.max.x + delta.x,
            boardIsland.max.y + yShift,
            boardIsland.max.z + delta.z,
          ),
        };
      })()
    : null;

  return {
    geometry,
    materialCount: 3,
    colliders,
    bounds,
    board,
    props: propBoxes,
  };
}
