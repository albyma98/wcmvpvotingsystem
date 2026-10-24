import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { compileScript, compileTemplate, parse } from '@vue/compiler-sfc'
import * as Vue from 'vue'
import * as VueSSR from 'vue/server-renderer'

const { descriptor } = parse(readFileSync(new URL('./MasterClubDomains.vue', import.meta.url), 'utf8'))
const compiledScript = compileScript(descriptor, { id: 'club-domains-test' })
const source = compiledScript.content.replace(/^import .* from .*\r?\n/gm, '').replace('export default', 'return')
const createComponent = new Function('Vue', 'fetch', 'onMounted', 'onBeforeUnmount', 'window', 'navigator', `const { computed, ref } = Vue;\n${source}`)
const template = compileTemplate({ source: descriptor.template.content, id: 'club-domains-test', ssr: true, ssrCssVars: [], compilerOptions: { bindingMetadata: compiledScript.bindings } })
assert.deepEqual(template.errors, [])
const helpers = {}
const renderCode = template.code.replace(/^import \{([^}]+)\} from "([^"]+)"\r?$/gm, (_, imports, module) => {
  for (const binding of imports.split(',')) {
    const [name, alias = name] = binding.trim().split(/\s+as\s+/)
    helpers[alias] = (module === 'vue' ? Vue : VueSSR)[name]
  }
  return ''
}).replace('export function ssrRender', 'return function ssrRender')
const render = new Function(...Object.keys(helpers), renderCode)(...Object.values(helpers))

function fixture(t, fetcher) {
  const props = { club: { id: 4, name: 'Aurora' } }
  const component = createComponent(Vue, fetcher, () => {}, () => {}, { confirm: () => true }, { clipboard: { writeText() {} } })
  const scope = Vue.effectScope()
  t.after(() => scope.stop())
  const state = scope.run(() => component.setup(props, { expose() {} }))
  return { state, html() { let html = ''; render({}, chunk => { html += chunk }, null, {}, props, Vue.proxyRefs(state), {}, {}); return html } }
}
const domains = [
  { id: 1, hostname: 'aurora.it', primary: true, status: 'pending', verificationName: '_clubhub.aurora.it', verificationValue: 'clubhub-verification=abc', message: 'Record TXT in attesa.' },
  { id: 2, hostname: 'www.aurora.it', primary: false, status: 'pending', verificationName: '_clubhub.aurora.it', verificationValue: 'clubhub-verification=abc', message: 'Record TXT in attesa.' },
]
const response = data => ({ ok: true, json: async () => structuredClone(data) })

test('Master associates the domain and www alias, keeps failed edits and renders one shared TXT record', async t => {
  const calls = []
  let fail = false
  const { state, html } = fixture(t, async (path, options) => {
    calls.push([path, options])
    if (fail) return { ok: false, json: async () => ({ error: 'domain_taken' }) }
    return response({ domains, target: 'mvp.wearingcash.it', targetIPs: ['203.0.113.10'] })
  })
  await state.load()
  state.hostname.value = '  aurora.it  '
  await state.add()
  assert.equal(calls[1][0], '/api/admin/master/clubs/4/domains')
  assert.equal(calls[1][1].method, 'POST')
  assert.equal(calls[1][1].credentials, 'include')
  assert.deepEqual(JSON.parse(calls[1][1].body), { hostname: 'aurora.it', includeWWW: true })
  assert.equal(state.hostname.value, '')
  const rendered = html()
  assert.equal((rendered.match(/class="dns-record"/g) || []).length, 1)
  assert.match(rendered, /_clubhub.aurora.it/)
  assert.match(rendered, /203.0.113.10/)
  assert.match(rendered, /In attesa dei DNS/)
  fail = true
  state.hostname.value = 'duplicate.it'
  await state.add()
  assert.equal(state.hostname.value, 'duplicate.it')
  assert.match(state.error.value, /già associato/)
})

test('verification, primary selection and unlink actions use the selected club and refresh the visible state', async t => {
  const calls = []
  const { state, html } = fixture(t, async (path, options) => {
    calls.push([path, options])
    return response({ domains: options.method === 'DELETE' ? [] : domains.map(domain => ({ ...domain, status: 'active' })), target: 'mvp.wearingcash.it' })
  })
  await state.load()
  calls.length = 0
  await state.change('POST', '/verify')
  assert.match(html(), /Autorizzato/)
  assert.match(html(), /href="https:\/\/aurora.it"/)
  await state.change('PUT', '/2/primary')
  await state.remove(domains[1])
  assert.deepEqual(calls.map(([path, options]) => [path, options.method]), [
    ['/api/admin/master/clubs/4/domains/verify', 'POST'],
    ['/api/admin/master/clubs/4/domains/2/primary', 'PUT'],
    ['/api/admin/master/clubs/4/domains/2', 'DELETE'],
  ])
  assert.deepEqual(state.domains.value, [])
})

test('a stale background refresh cannot overwrite a newer domain association', async t => {
  let finishLoad
  const { state } = fixture(t, async (path, options) => {
    if (!options.method) return new Promise(resolve => { finishLoad = resolve })
    return response({ domains, target: 'mvp.wearingcash.it' })
  })
  const pending = state.load()
  state.hostname.value = 'aurora.it'
  await state.add()
  finishLoad(response({ domains: [] }))
  await pending
  assert.equal(state.domains.value.length, 2)
})
