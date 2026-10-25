import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc, testVue } from '../../../test/sfcSetup.js';

function fixture(t, client) {
  const setup = setupSfc(new URL('./MasterGamificationSection.vue', import.meta.url), {
    vue: testVue, '../../api': { apiClient: client },
  }, {});
  t.after(() => setup.scope.stop());
  return setup.state;
}

test('master assigns multiple sports, switches to Tutti and can disable a game', async t => {
  const writes = [];
  let stored = ['Tutti'];
  const state = fixture(t, {
    get: async () => ({ data: [{ id: 'basketball', name: 'Basket Rush', sports: [...stored] }] }),
    put: async (path, body) => { writes.push({ path, body }); stored = [...body.sports]; return { data: { sports: stored } }; },
  });
  await state.load();
  let game = state.games.value[0];
  assert.equal(state.isDirty(game), false);
  state.toggle(game, 'Calcio', true);
  state.toggle(game, 'Pallavolo', true);
  await state.save(game);
  assert.deepEqual(writes[0], { path: '/admin/master/minigames/basketball', body: { sports: ['Calcio', 'Pallavolo'] } });
  assert.equal(game.saved, true);
  await state.load();
  game = state.games.value[0];
  assert.deepEqual(game.selected, ['Calcio', 'Pallavolo']);
  state.toggle(game, 'Tutti', true);
  assert.deepEqual(game.selected, ['Tutti']);
  await state.save(game);
  state.toggle(game, 'Tutti', false);
  await state.save(game);
  assert.deepEqual(writes.at(-1).body.sports, []);
  assert.equal(state.isDirty(game), false);
});

test('a failed save keeps pending assignments and retries without duplicate writes', async t => {
  let rejectSave = true;
  let calls = 0;
  const state = fixture(t, {
    get: async () => ({ data: [{ id: 'quiz', name: 'Quiz Lampo', sports: ['Tutti'] }] }),
    put: async (path, body) => { calls += 1; if (rejectSave) throw new Error('offline'); return { data: body }; },
  });
  await state.load();
  const game = state.games.value[0];
  state.toggle(game, 'Pallacanestro', true);
  await state.save(game);
  assert.deepEqual(game.selected, ['Pallacanestro']);
  assert.deepEqual(game.sports, ['Tutti']);
  assert.match(game.error, /Salvataggio non riuscito/);
  assert.equal(state.isDirty(game), true);
  rejectSave = false;
  await Promise.all([state.save(game), state.save(game)]);
  assert.equal(calls, 2);
  assert.equal(game.saved, true);
  assert.equal(game.error, '');
});
