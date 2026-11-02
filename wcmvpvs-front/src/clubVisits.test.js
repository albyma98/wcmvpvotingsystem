import test from 'node:test'
import assert from 'node:assert/strict'

async function fixture(t, storage) {
  t.mock.method(globalThis, 'fetch', async () => ({ ok: true }))
  const original = Object.getOwnPropertyDescriptor(globalThis, 'localStorage')
  Object.defineProperty(globalThis, 'localStorage', { configurable: true, value: storage })
  t.after(() => {
    if (original) Object.defineProperty(globalThis, 'localStorage', original)
    else delete globalThis.localStorage
  })
  return import(`./clubVisits.js?test=${crypto.randomUUID()}`)
}

test('browser identity survives reloads, each opening has its own ID, and club URLs are encoded', async t => {
  const values = new Map()
  const storage = { getItem: key => values.get(key), setItem: (key, value) => values.set(key, value) }
  const { recordClubVisit } = await fixture(t, storage)
  await recordClubVisit('aurora')
  await recordClubVisit('another club')
  const reloaded = await import(`./clubVisits.js?reload=${crypto.randomUUID()}`)
  await reloaded.recordClubVisit('aurora')
  const calls = fetch.mock.calls.map(call => call.arguments)
  assert.equal(calls[0][0], '/api/v1/clubs/aurora/visits')
  assert.equal(calls[1][0], '/api/v1/clubs/another%20club/visits')
  const bodies = calls.map(([, options]) => JSON.parse(options.body))
  assert.equal(new Set(bodies.map(body => body.visitorId)).size, 1)
  assert.equal(new Set(bodies.map(body => body.visitId)).size, 3)
  assert.equal(calls[0][1].credentials, 'omit')
  assert.equal(calls[0][1].method, 'POST')
})

test('blocked storage retains identity in memory and tracking failures are ignored', async t => {
  const { recordClubVisit } = await fixture(t, {
    getItem() { throw new Error('blocked') },
    setItem() { throw new Error('blocked') },
  })
  await recordClubVisit('aurora')
  await recordClubVisit('aurora')
  const bodies = fetch.mock.calls.map(call => JSON.parse(call.arguments[1].body))
  assert.equal(bodies[0].visitorId, bodies[1].visitorId)
  fetch.mock.mockImplementation(async () => { throw new Error('offline') })
  await assert.doesNotReject(recordClubVisit('aurora'))
})
