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

const COLUMN_POSITIONS: Record<number, number[]> = {
  1: [50],
  2: [35, 65],
  3: [20, 50, 80],
};

const DEFAULT_GRID_TOP = 12.5;
const DEFAULT_GRID_BOTTOM = 87.5;
const MIN_GRID_ROWS = 4;
const MAX_COLUMNS = 3;

const distributePlayersAcrossRows = (totalPlayers: number, rows: number) => {
  const basePerRow = Math.floor(totalPlayers / rows);
  let remainder = totalPlayers % rows;

  return Array.from({ length: rows }, (_, index) => {
    const count = basePerRow + (remainder > 0 ? 1 : 0);
    if (remainder > 0) {
      remainder -= 1;
    }
    return count;
  }).filter((count) => count > 0);
};

const createEquidistantLayout = (
  tiers: PlayerTier[],
  options: { rows?: number; top?: number; bottom?: number } = {},
): PlayerLayoutSlot[] => {
  const totalPlayers = tiers.length;
  const baseRows = options.rows ?? Math.max(MIN_GRID_ROWS, Math.ceil(totalPlayers / MAX_COLUMNS));
  const rows = totalPlayers >= 13 && baseRows % 2 !== 0 ? baseRows + 1 : baseRows;
  const top = options.top ?? DEFAULT_GRID_TOP;
  const bottom = options.bottom ?? DEFAULT_GRID_BOTTOM;

  const rowDistribution = distributePlayersAcrossRows(totalPlayers, rows);
  const verticalSpacing = rows > 1 ? (bottom - top) / (rows - 1) : 0;

  let tierIndex = 0;

  return rowDistribution.flatMap((playersInRow, rowIndex) => {
    const y = top + rowIndex * verticalSpacing;
    const xPositions = COLUMN_POSITIONS[playersInRow] ?? COLUMN_POSITIONS[MAX_COLUMNS];

    return Array.from({ length: playersInRow }, (_, columnIndex) => {
      const x = xPositions[columnIndex] ?? xPositions[xPositions.length - 1];
      const tier = tiers[tierIndex] ?? tiers[tiers.length - 1] ?? 'gold';
      tierIndex += 1;
      return {
        tier,
        position: { x, y },
      };
    });
  });
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
  12: createEquidistantLayout(LAYOUT_12_TIERS),
  13: createEquidistantLayout(LAYOUT_13_TIERS),
  14: createEquidistantLayout(LAYOUT_14_TIERS),
};

export const DEFAULT_ROSTER_SCHEMA = 13;
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
