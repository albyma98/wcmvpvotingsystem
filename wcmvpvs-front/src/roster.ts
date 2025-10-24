export interface RosterPlayer {
  id: number;
  name: string;
  firstName: string;
  lastName: string;
  role: string;
  number: number;
  tier: 'gold' | 'silver' | 'bronze';
  position: { x: number; y: number };
  avatar: string;
}

export const roster: RosterPlayer[] = [
  {
    id: 1,
    name: 'Matteo Paris',
    firstName: 'Matteo',
    lastName: 'Paris',
    role: 'Opposto',
    number: 10,
    tier: 'gold',
    position: { x: 20, y: 14 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 2,
    name: 'Giuseppe Longo',
    firstName: 'Giuseppe',
    lastName: 'Longo',
    role: 'Palleggiatore',
    number: 8,
    tier: 'gold',
    position: { x: 50, y: 14 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 3,
    name: 'Felice Sette',
    firstName: 'Felice',
    lastName: 'Sette',
    role: 'Centrale',
    number: 7,
    tier: 'silver',
    position: { x: 80, y: 14 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 4,
    name: 'Sebastiano Milan',
    firstName: 'Sebastiano',
    lastName: 'Milan',
    role: 'Schiacciatore',
    number: 9,
    tier: 'gold',
    position: { x: 20, y: 32 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 5,
    name: 'Leonardo Carta',
    firstName: 'Leonardo',
    lastName: 'Carta',
    role: 'Centrale',
    number: 15,
    tier: 'silver',
    position: { x: 50, y: 32 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 6,
    name: 'Romolo Mariano',
    firstName: 'Romolo',
    lastName: 'Mariano',
    role: 'Schiacciatore',
    number: 3,
    tier: 'gold',
    position: { x: 80, y: 32 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 7,
    name: 'Alessio Santangelo',
    firstName: 'Alessio',
    lastName: 'Santangelo',
    role: 'Libero',
    number: 1,
    tier: 'bronze',
    position: { x: 50, y: 50 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 8,
    name: 'Alberto Marra',
    firstName: 'Alberto',
    lastName: 'Marra',
    role: 'Opposto',
    number: 5,
    tier: 'bronze',
    position: { x: 20, y: 68 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 9,
    name: 'Cristian Frumuselu',
    firstName: 'Cristian',
    lastName: 'Frumuselu',
    role: 'Centrale',
    number: 6,
    tier: 'bronze',
    position: { x: 50, y: 68 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 10,
    name: 'Vincenzo Utro',
    firstName: 'Vincenzo',
    lastName: 'Utro',
    role: 'Palleggiatore',
    number: 33,
    tier: 'silver',
    position: { x: 80, y: 68 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 11,
    name: 'Francesco Pierri',
    firstName: 'Francesco',
    lastName: 'Pierri',
    role: 'Schiacciatore',
    number: 14,
    tier: 'bronze',
    position: { x: 20, y: 86 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 12,
    name: 'Alessandro Chinello',
    firstName: 'Alessandro',
    lastName: 'Chinello',
    role: 'Libero',
    number: 13,
    tier: 'silver',
    position: { x: 50, y: 86 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
  {
    id: 13,
    name: 'Sandi Persoglia',
    firstName: 'Sandi',
    lastName: 'Persoglia',
    role: 'Opposto',
    number: 11,
    tier: 'bronze',
    position: { x: 80, y: 86 },
    avatar: 'https://www.legavolley.it/Foto.aspx?Key=UTR-VIN-96&sqid=6733&heightImg=600',
  },
];

export const rosterById = new Map(roster.map((player) => [player.id, player]));

export function findRosterPlayer(playerId: number): RosterPlayer | undefined {
  return rosterById.get(playerId);
}
