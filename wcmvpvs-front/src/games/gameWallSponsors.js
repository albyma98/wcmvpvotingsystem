import { findAssignedSponsor } from '../sponsorPlacements.js';

export const gameWallGames = [
  { id: 'basketball', label: 'Basket Rush' },
  { id: 'playmaker', label: 'Playmaker' },
  { id: 'crossover', label: 'Crossover Rush' },
];

export function resolveGameWallSponsors(assignments, gameId, sponsors) {
  return {
    left: findAssignedSponsor(assignments[`${gameId}_left`], sponsors),
    right: findAssignedSponsor(assignments[`${gameId}_right`], sponsors),
  };
}
