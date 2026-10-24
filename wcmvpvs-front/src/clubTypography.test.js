import test from 'node:test'
import assert from 'node:assert/strict'
import { CLUB_FONTS, clubTypography } from './clubTypography.js'

test('legacy and unsupported font settings fall back safely to Inter', () => {
  const defaults = clubTypography()
  assert.match(defaults.style['--club-font-title'], /^"Inter"/)
  assert.match(defaults.style['--club-font-body'], /^"Inter"/)
  assert.equal((defaults.stylesheet.match(/family=/g) || []).length, 1)
  assert.deepEqual(clubTypography({ titleFont: 'unknown', bodyFont: '"; color:red' }), defaults)
})

test('title and body fonts load independently with only the selected web families', () => {
  const result = clubTypography({ titleFont: 'Barlow Condensed', bodyFont: 'Lato' })
  assert.match(result.style['--club-font-title'], /^"Barlow Condensed"/)
  assert.match(result.style['--club-font-body'], /^"Lato"/)
  const params = new URL(result.stylesheet).searchParams
  assert.deepEqual(params.getAll('family'), ['Barlow Condensed:wght@400;500;600;700;800;900', 'Lato:wght@400;700;900'])
  assert.equal(params.get('display'), 'swap')
  assert.equal(clubTypography({ titleFont: 'Georgia', bodyFont: 'Arial' }).stylesheet, '')
})

test('ENDURA uses its licensed stylesheet alongside Google Fonts without duplicate downloads', () => {
  const fonts = CLUB_FONTS.map(font => font.name === 'ENDURA'
    ? { ...font, disabled: false, stylesheet: 'https://fonts.example.test/endura.css' }
    : font)
  const mixed = clubTypography({ titleFont: 'ENDURA', bodyFont: 'Inter' }, fonts)
  assert.match(mixed.style['--club-font-title'], /^"ENDURA"/)
  assert.match(mixed.style['--club-font-body'], /^"Inter"/)
  assert.equal(mixed.stylesheets.length, 2)
  assert.ok(mixed.stylesheets.includes('https://fonts.example.test/endura.css'))
  assert.ok(!mixed.stylesheet.includes('ENDURA'))
  const same = clubTypography({ titleFont: 'ENDURA', bodyFont: 'ENDURA' }, fonts)
  assert.deepEqual(same.stylesheets, ['https://fonts.example.test/endura.css'])
  const unavailable = fonts.map(font => font.name === 'ENDURA' ? { ...font, disabled: true } : font)
  assert.match(clubTypography({ titleFont: 'ENDURA' }, unavailable).style['--club-font-title'], /^"Inter"/)
})
