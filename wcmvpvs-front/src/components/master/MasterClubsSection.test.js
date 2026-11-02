import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { compileScript, compileTemplate, parse } from '@vue/compiler-sfc'
import * as Vue from 'vue'
import * as VueSSR from 'vue/server-renderer'

const { descriptor } = parse(readFileSync(new URL('./MasterClubsSection.vue', import.meta.url), 'utf8'))
const script = compileScript(descriptor, { id: 'master-clubs-test' }).content
  .replace(/^import\s+([\s\S]*?)\s+from\s+(['"])([^'"]+)\2;?\r?\n/gm, (_, binding, quote, name) =>
    `const ${binding.trim().replace(/\s+as\s+/g, ': ')} = imports[${JSON.stringify(name)}];\n`)
  .replace('export default', 'return')
const createComponent = new Function('imports', 'fetch', script)
const template = compileTemplate({ source: descriptor.template.content, id: 'master-clubs-test', ssr: true, ssrCssVars: [] })
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

test('master renders per-club totals, uniques and zeros and refreshes with authenticated requests', async t => {
  const calls = []
  let totalVisits = 12345
  const component = createComponent({
    vue: { ...Vue, onMounted() {} },
    './MasterClubDomains.vue': { render: () => null },
  }, async (path, options) => {
    calls.push([path, options])
    return { ok: true, json: async () => ({ clubs: [
      { id: 1, slug: 'aurora', name: 'Aurora', totalVisits, uniqueVisits: 456 },
      { id: 2, slug: 'nuovo', name: 'Nuovo', totalVisits: 0, uniqueVisits: 0 },
    ] }) }
  })
  const scope = Vue.effectScope()
  t.after(() => scope.stop())
  const state = scope.run(() => component.setup({}, { expose() {} }))
  const html = () => VueSSR.renderToString(Vue.createSSRApp({
    setup: () => ({ ...Vue.proxyRefs(state) }), ssrRender,
    components: { MasterClubDomains: { render: () => null } },
  }))
  await state.load()
  const rendered = await html()
  assert.match(rendered, /12\.345 visite totali/)
  assert.match(rendered, /456 visite uniche/)
  assert.match(rendered, /0 visite totali/)
  assert.match(rendered, /0 visite uniche/)
  assert.deepEqual(calls[0], ['/api/admin/master/clubs', { credentials: 'include' }])
  totalVisits = 12346
  await state.load()
  assert.match(await html(), /12\.346 visite totali/)
})
