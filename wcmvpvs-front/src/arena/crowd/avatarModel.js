import * as THREE from "three";
import { mergeGeometries } from "three/addons/utils/BufferGeometryUtils.js";

function headGeometry(detail) {
  const g = new THREE.SphereGeometry(1, detail ? 24 : 8, detail ? 18 : 6),
    p = g.attributes.position;
  for (let i = 0; i < p.count; i++) {
    const y = p.getY(i);
    const cheek = 1 + 0.065 * Math.exp(-(((y + 0.25) / 0.35) ** 2));
    const chin = y < -0.4 ? 1 - (Math.abs(y) - 0.4) * 0.23 : 1;
    p.setX(i, p.getX(i) * cheek * chin);
  }
  g.computeVertexNormals();
  return g;
}
function shirtGeometry(detail) {
  return new THREE.LatheGeometry(
    [
      [-0.5, 0.34],
      [-0.48, 0.37],
      [-0.32, 0.38],
      [0.12, 0.58],
      [0.28, 0.66],
      [0.4, 0.55],
      [0.48, 0.27],
      [0.5, 0.23],
    ].map(([y, r]) => new THREE.Vector2(r, y)),
    detail ? 24 : 8,
  );
}
function hairGeometry(style, detail) {
  const pieces = [];
  const cap = new THREE.SphereGeometry(
    1.04,
    detail ? 24 : 7,
    detail ? 14 : 3,
    0,
    Math.PI * 2,
    0,
    1.65,
  );
  const vertices = cap.attributes.position;
  for (let i = 0; i < vertices.count; i++) {
    const x = vertices.getX(i),
      z = vertices.getZ(i),
      radial = Math.hypot(x, z);
    if (radial < 1e-6) {
      if (style === 0) vertices.setXYZ(i, 0.035, 1.16, -0.06);
      continue;
    }
    const theta = Math.acos(
      THREE.MathUtils.clamp(vertices.getY(i) / 1.04, -1, 1),
    );
    const front = Math.max(0, -z / radial) ** 2;
    const angle = (theta * (1.65 - front * (style === 0 ? 0.56 : 0.78))) / 1.65;
    const r = 1.04 * Math.sin(angle);
    const sweep =
      style === 0
        ? Math.max(0, Math.cos(angle)) * (0.12 + 0.07 * Math.sin(x * 2))
        : 0;
    vertices.setXYZ(
      i,
      (x / radial) * r + sweep * 0.3,
      1.04 * Math.cos(angle) + sweep,
      (z / radial) * r * 1.015 - sweep * 0.6,
    );
  }
  pieces.push(cap);
  const clump = (x, y, z, sx, sy, sz, turn = 0) => {
    const g = new THREE.SphereGeometry(1, detail ? 16 : 5, detail ? 10 : 3);
    g.scale(sx, sy, sz);
    g.rotateZ(turn);
    g.translate(x, y, z);
    pieces.push(g);
  };
  // The short cut is one swept continuous surface, without primitive tufts.
  if (style === 1) {
    for (const sign of [-1, 1])
      clump(sign * 0.82, 0.05, 0.05, 0.26, 0.78, 0.67, sign * 0.14);
    clump(0.25, 0.64, -0.73, 0.75, 0.36, 0.4, 0.3);
  }
  if (style === 2) {
    clump(0, -0.25, 0.6, 0.9, 1.05, 0.55);
    for (const sign of [-1, 1])
      clump(sign * 0.83, -0.23, 0.1, 0.25, 0.88, 0.45, sign * -0.1);
    clump(-0.2, 0.69, -0.73, 0.73, 0.29, 0.35, -0.15);
  }
  if (style === 3) {
    const count = detail ? 16 : 5;
    for (let i = 0; i < count; i++) {
      const angle = (i / count) * Math.PI * 2;
      clump(
        Math.cos(angle) * 0.74,
        0.54 + (i % 3) * 0.1,
        Math.sin(angle) * 0.72,
        detail ? 0.29 : 0.35,
        detail ? 0.32 : 0.38,
        detail ? 0.29 : 0.35,
      );
    }
    if (detail) clump(0, 0.98, 0, 0.55, 0.32, 0.54);
  }
  if (style === 4) {
    clump(0, -0.25, 0.98, 0.52, 0.9, 0.44, -0.15);
    clump(0.1, 0.7, -0.73, 0.73, 0.33, 0.37, 0.25);
  }
  const merged = mergeGeometries(pieces, false);
  pieces.forEach((p) => p.dispose());
  merged.computeVertexNormals();
  return merged;
}
function smileGeometry(detail) {
  const curve = new THREE.QuadraticBezierCurve3(
    new THREE.Vector3(-0.047, 0, 0),
    new THREE.Vector3(0, -0.019, -0.012),
    new THREE.Vector3(0.047, 0.005, 0),
  );
  return new THREE.TubeGeometry(
    curve,
    detail ? 10 : 4,
    0.006,
    detail ? 5 : 3,
    false,
  );
}
function handGeometry(sign) {
  const pieces = [];
  const palm = new THREE.SphereGeometry(1, 12, 8);
  palm.scale(0.55, 0.6, 0.32);
  pieces.push(palm);
  for (let i = 0; i < 4; i++) {
    const length = [0.34, 0.5, 0.46, 0.3][i];
    const finger = new THREE.CapsuleGeometry(0.115, length, 2, 7);
    finger.translate((i - 1.5) * 0.275, 0.51 + length * 0.2, -0.04);
    pieces.push(finger);
  }
  const thumb = new THREE.CapsuleGeometry(0.17, 0.32, 3, 8);
  thumb.rotateZ(-sign * 0.65);
  thumb.translate(sign * 0.59, 0.02, 0);
  pieces.push(thumb);
  const g = mergeGeometries(pieces, false);
  pieces.forEach((p) => p.dispose());
  return g;
}
function sneakerGeometry() {
  const g = new THREE.SphereGeometry(1, 16, 10),
    p = g.attributes.position;
  for (let i = 0; i < p.count; i++) {
    p.setX(i, p.getX(i) * (0.94 - p.getZ(i) * 0.06));
    p.setY(i, Math.max(-0.55, p.getY(i) * 0.85));
  }
  g.computeVertexNormals();
  return g;
}
function laceGeometry() {
  const pieces = [0, -0.23, -0.46].map((z) => {
    const y = Math.sqrt(1 - z * z) * 0.85;
    return new THREE.TubeGeometry(
      new THREE.QuadraticBezierCurve3(
        new THREE.Vector3(-0.48, y - 0.18, z),
        new THREE.Vector3(0, y + 0.2, z - 0.025),
        new THREE.Vector3(0.48, y - 0.18, z),
      ),
      6,
      0.024,
      4,
      false,
    );
  });
  const g = mergeGeometries(pieces, false);
  pieces.forEach((p) => p.dispose());
  return g;
}

