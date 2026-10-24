import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { compileScript, compileTemplate, parse } from '@vue/compiler-sfc'
import * as Vue from 'vue'
import * as VueSSR from 'vue/server-renderer'
import { clubHomeTiles } from '../clubHub.js'
import { CLUB_FONTS, clubTypography } from '../clubTypography.js'

const { descriptor } = parse(readFileSync(new URL('./ClubAdminPortal.vue', import.meta.url), 'utf8'))
const compiledScript = compileScript(descriptor, { id: 'club-cms-test' })
const script = compiledScript.content
  .replace(/^import .* from .*\r?\n/gm, '')
  .replace('export default', 'return')
const createComponent = new Function('Vue', 'clubHomeTiles', 'CLUB_FONTS', 'clubTypography', 'ClubTileIconUpload', 'ClubMediaField', 'fetch', 'setTimeout', 'onMounted', 'onUnmounted', 'confirm',
  `const { computed, reactive, ref, nextTick } = Vue;\n${script}`)

const template = compileTemplate({ source: descriptor.template.content, id: 'club-cms-test', ssr: true, ssrCssVars: [], compilerOptions: { bindingMetadata: compiledScript.bindings } })
assert.deepEqual(template.errors, [])
const helpers = {}
const renderCode = template.code.replace(/^import \{([^}]+)\} from "([^"]+)"\r?$/gm, (_, imports, module) => {
  for (const item of imports.split(',')) {
    const [name, alias = name] = item.trim().split(/\s+as\s+/)
    helpers[alias] = (module === 'vue' ? Vue : VueSSR)[name]
  }
  return ''
}).replace('export function ssrRender', 'return function ssrRender')
const render = new Function(...Object.keys(helpers), renderCode)(...Object.values(helpers))

async function fixture(t, failSave = false, config = {}) {
  let club = { name: 'Aurora', active: true, contacts: [{ name: 'Segreteria', email: 'info@aurora.it' }], tileIcons: {} }
  let sponsors = structuredClone(config.sponsors || [])
  const writes = []
  const requests = []
  const component = createComponent(Vue, clubHomeTiles, CLUB_FONTS, clubTypography, { render: () => null }, { render: () => null }, async (path, options) => {
    requests.push({ path, ...options, body: options?.body ? JSON.parse(options.body) : undefined })
    if (options?.method && config.failWrites) return { ok: false, status: 500, json: async () => ({ error: config.writeError || 'save_failed' }) }
    if (options?.method && config.waitForWrite) await config.waitForWrite
    let data = {}
    if (path.endsWith('/overview')) data = { club: structuredClone(club), teams: [{ id: 1, name: 'Under 16', active: true }], sponsors: structuredClone(sponsors), counts: { sponsors: sponsors.length } }
    if (path.endsWith('/teams')) data = { teams: [{ id: 1, name: 'Under 16', active: true }], players: config.players || {} }
    if (path.endsWith('/settings')) {
      writes.push(JSON.parse(options.body))
      if (failSave) return { ok: false, status: 400, json: async () => ({ error: 'invalid_contact_email' }) }
      club = JSON.parse(options.body)
    }
    if (path.includes('/sponsors') && options?.method) {
      const body = options.body ? JSON.parse(options.body) : null
      const id = Number(path.match(/\/sponsors\/(\d+)$/)?.[1])
      if (options.method === 'DELETE') sponsors = sponsors.filter(sponsor => sponsor.id !== id)
      else if (options.method === 'PUT') sponsors = sponsors.map(sponsor => sponsor.id === id ? { ...body, id } : sponsor)
      else { const nextID = Math.max(0, ...sponsors.map(sponsor => sponsor.id)) + 1; sponsors.push({ ...body, id: nextID }); data = { id: nextID, ok: true } }
    }
    return { ok: true, status: 200, json: async () => data }
  }, () => {}, () => {}, () => {}, () => config.confirm ?? true)
  const scope = Vue.effectScope()
  t.after(() => scope.stop())
  const state = scope.run(() => component.setup({ slug: 'aurora' }, { expose() {} }))
  await state.loadOverview()
  return { state, writes, requests }
}

