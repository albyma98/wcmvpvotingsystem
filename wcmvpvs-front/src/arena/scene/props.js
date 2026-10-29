import * as THREE from "three";
import { createBuilder, palette as p } from "./primitives";

export function createProps(parent) {
  const railings = new THREE.Group();
  railings.name = "court-railings";
  parent.add(railings);
  const b = createBuilder(railings);
  for (const z of [-8.3, 8.3]) b.rail([-12, z], [12, z]);
  for (const x of [-15, 15]) {
    b.rail([x, -8.3], [x, -2]);
    b.rail([x, 2], [x, 8.3]);
  }
  b.flush();
  const tech = createBuilder(parent);
  for (const x of [-7, 7]) {
    tech.box(p.white, [x, 0.8, 7.3], [1.9, 0.1, 0.85]);
    for (const dx of [-0.78, 0.78])
      for (const dz of [-0.3, 0.3])
        tech.box(p.white, [x + dx, 0.4, 7.3 + dz], [0.065, 0.8, 0.065]);
  }
  for (const z of [-3, 0, 3]) {
    tech.box(p.white, [-16.7, 0.8, z], [0.1, 1.6, 1.6]);
    tech.box("#9a9b8b", [-16.7, 0.12, z], [0.6, 0.24, 1.7]);
  }
  tech.flush();
  return railings;
}

export function createChairs(parent) {
  const group = new THREE.Group();
  group.name = "optional-photo-chairs";
  parent.add(group);
  const b = createBuilder(group);
  let count = 0;
  for (let row = 0; row < 5; row++)
    for (let col = 0; col < 8; col++) {
      if ((row * 7 + col) % 11 === 0) continue;
      const x = -12 + col * 2.85 + (row % 2) * 0.4,
        z = -7.8 + row * 3.15;
      const angle = Math.sin(row * 8 + col * 2) * 0.22;
      const point = (dx, y, dz) => [
        x + dx * Math.cos(angle) + dz * Math.sin(angle),
        y,
        z - dx * Math.sin(angle) + dz * Math.cos(angle),
      ];
      b.box(p.chair, point(0, 0.53, 0), [0.44, 0.07, 0.43], [0, angle, 0]);
      b.box(
        p.chair,
        point(0, 0.85, -0.19),
        [0.43, 0.45, 0.065],
        [-0.09, angle, 0],
      );
      for (const dx of [-0.18, 0.18])
        for (const dz of [-0.16, 0.16])
          b.beam(
            p.chair,
            point(dx * 1.35, 0.035, dz * 1.4),
            point(dx, 0.53, dz),
            0.027,
          );
      count++;
    }
  b.flush();
  group.userData.count = count;
  return group;
}
