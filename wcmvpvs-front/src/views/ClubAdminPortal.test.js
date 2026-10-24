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
const createComponent = new Function('Vue', 'clubHomeTiles', 'CLUB_FONTS', 'clubTypography', 'ClubTileIconUpload', 'fetch', 'setTimeout', 'onMounted',
  `const { computed, reactive, ref } = Vue;\n${script}`)

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

async function fixture(t, failSave = false) {
  let club = { name: 'Aurora', active: true, contacts: [{ name: 'Segreteria', email: 'info@aurora.it' }], tileIcons: {} }
  const writes = []
  const component = createComponent(Vue, clubHomeTiles, CLUB_FONTS, clubTypography, { render: () => null }, async (path, options) => {
    let data = {}
    if (path.endsWith('/overview')) data = { club: structuredClone(club), teams: [{ id: 1, name: 'Under 16', active: true }] }
    if (path.endsWith('/teams')) data = { teams: [{ id: 1, name: 'Under 16', active: true }], players: {} }
    if (path.endsWith('/settings')) {
      writes.push(JSON.parse(options.body))
      if (failSave) return { ok: false, status: 400, json: async () => ({ error: 'invalid_contact_email' }) }
      club = JSON.parse(options.body)
    }
    return { ok: true, status: 200, json: async () => data }
  }, () => {}, () => {})
  const scope = Vue.effectScope()
  t.after(() => scope.stop())
  const state = scope.run(() => component.setup({ slug: 'aurora' }, { expose() {} }))
  await state.loadOverview()
  return { state, writes }
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
  state.trackIconUpload(true)
  await state.saveSettings()
  assert.equal(writes.length, 0)
  state.trackIconUpload(false)
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
