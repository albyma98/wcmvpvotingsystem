import { assignSeats, createFillers, SEAT_BY_ID } from "../seating/seatMap.js";
import { createDemoProfiles } from "../crowd/demoProfiles.js";
import { DEMO_PLAYERS } from "../mvp/players.js";

export const STORAGE_KEY = "abx.arena-tifosi.demo.v1";
export const FAN_ACTIONS = {
  clap: {
    label: "Applaudi",
    recent: "Ha applaudito",
    duration: 3600,
    cooldown: 4500,
  },
  scarf: {
    label: "Alza la sciarpa",
    recent: "Ha alzato la sciarpa",
    duration: 5200,
    cooldown: 7000,
  },
  wave: {
    label: "Coreografia / Ola",
    recent: "Ha lanciato l’ola",
    duration: 8000,
    cooldown: 12000,
  },
};

export function createDemoSession(storage) {
  let saved = {};
  try {
    const value = JSON.parse(storage?.getItem(STORAGE_KEY) || "null");
    if (value?.version === 1) saved = value;
  } catch {}
  const profiles = createDemoProfiles();
  const baseInteractions = new Map(
    profiles.map((profile) => [profile.id, profile.interactions]),
  );
  const profileIds = new Set(profiles.map((profile) => profile.id));
  const playerIds = new Set(DEMO_PLAYERS.map((player) => player.id));
  const recentLabels = new Set([
    "In tribuna",
    "Ha votato l’MVP",
    ...Object.values(FAN_ACTIONS).map((action) => action.recent),
  ]);
  const assignments = assignSeats(profiles, saved.assignments);
  for (const profile of profiles) {
    profile.seatId = assignments[profile.id];
    if (Array.isArray(saved.activated) && saved.activated.includes(profile.id))
      profile.active = true;
    const recent = saved.recentActions?.[profile.id];
    if (recentLabels.has(recent)) profile.recentAction = recent;
    else if (playerIds.has(saved.votes?.[profile.id]))
      profile.recentAction = "Ha votato l’MVP";
    const count = saved.interactions?.[profile.id];
    if (Number.isInteger(count) && count >= 0 && count < 100000)
      profile.interactions += count;
  }
  let selectedId = profileIds.has(saved.selectedId)
    ? saved.selectedId
    : profiles[0].id;
  let votes = Object.fromEntries(
    Object.entries(saved.votes || {}).filter(
      ([id, player]) => profileIds.has(id) && playerIds.has(player),
    ),
  );
  let interactionCounts = Object.fromEntries(
    profiles.map((profile) => [
      profile.id,
      Math.max(0, profile.interactions - baseInteractions.get(profile.id)),
    ]),
  );
  let cooldowns = {};
  for (const [key, value] of Object.entries(saved.cooldowns || {})) {
    const [profileId, kind] = key.split(":");
    if (
      profileIds.has(profileId) &&
      Object.hasOwn(FAN_ACTIONS, kind) &&
      Number.isFinite(value) &&
      value > Date.now() &&
      value < Date.now() + 12000
    )
      cooldowns[key] = value;
  }
  let persistent = true;
  const fillers = createFillers(assignments);
  const actors = [
    ...profiles.map((profile) => ({
      id: profile.id,
      kind: "profile",
      profile,
      seat: SEAT_BY_ID.get(profile.seatId),
      seed: profile.seed,
    })),
    ...fillers,
  ];
  profiles.find((profile) => profile.id === selectedId).active = true;
  function save() {
    try {
      if (!storage) throw new Error("Storage unavailable");
      storage.setItem(
        STORAGE_KEY,
        JSON.stringify({
          version: 1,
          assignments,
          selectedId,
          votes,
          activated: profiles.filter((p) => p.active).map((p) => p.id),
          interactions: interactionCounts,
          recentActions: Object.fromEntries(
            profiles.map((profile) => [profile.id, profile.recentAction]),
          ),
          cooldowns,
        }),
      );
      persistent = true;
    } catch {
      persistent = false;
    }
  }
  function current() {
    return profiles.find((profile) => profile.id === selectedId);
  }
  function record(label) {
    const fan = current();
    fan.recentAction = label;
    fan.interactions++;
    interactionCounts[fan.id] = (interactionCounts[fan.id] || 0) + 1;
  }
  function snapshot() {
    return {
      selectedId,
      profiles: profiles.map((profile) => ({
        ...profile,
        seat: SEAT_BY_ID.get(profile.seatId),
      })),
      votes: { ...votes },
      cooldowns: { ...cooldowns },
      persistent,
    };
  }
  function select(id) {
    if (!profileIds.has(id)) return false;
    selectedId = id;
    current().active = true;
    save();
    return true;
  }
  function remaining(kind, now = Date.now()) {
    return Math.max(0, (cooldowns[`${selectedId}:${kind}`] || 0) - now);
  }
  function action(kind, now = Date.now()) {
    if (!Object.hasOwn(FAN_ACTIONS, kind))
      return { ok: false, reason: "Azione non disponibile" };
    const wait = remaining(kind, now);
    if (wait)
      return { ok: false, reason: "Attendi qualche secondo", remaining: wait };
    cooldowns[`${selectedId}:${kind}`] = now + FAN_ACTIONS[kind].cooldown;
    record(FAN_ACTIONS[kind].recent);
    save();
    return { ok: true, fanId: selectedId, kind };
  }
  function vote(playerId) {
    if (!playerIds.has(playerId))
      return { ok: false, reason: "Giocatore non disponibile" };
    if (votes[selectedId])
      return {
        ok: false,
        reason: "Questo profilo ha già votato",
        playerId: votes[selectedId],
      };
    votes[selectedId] = playerId;
    record("Ha votato l’MVP");
    save();
    return { ok: true, fanId: selectedId, playerId };
  }
  function reset() {
    votes = {};
    cooldowns = {};
    interactionCounts = {};
    const initial = createDemoProfiles();
    profiles.forEach((profile, i) => {
      profile.active = initial[i].active;
      profile.recentAction = "In tribuna";
      profile.interactions = initial[i].interactions;
    });
    selectedId = profiles[0].id;
    save();
  }
  save();
  return { actors, profiles, snapshot, select, action, vote, remaining, reset };
}
