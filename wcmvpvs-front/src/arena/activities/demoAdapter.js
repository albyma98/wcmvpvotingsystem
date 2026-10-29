import { CATALOG, dayInRome,weeklyFeatured } from "./model.js";
import { createDemoProfiles } from "../crowd/demoProfiles.js";
export const DEMO_ACTIVITY_KEY = "abx.arena.activities.demo.v3";
const config = {
  treasure: 20,
  exact: 50,
  winner: 15,
  growth: 100,
  growth_big: 500,
  challenge: 100,
  collective: 3,
  record: 10,
};
const milestones = [
  ["fans100", "Primi 100 tifosi", 100],
  ["fans500", "Primi 500 tifosi", 500],
  ["ola", "Prima coreografia collettiva", 3],
  ["challenge", "100 partecipazioni alla caccia", 100],
  ["champion", "Tribuna campione", 1],
  ["record", "Record di partecipazione", 10],
];
export function createDemoActivityAdapter(
  storage = window.localStorage,
  { dev = false, now = () => new Date() } = {},
) {
  let data;
  try {
    data = JSON.parse(storage.getItem(DEMO_ACTIVITY_KEY));
  } catch {}
  const initial = () => ({
    version: 3,
    wallet: 0,
    points: 0,
    owned: [],
    equipment: {},
    claims: [],
    votes: null,
    predictions: {},
    trophies: {ola:now().toISOString(),record:now().toISOString()},
    offset: 0,
    location: null,
    public: true,
    invitation: "",
    friends: [],
    match: {
      id: 1,
      home: "Joy Volley · DEMO",
      away: "Squadra ospite · DEMO",
      starts: new Date(now().getTime() + 86400000 * 3).toISOString(),
      score: "",
    },
  });
  if (data?.version !== 3) data = initial();
  const save = () => {
    storage.setItem(DEMO_ACTIVITY_KEY, JSON.stringify(data));
  };
  const date = () => new Date(now().getTime() + data.offset * 86400000);
  function snapshot() {
    const time = date(),
      day = dayInRome(time);
    let hash = 7;
    for (const c of day) hash = (hash * 31 + c.charCodeAt(0)) >>> 0;
    const prediction = data.predictions[data.match.id];
    return {
      now: time.toISOString(),
      treasure: {
        day,
        kind: hash % 4,
        location: data.location ?? hash % 10,
        reward: config.treasure,
        claimed: data.claims.includes(day),
      },
      catalog: CATALOG,
      featured:weeklyFeatured(time),
      owned: [...data.owned],
      equipment: { ...data.equipment },
      wallet: data.wallet,
      points: data.points,
      level: 1 + Math.floor(data.points / 100),
      member: {
        fan_id: 1,
        slot: 0,
        public: data.public,
        equipment: { ...data.equipment },
      },
      members: [
        {
          fan_id: 1,
          slot: 0,
          public: data.public,
          equipment: { ...data.equipment },
        },
      ],
      matches: [
        {
          ...data.match,
          cutoff: data.match.starts,
          open: time < new Date(data.match.starts) && !data.match.score,
          choice: prediction?.choice || "",
          reward: prediction?.reward || 0,
          settled: prediction?.reward != null,
        },
      ],
      trophies: milestones.map(([key, title, target]) => ({
        key,
        title,
        description:
          "Traguardo dimostrativo · nessuna statistica reale del club",
        target,
        value: data.trophies[key] ? target : 0,
        unlocked: data.trophies[key] || "",
      })),
      config,
      invitation: data.invitation,
      demo: true,
      voted: data.votes,
    };
  }
  async function command(cmd) {
    const state = snapshot();
    const item = CATALOG.find((i) => i.key === cmd.key);
    switch (cmd.action) {
      case "claim":
        if (
          cmd.day !== state.treasure.day ||
          cmd.location !== state.treasure.location
        )
          throw Error("Il tesoro di oggi è cambiato.");
        if (!data.claims.includes(cmd.day)) {
          data.claims.push(cmd.day);
          data.wallet += config.treasure;
          data.points += config.treasure;
        }
        break;
      case "buy":
        if (!item) throw Error("Accessorio non disponibile.");
        if (!data.owned.includes(item.key)) {
          if (
            state.level < item.level ||
            (item.availability === "weekly" &&
              !state.featured.includes(item.key))
          )
            throw Error("Accessorio bloccato.");
          if (data.wallet < item.cost) throw Error("Monetine insufficienti.");
          data.wallet -= item.cost;
          data.owned.push(item.key);
        }
        break;
      case "equip":
      case "remove":
        if (!item || !data.owned.includes(item.key))
          throw Error("Acquista prima questo accessorio.");
        if (cmd.action === "remove") delete data.equipment[item.slot];
        else data.equipment[item.slot] = item.key;
        break;
      case "predict":
        if (!state.matches[0].open) throw Error("Pronostici chiusi.");
        if (!["3-0", "3-1", "3-2", "2-3", "1-3", "0-3"].includes(cmd.score))
          throw Error("Risultato non valido.");
        data.predictions[data.match.id] = { choice: cmd.score };
        break;
      case "privacy":
        data.public = cmd.public;
        break;
      case "invite":
        if (!data.invitation) data.invitation = crypto.randomUUID();
        break;
      case "accept":
        if (cmd.code !== data.invitation) throw Error("Invito non valido.");
        break;
      case "vote":
        if (!data.votes) data.votes = cmd.key;
        break;
      case "wave":
      case "clap":
        break;
      default:
        throw Error("Azione non disponibile.");
    }
    save();
    return snapshot();
  }
  async function debug(action, value) {
    if (!dev) throw Error("Debug disabilitato.");
    switch (action) {
      case "day":
        data.offset++;
        data.location = null;
        break;
      case "location":
        data.location =
          ((data.location ?? snapshot().treasure.location) + 1) % 10;
        break;
      case "reset-claim":
        data.claims = data.claims.filter((d) => d !== snapshot().treasure.day);
        break;
      case "coins":
        data.wallet += 100;
        data.points += 100;
        break;
      case "unlock":
        if (!data.owned.includes(value)) data.owned.push(value);
        break;
      case "match":
        data.match = {
          ...data.match,
          id: data.match.id + 1,
          starts: new Date(date().getTime() + 86400000 * 3).toISOString(),
          score: "",
        };
        break;
      case "result":
        if (!["3-0", "3-1", "3-2", "2-3", "1-3", "0-3"].includes(value))
          throw Error("Risultato non valido");
        data.match.score = value;
        data.match.starts = new Date(date().getTime() - 1000).toISOString();
        {
          const p = data.predictions[data.match.id];
          if (p && p.reward == null) {
            p.reward =
              p.choice === value
                ? 50
                : (p.choice[0] === "3") === (value[0] === "3")
                  ? 15
                  : 0;
            data.points += p.reward;
          }
        }
        break;
      case "friend":
        data.friends.push({
          id: 1000 + data.friends.length,
          nickname: "AmicoDemo" + (data.friends.length + 1),
          slot: 70 + data.friends.length,
          public: true,
        });
        break;
      case "trophy":
        data.trophies[value] = date().toISOString();
        break;
      case "reset":
        data = initial();
        break;
      default:
        throw Error("Debug non valido");
    }
    save();
    return snapshot();
  }
  return {
    getState: async () => snapshot(),
    command,
    debug,
    getFans: async () => ({
      organization_id: 7,
      fans: [
        ...createDemoProfiles().map((p, index) => ({
          id: index + 1,
          nickname: p.nickname,
          slot: index,
          is_me: index === 0,
          public: index === 0 ? data.public : true,
          equipment: index === 0 ? data.equipment : {},
        })),
        ...data.friends,
      ],
    }),
  };
}
