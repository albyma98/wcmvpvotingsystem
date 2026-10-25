import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { compileTemplate, parse } from '@vue/compiler-sfc';
import * as Vue from 'vue';
import * as VueSSR from 'vue/server-renderer';
import { setupSfc, testVue, testWindow } from '../../../test/sfcSetup.js';
import * as journey from '../../fanJourney.js';

const { descriptor } = parse(readFileSync(new URL('./TapChallenge.vue', import.meta.url), 'utf8'));
const template = compileTemplate({ source: descriptor.template.content, id: 'tap-theme-test', ssr: true, ssrCssVars: [] });
assert.deepEqual(template.errors, []);
const helpers = {};
const renderCode = template.code.replace(/^import \{([^}]+)\} from "([^"]+)"\r?$/gm, (_, imports, module) => {
  for (const item of imports.split(',')) {
    const [name, alias = name] = item.trim().split(/\s+as\s+/);
    helpers[alias] = (module === 'vue' ? Vue : VueSSR)[name];
  }
  return '';
}).replace('export function ssrRender', 'return function ssrRender');
const ssrRender = new Function(...Object.keys(helpers), renderCode)(...Object.values(helpers));

function game(t, props = {}, api = {}) {
  const awards = [];
  const cleanup = [];
  const setup = setupSfc(new URL('./TapChallenge.vue', import.meta.url), {
    vue: { ...testVue, onBeforeUnmount: callback => cleanup.push(callback) },
    '../../fanJourney': journey,
    './ArcadeBasketball.vue': { render: () => Vue.h('svg', { 'data-ball': 'basketball' }) },
    './ArcadeVolleyball.vue': { render: () => Vue.h('svg', { 'data-ball': 'volleyball' }) },
    '../../services/coins': { awardTapChallengeCoins: async (payload) => { awards.push(payload); return { ok: true }; } },
    '../../api': { getFanSessionToken: () => '', ...api },
  }, { eventId: 42, cooldownSeconds: 0, completionReward: null, claiming: false, variant: 'basketball', ...props }, { window: testWindow(), crypto: { randomUUID: () => 'test-request' }, navigator: { vibrate() {} } });
  t.after(() => setup.scope.stop());
  const html = () => VueSSR.renderToString(Vue.createSSRApp({ setup: () => ({ ...Vue.proxyRefs(setup.state), ...setup.props }), ssrRender }));
  return { ...setup, awards, html, unmount: () => Promise.all(cleanup.map(callback => callback())) };
}

test('the volley variant renders only volleyball court, ball and copy, and earns eight coins after a real round', async t => {
  const volley = game(t, { variant: 'volleyball', completionReward: 8 });
  let html = await volley.html();
  assert.match(html, /RITMO SOTTO RETE/);
  assert.match(html, /TAP VOLLEY/);
  assert.match(html, /data-ball="volleyball"/);
  assert.doesNotMatch(html, /TAP BASKET|pallone da basket|PLAYGROUND|PARQUET|parquet|data-ball="basketball"/);
  await volley.state.startGame();
  volley.state.onTap();
  assert.equal(volley.state.tapCount.value, 1);
  html = await volley.html();
  assert.match(html, /Tocca il pallone da pallavolo/);
  await volley.state.finishGame();
  await volley.state.claimReward();
  await volley.state.claimReward();
  assert.deepEqual(volley.events.map(([name]) => name), ['round-started', 'round-completed', 'claim']);
  assert.equal(volley.events.at(-1)[1].coins, 8);
  assert.deepEqual(volley.awards, []);
  const basket = game(t);
  html = await basket.html();
  assert.match(html, /TAP BASKET/);
  assert.match(html, /data-ball="basketball"/);
  assert.doesNotMatch(html, /TAP VOLLEY/);
});

test('the volley live round submits and reads results using its own game identifier', async t => {
  const calls = [];
  const volley = game(t, { variant: 'volleyball', completionReward: 8 }, {
    submitTapLiveScore: async (...args) => { calls.push(['submit', ...args]); return { ok: true }; },
    fetchTapLiveResult: async (...args) => { calls.push(['result', ...args]); return { ok: true, data: { outcome: 'win', coins_earned: 30, my_score: 2, opponent_score: 1 } }; },
  });
  volley.state.status.value = 'playing';
  volley.state.liveStep.value = 'playing';
  volley.state.liveMatchId.value = 'volley-match';
  volley.state.tapCount.value = 2;
  await volley.state.finishGame();
  assert.deepEqual(calls, [['submit', 42, 'volley-match', 2, 'tap-volley'], ['result', 42, 'volley-match', 'tap-volley']]);
  assert.equal(volley.state.liveStep.value, 'result');
  assert.equal(volley.events.at(-1)[1].source, 'tap_challenge_live');
  assert.equal(volley.events.at(-1)[1].coins, 30);
});

test('closing a volley search cancels its queue before opening another game', async t => {
  const calls = [];
  const volley = game(t, { variant: 'volleyball' }, { cancelTapLiveQueue: async (...args) => { calls.push(args); return { ok: true }; } });
  volley.state.liveStep.value = 'searching';
  await volley.unmount();
  assert.deepEqual(calls, [[42, 'tap-volley']]);
});

test('fixed singleplayer emits only the journey reward once, even with zero taps, and bypasses the legacy wallet service', async (t) => {
  const setup = game(t, { completionReward: 8 });
  setup.state.status.value = 'finished';
  assert.equal(setup.state.earnedCoins.value, 8);
  assert.equal(setup.state.headline.value, 'Hai totalizzato 0 tap');
  await setup.state.claimReward();
  await setup.state.claimReward();
  assert.equal(setup.events.length, 1);
  assert.equal(setup.events[0][1].coins, 8);
  assert.deepEqual(setup.awards, []);
});

test('server saving blocks a fixed claim, and a new session can earn another reward', async (t) => {
  const setup = game(t, { completionReward: 8, claiming: true });
  setup.state.status.value = 'finished';
  await setup.state.claimReward();
  assert.deepEqual(setup.events, []);
  assert.equal(setup.state.isPrimaryDisabled.value, true);
  setup.props.claiming = false;
  await setup.state.claimReward();
  const firstRound = setup.events[0][1].roundId;
  await setup.state.startGame();
  setup.state.status.value = 'finished';
  await setup.state.claimReward();
  assert.deepEqual(setup.events.map(([name]) => name), ['claim', 'round-started', 'claim']);
  assert.notEqual(setup.events[2][1].roundId, firstRound);
  assert.deepEqual(setup.awards, []);
});

test('legacy singleplayer and server-awarded live matches retain their original claim paths', async (t) => {
  const legacy = game(t);
  legacy.state.status.value = 'finished';
  legacy.state.tapCount.value = 3;
  await legacy.state.claimReward();
  assert.equal(legacy.awards.length, 1);
  assert.equal(legacy.awards[0].amount, 3);
  assert.equal(legacy.events[0][1].source, 'tap_challenge');
  const live = game(t, { completionReward: 8, claiming: true });
  live.state.applyLiveResult({ outcome: 'win', coins_earned: 12, my_score: 5 });
  assert.equal(live.state.isFixedSingleplayer.value, false);
  assert.equal(live.events[0][1].source, 'tap_challenge_live');
  assert.equal(live.events[0][1].coins, 12);
  assert.deepEqual(live.awards, []);
});
