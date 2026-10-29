import * as THREE from "three";
import { createBuilder, palette as p } from "./primitives";

export function createProps(parent) {
  const railings = new THREE.Group();
  railings.name = "court-railings";
  parent.add(railings);
  const b = createBuilder(railings);
  b.rail([-16, -10.5], [9.6, -10.5]);
  b.rail([-16, -10.5], [-16, 8.5]);
  b.rail([9.7, -11.8], [9.7, 9.2]);
  b.flush();
  const tech = createBuilder(parent);
  for (const x of [-11, 5]) {
    tech.box(p.white, [x, 0.8, 8.55], [1.9, 0.1, 0.85]);
    for (const dx of [-0.78, 0.78])
      for (const dz of [-0.3, 0.3])
        tech.box(p.white, [x + dx, 0.4, 8.55 + dz], [0.065, 0.8, 0.065]);
  }
  for (const x of [-13, -8, -3, 2, 7]) {
    tech.box(p.white, [x, 0.8, -11.6], [1.6, 1.6, 0.1]);
    tech.box("#9a9b8b", [x, 0.12, -11.6], [1.7, 0.24, 0.6]);
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
