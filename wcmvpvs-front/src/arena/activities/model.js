export const TREASURE_SPOTS = [
  {
    position: [-12.5, 0.65, -7.6],
    hint: "Cerca vicino alla ringhiera, verso l’ingresso ovest.",
  },
  {
    position: [10.9, 1.55, -10.1],
    hint: "Un bagliore sull’angolo basso della tribuna senza seggiolini.",
  },
  {
    position: [-10.5, 3.4, 12.7],
    hint: "Tra giallo e blu, a metà della gradinata.",
  },
  {
    position: [13.8, 0.65, 6.5],
    hint: "Il bordo campo custodisce una sorpresa.",
  },
  {
    position: [-16.2, 0.7, -5.5],
    hint: "Una delle porte blu nasconde un indizio.",
  },
  {
    position: [7.8, 4.9, -14.5],
    hint: "Sali con lo sguardo sulla tribuna chiara.",
  },
  {
    position: [0.5, 0.75, 7.8],
    hint: "Guarda dietro la ringhiera sul lato dei seggiolini.",
  },
  { position: [10.6, 5.2, 14.9], hint: "In alto, nell’angolo giallo e blu." },
  {
    position: [-8.1, 2.2, -11],
    hint: "Segui le scale arancioni della tribuna chiara.",
  },
  {
    position: [15.4, 0.7, -8.2],
    hint: "Esplora il fondo del palazzetto, vicino all’ingresso est.",
  },
];
export const TREASURE_NAMES = [
  "Pallone dorato",
  "Monetina speciale",
  "Mini trofeo",
  "Sciarpa luminosa",
];
export const CATALOG = [
  ["club-shirt", "Maglia Joy Volley", "shirt", 20, 1, "always"],
  ["away-shirt", "Maglia alternativa", "shirt", 25, 1, "event"],
  ["hoodie", "Felpa", "shirt", 35, 1, "weekly"],
  ["sport-pants", "Pantaloni sportivi", "pants", 15, 1, "always"],
  ["scarf", "Sciarpa", "scarf", 20, 1, "always"],
  ["cap", "Cappellino", "hat", 20, 1, "always"],
  ["glasses", "Occhiali", "glasses", 30, 1, "always"],
  ["flag", "Bandierina", "flag", 35, 1, "weekly"],
  ["shoes", "Scarpe speciali", "shoes", 25, 1, "always"],
  ["hair", "Acconciatura alternativa", "hair", 20, 1, "always"],
  ["shirt-color", "Maglia verde lime", "shirt", 30, 2, "level"],
  ["celebration", "Esultanza speciale", "celebration", 40, 2, "level"],
].map(([key, title, slot, cost, level, availability]) => ({
  key,
  title,
  slot,
  cost,
  level,
  availability,
}));
export const ACTIVITIES = [
  {
    key: "treasure",
    icon: "pin",
    title: "Caccia al tesoro",
    description: "Esplora il palazzetto e trova il bagliore di oggi.",
  },
  {
    key: "wardrobe",
    icon: "fans",
    title: "Il mio guardaroba",
    description: "Prova i tuoi colori, indossa la tua passione.",
  },
  {
    key: "predictions",
    icon: "arena",
    title: "Pronostici",
    description: "La prossima sfida comincia dal tabellone.",
  },
  {
    key: "neighbors",
    icon: "fans",
    title: "I miei vicini",
    description: "La tua community, un posto alla volta.",
  },
  {
    key: "museum",
    icon: "trophy",
    title: "Museo",
    description: "Le conquiste che restano nella nostra storia.",
  },
];
export function dayInRome(date = new Date()) {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: "Europe/Rome",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(date);
}
export function closestNeighbors(actors, me) {
  if (!me) return [];
  return actors
    .filter((a) => a.id !== me.id && a.seat.stand === me.seat.stand)
    .map((a) => ({
      actor: a,
      d: Math.hypot(a.seat.x - me.seat.x, (a.seat.row - me.seat.row) * 0.6),
    }))
    .filter((x) => x.d < 3.6)
    .sort((a, b) => a.d - b.d)
    .map((x) => x.actor);
}
export function itemAvailable(item, state) {
  return (
    state.level >= item.level &&
    (item.availability!=='event'||state.matches.some(m=>new Date(m.starts)>=new Date(state.now)&&new Date(m.starts)<=new Date(new Date(state.now).getTime()+7*86400000)))&&
    (item.availability !== "weekly" || state.featured.includes(item.key))
  );
}
export function weeklyFeatured(now){const d=new Date(dayInRome(now)+'T00:00:00Z');d.setUTCDate(d.getUTCDate()+4-(d.getUTCDay()||7));const year=d.getUTCFullYear();const week=Math.ceil((((d-new Date(Date.UTC(year,0,1)))/86400000)+1)/7);return (year+week)%2===0?['hoodie','scarf','shoes']:['flag','cap','away-shirt']}
