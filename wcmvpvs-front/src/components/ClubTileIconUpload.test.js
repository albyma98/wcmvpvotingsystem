import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { compileScript, parse } from '@vue/compiler-sfc'
import * as Vue from 'vue'

const { descriptor } = parse(readFileSync(new URL('./ClubTileIconUpload.vue', import.meta.url), 'utf8'))
const script = compileScript(descriptor, { id: 'club-icon-test' }).content
  .replace(/^import .* from .*\r?\n/gm, '')
  .replace('export default', 'return')
const createComponent = new Function('Vue', 'onBeforeUnmount', 'URL', 'Image', 'document', `const { ref } = Vue;\n${script}`)

function fixture(t, fail = false) {
  const events = [], revoked = [], unmountCallbacks = []
  const canvas = { width: 0, height: 0, getContext: () => ({ drawImage() {} }), toDataURL: () => 'data:image/webp;base64,icon' }
  class Image {
    naturalWidth = 1200
    naturalHeight = 600
    set src(value) { queueMicrotask(() => fail ? this.onerror() : this.onload()) }
  }
  const component = createComponent(Vue, callback => unmountCallbacks.push(callback), { createObjectURL: () => 'blob:icon', revokeObjectURL: url => revoked.push(url) }, Image, { createElement: () => canvas })
  const scope = Vue.effectScope()
  t.after(() => scope.stop())
  const state = scope.run(() => component.setup({ label: 'Contatti', modelValue: '' }, { expose() {}, emit: (...args) => events.push(args) }))
  return { state, events, revoked, canvas, unmount: () => unmountCallbacks.forEach(callback => callback()) }
}

test('image upload prepares a bounded icon, releases its object URL and emits its saved value', async t => {
  const { state, events, revoked, canvas } = fixture(t)
  const input = { files: [{ type: 'image/png', size: 1024 }], value: 'selected.png' }
  await state.upload({ target: input })
  assert.deepEqual(events, [['uploading', true], ['update:modelValue', 'data:image/webp;base64,icon'], ['uploading', false]])
  assert.equal(canvas.width, 256)
  assert.equal(canvas.height, 128)
  assert.equal(input.value, '')
  assert.deepEqual(revoked, ['blob:icon'])
  assert.equal(state.loading.value, false)
})

test('invalid or unreadable uploads preserve the current icon and allow another attempt', async t => {
  const { state, events, revoked } = fixture(t, true)
  await state.upload({ target: { files: [{ type: 'text/plain', size: 1 }], value: 'wrong' } })
  assert.match(state.error.value, /file immagine/)
  await state.upload({ target: { files: [{ type: 'image/png', size: 6 * 1024 * 1024 }], value: 'large' } })
  assert.match(state.error.value, /5 MB/)
  await state.upload({ target: { files: [{ type: 'image/png', size: 1024 }], value: 'broken' } })
  assert.match(state.error.value, /leggere/)
  assert.ok(!events.some(([name]) => name === 'update:modelValue'))
  assert.deepEqual(revoked, ['blob:icon'])
  assert.equal(state.loading.value, false)
})

test('leaving the icon editor during upload releases the save lock and ignores the late image', async t => {
  const { state, events, revoked, unmount } = fixture(t)
  const pending = state.upload({ target: { files: [{ type: 'image/png', size: 1024 }], value: 'icon' } })
  unmount()
  await pending
  assert.deepEqual(events, [['uploading', true], ['uploading', false]])
  assert.deepEqual(revoked, ['blob:icon'])
})
