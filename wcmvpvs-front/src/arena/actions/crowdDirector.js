import { FAN_ACTIONS } from "./demoSession.js";

const clamp = (value) => Math.max(0, Math.min(1, value));
const pulse = (age, duration) =>
  age < 0 || age > duration ? 0 : Math.sin(Math.PI * clamp(age / duration));

export function createCrowdDirector(actors) {
  const byId = new Map(actors.map((actor) => [actor.id, actor]));
  let events = [];
  function trigger(kind, fanId, now) {
    const source = byId.get(fanId);
    if (
      !source ||
      source.kind !== "profile" ||
      (!Object.hasOwn(FAN_ACTIONS, kind) && kind !== "vote")
    )
      return false;
    const nearby = new Map();
    for (const actor of actors) {
      if (actor.seat.stand !== source.seat.stand) continue;
      const distance = Math.hypot(
        (actor.seat.x - source.seat.x) * 0.8,
        (actor.seat.row - source.seat.row) * 0.65,
      );
      if (distance < 3.8 || actor.id === fanId)
        nearby.set(actor.id, distance * 85);
    }
    events.push({
      kind,
      source,
      start: now,
      end: now + (kind === "vote" ? 5200 : FAN_ACTIONS[kind].duration),
      nearby,
    });
    events = events.slice(-6);
    return true;
  }
  function sample(actor, now) {
    for (let i = events.length - 1; i >= 0; i--) {
      const event = events[i];
      if (now > event.end) continue;
      if (event.kind === "wave") {
        const delay =
          ((actor.seat.x + 11.5) / 23) * 4500 +
          (actor.seat.stand === event.source.seat.stand ? 0 : 900);
        const strength = pulse(now - event.start - delay, 1600);
        if (strength > 0)
          return {
            standing: strength,
            raise: strength,
            scarf: 0,
            clap: 0,
            energy: strength,
          };
      } else if (event.nearby.has(actor.id)) {
        const age = now - event.start - event.nearby.get(actor.id);
        const strength = pulse(age, event.kind === "scarf" ? 4400 : 3200);
        if (strength > 0)
          return {
            standing: event.kind === "clap" ? 0 : strength,
            raise: event.kind === "scarf" ? strength : 0,
            scarf: event.kind === "scarf" ? strength : 0,
            clap: event.kind === "scarf" ? 0 : strength,
            energy: strength,
          };
      }
    }
    return null;
  }
  function activeFor(actor, now) {
    return events.some(
      (event) =>
        now <= event.end &&
        (event.kind === "wave" || event.nearby.has(actor.id)),
    );
  }
  function latest(now) {
    events = events.filter((event) => now <= event.end);
    return events.at(-1);
  }
  return {
    trigger,
    sample,
    activeFor,
    latest,
    reset: () => {
      events = [];
    },
  };
}
