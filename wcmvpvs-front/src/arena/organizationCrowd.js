import { SEATS, weightedSeats } from "./seating/seatMap.js";

// Server orders immutable profile IDs: new registrations append without moving
// existing fans. No random filler, local identity override or demo storage.
export function organizationCrowd(fans, organizationID, seatMap = SEATS) {
  const ordered = [...fans].sort((a, b) => a.id - b.id);
  const reserved = new Set(ordered.filter((fan) => Number.isInteger(fan.slot)).map((fan) => fan.slot));
  const available = reserved.size < ordered.length
    ? weightedSeats(seatMap.filter((_, slot) => !reserved.has(slot)), Number(organizationID) || 1)
    : [];
  const actors = ordered.flatMap((fan) => {
    const seat = Number.isInteger(fan.slot) ? seatMap[fan.slot] : available.shift();
    if (!seat) return [];
    return [{
      id: String(fan.id),
      kind: "profile",
      seed: fan.id * 71 + Number(organizationID),
      seat,
      equipment: fan.equipment || {},
      public: fan.public !== false,
      profile: {
        id: String(fan.id),
        nickname: fan.nickname || "Tifoso riservato",
        active: false,
      },
    }];
  });
  const me = ordered.find((fan) => fan.is_me);
  return {
    actors,
    selectedId: me ? String(me.id) : null,
    total: ordered.length,
  };
}
