import { findAssignedSponsor } from '../sponsorPlacements.js';

// Associa slug, minigioco e lato a un ID del catalogo /sponsors.
export const gameWallSponsorAssignments = {
  // 'slug-societa': { basketball: { left: 12, right: 34 } },
};

export function resolveGameWallSponsors(assignments, sponsors) {
  return {
    left: findAssignedSponsor(assignments?.left, sponsors),
    right: findAssignedSponsor(assignments?.right, sponsors),
  };
}
