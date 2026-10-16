import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc } from '../../test/sfcSetup.js';

function roster(t, overrides = {}) {
  const setup = setupSfc(new URL('./FanMvpRoster.vue', import.meta.url), { './FanPlayerCard.vue': {} }, {
    players: [{ id: 17, fullName: 'Marco Rossi', name: 'Rossi', number: 8 }],
    active: true, selectedPlayerId: null, pendingPlayerId: null,
    busy: false, canVote: true, clubLogoUrl: '', ...overrides,
  });
  t.after(() => setup.scope.stop());
  return setup;
}

test('select emits the real roster player without assigning a vote', (t) => {
  const setup = roster(t);
  setup.state.selectPlayer({ id: 17, name: 'Untrusted replacement' });
  assert.equal(setup.events.length, 1);
  assert.equal(setup.events[0][0], 'select');
  assert.equal(setup.events[0][1], setup.props.players[0]);
  assert.equal(setup.props.selectedPlayerId, null);
  assert.equal(setup.props.pendingPlayerId, null);
});

test('inactive, closed and pending rosters cannot emit another selection', (t) => {
  for (const overrides of [{ active: false }, { canVote: false }, { busy: true, pendingPlayerId: 17 }]) {
    const setup = roster(t, overrides);
    setup.state.selectPlayer(setup.props.players[0]);
    assert.deepEqual(setup.events, []);
  }
});

test('a confirmed selection remains highlighted while another player can be chosen', (t) => {
  const setup = roster(t, {
    selectedPlayerId: 17,
    players: [{ id: 17, name: 'Rossi' }, { id: 24, name: 'Bianchi' }],
  });
  assert.equal(setup.state.playerLabel(setup.props.players[0]), 'Rossi, Il tuo MVP');
  setup.state.selectPlayer(setup.props.players[1]);
  assert.equal(setup.events[0][1].id, 24);
  assert.equal(setup.props.selectedPlayerId, 17, 'a change request does not replace the confirmed vote');
});

test('missing players are ignored and selection needs no pointer event', (t) => {
  const setup = roster(t);
  setup.state.selectPlayer({ id: 999 });
  assert.deepEqual(setup.events, []);
  setup.state.selectPlayer(setup.props.players[0]);
  assert.equal(setup.events[0][1], setup.props.players[0]);
});

test('accessible labels distinguish pending and confirmed votes without inventing success', (t) => {
  const setup = roster(t);
  const player = setup.props.players[0];
  assert.equal(setup.state.playerLabel(player), 'Marco Rossi, maglia 8, Vota come MVP');
  setup.props.pendingPlayerId = 17;
  setup.props.busy = true;
  assert.equal(setup.state.playerLabel(player), 'Marco Rossi, maglia 8, Invio del voto in corso');
  setup.props.pendingPlayerId = null;
  setup.props.selectedPlayerId = 17;
  assert.equal(setup.state.playerLabel(player), 'Marco Rossi, maglia 8, Il tuo MVP');
  assert.equal(setup.state.samePlayer(null, null), false);
});