// One batch per geometry/material, shared across all actors at this LOD.
// Detailed slots are bounded by createCrowd; far slots never allocate a rig.
export function createAvatarBatch(
  parent,
  capacity,
  { detail = false, prefix = "fan" } = {},
) {
  const group = new THREE.Group();
  group.name = prefix + "-lod";
  parent.add(group);
  const cloth = new THREE.MeshStandardMaterial({ roughness: 0.8 }),
    jersey = new THREE.MeshStandardMaterial({ roughness: 0.84 }),
    skin = new THREE.MeshStandardMaterial({ roughness: 0.64 }),
    textile = new THREE.MeshStandardMaterial({ roughness: 0.98 });
  const sphere = new THREE.SphereGeometry(1, detail ? 16 : 5, detail ? 10 : 3),
    capsule = new THREE.CapsuleGeometry(
      0.5,
      1,
      detail ? 4 : 1,
      detail ? 12 : 5,
    ),
    box = new THREE.BoxGeometry(1, 1, 1);
  const nodes = [];
  function mesh(geometry, multiple, name, material = cloth) {
    const m = new THREE.InstancedMesh(geometry, material, capacity * multiple);
    m.name = name;
    m.frustumCulled = false;
    m.instanceMatrix.setUsage(THREE.DynamicDrawUsage);
    group.add(m);
    nodes.push({ mesh: m, multiple });
    return m;
  }
  const torso = mesh(shirtGeometry(detail), 1, prefix + "-shirts", jersey),
    heads = mesh(headGeometry(detail), 1, prefix + "-heads", skin);
  const limbs = mesh(capsule, 8, prefix + "-limbs-and-details", skin);
  const sleeves = detail
    ? mesh(
        new THREE.CylinderGeometry(0.43, 0.5, 2, 16),
        2,
        prefix + "-sleeves",
        cloth,
      )
    : null;
  const softCount = detail ? 36 : 18;
  const soft = mesh(sphere, softCount, prefix + "-soft-parts", skin);
  const scarves = mesh(box, 3, prefix + "-scarves", textile),
    accessories = mesh(box, 8, prefix + "-equipped-accessories", textile);
  const hairs = Array.from({ length: 5 }, (_, style) =>
    mesh(
      hairGeometry(style, detail),
      1,
      style === 0 ? prefix + "-hair-and-caps" : prefix + "-hair-" + style,
      skin,
    ),
  );
  const mouths = detail
    ? mesh(smileGeometry(true), 1, prefix + "-smiles", skin)
    : null;
  const hands = detail
    ? [-1, 1].map((sign) =>
        mesh(handGeometry(sign), 1, prefix + "-hand-" + sign, skin),
      )
    : null;
  const sneakers = detail
    ? mesh(sneakerGeometry(), 2, prefix + "-sneakers", cloth)
    : null;
  const laces = detail
    ? mesh(laceGeometry(), 2, prefix + "-laces", textile)
    : null;
  const glasses = detail
    ? mesh(
        new THREE.TorusGeometry(1, 0.085, 5, 16),
        2,
        prefix + "-glasses",
        skin,
      )
    : null;
  const collars = detail
    ? mesh(
        new THREE.TorusGeometry(1, 0.18, 5, 16),
        1,
        prefix + "-collars",
        textile,
      )
    : null;
  const dummy = new THREE.Object3D(),
    a = new THREE.Vector3(),
    b = new THREE.Vector3(),
    up = new THREE.Vector3(0, 1, 0),
    color = new THREE.Color();
  const headMeshes = new Set([heads, ...hairs, mouths, glasses]);
  let mappings = [],
    actors = [],
    looks = [],
    signatures = [],
    paint = false,
    actor,
    look,
    p,
    slot,
    cos,
    sin;
  function setActors(next, nextLooks) {
    actors = next;
    looks = nextLooks;
    signatures = [];
    const counts = [0, 0, 0, 0, 0];
    mappings = looks.map((l) => ({ style: l.hair, index: counts[l.hair]++ }));
    for (const node of nodes) node.mesh.count = actors.length * node.multiple;
    hairs.forEach((m, i) => (m.count = counts[i]));
  }
  function point(x, y, z, out, turn = 0) {
    const yaw = actor.seat.yaw + turn,
      c = Math.cos(yaw),
      s = Math.sin(yaw);
    return out.set(
      actor.seat.x +
        (x * c + z * s + Math.sin(actor.seat.yaw) * (p.forward || 0)) *
          look.width,
      actor.seat.y + y * look.height,
      actor.seat.z +
        (-x * s + z * c + Math.cos(actor.seat.yaw) * (p.forward || 0)) *
          look.width,
    );
  }
  function shape(m, index, x, y, z, sx, sy, sz, tint, turn = 0, roll = 0) {
    const facePart = m === soft ? index % softCount : -1;
    const equipmentPart = m === accessories ? index % 8 : -1;
    if (
      headMeshes.has(m) ||
      facePart === 1 ||
      facePart === 2 ||
      (facePart >= 12 && facePart <= (detail ? 30 : 15)) ||
      equipmentPart === 2 ||
      equipmentPart === 3
    ) {
      // Keep a slightly oversized cartoon head while giving shoulders a heroic silhouette.
      const center = p.hip + 0.565,
        scale = 0.91;
      x *= scale;
      y = center + (y - center) * scale;
      z *= scale;
      sx *= scale;
      sy *= scale;
      sz *= scale;
    }
    point(x, y, z, dummy.position, turn);
    dummy.rotation.set(0, actor.seat.yaw + turn, roll);
    dummy.scale.set(sx * look.width, sy * look.height, sz * look.width);
    dummy.updateMatrix();
    m.setMatrixAt(index, dummy.matrix);
    if (paint) m.setColorAt(index, color.set(tint));
  }
  function joint(index, from, to, width, tint, target = limbs, overlap = 0.9) {
    point(...from, a);
    point(...to, b);
    dummy.position.copy(a).add(b).multiplyScalar(0.5);
    const length = a.distanceTo(b);
    dummy.quaternion.setFromUnitVectors(up, b.sub(a).normalize());
    dummy.scale.set(
      width * look.width,
      (length + width * overlap) / 2,
      width * look.width,
    );
    dummy.updateMatrix();
    target.setMatrixAt(index, dummy.matrix);
    if (paint) target.setColorAt(index, color.set(tint));
  }
  function clear(index) {
    dummy.scale.set(0, 0, 0);
    dummy.position.set(0, 0, 0);
    dummy.rotation.set(0, 0, 0);
    dummy.updateMatrix();
    for (const n of nodes) {
      if (hairs.includes(n.mesh)) continue;
      for (let part = 0; part < n.multiple; part++)
        n.mesh.setMatrixAt(index * n.multiple + part, dummy.matrix);
    }
    const mapping = mappings[index];
    if (mapping) hairs[mapping.style].setMatrixAt(mapping.index, dummy.matrix);
  }
  function render(index, pose) {
    slot = index;
    actor = actors[index];
    look = looks[index];
    p = pose;
    const e = look.equipment,
      hip = p.hip,
      h = hip + 0.565;
    // Looks are replaced on wardrobe changes; poses never mutate them.
    paint = signatures[index] !== look;
    signatures[index] = look;
    shape(
      torso,
      index,
      0,
      hip + 0.205,
      0,
      0.34,
      0.38,
      0.225,
      look.shirt,
      p.turn,
    );
    shape(
      heads,
      index,
      0,
      h,
      -0.005,
      0.153,
      0.18,
      0.138,
      look.skin,
      p.headTurn,
    );
    const mapping = mappings[index];
    shape(
      hairs[mapping.style],
      mapping.index,
      0,
      h,
      -0.005,
      e.hat ? 0.149 : 0.156,
      e.hat ? 0.125 : 0.183,
      0.143,
      look.hairColor,
      p.headTurn,
    );
    let part = 0;
    for (const arm of p.arms) {
      if (detail) {
        dummy.scale.setScalar(0);
        dummy.updateMatrix();
        limbs.setMatrixAt(index * 8 + part, dummy.matrix);
        joint(
          index * 2 + part / 2,
          arm.shoulder,
          arm.elbow,
          0.096,
          look.shirt,
          sleeves,
          0,
        );
        part++;
      } else
        joint(index * 8 + part++, arm.shoulder, arm.elbow, 0.086, look.shirt);
      joint(index * 8 + part++, arm.elbow, arm.hand, 0.069, look.skin);
    }
    for (const leg of p.legs) {
      joint(index * 8 + part++, leg.hip, leg.knee, 0.105, look.pants);
      joint(index * 8 + part++, leg.knee, leg.foot, 0.084, look.pants);
    }
    const s = (part, x, y, z, sx, sy, sz, tint, turn = 0, roll = 0) =>
      shape(
        soft,
        index * softCount + part,
        x,
        y,
        z,
        sx,
        sy,
        sz,
        tint,
        turn,
        roll,
      );
    s(0, 0, hip + 0.398, 0, 0.063, 0.073, 0.062, look.skin);
    for (let side = 0; side < 2; side++) {
      const sign = side ? 1 : -1,
        arm = p.arms[side],
        leg = p.legs[side];
      s(
        1 + side,
        sign * 0.155,
        h,
        -0.002,
        0.031,
        0.046,
        0.026,
        look.skin,
        p.headTurn,
      );
      s(3 + side, ...arm.hand, 0.045, 0.053, 0.036, look.skin);
      s(
        5 + side,
        leg.foot[0],
        leg.foot[1] - 0.002,
        leg.foot[2] - 0.04,
        0.061,
        0.045,
        0.115,
        look.shoes,
      );
      s(
        7 + side,
        leg.foot[0],
        leg.foot[1] - 0.033,
        leg.foot[2] - 0.04,
        0.065,
        0.016,
        0.12,
        "#344450",
      );
      s(
        9 + side,
        sign * 0.19,
        hip + 0.33,
        0,
        0.051,
        0.055,
        0.062,
        look.shirt,
        p.turn,
      );
    }
    s(11, 0, hip - 0.035, 0, 0.153, 0.071, 0.105, look.pants);
    for (let side = 0; side < 2; side++) {
      s(
        (detail ? 32 : 16) + side,
        ...p.arms[side].elbow,
        0.039,
        0.042,
        0.039,
        look.skin,
      );
      if (detail)
        s(34 + side, ...p.legs[side].knee, 0.047, 0.05, 0.047, look.pants);
    }
    if (detail) {
      for (let side = 0; side < 2; side++) {
        const arm = p.arms[side],
          leg = p.legs[side];
        point(...arm.elbow, a);
        point(...arm.hand, b);
        a.copy(b).sub(a).normalize();
        dummy.position.copy(b).addScaledVector(a, 0.032 * look.height);
        dummy.quaternion.setFromUnitVectors(up, a);
        dummy.scale.set(
          0.054 * look.width,
          0.068 * look.height,
          0.043 * look.width,
        );
        dummy.updateMatrix();
        hands[side].setMatrixAt(index, dummy.matrix);
        if (paint) hands[side].setColorAt(index, color.set(look.skin));
        shape(
          sneakers,
          index * 2 + side,
          leg.foot[0],
          leg.foot[1] - 0.003,
          leg.foot[2] - 0.04,
          0.066,
          0.052,
          0.12,
          look.shoes,
        );
        shape(
          laces,
          index * 2 + side,
          leg.foot[0],
          leg.foot[1] - 0.003,
          leg.foot[2] - 0.04,
          0.066,
          0.052,
          0.12,
          "#f9f6ed",
        );
        s(
          3 + side,
          leg.foot[0],
          leg.foot[1] + 0.032,
          leg.foot[2] - 0.042,
          0.038,
          0.009,
          0.057,
          look.shirt,
        );
        s(
          5 + side,
          leg.foot[0],
          leg.foot[1] + 0.01,
          leg.foot[2] + 0.058,
          0.056,
          0.03,
          0.02,
          look.shirt,
        );
      }
      const iris = ["#576e78", "#705440", "#496d53"][Math.abs(actor.seed) % 3];
      for (let side = 0; side < 2; side++)
        shape(
          glasses,
          index * 2 + side,
          side ? 0.061 : -0.061,
          h + 0.026,
          -0.155,
          e.glasses ? 0.035 : 0,
          e.glasses ? 0.041 : 0,
          e.glasses ? 0.02 : 0,
          "#293744",
          p.headTurn,
        );
      for (let side = 0; side < 2; side++) {
        const sign = side ? 1 : -1,
          eyeX = sign * 0.061;
        s(
          12 + side,
          eyeX,
          h + 0.026,
          -0.125,
          0.026,
          0.034,
          0.024,
          "#fbf8f0",
          p.headTurn,
        );
        s(
          14 + side,
          eyeX + Math.sin(p.headTurn) * 0.006,
          h + 0.025,
          -0.147,
          0.015,
          0.022,
          0.011,
          iris,
          p.headTurn,
        );
        s(
          16 + side,
          eyeX + Math.sin(p.headTurn) * 0.006,
          h + 0.026,
          -0.156,
          0.009,
          0.014,
          0.006,
          "#202c33",
          p.headTurn,
        );
        s(
          18 + side,
          eyeX - 0.005,
          h + 0.036,
          -0.162,
          0.0045,
          0.006,
          0.003,
          "#fffef2",
          p.headTurn,
        );
        s(
          20 + side,
          eyeX,
          h + 0.087,
          -0.126,
          0.038,
          0.009,
          0.009,
          look.hairColor,
          p.headTurn,
          sign * (look.expression === 2 ? -0.22 : -0.09),
        );
        s(
          22 + side,
          sign * 0.097,
          h - 0.039,
          -0.109,
          0,
          0,
          0,
          look.skin,
          p.headTurn,
        );
        s(
          24 + side,
          sign * 0.174,
          h - 0.002,
          -0.005,
          0.009,
          0.025,
          0.015,
          "#c78c78",
          p.headTurn,
        );
      }
      s(26, 0, h - 0.024, -0.146, 0.017, 0.025, 0.023, look.skin, p.headTurn);
      s(
        27,
        0,
        h - 0.083,
        -0.127,
        look.expression === 2 ? 0.027 : 0,
        look.expression === 2 ? 0.021 : 0,
        0.009,
        "#713e39",
        p.headTurn,
      );
      s(
        28,
        0,
        h - 0.073,
        -0.136,
        look.expression === 2 ? 0.026 : 0,
        look.expression === 2 ? 0.007 : 0,
        0.006,
        "#f9f1de",
        p.headTurn,
      );
      s(
        29,
        0,
        h + 0.13,
        -0.047,
        e.hat ? 0.159 : 0,
        e.hat ? 0.105 : 0,
        e.hat ? 0.137 : 0,
        "#ddbc41",
        p.headTurn,
      );
      s(
        30,
        0,
        h + 0.064,
        -0.145,
        e.hat ? 0.15 : 0,
        e.hat ? 0.018 : 0,
        e.hat ? 0.1 : 0,
        look.shirt,
        p.headTurn,
      );
      s(
        31,
        0,
        hip + 0.37,
        0.04,
        e.shirt === "hoodie" ? 0.13 : 0,
        e.shirt === "hoodie" ? 0.11 : 0,
        0.105,
        look.shirt,
      );
      shape(
        mouths,
        index,
        0,
        h - 0.067,
        -0.134,
        look.expression === 2 ? 0 : 0.85,
        look.expression === 1 ? 0.14 : 0.7,
        1,
        "#744840",
        p.headTurn,
      );
      shape(
        collars,
        index,
        0,
        hip + 0.385,
        0,
        0.068,
        0.061,
        0.013,
        "#eee8d7",
        0,
        0,
      );
      dummy.rotation.x = Math.PI / 2;
      dummy.updateMatrix();
      collars.setMatrixAt(index, dummy.matrix);
    } else {
      for (let side = 0; side < 2; side++)
        s(
          12 + side,
          side ? 0.061 : -0.061,
          h + 0.026,
          -0.14,
          0.018,
          0.025,
          0.009,
          "#25313a",
          p.headTurn,
        );
      s(14, 0, h - 0.023, -0.145, 0.02, 0.021, 0.022, look.skin, p.headTurn);
      s(15, 0, h - 0.079, -0.13, 0.029, 0.006, 0.006, "#744840", p.headTurn);
    }
    for (let j = 0; j < 3; j++) {
      const raised = p.scarf;
      shape(
        scarves,
        index * 3 + j,
        j === 0 ? 0 : j === 1 ? -0.35 : 0.35,
        hip + 0.1 + 0.76 * raised - (j === 0 ? 0 : 0.08 * raised),
        -0.045,
        j === 0 ? (raised > 0.025 ? 0.79 : 0) : 0.065 * raised,
        j === 0 ? 0.09 * raised : 0.18 * raised,
        0.045 * raised,
        j === 0 ? "#3659a1" : "#e6c940",
      );
    }
    const extra = (part, x, y, z, sx, sy, sz, enabled, tint) =>
      shape(
        accessories,
        index * 8 + part,
        x,
        y,
        z,
        enabled ? sx : 0,
        enabled ? sy : 0,
        enabled ? sz : 0,
        tint,
      );
    extra(0, 0, hip + 0.365, -0.036, 0.32, 0.055, 0.2, e.scarf, "#dcbc40");
    extra(1, -0.115, hip + 0.23, -0.13, 0.075, 0.26, 0.035, e.scarf, "#3659a1");
    extra(
      2,
      0,
      h + 0.028,
      -0.16,
      detail ? 0.04 : 0.16,
      0.012,
      0.022,
      e.glasses,
      "#293744",
    );
    extra(
      3,
      0,
      h + 0.06,
      -0.13,
      0.25,
      0.025,
      0.15,
      e.hat && !detail,
      "#ddbc41",
    );
    extra(4, 0.29, hip + 0.45, -0.14, 0.018, 0.64, 0.018, e.flag, "#e5ded0");
    extra(5, 0.4, hip + 0.66, -0.14, 0.23, 0.17, 0.025, e.flag, "#e4c647");
    extra(
      6,
      0,
      hip + 0.37,
      0.05,
      0.24,
      0.2,
      0.12,
      e.shirt === "hoodie" && !detail,
      look.shirt,
    );
    extra(
      7,
      0,
      hip + 0.245,
      -0.145,
      0.17,
      0.018,
      0.012,
      e.shirt === "club-shirt",
      "#e5c444",
    );
    if (paint)
      for (const n of nodes)
        if (n.mesh.instanceColor) n.mesh.instanceColor.needsUpdate = true;
  }
  function flush() {
    for (const n of nodes) n.mesh.instanceMatrix.needsUpdate = true;
  }
  function setJerseyTexture(texture) {
    jersey.map = texture;
    jersey.needsUpdate = true;
  }
  return {
    group,
    setActors,
    render,
    clear,
    flush,
    setJerseyTexture,
    nodes,
    triangleBudget() {
      return nodes.reduce(
        (sum, n) =>
          sum +
          (n.mesh.count *
            (n.mesh.geometry.index?.count ||
              n.mesh.geometry.attributes.position.count)) /
            3,
        0,
      );
    },
  };
}
