import { createBuilder, palette as p } from "./primitives";

export function createStands(parent) {
  const b = createBuilder(parent);
  // The dominant concrete stand is on the short right end, as in the photograph.
  for (let row = 0; row < 19; row++) {
    const x = 10.6 + row * 0.38,
      y = (row + 1) * 0.31;
    b.box(row % 2 ? p.concrete : p.riser, [x, y / 2, -1], [0.38, y, 23]);
    for (const z of [-8.2, 5.7])
      b.box(p.orange, [x, y + 0.012, z], [0.36, 0.025, 0.82]);
  }
  b.rail([17.75, -12.5], [17.75, 10.5], 5.9);
  b.rail([10.2, -12.65], [17.7, -12.65], 0);
  for (const z of [-12.5, 10.5]) {
    b.beam(p.blue, [10.3, 1, z], [17.8, 6.8, z], 0.065);
    for (let i = 0; i < 10; i++)
      b.beam(
        p.blue,
        [10.5 + i * 0.77, i * 0.32, z],
        [10.5 + i * 0.77, 1.05 + i * 0.59, z],
        0.045,
      );
  }
  // Low foreground bleachers, yellow molded seats and blue handrails.
  for (let row = 0; row < 6; row++) {
    const z = 10.5 + row * 0.65,
      y = (row + 1) * 0.34;
    b.box(p.riser, [-3, y / 2, z], [28, y, 0.65]);
    for (let col = 0; col < 45; col++) {
      const x = -16.5 + col * 0.61;
      if (Math.abs(x + 9) < 0.7 || Math.abs(x - 3) < 0.7) continue;
      b.box(p.yellow, [x, y + 0.1, z], [0.53, 0.12, 0.45]);
      b.box(
        p.yellow,
        [x, y + 0.3, z + 0.22],
        [0.53, 0.34, 0.075],
        [-0.12, 0, 0],
      );
    }
  }
  b.rail([-17, 9.9], [10, 9.9], 0.15, 1.1);
  b.rail([-17, 14.1], [10, 14.1], 2.05, 1.05);
  for (const x of [-9, 3]) {
    b.beam(p.blue, [x, 1.2, 10], [x, 3.2, 14.1], 0.065);
    for (let i = 0; i < 5; i++)
      b.box(p.blue, [x, 0.9 + i * 0.32, 10.4 + i * 0.8], [0.055, 1.1, 0.055]);
  }
  b.flush();
}
