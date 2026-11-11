export const postVoteSponsorAssignments = {
  // 'slug-societa': { post_vote_left: 12, post_vote_right: 34 },
};

export function findAssignedSponsor(id, sponsors) {
  return id > 0 ? sponsors.find((sponsor) => sponsor.sponsorId === id) : undefined;
}