test('CMS saves reordered contacts and uploaded icons, then reloads and restores defaults', async t => {
  const { state, writes } = await fixture(t)
  state.activeTab.value = 'contacts'
  state.addContact()
  Object.assign(state.settings.contacts[1], { name: 'Mario Rossi', role: 'Presidente', phone: '+39 3331234567' })
  state.moveContact(1, -1)
  state.setTileIcon('team-1', 'data:image/png;base64,team')
  state.setTileIcon('contacts', 'data:image/png;base64,contacts')
  state.setTileIcon('sponsors', 'data:image/png;base64,sponsors')
  await state.saveSettings()
  assert.deepEqual(writes[0].contacts.map(contact => contact.name), ['Mario Rossi', 'Segreteria'])
  assert.equal(state.settings.contacts[0].phone, '+39 3331234567')
  assert.equal(state.settings.tileIcons.contacts, 'data:image/png;base64,contacts')
  assert.equal(state.editableTiles.value.find(tile => tile.id === 'contacts').iconImage, 'data:image/png;base64,contacts')
  assert.equal(state.editableTiles.value.find(tile => tile.id === 'sponsors').iconImage, 'data:image/png;base64,sponsors')
  state.settings.contacts.splice(0, 1)
  state.setTileIcon('team-1', '')
  state.setTileIcon('contacts', '')
  state.setTileIcon('sponsors', '')
  await state.saveSettings()
  assert.deepEqual(writes[1].tileIcons, {})
  assert.deepEqual(state.settings.contacts.map(contact => contact.name), ['Segreteria'])
  assert.equal(state.editableTiles.value.at(-1).iconImage, '')
})

test('CMS retains edits on validation errors and waits for image processing before saving', async t => {
  const { state, writes } = await fixture(t, true)
  state.settings.contacts[0].email = 'wrong'
  state.trackImageUpload(true)
  await state.saveSettings()
  assert.equal(writes.length, 0)
  state.trackImageUpload(false)
  await state.saveSettings()
  assert.match(state.error.value, /indirizzi email/)
  assert.equal(state.settings.contacts[0].email, 'wrong')
  assert.equal(state.busy.value, false)
})

test('CMS defaults legacy fonts and persists separate title and body selections', async t => {
  const { state, writes } = await fixture(t)
  assert.equal(state.settings.titleFont, 'Inter')
  assert.equal(state.settings.bodyFont, 'Inter')
  state.settings.titleFont = 'Oswald'
  state.settings.bodyFont = 'Lato'
  assert.match(state.typography.value.style['--club-font-title'], /^"Oswald"/)
  assert.match(state.typography.value.style['--club-font-body'], /^"Lato"/)
  await state.saveSettings()
  assert.equal(writes[0].titleFont, 'Oswald')
  assert.equal(writes[0].bodyFont, 'Lato')
  assert.equal(state.settings.titleFont, 'Oswald')
  assert.equal(state.settings.bodyFont, 'Lato')
  // Saving a different tab must also retain the published typography.
  state.activeTab.value = 'contacts'
  await state.saveSettings()
  assert.equal(writes[1].titleFont, 'Oswald')
  assert.equal(writes[1].bodyFont, 'Lato')
})

