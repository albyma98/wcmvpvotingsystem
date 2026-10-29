import * as THREE from "three";
import { createBuilder, palette as p, sign, canvasTexture } from "./primitives";

export function createEnvironment(parent) {
  const b = createBuilder(parent);
  const wallMaterial = new THREE.MeshStandardMaterial({
    color: p.wall,
    roughness: 0.95,
    side: THREE.FrontSide,
  });
  function wall(w, h, position, rotation) {
    const mesh = new THREE.Mesh(new THREE.PlaneGeometry(w, h), wallMaterial);
    mesh.position.set(...position);
    mesh.rotation.y = rotation;
    parent.add(mesh);
  }
  // Single-sided infill gives an architectural cutaway when viewed from outside.
  wall(36, 12, [0, 6, -13.5], 0);
  wall(27, 12, [18.3, 6, 0], -Math.PI / 2);
  wall(27, 12, [-18.3, 6, 0], Math.PI / 2);
  for (let x = -18; x <= 18; x += 4.5) {
    b.box(p.frame, [x, 5.9, -13.32], [0.2, 11.8, 0.25]);
    b.box(p.frame, [x, 12, -13.2], [0.35, 1, 0.5]);
  }
  for (const y of [0.2, 4.3, 8.6, 11.8])
    b.box(p.frame, [0, y, -13.28], [36, 0.2, 0.23]);
  // Fine horizontal acoustic-block coursing, kept to the wall in the photo.
  for (let y = 0.6; y < 11.8; y += 0.27)
    b.box("#c9c797", [0, y, -13.42], [35.8, 0.018, 0.025]);
  for (const x of [-15, -11, -7, -3, 4, 7]) {
    b.box(p.blue, [x, 1.15, -13.12], [1.55, 2.3, 0.13]);
    b.box(p.frame, [x, 2.36, -13.04], [1.75, 0.13, 0.12]);
    b.box("#c9cbbf", [x + 0.44, 1.12, -13.01], [0.25, 0.045, 0.04]);
  }
  sign(parent, "Gioia del Colle", [-8.5, 3.2, -13.1], 3.5, 0.63);
  // Tall clerestory windows on the end walls.
  for (const x of [-18.12, 18.12]) {
    for (let z = -12; z < 13; z += 1.5) {
      b.box(p.glass, [x, 12.9, z], [0.07, 2.25, 1.36]);
      b.box(p.darkWood, [x, 12.9, z - 0.72], [0.13, 2.5, 0.07]);
      b.box(p.darkWood, [x, 12.9, z], [0.13, 0.07, 1.5]);
    }
  }
  const roof = new THREE.Group();
  roof.name = "timber-roof";
  parent.add(roof);
  const roofBuilder = createBuilder(roof);
  const roofMaterial = new THREE.MeshStandardMaterial({
    color: p.darkWood,
    roughness: 0.94,
    side: THREE.FrontSide,
  });
  // Gabled timber canopy, dark inside; hidden from above for the high-angle view.
  for (const side of [-1, 1]) {
    const geometry = new THREE.BufferGeometry();
    const positions = new Float32Array([
      -18.5,
      16.7,
      0,
      18.5,
      16.7,
      0,
      18.5,
      12.4,
      side * 14.5,
      -18.5,
      12.4,
      side * 14.5,
    ]);
    geometry.setAttribute("position", new THREE.BufferAttribute(positions, 3));
    geometry.setIndex(side === 1 ? [0, 1, 2, 0, 2, 3] : [2, 1, 0, 3, 2, 0]);
    geometry.computeVertexNormals();
    roof.add(new THREE.Mesh(geometry, roofMaterial));
    for (let z = 0.5; z < 14.5; z += 0.65)
      roofBuilder.box(
        p.wood,
        [0, 16.7 - (z * 4.3) / 14.5 - 0.09, side * z],
        [37, 0.15, 0.1],
      );
  }
  for (let x = -17; x <= 18; x += 5.7) {
    for (const side of [-1, 1]) {
      roofBuilder.beam(p.wood, [x, 16.4, 0], [x, 12.1, side * 14.5], 0.3);
      roofBuilder.beam(p.wood, [x, 11.9, -13.4], [x, 11.9, 13.6], 0.22);
      roofBuilder.beam(p.wood, [x, 11.9, side * 9], [x, 16.4, 0], 0.14);
    }
  }
  roofBuilder.flush();
  // High lights on a blue-grey service rail, and simple hanging flags.
  b.beam(p.blue, [16.5, 11, -12], [16.5, 11, 12], 0.09);
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
  const lightGlowMaterial = new THREE.SpriteMaterial({
    map: lightGlow,
    transparent: true,
    depthWrite: false,
    toneMapped: false,
  });
  for (let z = -11; z < 12; z += 2.8) {
    b.box("#7d8583", [16.4, 11.1, z], [0.65, 0.18, 0.45]);
    const lamp = new THREE.Mesh(
      new THREE.PlaneGeometry(0.55, 0.34),
      new THREE.MeshBasicMaterial({ color: "#fff9dc", side: THREE.DoubleSide }),
    );
    lamp.position.set(16.25, 11.01, z);
    lamp.rotation.set(-Math.PI / 2, 0, 0);
    parent.add(lamp);
    const glow = new THREE.Sprite(lightGlowMaterial);
    glow.position.set(16.05, 10.92, z);
    glow.scale.set(1.05, 1.05, 1);
    parent.add(glow);
  }
  const flags = [
    ["#1c955c", "#f2f0dc", "#c53e3b"],
    ["#28468b", "#f2f0dc", "#cd4240"],
    ["#f5dd50", "#e6e6de", "#c7453c"],
  ];
  flags.forEach((colors, i) =>
    colors.forEach((color, j) =>
      b.box(color, [17.94, 10.2, -7 + i * 2.5 + j * 0.5], [0.08, 1.5, 0.49]),
    ),
  );
  b.flush();
  return roof;
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
