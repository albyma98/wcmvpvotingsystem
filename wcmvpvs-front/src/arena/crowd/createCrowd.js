import * as THREE from "three";
import { createCrowdDirector } from "../actions/crowdDirector.js";

const skin = ["#d8ab85", "#a96f4a", "#f0c9a4", "#714b37", "#bc8965", "#e1b798"];
const shirts = [
  "#324e92",
  "#d9b72f",
  "#edf0db",
  "#405b68",
  "#e1c447",
  "#647a9b",
];
const trousers = ["#263741", "#3a4852", "#58616a", "#344362"];
const hair = ["#3c2d23", "#66503a", "#24272a", "#a37d45"];
const LIMBS = 14;

// A whole crowd uses five instanced meshes, with no skeletons or imported models.
export function createCrowd(parent, actors, { selectedId, onSelect } = {}) {
  const group = new THREE.Group();
  group.name = "crowd";
  parent.add(group);
  const material = new THREE.MeshStandardMaterial({ roughness: 0.88 });
  const boxGeometry = new THREE.BoxGeometry(1, 1, 1);
  function mesh(geometry, count, name) {
    const object = new THREE.InstancedMesh(geometry, material, count);
    object.name = name;
    object.frustumCulled = false;
    object.instanceMatrix.setUsage(THREE.DynamicDrawUsage);
    group.add(object);
    return object;
  }
  const torso = mesh(boxGeometry, actors.length, "fan-shirts");
  const heads = mesh(
    new THREE.SphereGeometry(1, 7, 5),
    actors.length,
    "fan-heads",
  );
  const hats = mesh(boxGeometry, actors.length, "fan-hair-and-caps");
  const limbs = mesh(
    boxGeometry,
    actors.length * LIMBS,
    "fan-limbs-and-details",
  );
  const scarves = mesh(boxGeometry, actors.length * 3, "fan-scarves");
  const accessories = mesh(
    boxGeometry,
    actors.length * 8,
    "fan-equipped-accessories",
  );
  let neighbors = new Set();
  const activeMaterial = new THREE.MeshBasicMaterial({
    color: "#b4d96b",
    transparent: true,
    opacity: 0.7,
  });
  const dots = new THREE.InstancedMesh(
    new THREE.SphereGeometry(0.045, 5, 3),
    activeMaterial,
    actors.length,
  );
  dots.frustumCulled = false;
  group.add(dots);
  const color = new THREE.Color(),
    dummy = new THREE.Object3D();
  actors.forEach((actor, i) => {
    const index = actor.seed;
    const shirt = shirts[index % shirts.length],
      face = skin[index % skin.length];
    torso.setColorAt(i, color.set(shirt));
    heads.setColorAt(i, color.set(face));
    hats.setColorAt(
      i,
      color.set(index % 5 === 0 ? shirt : hair[index % hair.length]),
    );
    for (let j = 0; j < LIMBS; j++) {
      const shade =
        j < 4
          ? j % 2 === 0
            ? shirt
            : face
          : j < 8
            ? trousers[index % trousers.length]
            : j < 10
              ? "#e5e5d8"
              : j === 10
                ? face
                : j < 13
                  ? "#26302c"
                  : "#e7cf54";
      limbs.setColorAt(i * LIMBS + j, color.set(shade));
    }
    for (let j = 0; j < 3; j++)
      scarves.setColorAt(i * 3 + j, color.set(j === 0 ? "#304d99" : "#f2cf38"));
  });
  const highlightMaterial = new THREE.MeshBasicMaterial({
    color: "#bfe86a",
    transparent: true,
    opacity: 0.85,
    depthTest: false,
  });
  const ring = new THREE.Mesh(
    new THREE.TorusGeometry(0.31, 0.024, 5, 24),
    highlightMaterial,
  );
  ring.rotation.x = Math.PI / 2;
  ring.renderOrder = 6;
  group.add(ring);
  const pin = new THREE.Mesh(
    new THREE.ConeGeometry(0.095, 0.18, 6),
    highlightMaterial,
  );
  pin.rotation.z = Math.PI;
  pin.renderOrder = 6;
  group.add(pin);
  const sector = new THREE.Group();
  sector.name = "fan-action-sector";
  group.add(sector);
  const sectorMaterial = new THREE.MeshBasicMaterial({
    color: "#cfed7c",
    transparent: true,
    opacity: 0.3,
    depthWrite: false,
  });
  for (let i = 0; i < 5; i++)
    sector.add(
      new THREE.Mesh(new THREE.BoxGeometry(4.7, 0.024, 0.35), sectorMaterial),
    );
  sector.visible = false;
  const director = createCrowdDirector(actors),
    byId = new Map(actors.map((actor, i) => [actor.id, i]));
  const visible = new Uint8Array(actors.length).fill(1);
  const wasActing = new Set();
  let selected =
    selectedId === undefined
      ? actors.find((actor) => actor.kind === "profile")?.id
      : selectedId;
  let showFillers = true,
    onlyActive = false,
    lastUpdate = -Infinity,
    tick = 0;
  const reducedMotion = window.matchMedia(
    "(prefers-reduced-motion: reduce)",
  ).matches;
  const up = new THREE.Vector3(0, 1, 0),
    a = new THREE.Vector3(),
    b = new THREE.Vector3(),
    direction = new THREE.Vector3();
  const cosine = actors.map((actor) => Math.cos(actor.seat.yaw)),
    sine = actors.map((actor) => Math.sin(actor.seat.yaw));
  function point(i, x, y, z, target) {
    const seat = actors[i].seat;
    return target.set(
      seat.x + x * cosine[i] + z * sine[i],
      y,
      seat.z - x * sine[i] + z * cosine[i],
    );
  }
  function box(target, index, i, x, y, z, sx, sy, sz, turn = 0) {
    point(i, x, y, z, dummy.position);
    dummy.rotation.set(0, actors[i].seat.yaw + turn, 0);
    dummy.scale.set(sx, sy, sz);
    dummy.updateMatrix();
    target.setMatrixAt(index, dummy.matrix);
  }
  function link(index, i, start, end, width, depth) {
    point(i, ...start, a);
    point(i, ...end, b);
    dummy.position.copy(a).add(b).multiplyScalar(0.5);
    direction.copy(b).sub(a);
    const length = direction.length();
    dummy.quaternion.setFromUnitVectors(
      up,
      direction.multiplyScalar(1 / Math.max(length, 0.00001)),
    );
    dummy.scale.set(width, length, depth);
    dummy.updateMatrix();
    limbs.setMatrixAt(index, dummy.matrix);
  }
  function pose(i, now, force = false) {
    const actor = actors[i],
      seat = actor.seat,
      seed = actor.seed;
    if (!visible[i]) {
      dummy.scale.set(0, 0, 0);
      dummy.updateMatrix();
      for (const object of [torso, heads, hats, dots])
        object.setMatrixAt(i, dummy.matrix);
      for (let part = 0; part < LIMBS; part++)
        limbs.setMatrixAt(i * LIMBS + part, dummy.matrix);
      for (let part = 0; part < 3; part++)
        scarves.setMatrixAt(i * 3 + part, dummy.matrix);
      for (let part = 0; part < 8; part++)
        accessories.setMatrixAt(i * 8 + part, dummy.matrix);
      return;
    }
    const reaction = director.sample(actor, now);
    const equipment = actor.equipment || {};
    const t = now / 1000 + seed * 0.117,
      cycle = t % 24,
      animated = seed % 8 === 0 || actor.id === selected;
    const idleClap =
      !reducedMotion && animated && cycle > 9 && cycle < 11
        ? Math.sin(((cycle - 9) / 2) * Math.PI)
        : 0;
    const idleWave =
      !reducedMotion && animated && cycle > 17 && cycle < 19
        ? Math.sin(((cycle - 17) / 2) * Math.PI)
        : 0;
    const idleCheer =
      !reducedMotion && animated && cycle > 3 && cycle < 4.8
        ? Math.sin(((cycle - 3) / 1.8) * Math.PI) * 0.65
        : 0;
    const standing = Math.max(
      seed % 13 === 0 ? 1 : 0,
      reaction?.standing || 0,
      idleCheer * 0.45,
    );
    const clap = reaction?.clap || idleClap,
      raise = reaction?.raise || idleCheer,
      scarf = reaction?.scarf || 0;
    const bob = !reducedMotion && animated ? Math.sin(t * 1.15) * 0.009 : 0;
    const hip =
      seat.y +
      (seat.hasSeat ? 0.17 : 0.04) +
      (0.43 - (seat.hasSeat ? 0.17 : 0.04)) * standing +
      bob;
    const turn = !reducedMotion && animated ? Math.sin(t * 0.6) * 0.1 : 0;
    box(torso, i, i, 0, hip + 0.2, 0, 0.29, 0.34, 0.19, turn);
    box(heads, i, i, 0, hip + 0.49, -0.004, 0.125, 0.14, 0.12, turn);
    box(
      hats,
      i,
      i,
      0,
      hip + 0.585,
      equipment.hat ? -0.018 : seed % 5 === 0 ? -0.018 : 0.014,
      0.21,
      equipment.hair ? 0.15 : 0.06,
      seed % 5 === 0 ? 0.25 : 0.17,
      turn,
    );
    for (const sign of [-1, 1]) {
      const left = sign < 0,
        base = i * LIMBS + (left ? 0 : 2);
      let elbow = [sign * 0.2, hip + 0.15, -0.04],
        hand = [sign * 0.19, hip + 0.035, -0.13];
      if (clap > 0) {
        elbow = [sign * 0.22, hip + 0.2, -0.1];
        hand = [
          sign * (0.035 + 0.075 * (0.5 + 0.5 * Math.sin(t * 13))),
          hip + 0.25,
          -0.27,
        ];
      }
      const lift = raise || (!left ? idleWave : 0);
      if (lift > 0) {
        elbow = [sign * 0.25, hip + 0.16 + 0.32 * lift, -0.02];
        hand = [sign * (0.22 + 0.11 * lift), hip + 0.08 + 0.6 * lift, -0.02];
      }
      link(base, i, [sign * 0.15, hip + 0.33, 0], elbow, 0.082, 0.09);
      link(base + 1, i, elbow, hand, 0.065, 0.065);
      const knee = [
        sign * 0.085,
        hip - 0.035 - 0.18 * standing,
        -0.23 * (1 - standing) - 0.02,
      ];
      const ankle = [
        sign * 0.085,
        seat.y + 0.04,
        -0.25 * (1 - standing) - 0.025,
      ];
      link(
        i * LIMBS + (left ? 4 : 6),
        i,
        [sign * 0.085, hip, 0],
        knee,
        0.1,
        0.11,
      );
      link(i * LIMBS + (left ? 5 : 7), i, knee, ankle, 0.075, 0.08);
      box(
        limbs,
        i * LIMBS + (left ? 8 : 9),
        i,
        ankle[0],
        seat.y + 0.045,
        ankle[2] - 0.035,
        0.1,
        0.065,
        0.17,
      );
    }
    box(limbs, i * LIMBS + 10, i, 0, hip + 0.485, -0.125, 0.043, 0.048, 0.052);
    for (let eye = 0; eye < 2; eye++)
      box(
        limbs,
        i * LIMBS + 11 + eye,
        i,
        eye === 0 ? -0.045 : 0.045,
        hip + 0.52,
        -0.112,
        equipment.glasses ? 0.065 : 0.018,
        equipment.glasses ? 0.043 : 0.018,
        0.016,
      );
    box(limbs, i * LIMBS + 13, i, 0, hip + 0.23, -0.101, 0.24, 0.036, 0.018);
    box(
      scarves,
      i * 3,
      i,
      0,
      hip + 0.77,
      -0.02,
      0.79 * scarf,
      0.105 * scarf,
      0.045 * scarf,
    );
    for (let end = 0; end < 2; end++)
      box(
        scarves,
        i * 3 + 1 + end,
        i,
        end === 0 ? -0.36 : 0.36,
        hip + 0.65,
        -0.02,
        0.065 * scarf,
        0.2 * scarf,
        0.05 * scarf,
      );
    const extra = (part, x, y, z, sx, sy, sz, enabled) =>
      box(
        accessories,
        i * 8 + part,
        i,
        x,
        y,
        z,
        enabled ? sx : 0,
        enabled ? sy : 0,
        enabled ? sz : 0,
      );
    extra(0, 0, hip + 0.37, -0.04, 0.34, 0.065, 0.22, equipment.scarf);
    extra(1, -0.12, hip + 0.22, -0.13, 0.09, 0.27, 0.04, equipment.scarf);
    extra(2, 0, hip + 0.52, -0.13, 0.12, 0.024, 0.02, equipment.glasses);
    extra(3, 0, hip + 0.57, -0.13, 0.23, 0.035, 0.13, equipment.hat);
    extra(4, 0.3, hip + 0.42, -0.12, 0.023, 0.65, 0.023, equipment.flag);
    extra(5, 0.41, hip + 0.65, -0.12, 0.25, 0.19, 0.03, equipment.flag);
    extra(
      6,
      0,
      hip + 0.42,
      0.08,
      0.23,
      0.21,
      0.15,
      equipment.shirt === "hoodie",
    );
    extra(
      7,
      0,
      hip + 0.4,
      -0.1,
      0.2,
      0.03,
      0.03,
      equipment.shirt === "club-shirt",
    );
    const active =
      actor.kind === "profile" &&
      (actor.profile.active || neighbors.has(actor.id));
    box(
      dots,
      i,
      i,
      0,
      hip + 0.72,
      0,
      active ? 1 : 0,
      active ? 1 : 0,
      active ? 1 : 0,
    );
  }
  function dirty() {
    for (const object of [
      torso,
      heads,
      hats,
      limbs,
      scarves,
      dots,
      accessories,
    ])
      object.instanceMatrix.needsUpdate = true;
  }
  function filters(options = {}) {
    showFillers = options.fillers ?? showFillers;
    onlyActive = options.onlyActive ?? onlyActive;
    actors.forEach((actor, i) => {
      visible[i] =
        actor.kind === "filler"
          ? Number(showFillers && !onlyActive)
          : Number(
              !onlyActive || actor.profile.active || actor.id === selected,
            );
      pose(i, performance.now(), true);
    });
    dirty();
  }
  function update(now, camera) {
    const selectedIndex = byId.get(selected),
      actor = actors[selectedIndex];
    ring.visible = pin.visible = Boolean(actor);
    if (actor) {
      ring.position.set(actor.seat.x, actor.seat.y + 0.2, actor.seat.z);
      ring.scale.setScalar(
        1 + (reducedMotion ? 0 : Math.sin(now / 350) * 0.08),
      );
      pin.position.set(
        actor.seat.x,
        actor.seat.y + 1.25 + (reducedMotion ? 0 : Math.sin(now / 450) * 0.035),
        actor.seat.z,
      );
    }
    const latest = director.latest(now);
    sector.visible = Boolean(latest);
    if (latest) {
      const source = latest.source.seat;
      sectorMaterial.opacity =
        0.12 + Math.sin((now - latest.start) / 230) ** 2 * 0.18;
      sector.children.forEach((stripe, index) => {
        const row = Math.max(1, Math.min(19, source.row + index - 2));
        stripe.position.set(
          source.x,
          row * 0.31 + 0.19,
          source.side * (9.1 + (row - 1) * 0.38),
        );
      });
    }
    activeMaterial.opacity = reducedMotion
      ? 0.65
      : 0.55 + Math.sin(now / 1100) * 0.15;
    if (now - lastUpdate < 1000 / 30) return;
    lastUpdate = now;
    tick++;
    actors.forEach((actor, i) => {
      if (!visible[i]) return;
      const acting = director.activeFor(actor, now);
      if (
        actor.id !== selected &&
        actor.seed % 8 !== 0 &&
        !acting &&
        !wasActing.has(i)
      )
        return;
      const distance = camera.position.distanceToSquared(
        a.set(actor.seat.x, actor.seat.y, actor.seat.z),
      );
      if (actor.kind === "filler" && distance > 1600 && tick % 4 !== 0) return;
      pose(i, now);
      if (acting) wasActing.add(i);
      else wasActing.delete(i);
    });
    dirty();
  }
  const raycaster = new THREE.Raycaster(),
    pointer = new THREE.Vector2();
  function pick(clientX, clientY, camera, element) {
    const rect = element.getBoundingClientRect();
    pointer.set(
      ((clientX - rect.left) / rect.width) * 2 - 1,
      (-(clientY - rect.top) / rect.height) * 2 + 1,
    );
    raycaster.setFromCamera(pointer, camera);
    const hit = raycaster.intersectObjects([heads, torso], false)[0];
    if (
      hit &&
      actors[hit.instanceId]?.kind === "profile" &&
      visible[hit.instanceId]
    ) {
      onSelect?.(actors[hit.instanceId].id);
      return true;
    }
    return false;
  }
  actors.forEach((actor, i) => pose(i, 0, true));
  function applyEquipment(id, equipment) {
    const i = byId.get(id);
    if (i === undefined) return;
    const actor = actors[i];
    actor.equipment = { ...equipment };
    const e = actor.equipment;
    const shirt =
      {
        "club-shirt": "#304e92",
        "away-shirt": "#edcf38",
        hoodie: "#465c69",
        "shirt-color": "#a2c861",
      }[e.shirt] || shirts[actor.seed % shirts.length];
    torso.setColorAt(i, color.set(shirt));
    hats.setColorAt(
      i,
      color.set(e.hat ? "#e8c83e" : hair[actor.seed % hair.length]),
    );
    for (let j = 0; j < LIMBS; j++) {
      let tint =
        j < 4
          ? j % 2 === 0
            ? shirt
            : skin[actor.seed % skin.length]
          : j < 8
            ? e.pants
              ? "#243b6c"
              : trousers[actor.seed % trousers.length]
            : j < 10
              ? e.shoes
                ? "#c2e96c"
                : "#eee8d6"
              : j > 10 && j < 13
                ? "#253534"
                : shirt;
      limbs.setColorAt(i * LIMBS + j, color.set(tint));
    }
    for (let j = 0; j < 8; j++)
      accessories.setColorAt(
        i * 8 + j,
        color.set(
          [0, 1, 5, 7].includes(j) ? "#f3ce27" : j === 2 ? "#263b2d" : shirt,
        ),
      );
    for (const m of [torso, hats, limbs, accessories])
      m.instanceColor.needsUpdate = true;
    pose(i, performance.now(), true);
    dirty();
  }
  actors.forEach((actor) => applyEquipment(actor.id, actor.equipment || {}));
  dirty();
  return {
    setEquipment: applyEquipment,
    setNeighbors(ids) {
      neighbors = new Set(ids);
      actors.forEach((_, i) => pose(i, performance.now(), true));
      dirty();
    },
    update,
    pick,
    group,
    select(id) {
      if (byId.has(id)) {
        selected = id;
        filters();
      }
    },
    setFilters: filters,
    trigger(kind, id, now = performance.now()) {
      return director.trigger(kind, id, now);
    },
    reset() {
      director.reset();
      filters({ fillers: true, onlyActive: false });
    },
    actor(id) {
      return actors[byId.get(id)];
    },
    selectedPosition() {
      const actor = actors[byId.get(selected)];
      return actor
        ? new THREE.Vector3(actor.seat.x, actor.seat.y + 1.0, actor.seat.z)
        : null;
    },
  };
}