test('Homepage renders both labeled font selectors with their options before identity settings', async t => {
  const { state } = await fixture(t)
  state.activeTab.value = 'settings'
  let html = ''
  render({}, chunk => { html += chunk }, null, {}, { slug: 'aurora' }, Vue.proxyRefs(state), {}, {})
  for (const id of ['club-title-font', 'club-body-font']) {
    assert.ok(html.includes(`for="${id}"`))
    const select = html.match(new RegExp(`<select id="${id}"[^>]*>([\\s\\S]*?)</select>`))
    assert.ok(select, `missing ${id}`)
    assert.match(select[1], /value="Inter" selected/)
    assert.match(select[1], /value="ENDURA" disabled/)
    assert.equal((select[1].match(/<option /g) || []).length, CLUB_FONTS.length)
  }
  assert.ok(html.indexOf('club-title-font') < html.indexOf('Identità e homepage'))
})

const mutations = requests => requests.filter(request => ['POST', 'PUT', 'DELETE'].includes(request.method))

test('Structured roster preserves compound names, isolates edits and saves the chosen order', async t => {
  const original = { firstName: 'Maria Luisa', lastName: 'De Luca', jerseyNumber: 7, role: 'Centrale', imageUrl: '/maria.jpg' }
  const { state, requests } = await fixture(t, false, { players: { 1: [original] } })
  state.openRoster({ id: 1 })
  state.rosterPlayers.value[0].role = 'Capitana'
  assert.equal(state.playersByTeam.value[1][0].role, 'Centrale')
  state.addPlayer()
  Object.assign(state.rosterPlayers.value[1], { firstName: 'Anna', lastName: 'Rossi', jerseyNumber: 0 })
  state.moveItem(state.rosterPlayers.value, 1, -1)
  state.settings.tagline = 'Homepage ancora da salvare'
  await state.saveRoster()
  const saved = mutations(requests)[0]
  assert.equal(saved.path, '/api/v1/club-admin/aurora/teams/1/players')
  assert.deepEqual(saved.body.players, [
    { firstName: 'Anna', lastName: 'Rossi', jerseyNumber: 0, role: '', imageUrl: '', position: 0 },
    { firstName: 'Maria Luisa', lastName: 'De Luca', jerseyNumber: 7, role: 'Capitana', imageUrl: '/maria.jpg', position: 1 },
  ])
  assert.equal(state.settings.tagline, 'Homepage ancora da salvare')
  assert.equal(state.busy.value, false)
})

test('Empty athletes and invalid jersey numbers are rejected without dropping roster data', async t => {
  const { state, requests } = await fixture(t)
  state.openRoster({ id: 1 })
  state.addPlayer()
  await state.saveRoster()
  assert.match(state.error.value, /nome o il cognome/)
  state.rosterPlayers.value[0].firstName = 'Anna'
  state.rosterPlayers.value[0].jerseyNumber = -1
  await state.saveRoster()
  assert.equal(mutations(requests).length, 0)
  assert.equal(state.rosterPlayers.value[0].firstName, 'Anna')
})

test('Match editor serializes numeric partials for the API, including zero and away scores', async t => {
  const { state, requests } = await fixture(t)
  Object.assign(state.matchForm, { teamId: 1, opponent: 'Ospiti', scheduledAt: '2026-10-25T18:30', home: false, status: 'completed', scoreFor: 2, scoreAgainst: 0 })
  state.addSet()
  Object.assign(state.matchForm.sets[0], { scoreFor: 25, scoreAgainst: 0 })
  state.addSet()
  Object.assign(state.matchForm.sets[1], { scoreFor: 25, scoreAgainst: 18 })
  await state.saveMatch()
  const saved = mutations(requests)[0]
  assert.equal(saved.method, 'POST')
  assert.equal(saved.body.setsJson, '["25-0","25-18"]')
  assert.equal(saved.body.home, false)
  assert.equal(saved.body.scoreFor, 2)
  assert.ok(!('sets' in saved.body))
  assert.deepEqual(state.matchForm.sets, [])
})

