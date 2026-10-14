export const DEFAULT_ORGANIZATION_COLORS = Object.freeze({
  primary_color: '#e4212e',
  secondary_color: '#f5b321',
  tertiary_color: '#fff7f2',
});

export const isHexColor = (value) => /^#[0-9a-f]{6}$/i.test(value || '');

export function normalizeOrganizationColors(colors = {}) {
  return Object.fromEntries(Object.entries(DEFAULT_ORGANIZATION_COLORS).map(([key, fallback]) =>
    [key, isHexColor(colors?.[key]) ? colors[key].toLowerCase() : fallback],
  ));
}

function rgb(hex) {
  return [1, 3, 5].map((offset) => parseInt(hex.slice(offset, offset + 2), 16));
}

function mix(color, other, amount) {
  const target = rgb(other);
  return `#${rgb(color).map((channel, index) =>
    Math.round(channel * (1 - amount) + target[index] * amount).toString(16).padStart(2, '0'),
  ).join('')}`;
}

function luminance(color) {
  const channels = rgb(color).map((channel) => {
    const value = channel / 255;
    return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
  });
  return channels[0] * 0.2126 + channels[1] * 0.7152 + channels[2] * 0.0722;
}

function foreground(color) {
  const light = luminance(color);
  if ((light + 0.05) / (luminance('#23161a') + 0.05) >= 4.5) return '#23161a';
  return light > 0.179 ? '#000000' : '#ffffff';
}

function readableAccent(color, background) {
  const target = luminance(background) > 0.179 ? '#000000' : '#ffffff';
  for (let step = 0; step <= 10; step++) {
    const candidate = mix(color, target, step / 10);
    const light = luminance(candidate);
    const base = luminance(background);
    if ((Math.max(light, base) + 0.05) / (Math.min(light, base) + 0.05) >= 4.5) return candidate;
  }
  return target;
}

export function organizationThemeStyle(colors) {
  const { primary_color: primary, secondary_color: secondary, tertiary_color: background } = normalizeOrganizationColors(colors);
  const ink = foreground(background);
  const card = ink === '#ffffff' ? mix(background, '#000000', 0.04) : mix(background, '#ffffff', 0.5);
  return {
    '--bg': background,
    '--outer-bg': mix(background, '#000000', 0.04),
    '--card': card,
    '--ink': ink,
    '--muted': mix(ink, background, 0.3),
    '--line': mix(background, ink, 0.12),
    '--red': primary,
    '--red-text': readableAccent(primary, card),
    '--red-deep': mix(primary, '#000000', 0.18),
    '--red-soft': mix(card, primary, 0.12),
    '--red-line': mix(card, primary, 0.25),
    '--on-primary': foreground(primary),
    '--gold': secondary,
    '--gold-deep': readableAccent(secondary, card),
    '--gold-soft': mix(card, secondary, 0.16),
    '--on-secondary': foreground(secondary),
    '--hero-glow': mix(background, primary, 0.12),
  };
}
