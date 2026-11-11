export const postVoteSponsorSlots = [
  { id: 'post_vote_left', label: 'Banner a sinistra' },
  { id: 'post_vote_right', label: 'Banner a destra' },
];

export function findAssignedSponsor(id, sponsors) {
  return id > 0 ? sponsors.find((sponsor) => sponsor.sponsorId === id) : undefined;
}
