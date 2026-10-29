import * as THREE from "three";
import { canvasTexture, palette, createBuilder } from "../scene/primitives.js";
import { TREASURE_SPOTS } from "./model.js";

export function createActivityWorld(
  scene,
  { onActivity, onTreasure, reducedMotion = false },
) {
  const root = new THREE.Group();
  root.name = "arena-activities";
  scene.add(root);
  const gold = new THREE.MeshStandardMaterial({
    color: "#edc74d",
    metalness: 0.48,
    roughness: 0.34,
  });
  const white = new THREE.MeshStandardMaterial({ color: palette.white });
  const dark = new THREE.MeshStandardMaterial({ color: "#254639" });
  const lime = new THREE.MeshBasicMaterial({
    color: "#d6ff8c",
    transparent: true,
    opacity: 0.24,
    depthWrite: false,
  });
  const targets = [];
  let state,
    treasure,
    collecting = 0,
    last = 0;
  const add = (parent, geo, mat, pos, scale) => {
    const m = new THREE.Mesh(geo, mat);
    if (pos) m.position.set(...pos);
    if (scale) m.scale.set(...scale);
    parent.add(m);
    return m;
  };
  function trophy(parent, mat = gold) {
    const g = new THREE.Group();
    parent.add(g);
    add(g, new THREE.CylinderGeometry(0.36, 0.14, 0.55, 8), mat, [0, 0.45, 0]);
    add(g, new THREE.CylinderGeometry(0.07, 0.12, 0.3, 8), mat, [0, 0.05, 0]);
    add(g, new THREE.BoxGeometry(0.5, 0.12, 0.4), dark, [0, -0.16, 0]);
    for (const x of [-0.35, 0.35]) {
      const h = add(g, new THREE.TorusGeometry(0.2, 0.035, 5, 12), mat, [
        x,
        0.48,
        0,
      ]);
      h.scale.x = 0.75;
    }
    return g;
  }
  const museum = new THREE.Group();
  museum.name = "community-museum";
  root.add(museum);
  const cups = [];
  for (let i = 0; i < 6; i++) {
    const cabinet = new THREE.Group();
    cabinet.position.set(-14.5, 0, -5 + i * 2);
    museum.add(cabinet);
    add(cabinet, new THREE.BoxGeometry(1.25, 0.8, 1.2), dark, [0, 0.4, 0]);
    add(cabinet, new THREE.BoxGeometry(1.2, 0.07, 1.2), gold, [0, 0.85, 0]);
    const cup = trophy(cabinet);
    cup.position.y = 1.2;
    const frame = createBuilder(cabinet);
    for (const x of [-0.58, 0.58])
      for (const z of [-0.56, 0.56])
        frame.box("#b3beaa", [x, 1.55, z], [0.035, 1.4, 0.035]);
    frame.box(palette.white, [0, 2.25, 0], [1.25, 0.08, 1.25]);
    frame.flush();
    const glass = add(
      cabinet,
      new THREE.BoxGeometry(1.18, 1.3, 1.16),
      new THREE.MeshStandardMaterial({
        color: "#d9efe3",
        transparent: true,
        opacity: 0.07,
        depthWrite: false,
      }),
      [0, 1.55, 0],
    );
    const plaque = add(
      cabinet,
      new THREE.PlaneGeometry(1.13, 0.32),
      dark,
      [0.635, 0.5, 0],
    );
    plaque.rotation.y = Math.PI / 2;
    targets.push({ object: glass, kind: "museum", index: i });
    cups.push({ cup, plaque });
  }
  // The scoreboard shares the arena renderer; it is a physical panel at the end
  // of the court, with real club fixture names supplied by the activities state.
  const board = new THREE.Group();
  board.position.set(14.6, 0, 0);
  root.add(board);
  const frame = createBuilder(board);
  for (const z of [-1.9, 1.9])
    frame.box(palette.blue, [0, 1.5, z], [0.12, 3, 0.12]);
  frame.box(palette.blue, [0, 3, 0], [0.17, 2.6, 4.8]);
  frame.flush();
  const boardFace = add(
    board,
    new THREE.PlaneGeometry(4.6, 2.4),
    dark,
    [-0.1, 3, 0],
  );
  boardFace.rotation.y = -Math.PI / 2;
  targets.push({ object: boardFace, kind: "predictions" });
  function label(mesh, text, sub) {
    const map = canvasTexture(768, 384, (ctx, w, h) => {
      ctx.fillStyle = "#234435";
      ctx.fillRect(0, 0, w, h);
      ctx.textAlign = "center";
      ctx.fillStyle = "#d9edae";
      ctx.font = "600 26px sans-serif";
      ctx.fillText("ARENA TIFOSI", w / 2, 58);
      ctx.fillStyle = "#f9f7e6";
      ctx.font = "600 36px sans-serif";
      String(text)
        .split("\n")
        .forEach((line, i) => ctx.fillText(line, w / 2, 150 + i * 54, w - 40));
      ctx.fillStyle = "#c4d4b3";
      ctx.font = "23px sans-serif";
      ctx.fillText(sub, w / 2, h - 45, w - 40);
    });
    if (mesh.material.map) mesh.material.map.dispose();
    if (mesh.material === dark) mesh.material = new THREE.MeshBasicMaterial();
    mesh.material.map = map;
    mesh.material.needsUpdate = true;
  }
  function buildTreasure(kind) {
    const g = new THREE.Group();
    g.name = "daily-treasure";
    root.add(g);
    if (kind === 0) {
      add(g, new THREE.SphereGeometry(0.3, 12, 8), gold);
      for (const rotation of [
        [0, 0, 0],
        [Math.PI / 2, 0, 0],
        [0, Math.PI / 2, 0],
      ]) {
        const ring = add(g, new THREE.TorusGeometry(0.3, 0.014, 4, 18), white);
        ring.rotation.set(...rotation);
      }
    } else if (kind === 1) {
      const coin = add(
        g,
        new THREE.CylinderGeometry(0.32, 0.32, 0.09, 20),
        gold,
      );
      coin.rotation.x = Math.PI / 2;
      add(g, new THREE.BoxGeometry(0.07, 0.38, 0.12), white, [0, 0, -0.06]);
    } else if (kind === 2) {
      trophy(g);
      g.scale.setScalar(0.65);
    } else {
      add(g, new THREE.BoxGeometry(0.85, 0.14, 0.07), gold);
      for (const x of [-0.35, 0.35])
        add(g, new THREE.BoxGeometry(0.13, 0.38, 0.07), white, [x, -0.2, 0]);
    }
    const halo = add(g, new THREE.SphereGeometry(0.5, 12, 8), lime);
    halo.renderOrder = 1;
    return g;
  }
  function update(next) {
    state = next;
    collecting = 0;
    if (treasure) {
      root.remove(treasure);
      disposeGroup(treasure, false);
    }
    treasure = buildTreasure(state.treasure.kind);
    treasure.position.set(...TREASURE_SPOTS[state.treasure.location].position);
    treasure.visible = !state.treasure.claimed;
    const match =
      [...state.matches]
        .filter((m) => m.open)
        .sort((a, b) => new Date(a.starts) - new Date(b.starts))[0] ||
      state.matches[0];
    label(
      boardFace,
      match ? match.home + "\n" + match.away : "Calendario in aggiornamento",
      "TOCCA PER I PRONOSTICI",
    );
    cups.forEach(({ cup, plaque }, i) => {
      const t = state.trophies[i];
      cup.traverse((o) => {
        if (o.isMesh && o.material !== dark)
          o.material = t?.unlocked ? gold : white;
      });
      cup.scale.setScalar(t?.unlocked ? 1 : 0.75);
      label(
        plaque,
        t?.title || "Traguardo",
        t?.unlocked
          ? "SBLOCCATO · " + t.unlocked.slice(0, 10)
          : "DA CONQUISTARE",
      );
    });
  }
  const ray = new THREE.Raycaster(),
    pointer = new THREE.Vector2(),
    sphere = new THREE.Sphere(),
    center = new THREE.Vector3(),
    hitpoint = new THREE.Vector3();
  function pick(x, y, camera, element, architecture) {
    if (!state) return false;
    const rect = element.getBoundingClientRect();
    pointer.set(
      ((x - rect.left) / rect.width) * 2 - 1,
      (-(y - rect.top) / rect.height) * 2 + 1,
    );
    ray.setFromCamera(pointer, camera);
    if (treasure.visible && !collecting) {
      treasure.getWorldPosition(center);
      sphere.set(
        center,
        Math.max(0.55, center.distanceTo(camera.position) * 0.027),
      );
      if (ray.ray.intersectSphere(sphere, hitpoint)) {
        const obstruction = ray.intersectObject(architecture, true)[0];
        if (
          !obstruction ||
          obstruction.distance >= camera.position.distanceTo(center) - 0.6
        ) {
          onTreasure?.();
          return true;
        }
      }
    }
    const hits = ray.intersectObjects(
      targets.map((t) => t.object),
      false,
    );
    if (hits[0]) {
      const target = targets.find((t) => t.object === hits[0].object);
      onActivity?.(target.kind, target.index);
      return true;
    }
    return false;
  }
  function animate(now) {
    if (!state) return;
    if (!reducedMotion && treasure.visible) {
      treasure.rotation.y = now / 1100;
      treasure.position.y =
        TREASURE_SPOTS[state.treasure.location].position[1] +
        Math.sin(now / 700) * 0.07;
    }
    if (collecting) {
      const t = Math.min(1, (now - collecting) / 500);
      treasure.scale.setScalar(Math.max(0.01, 1 - t));
      if (t === 1) treasure.visible = false;
    }
    last = now;
  }
  function disposeGroup(group, all) {
    group.traverse((o) => {
      o.geometry?.dispose();
      if (o.material?.map) o.material.map.dispose();
      if (all && o.material) o.material.dispose();
    });
  }
  return {
    update,
    pick,
    animate,
    collect() {
      collecting = performance.now();
    },
    focus(kind, index) {
      if (kind === "museum")
        return {
          x: -14,
          y: 1,
          z: Number.isInteger(index) ? -5 + index * 2 : 0,
          side: -1,
        };
      if (kind === "treasure") {
        const p = TREASURE_SPOTS[state.treasure.location].position;
        return { x: p[0], y: p[1], z: p[2], side: p[2] < 0 ? -1 : 1 };
      }
      return { x: 12, y: 1, z: 0, side: 1 };
    },
    dispose() {
      root.removeFromParent();
      disposeGroup(root, true);
      gold.dispose();
      white.dispose();
      dark.dispose();
      lime.dispose();
    },
  };
}
