import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { FAN_LEGAL_VERSION, FAN_MINIMUM_AGE, fanRegistrationAcceptance, legalTextBlocks } from './fanRegistration.js';

test('the published legal bundle and server use the same version and age threshold', () => {
  const server = readFileSync(new URL('../../wcmvpvs-back/service/database/fan_registration.go', import.meta.url), 'utf8');
  assert.ok(server.includes(`FanLegalVersion = "${FAN_LEGAL_VERSION}"`));
  assert.ok(server.includes(`FanMinimumAge = ${FAN_MINIMUM_AGE}`));
  for (const name of ['fan-terms', 'fan-privacy']) {
    const text = readFileSync(new URL(`./legal/${name}.md`, import.meta.url), 'utf8');
    assert.ok(text.includes(FAN_LEGAL_VERSION));
    const blocks = legalTextBlocks(text, 'Club <Basket>');
    assert.ok(blocks.some((block) => block.text.includes('Club <Basket>')));
    assert.ok(!blocks.some((block) => block.text.includes('{{clubName}}')));
  }
});

test('registration never infers acceptance or gadget consent from a truthy value', () => {
  assert.equal(fanRegistrationAcceptance({ acceptedTerms: 'yes', ageEligibilityDeclared: 1 }).accepted_terms, false);
  assert.equal(fanRegistrationAcceptance({}).age_eligibility_declared, false);
  const payload = fanRegistrationAcceptance({ acceptedTerms: true, ageEligibilityDeclared: true });
  assert.equal(payload.accepted_terms, true);
  assert.equal('marketing_consent' in payload, false);
  assert.equal('enter_lottery' in payload, false);
});

test('the new experience never initializes PostHog or creates an analytics device identifier without analytics opt-in', () => {
  const source = readFileSync(new URL('./lib/posthog.js', import.meta.url), 'utf8')
    .replace(/^import .*;\r?\n/gm, '').replace(/^export /gm, '')
    .replaceAll('import.meta.env', '({ VITE_POSTHOG_KEY: "test" })');
  let initialized = 0;
  let identified = 0;
  const window = { location: { pathname: '/club/new' } };
  const api = new Function('posthog', 'getOrCreateDeviceId', 'window', `${source}\nreturn { initPostHog, getPostHog, isPostHogReady };`)(
    { init: () => { initialized++; } }, () => { identified++; return 'device'; }, window,
  );
  assert.equal(api.initPostHog(), null);
  assert.equal(api.getPostHog(), null);
  assert.equal(api.isPostHogReady(), false);
  assert.equal(initialized, 0);
  assert.equal(identified, 0);
});
