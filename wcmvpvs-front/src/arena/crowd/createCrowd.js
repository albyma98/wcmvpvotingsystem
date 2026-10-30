import * as THREE from "three";
import { createCrowdDirector } from "../actions/crowdDirector.js";
import { createAvatarBatch } from "./avatarModel.js";
import { avatarLook } from "./avatarLook.js";
import { avatarPose } from "./avatarPose.js";
export const MAX_DETAILED_AVATARS = 8;
export function createCrowd(
  parent,
  actors,
  { selectedId, onSelect, lod = "auto", markers = true } = {},
) {
  const group = new THREE.Group();
  group.name = "crowd";
  parent.add(group);
  const far = createAvatarBatch(group, actors.length, { prefix: "fan" });
  const near = createAvatarBatch(group, MAX_DETAILED_AVATARS, {
    detail: true,
    prefix: "fan-detail",
  });
  const looks = actors.map(avatarLook),
    poses = actors.map((a) => avatarPose(a, 0, null, true, false));
  const indexById = new Map(actors.map((a, i) => [a.id, i])),
    director = createCrowdDirector(actors);
  const visible = new Uint8Array(actors.length).fill(1),
    wasActing = new Set();
  let selected =
    selectedId === undefined
      ? actors.find((a) => a.kind === "profile")?.id
      : selectedId;
  let detailIndices = [],
    detailSet = new Set(),
    neighbors = new Set(),
    showFillers = true,
    onlyActive = false,
    lastUpdate = -Infinity,
    lastLod = -Infinity,
    tick = 0,
    forcedLod = lod;
  const reducedMotion = window.matchMedia(
    "(prefers-reduced-motion: reduce)",
  ).matches;
  const markerMaterial = new THREE.MeshBasicMaterial({
    color: "#b4d477",
    transparent: true,
    opacity: 0.7,
    depthTest: false,
  });
  const ring = new THREE.Mesh(
    new THREE.TorusGeometry(0.29, 0.016, 6, 32),
    markerMaterial,
  );
  ring.rotation.x = -Math.PI / 2;
  ring.renderOrder = 6;
  group.add(ring);
  const pin = new THREE.Mesh(
    new THREE.ConeGeometry(0.055, 0.12, 10),
    markerMaterial,
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
    opacity: 0.2,
    depthWrite: false,
  });
  for (let i = 0; i < 5; i++)
    sector.add(
      new THREE.Mesh(new THREE.BoxGeometry(4.7, 0.018, 0.35), sectorMaterial),
    );
  sector.visible = false;
  const dotMaterial = new THREE.MeshBasicMaterial({ color: "#bfdc91" });
  const dots = new THREE.InstancedMesh(
    new THREE.SphereGeometry(0.035, 5, 3),
    dotMaterial,
    actors.length,
  );
  dots.name = "fan-presence-dots";
  dots.frustumCulled = false;
  group.add(dots);
  const dummy = new THREE.Object3D(),
    position = new THREE.Vector3();
  let lastCamera = null;
  far.setActors(actors, looks);
  near.setActors([], []);
  const stats = { far: actors.length, detailed: 0, triangles: 0, poseMs: 0 };
  function drawFar(i) {
    if (!visible[i] || detailSet.has(i)) far.clear(i);
    else far.render(i, poses[i]);
    const actor = actors[i],
      active =
        visible[i] &&
        actor.kind === "profile" &&
        (actor.profile.active || neighbors.has(actor.id));
    dummy.position.set(
      actor.seat.x,
      actor.seat.y + (poses[i].hip + 0.91) * looks[i].height,
      actor.seat.z,
    );
    dummy.rotation.set(0, 0, 0);
    dummy.scale.setScalar(active ? 1 : 0);
    dummy.updateMatrix();
    dots.setMatrixAt(i, dummy.matrix);
  }
  function redrawAll() {
    for (let i = 0; i < actors.length; i++) drawFar(i);
    detailIndices.forEach((i, slot) => near.render(slot, poses[i]));
    far.flush();
    near.flush();
    dots.instanceMatrix.needsUpdate = true;
    stats.triangles = far.triangleBudget() + near.triangleBudget();
  }
  function selectDetail(camera, now, force = false) {
    if (!force && now - lastLod < 140) return;
    lastLod = now;
    const selectedIndex = indexById.get(selected);
    const candidates = [];
    if (forcedLod !== "far")
      for (let i = 0; i < actors.length; i++)
        if (visible[i]) {
          const actor = actors[i];
          const distance = camera.position.distanceToSquared(
            position.set(actor.seat.x, actor.seat.y + 0.55, actor.seat.z),
          );
          if (
            i === selectedIndex ||
            forcedLod === "near" ||
            distance < (detailSet.has(i) ? 64 : 36)
          )
            candidates.push({
              i,
              distance: i === selectedIndex ? -1 : distance,
            });
        }
    const next = candidates
      .sort((a, b) => a.distance - b.distance)
      .slice(0, window.innerWidth < 700 ? 4 : MAX_DETAILED_AVATARS)
      .map((x) => x.i);
    if (next.join() !== detailIndices.join() || force) {
      detailIndices = next;
      detailSet = new Set(next);
      near.setActors(
        next.map((i) => actors[i]),
        next.map((i) => looks[i]),
      );
      redrawAll();
    }
    stats.detailed = detailIndices.length;
    stats.far = actors.length - detailIndices.length;
  }
  function setFilters(options = {}) {
    showFillers = options.fillers ?? showFillers;
    onlyActive = options.onlyActive ?? onlyActive;
    actors.forEach((a, i) => {
      visible[i] =
        a.kind === "filler"
          ? Number(showFillers && !onlyActive)
          : Number(!onlyActive || a.profile.active || a.id === selected);
    });
    if (lastCamera) selectDetail(lastCamera, performance.now(), true);
    redrawAll();
  }
  function update(now, camera) {
    lastCamera = camera;
    const selectedIndex = indexById.get(selected),
      actor = actors[selectedIndex];
    ring.visible = pin.visible =
      markers && Boolean(actor && visible[selectedIndex]);
    if (actor) {
      ring.position.set(actor.seat.x, actor.seat.y + 0.02, actor.seat.z);
      pin.position.set(
        actor.seat.x +
          Math.sin(actor.seat.yaw) *
            poses[selectedIndex].forward *
            looks[selectedIndex].width,
        actor.seat.y +
          (poses[selectedIndex].hip + 0.99) * looks[selectedIndex].height,
        actor.seat.z +
          Math.cos(actor.seat.yaw) *
            poses[selectedIndex].forward *
            looks[selectedIndex].width,
      );
    }
    selectDetail(camera, now);
    const latest = director.latest(now);
    sector.visible = Boolean(latest);
    if (latest) {
      const source = latest.source.seat;
      sectorMaterial.opacity =
        0.1 + Math.sin((now - latest.start) / 300) ** 2 * 0.1;
      sector.children.forEach((stripe, index) => {
        const row = Math.max(1, Math.min(19, source.row + index - 2));
        stripe.position.set(
          source.x,
          row * 0.31 + 0.015,
          source.side * (9.1 + (row - 1) * 0.38),
        );
      });
    }
    if (now - lastUpdate < 1000 / 30) return;
    lastUpdate = now;
    tick++;
    const start = performance.now();
    actors.forEach((a, i) => {
      if (!visible[i]) return;
      const detailed = detailSet.has(i);
      if (detailed) return;
      const reaction = director.sample(a, now),
        active = Boolean(reaction);
      if (
        !detailed &&
        a.id !== selected &&
        !active &&
        !wasActing.has(i) &&
        a.seed % 8 !== 0
      )
        return;
      if (!detailed && tick % 2 !== 0 && !wasActing.has(i)) return;
      poses[i] = avatarPose(
        a,
        now,
        reaction,
        reducedMotion,
        detailed || a.id === selected || a.seed % 8 === 0,
      );
      drawFar(i);
      if (active) wasActing.add(i);
      else wasActing.delete(i);
    });
    detailIndices.forEach((i, slot) => {
      poses[i] = avatarPose(
        actors[i],
        now,
        director.sample(actors[i], now),
        reducedMotion,
        true,
      );
      near.render(slot, poses[i]);
    });
    far.flush();
    near.flush();
    dots.instanceMatrix.needsUpdate = true;
    stats.poseMs = performance.now() - start;
  }
  function setEquipment(id, equipment) {
    const i = indexById.get(id);
    if (i === undefined) return;
    const previous = looks[i];
    actors[i].equipment = { ...equipment };
    const next = avatarLook(actors[i]);
    if (JSON.stringify(previous) === JSON.stringify(next)) return;
    looks[i] = next;
    near.setActors(
      detailIndices.map((i) => actors[i]),
      detailIndices.map((i) => looks[i]),
    );
    if (previous.hair !== next.hair) {
      far.setActors(actors, looks);
      redrawAll();
    } else {
      drawFar(i);
      detailIndices.forEach((index, slot) => near.render(slot, poses[index]));
      far.flush();
      near.flush();
    }
  }
  function setAppearance(id, appearance) {
    const i = indexById.get(id);
    if (i === undefined) return;
    actors[i].appearance = { ...appearance };
    setEquipment(id, actors[i].equipment || {});
  }
  const raycaster = new THREE.Raycaster(),
    pointer = new THREE.Vector2(),
    sphere = new THREE.Sphere(),
    center = new THREE.Vector3(),
    hit = new THREE.Vector3();
  function pick(x, y, camera, element) {
    const rect = element.getBoundingClientRect();
    pointer.set(
      ((x - rect.left) / rect.width) * 2 - 1,
      (-(y - rect.top) / rect.height) * 2 + 1,
    );
    raycaster.setFromCamera(pointer, camera);
    let best = null,
      distance = Infinity;
    actors.forEach((actor, i) => {
      if (!visible[i] || actor.kind !== "profile") return;
      center.set(
        actor.seat.x +
          Math.sin(actor.seat.yaw) * poses[i].forward * looks[i].width,
        actor.seat.y + (poses[i].hip + 0.47) * looks[i].height,
        actor.seat.z +
          Math.cos(actor.seat.yaw) * poses[i].forward * looks[i].width,
      );
      sphere.set(center, 0.28 * looks[i].width);
      if (raycaster.ray.intersectSphere(sphere, hit)) {
        const d = camera.position.distanceToSquared(hit);
        if (d < distance) {
          distance = d;
          best = actor;
        }
      }
    });
    if (best) {
      onSelect?.(best.id);
      return true;
    }
    return false;
  }
  redrawAll();
  return {
    group,
    update,
    pick,
    setEquipment,
    setAppearance,
    setJerseyTexture(texture) {
      far.setJerseyTexture(texture);
      near.setJerseyTexture(texture);
    },
    setNeighbors(ids) {
      neighbors = new Set(ids);
      redrawAll();
    },
    setFilters,
    select(id) {
      if (indexById.has(id)) {
        selected = id;
        setFilters();
        lastLod = -Infinity;
      }
    },
    trigger: (kind, id, now = performance.now()) =>
      director.trigger(kind, id, now),
    reset() {
      director.reset();
      setFilters({ fillers: true, onlyActive: false });
    },
    actor: (id) => actors[indexById.get(id)],
    selectedPosition() {
      const i = indexById.get(selected),
        a = actors[i];
      return a
        ? new THREE.Vector3(
            a.seat.x + Math.sin(a.seat.yaw) * poses[i].forward * looks[i].width,
            a.seat.y + (poses[i].hip + 0.47) * looks[i].height,
            a.seat.z + Math.cos(a.seat.yaw) * poses[i].forward * looks[i].width,
          )
        : null;
    },
    setLod(value) {
      forcedLod = value;
      lastLod = -Infinity;
    },
    stats: () => ({ ...stats }),
  };
}
