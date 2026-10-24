import test from 'node:test'
import assert from 'node:assert/strict'
import { clubTypography } from './clubTypography.js'

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
