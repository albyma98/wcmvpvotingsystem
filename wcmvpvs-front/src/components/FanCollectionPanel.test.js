import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc, testVue } from '../../test/sfcSetup.js';

function panel(t, collection = null, overrides = {}) {
  const setup = setupSfc(new URL('./FanCollectionPanel.vue', import.meta.url), { vue: testVue }, {
    nickname: 'Luca Bianchi', clubName: 'Basket Club', clubLogoUrl: '', registered: true,
    collection, loading: false, busy: false, error: '', ...overrides,
  });
  t.after(() => setup.scope.stop());
  return setup;
}

test('collection and equipped frame come exclusively from server props', (t) => {
  const setup = panel(t, {
    equipped_frame: '', matches_followed: 2,
    badges: [{ key: 'active_fan', title: 'Tifoso Attivo', event_id: 42 }],
    items: [{ key: 'club_frame', kind: 'frame', event_id: 0 }, { key: 'match_card', kind: 'match_card', event_id: 42 }],
  });
  assert.equal(setup.state.matchesFollowed.value, 2);
  assert.equal(setup.state.badges.value.length, 1);
  assert.equal(setup.state.memories.value.length, 1);
  setup.state.equipFrame('club_frame');
  assert.deepEqual(setup.events, [['equip', 'club_frame']]);
  assert.equal(setup.state.equippedFrame.value, '', 'emitting does not optimistically equip the frame');
  setup.props.collection = { ...setup.props.collection, equipped_frame: 'club_frame' };
  assert.equal(setup.state.equippedFrame.value, 'club_frame');
  setup.state.equipFrame('');
  assert.deepEqual(setup.events.at(-1), ['equip', '']);
  assert.equal(setup.state.equippedFrame.value, 'club_frame', 'removing also waits for the server');
});

test('unowned frames, duplicate equip and busy interactions do not emit a mutation', (t) => {
  const setup = panel(t, { equipped_frame: '', items: [] });
  setup.state.equipFrame('club_frame');
  setup.state.equipFrame('unknown');
  setup.state.equipFrame('');
  assert.deepEqual(setup.events, []);
  setup.props.collection = { equipped_frame: 'club_frame', items: [{ key: 'club_frame', kind: 'frame' }] };
  setup.state.equipFrame('club_frame');
  setup.props.busy = true;
  setup.state.equipFrame('');
  setup.props.busy = false;
  setup.props.loading = true;
  setup.state.equipFrame('');
  assert.deepEqual(setup.events, []);
});

test('empty profiles invent no badges, match history or player images', (t) => {
  const setup = panel(t, null, { nickname: '  ', registered: false });
  assert.equal(setup.state.displayNickname.value, 'Tifoso');
  assert.equal(setup.state.matchesFollowed.value, 0);
  assert.equal(setup.state.hasCollection.value, false);
  assert.deepEqual(setup.state.badges.value, []);
  assert.deepEqual(setup.state.memories.value, []);
  assert.deepEqual(setup.state.frames.value, []);
  assert.equal(setup.state.formatDate('invalid'), '');
  assert.equal(setup.state.matchLabel({ club_name: 'Club storico', opponent: 'Rivals' }), 'Club storico vs Rivals');
  assert.equal(setup.state.formatDate('2026-10-16'), '16 ott 2026');
  assert.equal(setup.state.initials.value, 'T');
});
