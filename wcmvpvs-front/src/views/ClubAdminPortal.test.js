import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { compileScript, parse } from '@vue/compiler-sfc'
import * as Vue from 'vue'
import { clubHomeTiles } from '../clubHub.js'

const { descriptor } = parse(readFileSync(new URL('./ClubAdminPortal.vue', import.meta.url), 'utf8'))
const script = compileScript(descriptor, { id: 'club-cms-test' }).content
  .replace(/^import .* from .*\r?\n/gm, '')
  .replace('export default', 'return')
const createComponent = new Function('Vue', 'clubHomeTiles', 'ClubTileIconUpload', 'fetch', 'setTimeout', 'onMounted',
  `const { computed, reactive, ref } = Vue;\n${script}`)

async function fixture(t, failSave = false) {
  let club = { name: 'Aurora', active: true, contacts: [{ name: 'Segreteria', email: 'info@aurora.it' }], tileIcons: {} }
  const writes = []
  const component = createComponent(Vue, clubHomeTiles, {}, async (path, options) => {
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
  await state.saveSettings()
  assert.deepEqual(writes[0].contacts.map(contact => contact.name), ['Mario Rossi', 'Segreteria'])
  assert.equal(state.settings.contacts[0].phone, '+39 3331234567')
  assert.equal(state.settings.tileIcons.contacts, 'data:image/png;base64,contacts')
  assert.equal(state.editableTiles.value.at(-1).iconImage, 'data:image/png;base64,contacts')
  state.settings.contacts.splice(0, 1)
  state.setTileIcon('team-1', '')
  state.setTileIcon('contacts', '')
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
