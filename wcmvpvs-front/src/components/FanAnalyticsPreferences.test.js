import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { legalTextBlocks } from '../fanRegistration.js';
import { setupSfc, testVue, testWindow } from '../../test/sfcSetup.js';

function fixture(t, initial = null, navigator = {}) {
  let preference = initial;
  const writes = [];
  const setup = setupSfc(new URL('./FanAnalyticsPreferences.vue', import.meta.url), {
    vue: testVue,
    '../lib/posthog': { getFanAnalyticsConsent: () => preference, setFanAnalyticsConsent: value => { writes.push(value); preference = value; return value; } },
    '../fanRegistration': { legalTextBlocks },
    '../legal/fan-privacy.md?raw': readFileSync(new URL('../legal/fan-privacy.md', import.meta.url), 'utf8'),
  }, { clubName: 'Club di prova', themeStyle: {} }, { window: testWindow(), navigator });
  t.after(() => setup.scope.stop());
  return { ...setup, writes, setExternalPreference(value) { preference = value; } };
}

test('external analytics needs a positive choice and the complete notice is available before choosing', t => {
  const setup = fixture(t);
  assert.equal(setup.state.isOpen.value, true);
  assert.deepEqual(setup.writes, []);
  assert.ok(setup.state.legalBlocks.value.some(block => block.text.includes('PostHog')));
  setup.state.closePreferences();
  assert.deepEqual(setup.writes, []);
  assert.deepEqual(setup.events, []);
});

test('the preference can be accepted, reopened and revoked without any form or account data', async t => {
  const setup = fixture(t);
  setup.state.choose(true);
  assert.deepEqual(setup.writes, [true]);
  assert.deepEqual(setup.events, [['changed', true]]);
  assert.equal(setup.state.isOpen.value, false);
  await setup.state.openPreferences();
  setup.state.choose(false);
  assert.deepEqual(setup.writes, [true, false]);
  assert.deepEqual(setup.events, [['changed', true], ['changed', false]]);
  setup.state.choose(false);
  assert.equal(setup.events.length, 2, 'saving the same choice does not start another measured visit');
});

test('DNT and GPC disable acceptance while allowing the experience to continue without analytics', t => {
  for (const browser of [{ doNotTrack: '1' }, { globalPrivacyControl: true }]) {
    const setup = fixture(t, null, browser);
    assert.equal(setup.state.browserPreference.value, true);
    setup.state.choose(true);
    assert.deepEqual(setup.writes, []);
    setup.state.choose(false);
    assert.deepEqual(setup.writes, [false]);
    assert.deepEqual(setup.events, [['changed', false]]);
  }
});

test('a choice changed in another tab synchronizes the consent clock without creating another choice', t => {
  const setup = fixture(t);
  setup.setExternalPreference(true);
  setup.state.synchronizePreference({ key: 'arenaboostx:fan-analytics-consent:v1' });
  setup.state.synchronizePreference({ key: 'arenaboostx:fan-analytics-consent:v1' });
  setup.setExternalPreference(false);
  setup.state.synchronizePreference({ key: 'unrelated-storage-key' });
  setup.state.synchronizePreference({ key: 'arenaboostx:fan-analytics-consent:v1' });
  assert.deepEqual(setup.events, [['changed', true], ['changed', false]]);
  assert.deepEqual(setup.writes, []);
});