test('Existing matches load partials into dedicated fields and update their existing id', async t => {
  const { state, requests } = await fixture(t)
  await state.editMatch({ id: 32, teamId: 1, opponent: 'Volley', scheduledAt: '2026-10-25T18:30:00', status: 'completed', scoreFor: 1, scoreAgainst: 0, setsJson: '["25-18"]' })
  assert.equal(state.matchForm.scheduledAt, '2026-10-25T18:30')
  assert.equal(state.matchForm.sets[0].scoreAgainst, 18)
  assert.equal(state.hasUnsavedChanges.value, false)
  state.matchForm.sets[0].scoreAgainst = 20
  await state.saveMatch()
  const saved = mutations(requests)[0]
  assert.equal(saved.path, '/api/v1/club-admin/aurora/matches/32')
  assert.equal(saved.method, 'PUT')
  assert.equal(saved.body.setsJson, '["25-20"]')
})

test('Incomplete partials are rejected and malformed legacy data does not erase the current editor', async t => {
  const { state, requests } = await fixture(t)
  Object.assign(state.matchForm, { teamId: 1, opponent: 'Volley', scheduledAt: '2026-10-25T18:30', status: 'live' })
  state.addSet()
  await state.saveMatch()
  assert.equal(mutations(requests).length, 0)
  assert.match(state.error.value, /entrambi i punteggi/)
  await state.editMatch({ id: 2, opponent: 'Dati errati', setsJson: '{broken}' })
  assert.equal(state.matchForm.opponent, 'Volley')
  assert.match(state.error.value, /parziali.*non sono leggibili/)
})

test('Content and gallery edits update existing records and preserve visibility and ordering', async t => {
  const { state, requests } = await fixture(t)
  await state.editPost({ id: 12, title: 'Open day', kind: 'news', active: false, publishedAt: '2026-10-25T10:30:00Z' })
  assert.equal(state.postForm.publishedAt, '2026-10-25')
  state.postForm.title = 'Open day aggiornato'
  await state.savePost()
  await state.editGallery({ id: 24, imageUrl: '/foto.jpg', title: 'Torneo', position: 8, active: false, eventDate: '2026-10-25T00:00:00Z' })
  await state.saveGallery()
  const saved = mutations(requests)
  assert.equal(saved[0].path, '/api/v1/club-admin/aurora/posts/12')
  assert.equal(saved[0].body.title, 'Open day aggiornato')
  assert.equal(saved[0].body.active, false)
  assert.equal(saved[1].path, '/api/v1/club-admin/aurora/gallery/24')
  assert.equal(saved[1].body.position, 8)
  assert.equal(saved[1].body.eventDate, '2026-10-25')
})

test('Write errors keep data and release busy state across all content editors', async t => {
  const { state, requests } = await fixture(t, false, { failWrites: true })
  state.teamForm.name = 'Under 18'
  await state.saveTeam()
  assert.equal(state.teamForm.name, 'Under 18')
  state.postForm.title = 'News da salvare'
  await state.savePost()
  assert.equal(state.postForm.title, 'News da salvare')
  state.galleryForm.imageUrl = '/foto.jpg'
  await state.saveGallery()
  assert.equal(state.galleryForm.imageUrl, '/foto.jpg')
  assert.equal(state.busy.value, false)
  assert.match(state.error.value, /dati inseriti sono conservati/)
  assert.equal(mutations(requests).length, 3)
})

test('Pending saves block repeated submissions', async t => {
  let finishWrite
  const waitForWrite = new Promise(resolve => { finishWrite = resolve })
  const { state, requests } = await fixture(t, false, { waitForWrite })
  state.teamForm.name = 'Under 18'
  const saving = state.saveTeam()
  await state.saveTeam()
  assert.equal(state.busy.value, true)
  assert.equal(mutations(requests).length, 1)
  finishWrite()
  await saving
  assert.equal(state.busy.value, false)
})

