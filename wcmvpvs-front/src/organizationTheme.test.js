import test from 'node:test';
import assert from 'node:assert/strict';
import { DEFAULT_ORGANIZATION_COLORS, normalizeOrganizationColors, organizationThemeStyle } from './organizationTheme.js';

function luminance(hex) {
  const channels = [1, 3, 5].map((offset) => {
    const value = parseInt(hex.slice(offset, offset + 2), 16) / 255;
    return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
  });
  return channels[0] * 0.2126 + channels[1] * 0.7152 + channels[2] * 0.0722;
}

function contrast(first, second) {
  const a = luminance(first), b = luminance(second);
  return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05);
}

test('invalid or missing public colors fall back independently', () => {
  assert.deepEqual(normalizeOrganizationColors(null), DEFAULT_ORGANIZATION_COLORS);
  assert.deepEqual(normalizeOrganizationColors({ primary_color: '#ABCDEF', secondary_color: 'red', tertiary_color: '#123' }), {
    ...DEFAULT_ORGANIZATION_COLORS, primary_color: '#abcdef',
  });
});

test('light, dark and midtone palettes retain readable button and surface text', () => {
  for (const primary of ['#ffffff', '#000000', '#777777', '#ffff00', '#0000ff', '#e4212e']) {
    for (const background of ['#ffffff', '#000000', '#707070', '#777777', '#999999', '#fff7f2']) {
      const style = organizationThemeStyle({ primary_color: primary, secondary_color: primary, tertiary_color: background });
      assert.equal(style['--red'], primary);
      assert.equal(style['--gold'], primary);
      assert.equal(style['--bg'], background);
      for (const [text, surface] of [
        ['--on-primary', '--red'], ['--on-secondary', '--gold'],
        ['--ink', '--bg'], ['--ink', '--card'], ['--red-text', '--card'], ['--gold-deep', '--card'],
      ]) {
        assert.ok(contrast(style[text], style[surface]) >= 4.5, `${primary}/${background}: ${text} on ${surface}`);
      }
    }
  }
});
