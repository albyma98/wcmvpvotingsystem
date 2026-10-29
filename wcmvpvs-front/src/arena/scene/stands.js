import { Group } from "three";
import { createBuilder, palette as p } from "./primitives.js";
import { arenaLayout } from "./layout.js";

export function createStands(parent) {
  const stand = arenaLayout.stand;
  const halfLength = stand.length / 2;
  const lastRow = (stand.rows - 1) * stand.rowDepth;
  const top = stand.rows * stand.rowRise;

  // Identical concrete terraces on the two LONG court sides.
  // Only the southern stand receives seats. Neither short end has stands.
  for (const side of [-1, 1]) {
    const group = new Group();
    group.name = side < 0 ? "bare-long-side-stand" : "seated-long-side-stand";
    group.position.z = side * stand.frontZ;
    group.rotation.y = side < 0 ? Math.PI : 0;
    parent.add(group);
    const b = createBuilder(group);

    for (let row = 0; row < stand.rows; row++) {
      const z = row * stand.rowDepth;
      const y = (row + 1) * stand.rowRise;
      b.box(
        row % 2 ? p.concrete : p.riser,
        [0, y / 2, z],
        [stand.length, y, stand.rowDepth],
      );
      for (const x of stand.aisles) {
        b.box(
          p.orange,
          [x, y + 0.012, z],
          [0.82, 0.025, stand.rowDepth - 0.02],
        );
      }

      if (side > 0) {
        for (let col = 0; col < 41; col++) {
          const x = -11 + col * 0.55;
          if (stand.aisles.some((aisle) => Math.abs(x - aisle) < 0.7)) continue;
          const color =
            (Math.floor(col / 6) + Math.floor(row / 4)) % 2 ? p.blue : p.yellow;
          b.box(color, [x, y + 0.1, z], [0.47, 0.1, 0.29]);
          b.box(
            color,
            [x, y + 0.28, z + 0.14],
            [0.47, 0.34, 0.065],
            [-0.12, 0, 0],
          );
        }
      }
    }

    b.rail([-halfLength, lastRow + 0.24], [halfLength, lastRow + 0.24], top);
    for (const x of [-halfLength - 0.06, halfLength + 0.06]) {
      b.beam(
        p.blue,
        [x, stand.rowRise + 1.05, 0],
        [x, top + 1.05, lastRow],
        0.065,
      );
      b.beam(
        p.blue,
        [x, stand.rowRise + 0.55, 0],
        [x, top + 0.55, lastRow],
        0.055,
      );
      for (let row = 0; row < stand.rows; row += 2) {
        const y = (row + 1) * stand.rowRise;
        b.box(
          p.blue,
          [x, y + 0.525, row * stand.rowDepth],
          [0.055, 1.05, 0.055],
        );
      }
    }
    b.flush();
  }
}
