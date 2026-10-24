import test from 'node:test'
import assert from 'node:assert/strict'
import { clubContactPhoneLink, clubHomeTiles } from './clubHub.js'

test('home and CMS share the same tiles, with contacts spanning the former News and Gallery', () => {
  const teams = Array.from({ length: 5 }, (_, i) => ({ id: i + 1, name: `Squadra ${i + 1}`, logoUrl: `/logo-${i + 1}.png` }))
  const tiles = clubHomeTiles({ teams, club: { contacts: [{ name: 'Segreteria' }], tileIcons: { 'team-1': 'custom-team', matches: 'custom-matches', announcements: 'custom-announcements', contacts: 'custom-contacts' } } })
  assert.deepEqual(tiles.map(tile => tile.id), ['team-1', 'team-2', 'team-3', 'team-4', 'matches', 'announcements', 'contacts'])
  assert.equal(tiles.at(-1).wide, true)
  assert.equal(tiles.at(-1).panel, 'contacts')
  assert.equal(tiles.at(-1).sub, '1 contatto della società')
  assert.equal(tiles[0].iconImage, 'custom-team')
  assert.equal(tiles[1].iconImage, '/logo-2.png')
  assert.equal(tiles[4].iconImage, 'custom-matches')
  assert.equal(tiles[5].iconImage, 'custom-announcements')
  assert.equal(tiles[6].iconImage, 'custom-contacts')
  const defaults = clubHomeTiles({ teams })
  assert.equal(defaults[0].iconImage, '/logo-1.png')
  assert.equal(defaults.at(-1).iconImage, '')
  assert.ok(clubHomeTiles().find(tile => tile.id === 'contacts'))
})

test('telephone links retain international dialing and remove formatting', () => {
  assert.equal(clubContactPhoneLink('+39 (012) 345-6789'), 'tel:+390123456789')
  assert.equal(clubContactPhoneLink('Numero in aggiornamento'), '')
})
