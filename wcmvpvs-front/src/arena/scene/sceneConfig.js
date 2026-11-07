import { SEATS } from "../seating/seatMap.js";

const rotations = { long_1: 0, long_2: Math.PI, short_1: -Math.PI / 2, short_2: Math.PI / 2 };
const colors = {
  parquet: "#d7b57a", lines: "#ffffff", paint: "#a21e30",
  concrete: "#a5a4a1", seats: "#a21e30", metal: "#52555c",
  glass: "#c8e5eb", wall: "#dfded7",
};

function between(value, min, max) {
  return Number.isFinite(value) && value >= min && value <= max;
}

function point(value) {
  return Array.isArray(value) && value.length === 3 &&
    value.every((coordinate) => between(coordinate, -100, 100));
}

function color(value, fallback) {
  return typeof value === "string" && /^#[\da-f]{6}$/i.test(value) ? value : fallback;
}

export function readArenaScene(config) {
  if (config?.schema_version !== 1 || config.court?.sport !== "basket") return null;
  const { court, layout, camera } = config;
  if (!between(court.length, 24, 36) || !between(court.width, 13, 19) ||
      !between(layout?.length, court.length + 4, 65) ||
      !between(layout?.width, court.width + 4, 55) ||
      !between(layout?.height, 6, 25) || !Array.isArray(config.sectors) ||
      config.sectors.length > 40) return null;

  const materials = Object.fromEntries(Object.entries(colors).map(([key, fallback]) =>
    [key, color(config.materials?.[key], fallback)],
  ));
  const sectors = [];
  let seatCount = 0;
  for (const sector of config.sectors) {
    if (sector?.is_active === false) continue;
    if (!sector || !Object.hasOwn(rotations, sector.side) ||
        typeof sector.slug !== "string" || !sector.slug || typeof sector.label !== "string" ||
        sectors.some(({ slug }) => slug === sector.slug) ||
        !Number.isInteger(sector.rows_count) || !between(sector.rows_count, 1, 35) ||
        !Number.isInteger(sector.seats_per_row) || !between(sector.seats_per_row, 1, 90) ||
        !between(sector.length, 1, 45) || !between(sector.center_offset, -20, 20) ||
        !between(sector.front_offset, 1, 32) || !between(sector.base_height, 0, 12) ||
        !between(sector.row_depth, 0.2, 1.5) || !between(sector.row_rise, 0, 0.8)) return null;

    const shortSide = sector.side.startsWith("short");
    const rotation = rotations[sector.side];
    const glass = config.stands?.[sector.side]?.glass_divider;
    sectors.push({
      ...sector,
      shortSide,
      rotation,
      glass: sector.has_glass_divider ? {
        color: color(glass?.color, materials.glass),
        height: between(glass?.height, 0.2, 2) ? glass.height : 0.95,
        opacity: between(glass?.opacity, 0.05, 0.8) ? glass.opacity : 0.32,
      } : null,
    });
    seatCount += sector.rows_count * sector.seats_per_row;
    if (seatCount > 6000) return null;
  }
  const treasure = config.activity_anchors?.treasure;
  return {
    layout, court, materials, sectors,
    camera: point(camera?.position) && point(camera?.target) ? {
      position: camera.position,
      target: camera.target,
      fov: between(camera.fov, 30, 90) ? camera.fov : 57,
    } : null,
    activityAnchors: {
      museum: point(config.activity_anchors?.museum) ? config.activity_anchors.museum : null,
      predictions: point(config.activity_anchors?.predictions) ? config.activity_anchors.predictions : null,
      treasure: Array.isArray(treasure) && treasure.length === 10 &&
        treasure.every((spot) => point(spot?.position) && typeof spot.hint === "string") ? treasure : null,
    },
  };
}

export function sectorSeats(sector) {
  const spacing = sector.length / sector.seats_per_row;
  const cos = Math.cos(sector.rotation), sin = Math.sin(sector.rotation);
  const seats = [];
  for (let row = 0; row < sector.rows_count; row++) {
    const y = sector.base_height + row * sector.row_rise;
    const depth = sector.front_offset + row * sector.row_depth;
    for (let column = 0; column < sector.seats_per_row; column++) {
      const offset = sector.center_offset + (column + 0.5) * spacing - sector.length / 2;
      seats.push({
        id: `${sector.slug}-${row + 1}-${column + 1}`,
        stand: sector.side, sector: sector.label, column, row: row + 1, number: column + 1,
        side: sector.side === "long_2" ? -1 : 1,
        x: offset * cos + depth * sin, y, z: depth * cos - offset * sin,
        yaw: sector.rotation, hasSeat: sector.seat_style === "chairs",
        offset, depth,
      });
    }
  }
  return seats;
}

export function sceneSeatMap(scene) {
  if (!scene) return SEATS;
  const seats = scene.sectors
    .filter((sector) => !["press", "dirigenza"].includes(sector.sector_type))
    .flatMap(sectorSeats);
  if (!seats.length) return [];
  // Keep the same virtual slot numbers when the organization changes scenes.
  // Spread those slots evenly through the configured stands when there are extra seats.
  return seats.length <= SEATS.length ? seats : SEATS.map(
    (_, index) => seats[Math.floor(index * seats.length / SEATS.length)],
  );
}
