import { SEATS, weightedSeats } from "./seating/seatMap.js";

// Server orders immutable profile IDs: new registrations append without moving
// existing fans. No random filler, local identity override or demo storage.
export function organizationCrowd(fans, organizationID) {
  const seats = weightedSeats(SEATS, Number(organizationID) || 1);
  const ordered = [...fans].sort((a, b) => a.id - b.id);
  const actors = ordered.slice(0, seats.length).map((fan, index) => ({
    id: String(fan.id),
    kind: "profile",
    seed: fan.id * 71 + Number(organizationID),
    seat: seats[index],
    profile: { id: String(fan.id), nickname: fan.nickname, active: false },
  }));
  const me = ordered.find((fan) => fan.is_me);
  return {
    actors,
    selectedId: me ? String(me.id) : null,
    total: ordered.length,
  };
}
