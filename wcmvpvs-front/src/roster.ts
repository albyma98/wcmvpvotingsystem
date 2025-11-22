export type PlayerTier = 'gold' | 'silver' | 'bronze';

export interface PlayerLayoutSlot {
  tier: PlayerTier;
  position: { x: number; y: number };
}

export interface PublicPlayer {
  id: number;
  first_name?: string;
  last_name?: string;
  role?: string;
  jersey_number?: number | string | null;
  image_url?: string | null;
  is_called_up?: boolean;
}

export interface LayoutPlayer {
  id: number;
  name: string;
  firstName: string;
  lastName: string;
  role: string;
  number: number | null;
  avatar: string;
  tier: PlayerTier;
  position: { x: number; y: number };
  raw: PublicPlayer;
}

const COLUMN_POSITIONS = [20, 50, 80];

const DEFAULT_GRID_TOP = 12.5;
const DEFAULT_GRID_BOTTOM = 87.5;
const GRID_ROWS = 5;
export const DEFAULT_ROSTER_SCHEMA = 13;

const createRowPositions = (rows: number) => {
  if (rows <= 1) {
    return [DEFAULT_GRID_TOP];
  }
  const spacing = (DEFAULT_GRID_BOTTOM - DEFAULT_GRID_TOP) / (rows - 1);
  return Array.from({ length: rows }, (_, index) => DEFAULT_GRID_TOP + spacing * index);
};

const ROW_POSITIONS = createRowPositions(GRID_ROWS);

const BASE_COORDINATES = [
  { row: 0, column: 0 },
  { row: 0, column: 1 },
  { row: 0, column: 2 },
  { row: 1, column: 0 },
  { row: 1, column: 1 },
  { row: 1, column: 2 },
  { row: 3, column: 0 },
  { row: 3, column: 1 },
  { row: 3, column: 2 },
  { row: 4, column: 0 },
  { row: 4, column: 1 },
  { row: 4, column: 2 },
];

const LAYOUT_COORDINATES: Record<number, { row: number; column: number }[]> = {
  12: BASE_COORDINATES,
  13: [
    ...BASE_COORDINATES.slice(0, 6),
    { row: 2, column: 1 },
    ...BASE_COORDINATES.slice(6),
  ],
  14: [
    ...BASE_COORDINATES.slice(0, 6),
    { row: 2, column: 0 },
    { row: 2, column: 2 },
    ...BASE_COORDINATES.slice(6),
  ],
};

const createRosterLayout = (tiers: PlayerTier[], schema: number): PlayerLayoutSlot[] => {
  const coordinates = LAYOUT_COORDINATES[schema] ?? LAYOUT_COORDINATES[DEFAULT_ROSTER_SCHEMA];

  return coordinates.map((slot, index) => ({
    tier: tiers[index] ?? tiers[tiers.length - 1] ?? 'gold',
    position: {
      x: COLUMN_POSITIONS[slot.column] ?? COLUMN_POSITIONS[COLUMN_POSITIONS.length - 1],
      y: ROW_POSITIONS[slot.row] ?? ROW_POSITIONS[ROW_POSITIONS.length - 1],
    },
  }));
};

const LAYOUT_12_TIERS: PlayerTier[] = [
  'gold',
  'gold',
  'silver',
  'silver',
  'gold',
  'gold',
  'bronze',
  'bronze',
  'silver',
  'silver',
  'bronze',
  'bronze',
];

const LAYOUT_13_TIERS: PlayerTier[] = [
  'gold',
  'gold',
  'silver',
  'silver',
  'gold',
  'gold',
  'bronze',
  'bronze',
  'silver',
  'silver',
  'bronze',
  'bronze',
  'bronze',
];

const LAYOUT_14_TIERS: PlayerTier[] = [
  'gold',
  'gold',
  'silver',
  'silver',
  'gold',
  'gold',
  'bronze',
  'bronze',
  'silver',
  'silver',
  'bronze',
  'bronze',
  'gold',
  'gold',
];

