import * as THREE from "three";
import { canvasTexture, createBuilder } from "./primitives.js";
import { sectorSeats } from "./sceneConfig.js";

function createBasketballCourt(parent, { court, layout, materials }, arenaName) {
  const halfLength = court.length / 2, halfWidth = court.width / 2;
  const map = canvasTexture(2048, 1536, (ctx, width, height) => {
    ctx.translate(width / 2, height / 2);
    ctx.scale(width / layout.length, height / layout.width);
    ctx.fillStyle = materials.parquet;
    ctx.fillRect(-layout.length / 2, -layout.width / 2, layout.length, layout.width);
    for (let z = -layout.width / 2; z < layout.width / 2; z += 0.24) {
      ctx.fillStyle = "rgba(103, 62, 28, .09)";
      ctx.fillRect(-layout.length / 2, z, layout.length, 0.015);
      const shift = (Math.round(z / 0.24) % 3) * 0.52;
      for (let x = -layout.length / 2 + 1.6; x < layout.length / 2 - 1; x += 1.6)
        ctx.fillRect(x + shift, z, 0.02, 0.24);
    }
    ctx.strokeStyle = materials.lines;
    ctx.lineWidth = 0.055;
    ctx.strokeRect(-halfLength, -halfWidth, court.length, court.width);
    ctx.beginPath();
    ctx.moveTo(0, -halfWidth);
    ctx.lineTo(0, halfWidth);
    ctx.stroke();
    ctx.beginPath();
    ctx.arc(0, 0, 1.8, 0, 2 * Math.PI);
    ctx.stroke();
    if (court.has_center_logo && arenaName) {
      ctx.fillStyle = materials.paint;
      ctx.font = "bold 0.55px sans-serif";
      ctx.textAlign = "center";
      ctx.textBaseline = "middle";
      ctx.fillText(arenaName.toUpperCase(), 0, 0, 3.25);
    }
    for (const side of [-1, 1]) {
      ctx.save();
      ctx.scale(side, 1);
      const basket = -halfLength + 1.575;
      const corner = halfWidth - 0.9;
      const angle = Math.asin(Math.min(corner / 6.75, 1));
      const arcEnd = basket + 6.75 * Math.cos(angle);
      ctx.fillStyle = materials.paint;
      ctx.fillRect(-halfLength, -2.45, 5.8, 4.9);
      ctx.strokeRect(-halfLength, -2.45, 5.8, 4.9);
      ctx.beginPath();
      ctx.arc(-halfLength + 5.8, 0, 1.8, -Math.PI / 2, Math.PI / 2);
      ctx.stroke();
      ctx.beginPath();
      ctx.moveTo(-halfLength, -corner);
      ctx.lineTo(arcEnd, -corner);
      ctx.arc(basket, 0, 6.75, -angle, angle);
      ctx.lineTo(-halfLength, corner);
      ctx.stroke();
      ctx.restore();
    }
  });
  const floor = new THREE.Mesh(
    new THREE.PlaneGeometry(layout.length, layout.width),
    new THREE.MeshStandardMaterial({ map, roughness: 0.45 }),
  );
  floor.rotation.x = -Math.PI / 2;
  floor.position.y = 0.015;
  parent.add(floor);

  const hoops = createBuilder(parent);
  const rimMaterial = new THREE.MeshStandardMaterial({ color: materials.paint });
  const rimGeometry = new THREE.TorusGeometry(0.23, 0.024, 8, 28);
  for (const side of [-1, 1]) {
    const support = side * (halfLength + 0.85);
    const board = side * (halfLength - 1.2);
    hoops.box(materials.metal, [support, 1.6, 0], [0.18, 3.2, 0.2]);
    hoops.beam(materials.metal, [support, 3.2, 0], [board, 3.32, 0], 0.12);
    hoops.box(materials.lines, [board, 3.45, 0], [0.07, 1.05, 1.8]);
    hoops.box(materials.paint, [board - side * 0.05, 3.28, 0], [0.025, 0.36, 0.59]);
    const rim = new THREE.Mesh(rimGeometry, rimMaterial);
    rim.position.set(side * (halfLength - 1.575), 3.05, 0);
    rim.rotation.x = -Math.PI / 2;
    parent.add(rim);
  }
  hoops.flush();
}

