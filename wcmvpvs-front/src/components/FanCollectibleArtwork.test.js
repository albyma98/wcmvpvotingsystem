import test from 'node:test';
import assert from 'node:assert/strict';
import { setupSfc, testVue } from '../../test/sfcSetup.js';

const artworkImports = {
  vue: testVue,
  '../assets/fan-collectibles/avatar-frame.webp': '/art/frame.webp',
  '../assets/fan-collectibles/avatar-frame-small.webp': '/art/frame-small.webp',
  '../assets/fan-collectibles/match-card.webp': '/art/card.webp',
};

test('avatar uses actual initials, readable defaults and a bounded numeric size', (t) => {
  const setup = setupSfc(new URL('./FanAvatar.vue', import.meta.url), artworkImports, { name: 'Luca Bianchi', framed: false, size: 40 });
  t.after(() => setup.scope.stop());
  assert.equal(setup.state.initials.value, 'LB');
  assert.equal(setup.state.avatarStyle.value['--avatar-size'], '40px');
  assert.equal(setup.state.accessibleLabel.value, 'Luca Bianchi');
  setup.props.framed = true;
  assert.match(setup.state.accessibleLabel.value, /cornice della società indossata/);
  setup.props.name = '   ';
  setup.props.size = 'url(invalid)';
  assert.equal(setup.state.initials.value, 'T');
  assert.equal(setup.state.avatarSize.value, 44);
});

test('the souvenir preserves the purchased match metadata across later events', (t) => {
  const setup = setupSfc(new URL('./FanMatchCard.vue', import.meta.url), artworkImports, {
    memory: { club_name: 'Società della partita', opponent: 'Avversaria storica', event_date: '2026-10-16' },
    clubName: 'Nome corrente', clubLogoUrl: '/club.svg', compact: true,
  });
  t.after(() => setup.scope.stop());
  assert.equal(setup.state.displayClubName.value, 'Società della partita');
  assert.equal(setup.state.opponent.value, 'Avversaria storica');
  assert.equal(setup.state.eventDateLabel.value, '16 ott 2026');
  assert.equal(setup.state.eventDateISO.value, '2026-10-16');
  assert.equal(setup.state.safeClubLogoUrl.value, '/club.svg');
  setup.props.clubName = 'Altro nome corrente';
  assert.match(setup.state.accessibleLabel.value, /Società della partita, contro Avversaria storica/);
});

test('the card never invents an opponent, date or club logo when data is missing', (t) => {
  const setup = setupSfc(new URL('./FanMatchCard.vue', import.meta.url), artworkImports, {
    memory: { opponent: 123, event_date: 'invalid' }, clubName: '', clubLogoUrl: '', compact: false,
  });
  t.after(() => setup.scope.stop());
  assert.equal(setup.state.displayClubName.value, 'La tua squadra');
  assert.equal(setup.state.opponent.value, '');
  assert.equal(setup.state.eventDateLabel.value, '');
  assert.equal(setup.state.safeClubLogoUrl.value, '');
});