export const PLAYER_LAYOUTS: Record<number, PlayerLayoutSlot[]> = {
  12: createRosterLayout(LAYOUT_12_TIERS, 12),
  13: createRosterLayout(LAYOUT_13_TIERS, 13),
  14: createRosterLayout(LAYOUT_14_TIERS, 14),
};

export const PLAYER_LAYOUT: PlayerLayoutSlot[] = PLAYER_LAYOUTS[DEFAULT_ROSTER_SCHEMA];
export const MAX_PLAYER_SLOTS = Math.max(
  ...Object.values(PLAYER_LAYOUTS).map((layout) => layout.length),
);

const FALLBACK_POSITIONS: PlayerLayoutSlot[] = Array.from({ length: 20 }, (_, index) => {
  const columns = 4;
  const spacingX = 25;
  const spacingY = 15;
  const row = Math.floor(index / columns);
  const column = index % columns;
  return {
    tier: 'gold',
    position: {
      x: 15 + column * spacingX,
      y: 20 + row * spacingY,
    },
  };
});

const sanitizeText = (value?: string | null) => (typeof value === 'string' ? value.trim() : '');

const toNumberOrNull = (value: number | string | null | undefined) => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
};

function resolveRosterSchema(schema?: number): number {
  if (schema === 12 || schema === 13 || schema === 14) {
    return schema;
  }
  return DEFAULT_ROSTER_SCHEMA;
}

export function mapPlayersToLayout(
  players: PublicPlayer[],
  options: { layoutSchema?: number } = {},
): LayoutPlayer[] {
  if (!Array.isArray(players)) {
    return [];
  }

  const allowedPlayers = players.filter((player) => player?.is_called_up === true);

  const layoutSchema = resolveRosterSchema(options.layoutSchema);
  const layout = PLAYER_LAYOUTS[layoutSchema] ?? PLAYER_LAYOUTS[DEFAULT_ROSTER_SCHEMA];

  const sorted = [...allowedPlayers]
    .map((player) => ({
      ...player,
      jersey_number: toNumberOrNull(player.jersey_number),
    }))
    .sort((a, b) => {
      const jerseyA = typeof a.jersey_number === 'number' ? a.jersey_number : Number.MAX_SAFE_INTEGER;
      const jerseyB = typeof b.jersey_number === 'number' ? b.jersey_number : Number.MAX_SAFE_INTEGER;

      if (jerseyA !== jerseyB) {
        return jerseyA - jerseyB;
      }

      const lastA = sanitizeText(a.last_name).toLowerCase();
      const lastB = sanitizeText(b.last_name).toLowerCase();
      if (lastA !== lastB) {
        return lastA.localeCompare(lastB);
      }

      const firstA = sanitizeText(a.first_name).toLowerCase();
      const firstB = sanitizeText(b.first_name).toLowerCase();
      if (firstA !== firstB) {
        return firstA.localeCompare(firstB);
      }

      return a.id - b.id;
    })
    .slice(0, layout.length);

  return sorted.map((player, index) => {
    const slot = layout[index] ?? FALLBACK_POSITIONS[index] ?? FALLBACK_POSITIONS[0];
    const firstName = sanitizeText(player.first_name);
    const lastName = sanitizeText(player.last_name);
    const baseName = `${firstName} ${lastName}`.trim();
    const fallbackName = baseName || `Giocatore ${index + 1}`;
    const role = sanitizeText(player.role);
    const number = typeof player.jersey_number === 'number' ? player.jersey_number : null;
    const avatar = sanitizeText(player.image_url);

    return {
      id: Number.isFinite(player.id) ? player.id : index + 1,
      name: fallbackName,
      firstName,
      lastName,
      role,
      number,
      avatar,
      tier: slot.tier,
      position: slot.position,
      raw: player,
    };
  });
}
