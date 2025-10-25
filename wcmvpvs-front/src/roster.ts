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

export const PLAYER_LAYOUT: PlayerLayoutSlot[] = [
  { tier: 'gold', position: { x: 20, y: 14 } },
  { tier: 'gold', position: { x: 50, y: 14 } },
  { tier: 'silver', position: { x: 80, y: 14 } },
  { tier: 'gold', position: { x: 20, y: 32 } },
  { tier: 'silver', position: { x: 50, y: 32 } },
  { tier: 'gold', position: { x: 80, y: 32 } },
  { tier: 'bronze', position: { x: 50, y: 50 } },
  { tier: 'bronze', position: { x: 20, y: 68 } },
  { tier: 'bronze', position: { x: 50, y: 68 } },
  { tier: 'silver', position: { x: 80, y: 68 } },
  { tier: 'bronze', position: { x: 20, y: 86 } },
  { tier: 'silver', position: { x: 50, y: 86 } },
  { tier: 'bronze', position: { x: 80, y: 86 } },
];

const FALLBACK_POSITIONS: PlayerLayoutSlot[] = Array.from({ length: 20 }, (_, index) => {
  const columns = 4;
  const spacingX = 25;
  const spacingY = 18;
  const row = Math.floor(index / columns);
  const column = index % columns;
  return {
    tier: 'gold',
    position: {
      x: 15 + column * spacingX,
      y: 18 + row * spacingY,
    },
  };
});

const sanitizeText = (value?: string | null) => (typeof value === 'string' ? value.trim() : '');

const toNumberOrNull = (value: number | string | null | undefined) => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
};

export function mapPlayersToLayout(players: PublicPlayer[]): LayoutPlayer[] {
  if (!Array.isArray(players)) {
    return [];
  }

  const sorted = [...players]
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
    .slice(0, PLAYER_LAYOUT.length);

  return sorted.map((player, index) => {
    const slot = PLAYER_LAYOUT[index] ?? FALLBACK_POSITIONS[index] ?? FALLBACK_POSITIONS[0];
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
