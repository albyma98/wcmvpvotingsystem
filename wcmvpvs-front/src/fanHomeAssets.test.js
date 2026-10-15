import test from 'node:test';
import assert from 'node:assert/strict';
import { preloadFanHomeImages } from './fanHomeAssets.js';

test('preloading deduplicates images and waits for decoding as well as download', async () => {
  const images = [];
  let finishDecode;
  class FakeImage {
    constructor() { images.push(this); }
    decode() { return new Promise((resolve) => { finishDecode = resolve; }); }
  }
  let ready = false;
  const loading = preloadFanHomeImages(['/player.png', '/player.png', '', '/sponsor.png'], { ImageClass: FakeImage }).then(() => { ready = true; });
  assert.equal(images.length, 2);
  const loaded = images[0].onload();
  images[1].onerror();
  await new Promise(setImmediate);
  assert.equal(ready, false);
  finishDecode();
  await loaded;
  await loading;
  assert.equal(ready, true);
});

test('broken images and rejected decoding allow the splash to finish', async () => {
  const images = [];
  class FakeImage {
    constructor() { images.push(this); }
    decode() { return Promise.reject(new Error('Invalid image')); }
  }
  const loading = preloadFanHomeImages(['/broken.png', '/invalid.png'], { ImageClass: FakeImage });
  images[0].onerror();
  await images[1].onload();
  await loading;
  assert.ok(images.every((image) => image.onload === null && image.onerror === null));
});

test('aborting a replaced home releases pending image listeners', async () => {
  const images = [];
  class FakeImage { constructor() { images.push(this); } }
  const controller = new AbortController();
  const loading = preloadFanHomeImages(['/slow.png'], { ImageClass: FakeImage, signal: controller.signal });
  controller.abort();
  await loading;
  assert.equal(images[0].onload, null);
  assert.equal(images[0].onerror, null);
});
