import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { compileScript, parse } from '@vue/compiler-sfc';
import * as Vue from 'vue';

// Exercise the real component; replace only HTTP and browser image decoding.
const { descriptor } = parse(readFileSync(new URL('./PlatformBrandingSection.vue', import.meta.url), 'utf8'));
const source = compileScript(descriptor, { id: 'platform-branding-test' }).content
  .replace(/^import .* from 'vue';?\r?\n/gm, '')
  .replace(/^import .* from '\.\.\/\.\.\/api';?\r?\n/gm, '')
  .replace('export default', 'return');
const componentFactory = new Function('Vue', 'apiClient', 'Image', 'URL', 'document',
  `const { computed, onBeforeUnmount, onMounted, reactive, ref } = Vue;\n${source}`);

function fixture(options = {}) {
  const requests = [], images = [], revoked = [], mounted = [], unmounted = [];
  const apiClient = {
    async get(path, config) {
      requests.push({ method: 'GET', path, config });
      return options.get ? options.get(path, config) : { data: { company_name: 'ArenaBoostX', logo_data: '' } };
    },
    async put(path, payload) {
      requests.push({ method: 'PUT', path, payload });
      return options.put ? options.put(path, payload) : { data: payload };
    },
  };
  class StubImage {
    naturalWidth = options.width ?? 1000;
    naturalHeight = options.height ?? 500;
    set src(value) { images.push(this); }
  }
  const canvas = {
    width: 0, height: 0,
    getContext: () => ({ clearRect() {}, drawImage() {} }),
    toDataURL: type => options.encode ? options.encode(canvas, type) : `data:${type};base64,bG9nbw==`,
  };
  const component = componentFactory(
    { ...Vue, onMounted: callback => mounted.push(callback), onBeforeUnmount: callback => unmounted.push(callback) },
    apiClient, StubImage,
    { createObjectURL: () => 'blob:logo', revokeObjectURL: value => revoked.push(value) },
    { createElement: () => canvas },
  );
  const state = component.setup({}, { expose() {} });
  const selectFile = (type = 'image/png', size = 1000) => state.readLogo({ target: { files: [{ type, size }], value: 'logo.png' } });
  return { state, requests, images, revoked, canvas, mounted, unmounted, selectFile };
}

test('loads global branding, trims company name and saves a removed logo explicitly', async () => {
  const { state, requests } = fixture({
    get: async () => ({ data: { company_name: 'Original company', logo_data: 'data:image/png;base64,bG9nbw==' } }),
  });
  await state.load();
  state.form.company_name = '  ArenaBoostX Italia  ';
  state.removeLogo();
  await state.save();
  assert.deepEqual(requests[1], {
    method: 'PUT', path: '/admin/master/report-branding',
    payload: { company_name: 'ArenaBoostX Italia', logo_data: '' },
  });
  assert.equal(state.dirty.value, false);
  assert.match(state.success.value, /salvata/);
});

test('save failure retains edits and reset restores the persisted identity', async () => {
  const { state } = fixture({ put: async () => { throw new Error('offline'); } });
  await state.load();
  state.form.company_name = 'Nuovo nome';
  await state.save();
  assert.equal(state.form.company_name, 'Nuovo nome');
  assert.equal(state.dirty.value, true);
  assert.equal(state.saving.value, false);
  assert.match(state.error.value, /Salvataggio non riuscito/);
  state.reset();
  assert.equal(state.form.company_name, 'ArenaBoostX');
  assert.equal(state.dirty.value, false);
});

test('rejects unsupported formats, large source files and oversized decoded images', async () => {
  const { state, images, revoked, selectFile } = fixture({ width: 2049 });
  await selectFile('image/svg+xml');
  assert.match(state.error.value, /PNG, JPEG o WebP/);
  assert.equal(images.length, 0);
  await selectFile('image/png', 4 * 1024 * 1024 + 1);
  assert.match(state.error.value, /4 MB/);
  const decode = selectFile();
  images[0].onload();
  await decode;
  assert.match(state.error.value, /2048 pixel/);
  assert.equal(state.form.logo_data, '');
  assert.deepEqual(revoked, ['blob:logo']);
});

test('reset during decoding prevents a stale logo from replacing the saved version', async () => {
  const { state, images, revoked, selectFile } = fixture();
  await state.load();
  const decode = selectFile();
  assert.equal(state.processingImage.value, true);
  state.reset();
  images[0].onload();
  await decode;
  assert.equal(state.form.logo_data, '');
  assert.equal(state.processingImage.value, false);
  assert.equal(state.dirty.value, false);
  assert.deepEqual(revoked, ['blob:logo']);
});

test('converts WebP to PNG, scales proportionally and enforces encoded size', async () => {
  const { state, images, canvas, selectFile } = fixture({
    encode: (image, type) => `data:${type};base64,${'x'.repeat(image.width > 800 ? 600000 : 300000)}`,
  });
  const decode = selectFile('image/webp');
  images[0].onload();
  await decode;
  assert.equal(canvas.width, 800);
  assert.equal(canvas.height, 400);
  assert.ok(state.form.logo_data.startsWith('data:image/png;base64,'));
  assert.ok(state.form.logo_data.length <= 512 * 1024);
  assert.equal(state.error.value, '');
});

test('unmount ignores a late load response and cancels its request', async () => {
  let finish;
  const { state, requests, unmounted } = fixture({ get: () => new Promise(resolve => { finish = resolve; }) });
  const pending = state.load();
  unmounted[0]();
  assert.equal(requests[0].config.signal.aborted, true);
  finish({ data: { company_name: 'Stale response', logo_data: '' } });
  await pending;
  assert.equal(state.form.company_name, 'ArenaBoostX');
  assert.equal(state.loaded.value, false);
});
