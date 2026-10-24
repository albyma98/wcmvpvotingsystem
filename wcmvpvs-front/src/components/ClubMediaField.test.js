import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { compileScript, compileTemplate, parse } from '@vue/compiler-sfc'
import * as Vue from 'vue'

const { descriptor } = parse(readFileSync(new URL('./ClubMediaField.vue', import.meta.url), 'utf8'))
const compiled = compileScript(descriptor, { id: 'club-media-test' })
const script = compiled.content.replace(/^import .* from .*\r?\n/gm, '').replace('export default', 'return')
const createComponent = new Function('Vue', 'onBeforeUnmount', 'URL', 'Image', 'document', `const { computed, ref, watch } = Vue;\n${script}`)
assert.deepEqual(compileTemplate({ source: descriptor.template.content, id: 'club-media-test', compilerOptions: { bindingMetadata: compiled.bindings } }).errors, [])

function fixture(t, options = {}) {
  const events = [], revoked = [], unmountCallbacks = []
  const canvas = { width: 0, height: 0, getContext: () => ({ drawImage() {}, clearRect() {} }), toDataURL: () => options.oversize ? 'data:image/png;base64,' + 'x'.repeat(310 * 1024) : 'data:image/webp;base64,photo' }
  class Image {
    naturalWidth = 3200
    naturalHeight = 1800
    set src(value) { queueMicrotask(() => options.unreadable ? this.onerror() : this.onload()) }
  }
  const component = createComponent(Vue, callback => unmountCallbacks.push(callback), { createObjectURL: () => 'blob:photo', revokeObjectURL: url => revoked.push(url) }, Image, { createElement: () => canvas })
  const scope = Vue.effectScope()
  t.after(() => scope.stop())
  const props = Vue.reactive({ label: 'Fotografia', modelValue: '/originale.jpg', maxSide: 1600, disabled: false })
  const state = scope.run(() => component.setup(props, { expose() {}, emit: (...args) => events.push(args) }))
  return { state, events, revoked, canvas, props, unmount: () => unmountCallbacks.forEach(callback => callback()) }
}

test('Photo uploads preserve aspect ratio, optimize size and release the save lock and object URL', async t => {
  const { state, events, revoked, canvas } = fixture(t)
  const input = { files: [{ type: 'image/jpeg', size: 2048 }], value: 'photo.jpg' }
  await state.upload({ target: input })
  assert.deepEqual(events, [['uploading', true], ['update:modelValue', 'data:image/webp;base64,photo'], ['uploading', false]])
  assert.equal(canvas.width, 1600)
  assert.equal(canvas.height, 900)
  assert.equal(input.value, '')
  assert.deepEqual(revoked, ['blob:photo'])
})

test('Invalid, oversized and unreadable media keep the current image', async t => {
  const { state, events, props, revoked } = fixture(t, { unreadable: true })
  await state.upload({ target: { files: [{ type: 'image/svg+xml', size: 30 }], value: 'svg' } })
  assert.match(state.error.value, /JPG, PNG/)
  await state.upload({ target: { files: [{ type: 'image/jpeg', size: 11 * 1024 * 1024 }], value: 'large' } })
  assert.match(state.error.value, /10 MB/)
  await state.upload({ target: { files: [{ type: 'image/jpeg', size: 20 }], value: 'broken' } })
  assert.match(state.error.value, /leggere/)
  assert.equal(props.modelValue, '/originale.jpg')
  assert.equal(events.filter(([name]) => name === 'update:modelValue').length, 0)
  assert.deepEqual(revoked, ['blob:photo'])
  assert.equal(state.loading.value, false)
})

test('Complex images never create unbounded inline payloads', async t => {
  const { state, events, canvas } = fixture(t, { oversize: true })
  await state.upload({ target: { files: [{ type: 'image/png', size: 100 }], value: 'complex' } })
  assert.equal(canvas.width, 800)
  assert.match(state.error.value, /troppo complessa/)
  assert.ok(!events.some(([name]) => name === 'update:modelValue'))
  assert.equal(events.at(-1)[0], 'uploading')
  assert.equal(events.at(-1)[1], false)
})

test('Unmounted fields ignore late uploads and release the parent save lock exactly once', async t => {
  const { state, events, unmount, revoked } = fixture(t)
  const pending = state.upload({ target: { files: [{ type: 'image/png', size: 100 }], value: 'photo' } })
  unmount()
  await pending
  assert.deepEqual(events, [['uploading', true], ['uploading', false]])
  assert.deepEqual(revoked, ['blob:photo'])
})

test('Manual links validate supported sources without exposing inline data', async t => {
  const { state, events, props } = fixture(t)
  let validity = ''
  const input = { setCustomValidity(value) { validity = value } }
  state.setLink('javascript:alert(1)', input)
  assert.match(validity, /collegamento/)
  assert.equal(events.length, 0)
  state.setLink(' https://club.example/foto.jpg ', input)
  assert.equal(validity, '')
  assert.deepEqual(events.at(-1), ['update:modelValue', 'https://club.example/foto.jpg'])
  state.setLink('/immagini/foto.jpg', input)
  assert.deepEqual(events.at(-1), ['update:modelValue', '/immagini/foto.jpg'])
  props.modelValue = 'data:image/webp;base64,photo'
  assert.equal(state.remoteValue.value, '')
})