export function createConfiguredArena(parent, scene, arenaName) {
  const { layout, materials, sectors } = scene;
  createBasketballCourt(parent, scene, arenaName);
  const chairs = new THREE.Group(), railings = new THREE.Group(), roof = new THREE.Group();
  chairs.userData.count = 0;
  parent.add(chairs, railings, roof);

  for (const sector of sectors) {
    const group = new THREE.Group();
    group.name = sector.slug;
    group.rotation.y = sector.rotation;
    parent.add(group);
    const stand = createBuilder(group);
    for (let row = 0; row < sector.rows_count; row++) {
      const top = sector.base_height + row * sector.row_rise;
      if (top > 0) stand.box(
        materials.concrete,
        [sector.center_offset, top / 2, sector.front_offset + row * sector.row_depth],
        [sector.length, top, sector.row_depth],
      );
    }
    stand.flush();

    if (sector.seat_style === "chairs") {
      const seats = new THREE.Group();
      seats.rotation.y = sector.rotation;
      chairs.add(seats);
      const bench = createBuilder(seats);
      const width = Math.min(0.44, 0.83 * sector.length / sector.seats_per_row);
      for (const { offset, depth, y } of sectorSeats(sector)) {
        bench.box(materials.seats, [offset, y + 0.1, depth], [width, 0.1, 0.27]);
        bench.box(materials.seats, [offset, y + 0.27, depth + 0.13], [width, 0.29, 0.07]);
        chairs.userData.count++;
      }
      bench.flush();
    }
    if (sector.glass) {
      const { color, height, opacity } = sector.glass;
      const barrier = new THREE.Group();
      barrier.rotation.y = sector.rotation;
      railings.add(barrier);
      const front = sector.front_offset - sector.row_depth / 2;
      const pane = new THREE.Mesh(
        new THREE.BoxGeometry(sector.length, height, 0.035),
        new THREE.MeshStandardMaterial({ color, transparent: true, opacity, depthWrite: false }),
      );
      pane.position.set(sector.center_offset, sector.base_height + height / 2, front);
      barrier.add(pane);
      const frame = createBuilder(barrier);
      const posts = Math.ceil(sector.length / 2.4);
      for (let i = 0; i <= posts; i++) frame.box(
        materials.metal,
        [sector.center_offset + (i / posts - 0.5) * sector.length, sector.base_height + height / 2, front],
        [0.055, height, 0.06],
      );
      frame.box(materials.metal, [sector.center_offset, sector.base_height + height, front], [sector.length, 0.045, 0.055]);
      frame.flush();
    }
  }

  const walls = [];
  for (const axis of ["x", "z"]) {
    for (const side of [-1, 1]) {
      const group = new THREE.Group();
      const alongLength = axis === "z";
      const limit = (alongLength ? layout.width : layout.length) / 2;
      const wall = createBuilder(group);
      wall.box(materials.wall,
        alongLength ? [0, layout.height / 2, side * limit] : [side * limit, layout.height / 2, 0],
        alongLength ? [layout.length, layout.height, 0.24] : [0.24, layout.height, layout.width],
      );
      wall.flush();
      parent.add(group);
      walls.push({ group, axis, side, limit });
    }
  }
  const beams = createBuilder(roof);
  for (let i = -3; i <= 3; i++) {
    beams.box(materials.metal, [i * layout.length / 8, layout.height - 0.28, 0], [0.16, 0.25, layout.width]);
  }
  beams.box(materials.wall, [0, layout.height, 0], [layout.length, 0.22, layout.width]);
  beams.flush();
  return { roof, walls, chairs, railings };
}