test('Media processing blocks saving and switching records until the image is ready', async t => {
  const { state, requests } = await fixture(t)
  state.postForm.title = 'Comunicazione'
  state.trackImageUpload(true)
  await state.savePost()
  await state.editPost({ id: 4, title: 'Altro contenuto' })
  assert.equal(state.postForm.title, 'Comunicazione')
  assert.equal(mutations(requests).length, 0)
  assert.equal(state.processing.value, true)
  state.trackImageUpload(false)
  await state.savePost()
  assert.equal(mutations(requests).length, 1)
})

test('Reauthentication keeps unsaved homepage and content edits after an expired session', async t => {
  const { state } = await fixture(t)
  state.settings.tagline = 'Nuova identità'
  state.postForm.title = 'Contenuto in lavorazione'
  state.authenticated.value = false
  let prevented = false
  state.beforeUnload({ preventDefault() { prevented = true } })
  assert.equal(prevented, true)
  await state.loginSubmit()
  assert.equal(state.authenticated.value, true)
  assert.equal(state.settings.tagline, 'Nuova identità')
  assert.equal(state.postForm.title, 'Contenuto in lavorazione')
})

test('Collection filters combine text, team and publication state independently', async t => {
  const { state } = await fixture(t)
  state.overview.value.posts = [{ id: 1, title: 'Torneo regionale', teamId: 1, active: true }, { id: 2, title: 'Torneo regionale', teamId: 2, active: false }]
  Object.assign(state.filters.posts, { query: '  REGIONALE ', teamId: 1, state: 'true' })
  assert.deepEqual(state.filteredPosts.value.map(item => item.id), [1])
  state.filters.posts.state = 'false'
  assert.equal(state.filteredPosts.value.length, 0)
  state.activeTab.value = 'posts'
  state.resetFilters()
  assert.equal(state.filteredPosts.value.length, 2)
  state.overview.value.matches = [{ id: 3, opponent: 'Volley', teamId: 1, status: 'live' }, { id: 4, opponent: 'Volley', teamId: 1, status: 'scheduled' }]
  state.filters.matches.state = 'live'
  assert.deepEqual(state.filteredMatches.value.map(item => item.id), [3])
})

test('Deletion waits for explicit confirmation and clears the editor for a deleted record', async t => {
  const { state, requests } = await fixture(t)
  const item = { id: 4, title: 'Open day' }
  await state.askDelete('posts', item)
  assert.equal(mutations(requests).length, 0)
  state.cancelDelete()
  await state.confirmDelete()
  assert.equal(mutations(requests).length, 0)
  await state.editPost(item)
  await state.askDelete('posts', item)
  await state.confirmDelete()
  assert.equal(mutations(requests)[0].method, 'DELETE')
  assert.equal(mutations(requests)[0].path, '/api/v1/club-admin/aurora/posts/4')
  assert.equal(state.postForm.id, 0)
  assert.equal(state.pendingDelete.value, null)
})

test('Unsaved edits survive canceled resets and trigger the browser leave guard', async t => {
  const { state } = await fixture(t, false, { confirm: false })
  state.postForm.title = 'Da conservare'
  assert.equal(state.hasUnsavedChanges.value, true)
  state.clearEditor('post')
  assert.equal(state.postForm.title, 'Da conservare')
  await state.editPost({ id: 4, title: 'Altro contenuto' })
  assert.equal(state.postForm.title, 'Da conservare')
  let prevented = false
  const event = { preventDefault() { prevented = true } }
  state.beforeUnload(event)
  assert.equal(prevented, true)
})

