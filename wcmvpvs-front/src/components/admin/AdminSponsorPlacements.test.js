import test from 'node:test';
import assert from 'node:assert/strict';
import { nextTick } from 'vue';
import { setupSfc, testVue } from '../../../test/sfcSetup.js';
import { gameWallGames } from '../../games/gameWallSponsors.js';
import { postVoteSponsorSlots } from '../../sponsorPlacements.js';

test('the admin loads, saves and clears wall assignments with the society scope', async (t) => {
  const endpoint = '/admin/sponsors/game-walls';
  const requests = [];
  let stored = { basketball_left: 12 };
  const { state, props, scope } = setupSfc(new URL('./AdminSponsorPlacements.vue', import.meta.url), {
    vue: testVue,
    '../../games/gameWallSponsors': { gameWallGames },
    '../../sponsorPlacements': { postVoteSponsorSlots },
    '../../api': { apiClient: {
      async get(path, config) {
        requests.push([path, config.headers['X-Organization-Slug']]);
        return { data: config.headers['X-Organization-Slug'] === 'home' ? stored : {} };
      },
      async put(path, data, config) {
        requests.push([path, config.headers['X-Organization-Slug']]);
        stored = { ...data };
      },
    } },
  }, {
    authHeaders: { headers: { 'X-Organization-Slug': 'home' } },
    placementGroup: 'game-walls',
    sponsors: [{ id: 12, isActive: true }, { id: 34, isActive: true }],
  });
  t.after(() => scope.stop());

  await nextTick();
  assert.equal(state.draft.value.basketball_left, 12);
  assert.equal(state.draft.value.playmaker_right, 0);
  state.draft.value.basketball_left = 0;
  state.draft.value.playmaker_right = 34;
  await state.submit();
  assert.equal(stored.basketball_left, 0);
  assert.equal(stored.playmaker_right, 34);
  assert.equal(state.success.value, true);
  assert.equal(state.dirty.value, false);

  props.sponsors = [{ id: 12, isActive: true }];
  await nextTick();
  assert.equal(state.draft.value.playmaker_right, 0);
  assert.equal(state.dirty.value, false);

  props.authHeaders = { headers: { 'X-Organization-Slug': 'away' } };
  await nextTick();
  assert.equal(state.draft.value.basketball_left, 0);
  assert.equal(state.draft.value.playmaker_right, 0);
  assert.equal(state.success.value, false);
  assert.deepEqual(requests, [[endpoint, 'home'], [endpoint, 'home'], [endpoint, 'away']]);
});

test('the MVP form saves only its two banner slots and can clear either one', async (t) => {
  const requests = [];
  const { state, scope } = setupSfc(new URL('./AdminSponsorPlacements.vue', import.meta.url), {
    vue: testVue,
    '../../games/gameWallSponsors': { gameWallGames },
    '../../sponsorPlacements': { postVoteSponsorSlots },
    '../../api': { apiClient: {
      async get(path) {
        requests.push([path]);
        return { data: { post_vote_left: 12 } };
      },
      async put(path, data) { requests.push([path, data]); },
    } },
  }, {
    placementGroup: 'post-vote',
    authHeaders: { headers: { 'X-Organization-Slug': 'home' } },
    sponsors: [{ id: 12, isActive: true }, { id: 34, isActive: true }],
  });
  t.after(() => scope.stop());
  await nextTick();
  assert.deepEqual(state.draft.value, { post_vote_left: 12, post_vote_right: 0 });
  state.draft.value.post_vote_left = 0;
  state.draft.value.post_vote_right = 34;
  await state.submit();
  assert.deepEqual(requests, [
    ['/admin/sponsors/post-vote'],
    ['/admin/sponsors/post-vote', { post_vote_left: 0, post_vote_right: 34 }],
  ]);
  assert.equal(state.dirty.value, false);
  assert.equal(state.success.value, true);
});
