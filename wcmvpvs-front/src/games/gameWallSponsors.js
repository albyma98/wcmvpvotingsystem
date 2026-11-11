// Associa slug, minigioco e lato a un ID del catalogo /sponsors.
export const gameWallSponsorAssignments = {
  // 'slug-societa': { basketball: { left: 12, right: 34 } },
};

export function resolveGameWallSponsors(assignments, sponsors) {
  const findSponsor = (id) => id > 0 ? sponsors.find((sponsor) => sponsor.sponsorId === id) : undefined;
  return {
    left: findSponsor(assignments?.left),
    right: findSponsor(assignments?.right),
  };
}