test('Every collection renders a real labeled form; roster and partials expose structured fields', async t => {
  const { state } = await fixture(t)
  for (const tab of ['teams', 'posts', 'gallery', 'matches', 'contacts', 'sponsors']) {
    state.activeTab.value = tab
    let html = ''
    render({}, chunk => { html += chunk }, null, {}, { slug: 'aurora' }, Vue.proxyRefs(state), {}, {})
    assert.match(html, /<form /, tab)
    assert.match(html, /<label/, tab)
    assert.match(html, /<fieldset/, tab)
    assert.ok(!html.includes('Parziali JSON'))
  }
  state.activeTab.value = 'teams'
  state.openRoster({ id: 1 })
  state.addPlayer()
  let roster = ''
  render({}, chunk => { roster += chunk }, null, {}, { slug: 'aurora' }, Vue.proxyRefs(state), {}, {})
  assert.match(roster, />Nome<input/)
  assert.match(roster, />Cognome<input/)
  assert.ok(!roster.includes('Una persona per riga'))
  state.activeTab.value = 'matches'
  state.matchForm.status = 'completed'
  state.addSet()
  let match = ''
  render({}, chunk => { match += chunk }, null, {}, { slug: 'aurora' }, Vue.proxyRefs(state), {}, {})
  assert.match(match, /aria-label="Parziale 1, squadra del club"/)
  assert.match(match, /aria-label="Parziale 1, avversario"/)
})

test('Sponsor CMS creates, reloads, edits and deletes a sponsor without overwriting homepage drafts', async t => {
  const { state, requests } = await fixture(t)
  state.activeTab.value = 'sponsors'
  state.settings.tagline = 'Homepage in lavorazione'
  Object.assign(state.sponsorForm, { name: '  Partner del club  ', logoUrl: ' /partner.png ', website: ' https://partner.example ', description: '', position: 3, active: false })
  await state.saveSponsor()
  const created = mutations(requests)[0]
  assert.equal(created.method, 'POST')
  assert.equal(created.path, '/api/v1/club-admin/aurora/sponsors')
  assert.equal(created.body.name, 'Partner del club')
  assert.equal(created.body.logoUrl, '/partner.png')
  assert.equal(created.body.website, 'https://partner.example')
  assert.equal(created.body.description, '')
  assert.equal(created.body.position, 3)
  assert.equal(created.body.active, false)
  assert.equal(state.overview.value.sponsors.length, 1)
  assert.equal(state.overview.value.counts.sponsors, 1)
  assert.equal(state.sponsorForm.id, 0)
  assert.equal(state.sponsorForm.name, '')
  assert.equal(state.settings.tagline, 'Homepage in lavorazione')

  const sponsor = state.overview.value.sponsors[0]
  await state.editSponsor(sponsor)
  state.sponsorForm.description = '  Sostiene la prima squadra  '
  state.sponsorForm.active = true
  state.sponsorForm.position = 0
  assert.equal(sponsor.description, '')
  await state.saveSponsor()
  const updated = mutations(requests)[1]
  assert.equal(updated.method, 'PUT')
  assert.equal(updated.path, `/api/v1/club-admin/aurora/sponsors/${sponsor.id}`)
  assert.equal(updated.body.description, 'Sostiene la prima squadra')
  assert.equal(updated.body.position, 0)
  assert.equal(updated.body.active, true)
  assert.equal(state.overview.value.sponsors[0].description, updated.body.description)

  await state.editSponsor(state.overview.value.sponsors[0])
  await state.askDelete('sponsors', sponsor)
  assert.equal(mutations(requests).length, 2)
  await state.confirmDelete()
  const deleted = mutations(requests)[2]
  assert.equal(deleted.method, 'DELETE')
  assert.equal(deleted.path, `/api/v1/club-admin/aurora/sponsors/${sponsor.id}`)
  assert.equal(state.sponsorForm.id, 0)
  assert.deepEqual(state.overview.value.sponsors, [])
  assert.equal(state.pendingDelete.value, null)
  assert.equal(state.settings.tagline, 'Homepage in lavorazione')
})

