import * as THREE from "three";
import { createBuilder, palette as p, sign, canvasTexture } from "./primitives.js";
import { arenaLayout } from "./layout.js";

export function createEnvironment(parent) {
  const b = createBuilder(parent);
  const walls = [];
  const { length, width, endWallX, sideWallZ } = arenaLayout;
  const wallMaterial = new THREE.MeshStandardMaterial({
    color: p.wall,
    roughness: 0.95,
    side: THREE.FrontSide,
  });
  function wall(w, h, position, rotation, owner) {
    const mesh = new THREE.Mesh(new THREE.PlaneGeometry(w, h), wallMaterial);
    mesh.position.set(...position);
    mesh.rotation.y = rotation;
    owner.add(mesh);
  }

  // Four enclosing walls; exterior views retain the architectural cutaway.
  for (const side of [-1, 1]) {
    const group = new THREE.Group();
    group.name = `long-side-wall-${side}`;
    parent.add(group);
    walls.push({ group, axis: "z", side, limit: sideWallZ });
    const b = createBuilder(group);
    const z = side * sideWallZ;
    wall(length + 0.6, 12, [0, 6, z], side < 0 ? 0 : Math.PI, group);
    for (let x = -18; x <= 18; x += 4.5) {
      b.box(p.frame, [x, 5.9, z - side * 0.18], [0.2, 11.8, 0.25]);
    }
    for (const y of [0.2, 4.3, 8.6, 11.8]) {
      b.box(p.frame, [0, y, z - side * 0.22], [length, 0.2, 0.23]);
    }
    for (let y = 0.6; y < 11.8; y += 0.27) {
      b.box("#c9c797", [0, y, z - side * 0.08], [length - 0.2, 0.018, 0.025]);
    }
    b.flush();
  }

  // Exactly two blue doors on EACH short end, away from the stands.
  for (const side of [-1, 1]) {
    const group = new THREE.Group();
    group.name = `short-side-wall-${side}`;
    parent.add(group);
    walls.push({ group, axis: "x", side, limit: endWallX });
    const b = createBuilder(group);
    const doorBuilder = b;
    const x = side * endWallX;
    wall(width, 12, [x, 6, 0], (-side * Math.PI) / 2, group);
    for (let z = -16; z <= 16; z += 4) {
      b.box(p.frame, [x - side * 0.18, 5.9, z], [0.25, 11.8, 0.2]);
    }
    for (const y of [0.2, 4.3, 8.6, 11.8]) {
      b.box(p.frame, [x - side * 0.22, y, 0], [0.23, 0.2, width]);
    }
    for (const z of arenaLayout.doorOffsets) {
      const innerX = x - side * 0.22;
      doorBuilder.box(p.blue, [innerX, 1.15, z], [0.13, 2.3, 1.55]);
      doorBuilder.box(
        p.frame,
        [innerX - side * 0.08, 2.36, z],
        [0.13, 0.13, 1.75],
      );
      doorBuilder.box(
        "#c9cbbf",
        [innerX - side * 0.1, 1.12, z + 0.44],
        [0.04, 0.045, 0.25],
      );
    }

    // End-wall clerestory follows the timber gable.
    for (let z = -16; z <= 16; z += 1.5) {
      const top = 16.7 - (Math.abs(z) * 4.3) / 18 - 0.4;
      const height = top - 11.95;
      b.box(
        p.glass,
        [x - side * 0.18, 11.95 + height / 2, z],
        [0.07, height, 1.36],
      );
      b.box(
        p.darkWood,
        [x - side * 0.2, 11.95 + height / 2, z - 0.72],
        [0.13, height, 0.07],
      );
      b.box(p.darkWood, [x - side * 0.2, 12.4, z], [0.13, 0.07, 1.5]);
    }
    b.flush();
  }
  sign(
    walls.find((wall) => wall.axis === "x" && wall.side === -1).group,
    "Gioia del Colle",
    [-endWallX + 0.25, 3.2, 0],
    3.5,
    0.63,
    Math.PI / 2,
  );

  const roof = new THREE.Group();
  roof.name = "timber-roof";
  parent.add(roof);
  const roofBuilder = createBuilder(roof);
  const roofMaterial = new THREE.MeshStandardMaterial({
    color: p.darkWood,
    roughness: 0.94,
    side: THREE.FrontSide,
  });
  const eaveZ = sideWallZ + 0.5;
  for (const side of [-1, 1]) {
    const geometry = new THREE.BufferGeometry();
    geometry.setAttribute(
      "position",
      new THREE.BufferAttribute(
        new Float32Array([
          -18.5,
          16.7,
          0,
          18.5,
          16.7,
          0,
          18.5,
          12.4,
          side * eaveZ,
          -18.5,
          12.4,
          side * eaveZ,
        ]),
        3,
      ),
    );
    geometry.setIndex(side === 1 ? [0, 1, 2, 0, 2, 3] : [2, 1, 0, 3, 2, 0]);
    geometry.computeVertexNormals();
    roof.add(new THREE.Mesh(geometry, roofMaterial));
    for (let z = 0.5; z < eaveZ; z += 0.65) {
      roofBuilder.box(
        p.wood,
        [0, 16.7 - (z * 4.3) / eaveZ - 0.09, side * z],
        [37, 0.15, 0.1],
      );
    }
  }
  for (let x = -17; x <= 18; x += 5.7) {
    roofBuilder.beam(p.wood, [x, 11.9, -sideWallZ], [x, 11.9, sideWallZ], 0.22);
    for (const side of [-1, 1]) {
      roofBuilder.beam(p.wood, [x, 16.4, 0], [x, 12.1, side * eaveZ], 0.3);
      roofBuilder.beam(p.wood, [x, 11.9, side * 11.5], [x, 16.4, 0], 0.14);
    }
  }
  roofBuilder.flush();

  // Lights and flags are above the long side with BARE concrete terraces.
  const lightZ = -sideWallZ + 0.6;
  b.beam(p.blue, [-14, 11, lightZ], [14, 11, lightZ], 0.09);
  const lightGlow = canvasTexture(64, 64, (ctx, w, h) => {
    const gradient = ctx.createRadialGradient(
      w / 2,
      h / 2,
      0,
      w / 2,
      h / 2,
      w / 2,
    );
    gradient.addColorStop(0, "rgba(255,255,250,1)");
    gradient.addColorStop(0.18, "rgba(255,255,242,.95)");
    gradient.addColorStop(0.4, "rgba(255,247,211,.25)");
    gradient.addColorStop(1, "rgba(255,247,211,0)");
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, w, h);
  });
  const glowMaterial = new THREE.SpriteMaterial({
    map: lightGlow,
    transparent: true,
    depthWrite: false,
    toneMapped: false,
  });
  const lampGeometry = new THREE.PlaneGeometry(0.55, 0.34);
  const lampMaterial = new THREE.MeshBasicMaterial({
    color: "#fff9dc",
    side: THREE.DoubleSide,
  });
  for (let x = -13; x <= 13; x += 2.8) {
    b.box("#7d8583", [x, 11.1, lightZ], [0.65, 0.18, 0.45]);
    const lamp = new THREE.Mesh(lampGeometry, lampMaterial);
    lamp.position.set(x, 11.01, lightZ + 0.1);
    lamp.rotation.x = -Math.PI / 2;
    parent.add(lamp);
    const glow = new THREE.Sprite(glowMaterial);
    glow.position.set(x, 10.92, lightZ + 0.25);
    glow.scale.set(1.05, 1.05, 1);
    parent.add(glow);
  }
  const flags = new THREE.Group();
  flags.name = "flags-above-bare-stand";
  parent.add(flags);
  const flagBuilder = createBuilder(flags);
  [
    ["#1c955c", "#f2f0dc", "#c53e3b"],
    ["#28468b", "#f2f0dc", "#cd4240"],
    ["#f5dd50", "#e6e6de", "#c7453c"],
  ].forEach((colors, i) =>
    colors.forEach((color, j) => {
      flagBuilder.box(
        color,
        [-3.5 + i * 3 + j * 0.5, 9.5, -sideWallZ + 0.3],
        [0.49, 1.5, 0.08],
      );
    }),
  );
  flagBuilder.flush();
  b.flush();
  return { roof, walls };
}

export function createLighting(scene) {
  scene.add(new THREE.HemisphereLight("#fff7df", "#858477", 2.25));
  const key = new THREE.DirectionalLight("#fff8e5", 2.5);
  key.position.set(-6, 16, 9);
  scene.add(key);
  const fill = new THREE.DirectionalLight("#e5edff", 1.0);
  fill.position.set(15, 10, -9);
  scene.add(fill);
}
