import test from 'node:test'
import assert from 'node:assert/strict'
import { clubContactPhoneLink, clubHomeTiles, sortClubMatches } from './clubHub.js'

test('home and CMS share eight tiles, with contacts and sponsors side by side', () => {
  const teams = Array.from({ length: 5 }, (_, i) => ({ id: i + 1, name: `Squadra ${i + 1}`, logoUrl: `/logo-${i + 1}.png` }))
  const tiles = clubHomeTiles({ teams, club: { contacts: [{ name: 'Segreteria' }], tileIcons: { 'team-1': 'custom-team', matches: 'custom-matches', announcements: 'custom-announcements', contacts: 'custom-contacts', sponsors: 'custom-sponsors' } } })
  assert.deepEqual(tiles.map(tile => tile.id), ['team-1', 'team-2', 'team-3', 'team-4', 'matches', 'announcements', 'contacts', 'sponsors'])
  assert.ok(tiles.every(tile => !tile.wide))
  assert.equal(tiles[6].panel, 'contacts')
  assert.equal(tiles[6].sub, '1 contatto della società')
  assert.equal(tiles[7].panel, 'sponsors')
  assert.equal(tiles[7].label, 'I nostri Sponsor')
  assert.equal(tiles[0].iconImage, 'custom-team')
  assert.equal(tiles[1].iconImage, '/logo-2.png')
  assert.equal(tiles[4].iconImage, 'custom-matches')
  assert.equal(tiles[5].iconImage, 'custom-announcements')
  assert.equal(tiles[6].iconImage, 'custom-contacts')
  assert.equal(tiles[7].iconImage, 'custom-sponsors')
  const defaults = clubHomeTiles({ teams })
  assert.equal(defaults[0].iconImage, '/logo-1.png')
  assert.equal(defaults.at(-1).iconImage, '')
  assert.ok(clubHomeTiles().find(tile => tile.id === 'contacts'))
})

test('telephone links retain international dialing and remove formatting', () => {
  assert.equal(clubContactPhoneLink('+39 (012) 345-6789'), 'tel:+390123456789')
  assert.equal(clubContactPhoneLink('Numero in aggiornamento'), '')
})

test('match calendars sort by actual date with undated matches last without changing the payload', () => {
  const matches = Object.freeze([
    { id: 4, scheduledAt: '2026-11-10T18:00:00Z' },
    { id: 5, scheduledAt: '' },
    { id: 2, scheduledAt: '2026-10-29T20:00:00+02:00' },
    { id: 3, scheduledAt: '2026-10-29T19:00:00Z' },
    { id: 1, scheduledAt: '2026-10-20T18:00:00Z' },
    { id: 6, scheduledAt: 'Data da definire' },
  ])
  assert.deepEqual(sortClubMatches(matches).map(match => match.id), [1, 2, 3, 4, 5, 6])
  assert.deepEqual(matches.map(match => match.id), [4, 5, 2, 3, 1, 6])
  assert.deepEqual(sortClubMatches(), [])
})
