import test from 'node:test'
import assert from 'node:assert/strict'
import { setupSfc, testVue } from '../../test/sfcSetup.js'
import * as domain from '../clubDomain.js'
import * as club from '../clubHub.js'
import * as typography from '../clubTypography.js'

const match = { id: 31, status: 'live', home: false, revision: 1, opponent: 'Ospiti', scoreFor: 0, scoreAgainst: 1 }
const player = { id: 7, firstName: 'Maria Luisa', lastName: 'De Luca', jerseyNumber: 7, role: 'Centrale' }
const ballot = { club: { name: 'Aurora' }, team: { name: 'Under 19' }, match, players: [player] }
const response = (status, data) => ({ status, ok: status < 300, json: async () => structuredClone(data) })

function setup(t, fetch, customDomain = true, matchId = '') {
  const redirects = []
  const { state, scope } = setupSfc(new URL('./ClubMVPView.vue', import.meta.url), {
    vue: { ...testVue, onUnmounted() {} },
    '../clubDomain': domain,
    '../clubHub': { ...club, connectClubMatches: () => () => {} },
    '../clubTypography': typography,
  }, { slug: 'aurora', matchId, customDomain }, { fetch, window: { location: { replace: path => redirects.push(path) } } })
  t.after(() => scope.stop())
  return { state, redirects }
}

test('QR locks to the resolved match and confirms one successful submission', async t => {
  const requests = []
  let finish
  const pending = new Promise(resolve => { finish = resolve })
  const { state } = setup(t, async (path, options) => {
    requests.push([path, options])
    if (options.method === 'POST') return pending
    return response(200, ballot)
  })
  await state.refresh()
  assert.equal(requests[0][0], '/api/v1/clubs/aurora/mvp')
  assert.deepEqual(state.sides.value.map(side => side.name), ['Ospiti', 'Aurora'])
  state.selectedId.value = player.id
  const vote = state.vote()
  await state.vote()
  assert.equal(state.sending.value, true)
  assert.equal(state.confirmed.value, false)
  assert.equal(requests.length, 2)
  assert.equal(requests[1][0], '/api/v1/clubs/aurora/matches/31/mvp/vote')
  assert.deepEqual(JSON.parse(requests[1][1].body), { playerId: 7 })
  finish(response(201, { voted: player }))
  await vote
  assert.equal(state.confirmed.value, true)
  assert.deepEqual(state.data.value.voted, player)
  state.onScore({ match: { ...match, revision: 2, scoreFor: 1, scoreAgainst: 2 } })
  assert.equal(club.clubScoreLabel(state.match.value), '2 – 1')
  await state.refresh()
  assert.equal(requests[2][0], '/api/v1/clubs/aurora/matches/31/mvp')
  assert.deepEqual(state.data.value.voted, player)
  assert.equal(state.match.value.revision, 2)
})

test('a duplicate response displays the previous player and a removed call-up clears the selection', async t => {
  let get = ballot
  let post = response(422, { error: 'invalid_player' })
  const other = { ...player, id: 9 }
  const { state } = setup(t, async (path, options) => options.method ? post : response(200, get))
  await state.refresh()
  state.selectedId.value = player.id
  get = { ...ballot, players: [other] }
  await state.vote()
  await state.refresh()
  assert.equal(state.selectedId.value, null)
  assert.match(state.error.value, /convocati sono cambiati/)
  post = response(409, { error: 'already_voted', voted: player })
  state.selectedId.value = other.id
  await state.vote()
  assert.deepEqual(state.data.value.voted, player)
  assert.equal(state.confirmed.value, false)
})

test('closed or unavailable ballots replace the URL with the appropriate ClubHub home', async t => {
  const opened = setup(t, async () => response(200, ballot))
  await opened.state.refresh()
  opened.state.onScore({ match: { ...match, revision: 2, status: 'completed' } })
  assert.deepEqual(opened.redirects, ['/'])

  const unavailable = setup(t, async () => response(409, { error: 'mvp_closed' }), false, '31')
  await unavailable.state.refresh()
  assert.deepEqual(unavailable.redirects, ['/club/aurora'])

  const inFlight = setup(t, async (path, options) =>
    options.method ? response(409, { error: 'mvp_closed' }) : response(200, ballot))
  await inFlight.state.refresh()
  inFlight.state.selectedId.value = player.id
  await inFlight.state.vote()
  assert.deepEqual(inFlight.redirects, ['/'])
  assert.equal(inFlight.state.confirmed.value, false)
})
