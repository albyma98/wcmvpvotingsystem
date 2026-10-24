import test from 'node:test'
import assert from 'node:assert/strict'
import { clubDomainRoute, clubHomePath, resolveClubDomain } from './clubDomain.js'

test('domain resolution uses the current host and never trusts a browser-supplied club identifier', async () => {
  const calls = []
  const signal = new AbortController().signal
  const domain = { clubSlug: 'aurora-volley', primaryHostname: 'auroravolley.it' }
  assert.deepEqual(await resolveClubDomain(async (...args) => {
    calls.push(args)
    return { status: 200, ok: true, json: async () => domain }
  }, signal), domain)
  assert.equal(calls[0][0], '/api/v1/club-domain')
  assert.equal(calls[0][1].credentials, 'same-origin')
  assert.equal(calls[0][1].cache, 'no-store')
  assert.equal(calls[0][1].signal, signal)
  assert.equal(await resolveClubDomain(async () => ({ status: 204 })), null)
})

test('failed or malformed domain resolution does not silently open a different public portal', async () => {
  await assert.rejects(resolveClubDomain(async () => ({ status: 503, ok: false })))
  await assert.rejects(resolveClubDomain(async () => ({ status: 200, ok: true, json: async () => ({ clubSlug: '../other', primaryHostname: 'aurora.it' }) })))
  await assert.rejects(resolveClubDomain(async () => { throw new Error('network') }))
})

test('custom domains open the club at the root and preserve team and CMS navigation', () => {
  const domain = { clubSlug: 'aurora' }
  assert.deepEqual(clubDomainRoute(domain, '/'), { view: 'club', clubSlug: 'aurora', teamSlug: '' })
  assert.deepEqual(clubDomainRoute(domain, '/team/under-18'), { view: 'club', clubSlug: 'aurora', teamSlug: 'under-18' })
  assert.equal(clubDomainRoute(domain, '/club/aurora/team/under-18').teamSlug, 'under-18')
  assert.equal(clubDomainRoute(domain, '/club/aurora').view, 'club')
  assert.equal(clubDomainRoute(domain, '/club-admin/aurora').view, 'club-admin')
  assert.equal(clubHomePath('aurora', true), '/')
  assert.equal(clubHomePath('aurora'), '/club/aurora')
  assert.equal(clubDomainRoute(null, '/club/other'), null)
})

test('custom domains cannot display another club, the Master or unrelated match-day pages', () => {
  for (const path of ['/club/other', '/club-admin/other', '/admin/master', '/another/new', '/team/%ZZ', '/unknown', '/team/one/extra']) {
    assert.equal(clubDomainRoute({ clubSlug: 'aurora' }, path).view, 'club-not-found', path)
  }
})
