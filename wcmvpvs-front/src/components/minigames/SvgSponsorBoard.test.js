import test from 'node:test';
import assert from 'node:assert/strict';
import { nextTick } from 'vue';
import { setupSfc, testVue } from '../../../test/sfcSetup.js';

test('fits logos by their intrinsic proportions and falls back while a replacement loads', async (t) => {
  const logos = [];
  class Logo {
    constructor() { logos.push(this); }

    load(width, height) {
      this.naturalWidth = width;
      this.naturalHeight = height;
      this.onload?.();
    }
  }

  const { state, props, scope } = setupSfc(new URL('./SvgSponsorBoard.vue', import.meta.url), {
    vue: { ...testVue, useId: () => 'test' },
  }, { enabled: true, imageUrl: '/square.svg', width: 100, height: 40 }, { Image: Logo });
  t.after(() => scope.stop());

  assert.equal(state.imageBounds.value, null);
  logos[0].load(64, 64);
  assert.deepEqual(state.imageBounds.value, { x: 34, y: 4, width: 32, height: 32 });

  props.imageUrl = '/wide.svg';
  await nextTick();
  logos[0].load(64, 64);
  assert.equal(state.imageBounds.value, null);

  logos[1].load(192, 64);
  const { x, y, width, height } = state.imageBounds.value;
  assert.equal(width / height, 3);
  assert.equal(x + width / 2, 50);
  assert.equal(y + height / 2, 20);

  props.enabled = false;
  await nextTick();
  assert.equal(state.imageBounds.value, null);
  assert.equal(logos.length, 2);
});