test('Sponsor validation and media processing prevent invalid or premature writes', async t => {
  const { state, requests } = await fixture(t)
  await state.saveSponsor()
  assert.match(state.error.value, /nome dello sponsor/)
  state.sponsorForm.name = 'Partner'
  await state.saveSponsor()
  assert.match(state.error.value, /logo dello sponsor/)
  state.sponsorForm.logoUrl = '/logo.png'
  for (const website of ['', 'partner.example', 'javascript:alert(1)', 'https:///', 'https://user:password@partner.example', 'https://partner.example/a b', 'https://partner.example/\\path']) {
    state.sponsorForm.website = website
    await state.saveSponsor()
    assert.match(state.error.value, /sito web valido/)
  }
  state.sponsorForm.website = 'https://partner.example'
  for (const position of ['', -1, 1.5, 1000001]) {
    state.sponsorForm.position = position
    await state.saveSponsor()
    assert.match(state.error.value, /ordine dello sponsor/)
  }
  state.sponsorForm.position = 0
  state.sponsorForm.description = 'a'.repeat(3001)
  await state.saveSponsor()
  assert.match(state.error.value, /descrizione/)
  state.sponsorForm.description = ''
  state.trackImageUpload(true)
  await state.saveSponsor()
  await state.editSponsor({ id: 5, name: 'Altro partner' })
  assert.equal(state.sponsorForm.name, 'Partner')
  assert.equal(mutations(requests).length, 0)
  state.trackImageUpload(false)
  await state.saveSponsor()
  assert.equal(mutations(requests).length, 1)
})

test('Sponsor failures retain drafts, apply leave guards and expose backend validation messages', async t => {
  const { state, requests } = await fixture(t, false, { failWrites: true, writeError: 'invalid_sponsor_logo', confirm: false })
  Object.assign(state.sponsorForm, { name: 'Sponsor da conservare', logoUrl: '/logo.png', website: 'https://partner.example', description: 'Bozza', active: false })
  await state.saveSponsor()
  assert.equal(mutations(requests).length, 1)
  assert.match(state.error.value, /logo valido/)
  assert.equal(state.sponsorForm.name, 'Sponsor da conservare')
  assert.equal(state.sponsorForm.description, 'Bozza')
  assert.equal(state.busy.value, false)
  assert.equal(state.hasUnsavedChanges.value, true)
  state.clearEditor('sponsor')
  assert.equal(state.sponsorForm.name, 'Sponsor da conservare')
  await state.editSponsor({ id: 8, name: 'Altro sponsor' })
  assert.equal(state.sponsorForm.name, 'Sponsor da conservare')
  let prevented = false
  state.beforeUnload({ preventDefault() { prevented = true } })
  assert.equal(prevented, true)
})

test('Sponsor list combines search and publication filters and renders existing logos and edit actions', async t => {
  const sponsors = [
    { id: 1, name: 'Sport Partner', logoUrl: '/sport.png', website: 'https://sport.example', description: 'Prima squadra', position: 0, active: true },
    { id: 2, name: 'Market', logoUrl: '/market.png', website: 'https://market.example', description: 'Partner giovanili', position: 1, active: false },
  ]
  const { state } = await fixture(t, false, { sponsors })
  state.activeTab.value = 'sponsors'
  Object.assign(state.filters.sponsors, { query: 'partner', state: 'false' })
  assert.deepEqual(state.filteredSponsors.value.map(sponsor => sponsor.id), [2])
  state.resetFilters()
  assert.deepEqual(state.filteredSponsors.value.map(sponsor => sponsor.id), [1, 2])
  let html = ''
  render({}, chunk => { html += chunk }, null, {}, { slug: 'aurora' }, Vue.proxyRefs(state), {}, {})
  assert.match(html, /Nome sponsor \*/)
  assert.match(html, /type="url" required/)
  assert.match(html, /Descrizione \(opzionale\)/)
  assert.match(html, /Ordine di visualizzazione/)
  assert.match(html, /src="\/sport.png"/)
  assert.match(html, /aria-label="Modifica sponsor Sport Partner"/)
  assert.match(html, /aria-label="Elimina sponsor Market"/)
  assert.ok(!html.includes('<label>Squadra<select'))
})
