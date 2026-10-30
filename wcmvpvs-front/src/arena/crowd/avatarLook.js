export const SKIN_COLORS = [
  "#d6a27d",
  "#a97050",
  "#edc3a0",
  "#754b3b",
  "#bd8867",
  "#e4b79b",
];
export const SHIRT_COLORS = [
  "#36599c",
  "#dfbe40",
  "#e7e9de",
  "#4a727b",
  "#bd6557",
  "#78769a",
];
export const HAIR_COLORS = [
  "#382a25",
  "#674532",
  "#282b30",
  "#bf9156",
  "#865d46",
  "#d4c4a3",
];
export const HAIR_NAMES = ["Corti", "Medi", "Lunghi", "Ricci"];
export const BODY_NAMES = ["Regolare", "Morbida", "Slanciata"];
export const EXPRESSION_NAMES = ["Sorriso", "Sereno", "Entusiasta"];
export function avatarLook(actor) {
  const seed = Math.abs(Math.trunc(actor.seed || 1)),
    e = actor.equipment || {},
    a = actor.appearance || {};
  const option = (key, max, fallback) => {
    const value = a[key] ?? Number(e["look-" + key] ?? NaN);
    return Number.isInteger(value) && value >= 0 && value < max
      ? value
      : fallback;
  };
  const body = option("body", 3, seed % 3);
  return {
    body,
    width: [1, 1.12, 0.91][body],
    height: [1, 0.97, 1.055][body],
    hair: e.hair ? 4 : option("hair", 4, Math.floor(seed / 3) % 4),
    skin: SKIN_COLORS[option("skin", 6, seed % 6)],
    hairColor: HAIR_COLORS[option("hairColor", 6, Math.floor(seed / 5) % 6)],
    shirt:
      {
        "club-shirt": "#3558a0",
        "away-shirt": "#eccb43",
        hoodie: "#506776",
        "shirt-color": "#a5c85c",
      }[e.shirt] || SHIRT_COLORS[option("shirt", 6, Math.floor(seed / 7) % 6)],
    pants: e.pants ? "#293e66" : ["#2e4050", "#49566a", "#545b63"][seed % 3],
    shoes: e.shoes ? "#b2d870" : "#f0eee3",
    expression: option("expression", 3, Math.floor(seed / 11) % 3),
    equipment: e,
  };
}
