import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { compileScript, compileTemplate, parse } from '@vue/compiler-sfc'
import * as Vue from 'vue'
import * as VueSSR from 'vue/server-renderer'
import * as clubHub from '../clubHub.js'
import * as organizationTheme from '../organizationTheme.js'
import * as clubTypography from '../clubTypography.js'
import * as clubDomain from '../clubDomain.js'

const { descriptor } = parse(readFileSync(new URL('./ClubHomeView.vue', import.meta.url), 'utf8'))
const script = compileScript(descriptor, { id: 'club-home-test' }).content
  .replace(/^import\s+([\s\S]*?)\s+from\s+(['"])([^'"]+)\2;?\r?\n/gm, (_, binding, quote, name) =>
    `const ${binding.trim().replace(/\s+as\s+/g, ': ')} = imports[${JSON.stringify(name)}];\n`)
  .replace('export default', 'return')
const createComponent = new Function('imports', 'fetch', script)
const template = compileTemplate({ source: descriptor.template.content, id: 'club-home-test', ssr: true, ssrCssVars: [] })
assert.deepEqual(template.errors, [])
const helpers = {}
const renderCode = template.code.replace(/^import \{([^}]+)\} from "([^"]+)"\r?$/gm, (_, bindings, module) => {
  for (const binding of bindings.split(',')) {
    const [name, alias = name] = binding.trim().split(/\s+as\s+/)
    helpers[alias] = (module === 'vue' ? Vue : VueSSR)[name]
  }
  return ''
}).replace('export function ssrRender', 'return function ssrRender')
const ssrRender = new Function(...Object.keys(helpers), renderCode)(...Object.values(helpers))

async function fixture(t, teamSlug = '') {
  const payload = {
    club: { name: 'Club Aurora', titleFont: 'Arial', bodyFont: 'Arial' },
    team: { name: 'Under 18' },
    matches: [
      { id: 3, scheduledAt: '2026-11-15T18:00:00Z', status: 'scheduled', opponent: 'Ospiti novembre', location: 'Pala Aurora' },
      { id: 1, scheduledAt: '2026-10-20T18:00:00Z', status: 'completed', opponent: 'Ospiti ottobre' },
      { id: 2, scheduledAt: '2026-10-31T18:00:00Z', status: 'scheduled', opponent: 'Prossimi ospiti', location: 'Palazzetto comunale' },
    ],
  }
  const component = createComponent({
    vue: { ...Vue, onMounted() {} },
    '../clubHub': clubHub,
    '../organizationTheme': organizationTheme,
    '../clubTypography': clubTypography,
    '../clubDomain': clubDomain,
    '../components/ClubContentModal.vue': { render: () => null },
    '../components/ClubGalleryLightbox.vue': { render: () => null },
  }, async () => ({ ok: true, json: async () => payload }))
  const props = Vue.reactive({ slug: 'aurora', teamSlug, customDomain: false })
  const scope = Vue.effectScope()
  t.after(() => scope.stop())
  const state = scope.run(() => component.setup(props, { expose() {} }))
  await state.load()
  const html = () => VueSSR.renderToString(Vue.createSSRApp({
    components: { ClubContentModal: { render: () => null }, ClubGalleryLightbox: { render: () => null } },
    setup: () => ({ ...Vue.proxyRefs(state), ...props }),
    ssrRender,
  }))
  return { state, html }
}

test('category page renders fixtures chronologically even when the server returns unordered dates', async t => {
  const { state, html } = await fixture(t, 'under-18')
  const rendered = await html()
  assert.ok(rendered.indexOf('Ospiti ottobre') < rendered.indexOf('Prossimi ospiti'))
  assert.ok(rendered.indexOf('Prossimi ospiti') < rendered.indexOf('Ospiti novembre'))
  assert.deepEqual(state.data.value.matches.map(match => match.id), [3, 1, 2])
})

test('home features the nearest scheduled match with only the venue in its details line', async t => {
  const { state, html } = await fixture(t)
  assert.equal(state.featuredMatch.value.id, 2)
  const rendered = await html()
  assert.match(rendered, /PROSSIMA PARTITA/)
  assert.match(rendered, /<small>Palazzetto comunale<\/small>/)
  assert.doesNotMatch(rendered, /Club Aurora · Palazzetto comunale/)
  state.data.value.matches[0].status = 'live'
  assert.equal(state.featuredMatch.value.id, 3)
  state.data.value.matches.forEach(match => { match.status = 'completed' })
  assert.equal(state.featuredMatch.value.id, 3)
})
