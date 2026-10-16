import test from 'node:test';
import assert from 'node:assert/strict';
import * as journey from '../fanJourney.js';
import * as brand from '../organizationTheme.js';
import * as assets from '../fanHomeAssets.js';
import { setupSfc, testVue, testWindow } from '../../test/sfcSetup.js';

const player = { id: 7, name: 'Rossi', lastName: 'Rossi', number: '7', image: '' };

function walletData(overrides = {}) {
  return {
    wallet: 0, points: 0, movements: [], catalog: [],
    collection: { equipped_frame: '', items: [], badges: [], matches_followed: 0 },
    ...overrides,
  };
}

const flush = async () => { for (let index = 0; index < 8; index += 1) await Promise.resolve(); };
function deferred() {
  let resolve;
  const promise = new Promise((done) => { resolve = done; });
  return { promise, resolve };
}

function fixture(overrides = {}, options = {}) {
  const saves = [];
  let serverWallet = 0;
  const serverAwards = new Set();
  const window = testWindow();
  const api = {
    apiClient: { get: async () => ({ data: { top3: [] } }) },
    fetchFanProfile: async () => ({ ok: true, data: { registered: false, guest_coins: 0 } }),
    fetchVoteStatus: async () => ({ ok: true, hasVoted: false }),
    fetchQuickQuiz: async () => ({ ok: true, data: { questions: [{ id: 1 }] } }),
    vote: async () => ({ ok: true }),
    syncGuestCoins: async (eventId, coins) => { saves.push({ eventId, coins }); return { ok: true, data: { wallet: coins } }; },
    claimFanJourneyReward: async (eventId, key) => {
      const amounts = { vote: 10, quiz: 15, reaction: 10, tap: 8, 'sponsor-rush': 12, 'memory-flash': 8 };
      const coins = serverAwards.has(key) ? 0 : amounts[key] || 0;
      serverWallet += coins; serverAwards.add(key); saves.push({ eventId, coins: serverWallet });
      return { ok: true, data: { wallet: serverWallet, points: serverWallet, coins_earned: coins, reward_amount: amounts[key] || 0 } };
    },
    fetchFanWallet: async () => ({ ok: true, data: walletData({ wallet: serverWallet, points: serverWallet }) }),
    purchaseFanItem: async () => ({ ok: false }), equipFanItem: async () => ({ ok: false }),
    redeemFanReward: async () => ({ ok: false }), updateFanNickname: async () => ({ ok: false }),
    resolveApiUrl: (path) => path, getOrganizationSlug: () => 'club',
    getDeviceHeaders: () => ({ 'X-Device-ID': 'test-device' }), getFanSessionHeaders: () => ({ 'X-Fan-Session': 'test-session' }),
    registerFanProfile: async () => ({ ok: false }),
    ...overrides,
  };
  const setup = setupSfc(new URL('./FanLiveCento.vue', import.meta.url), {
    vue: testVue, '../api': api, '../organizationTheme': brand, '../fanJourney': journey,
    '../fanHomeAssets': options.assets || assets,
    '../components/EventFeedbackModal.vue': {},
    '../components/FanPlayerCard.vue': {},
    '../components/FanCollectionPanel.vue': {},
  }, { eventId: 42, organizationSlug: 'club', initialDataReady: true, activeEvent: { is_active: true, votes_closed: false }, ...options.props }, { window });
  return { ...setup, saves, window, api };
}

test('home splash waits for the session wallet, saved vote, and every home image', async (t) => {
  let finishProfile;
  let finishImages;
  let imageUrls;
  const { state, scope, events } = fixture({
    fetchFanProfile: () => new Promise((resolve) => { finishProfile = resolve; }),
    apiClient: { get: async (path) => ({ data:
      path === '/organization-theme' ? { primary_color: '#123456', logo_url: '/club.png', name: 'Club' }
        : path === '/public/players' ? [{ id: 7, image_url: '/player.png' }]
          : path === '/sponsors' ? [{ id: 1, logo_data: '/sponsor.png' }] : {},
    }) },
  }, { assets: { preloadFanHomeImages: (urls) => {
    imageUrls = urls;
    return new Promise((resolve) => { finishImages = resolve; });
  } } });
  t.after(() => scope.stop());
  const loading = state.bootstrap();
  await new Promise(setImmediate);
  assert.equal(state.isHomeLoading.value, true);
  assert.equal(imageUrls, undefined);
  finishProfile({ ok: true, data: { registered: true, wallet: 75 } });
  while (!finishImages) await new Promise(setImmediate);
  assert.equal(state.totalCoins.value, 75);
  assert.equal(state.isHomeLoading.value, true);
  for (const url of ['/club.png', '/player.png', '/sponsor.png']) assert.ok(imageUrls.includes(url));
  assert.ok(events.some(([name, value]) => name === 'brand-change' && value.colors.primary_color === '#123456'));
  finishImages();
  await loading;
  assert.equal(state.isHomeLoading.value, false);
  assert.deepEqual(events.at(-1), ['loading-change', false]);
});

test('home stays covered until the parent has resolved the active event and saved player', async (t) => {
  let playerRequests = 0;
  const { state, scope } = fixture({ apiClient: { get: async (path) => {
    if (path === '/public/players') playerRequests += 1;
    return { data: {} };
  } } }, { props: { initialDataReady: false } });
  t.after(() => scope.stop());
  await state.bootstrap();
  assert.equal(state.isHomeLoading.value, true);
  assert.equal(playerRequests, 0);
});

test('an earlier bootstrap cannot dismiss the splash while a newer one is loading', async (t) => {
  const downloads = [];
  const { state, scope } = fixture({}, { assets: { preloadFanHomeImages: () => new Promise((resolve) => downloads.push(resolve)) } });
  t.after(() => scope.stop());
  const first = state.bootstrap();
  while (downloads.length < 1) await new Promise(setImmediate);
  const second = state.bootstrap();
  while (downloads.length < 2) await new Promise(setImmediate);
  downloads[0]();
  await first;
  assert.equal(state.isHomeLoading.value, true);
  downloads[1]();
  await second;
  assert.equal(state.isHomeLoading.value, false);
});

test('confirmed vote leads to quiz and an earned first badge without duplicate credits', async (t) => {
  const { state, scope, saves } = fixture();
  t.after(() => scope.stop());
  await state.castVote(player);
  assert.equal(state.activeSheet.value, 'vote-result');
  assert.equal(state.totalCoins.value, 10);
  assert.equal(state.voteBonusCoins.value, 10);
  assert.equal(state.progress.value.remaining, 15);
  state.quizAvailable.value = true;
  state.openJourneyNext();
  assert.equal(state.activeGameId.value, 'quiz');
  await state.onGameClaim({ coins: 15 });
  assert.equal(state.totalCoins.value, 25);
  assert.equal(state.progress.value.unlocked, true);
  assert.equal(state.celebrationKind.value, 'quiz');
  state.openGame('quiz');
  await state.onGameClaim({ coins: 15 });
  await state.castVote(player);
  assert.equal(saves.length, 2);
  assert.equal(state.totalCoins.value, 25);
});

test('failed vote shows an error and grants nothing', async (t) => {
  const { state, scope, saves } = fixture({ vote: async () => ({ ok: false, message: 'Votazioni chiuse' }) });
  t.after(() => scope.stop());
  await state.castVote(player);
  assert.equal(state.hasVoted.value, false);
  assert.equal(state.voteError.value, 'Votazioni chiuse');
  assert.equal(state.progress.value.earned, 0);
  assert.equal(saves.length, 0);
  assert.notEqual(state.activeSheet.value, 'vote-result');
});

test('playing after the MVP vote advances the home progress bar and unlocks the badge', async (t) => {
  const { state, scope } = fixture();
  t.after(() => scope.stop());
  await state.castVote(player);
  state.openGame('reaction');
  await state.onGameClaim({ coins: 10 });
  assert.equal(state.totalCoins.value, 20);
  assert.equal(state.progress.value.current, 20);
  assert.equal(state.progress.value.remaining, 5);
  assert.equal(state.progress.value.percent, 80);
  state.openGame('memory-flash');
  await state.onGameClaim({ coins: 8 });
  assert.equal(state.totalCoins.value, 28);
  assert.equal(state.progress.value.current, 25);
  assert.equal(state.progress.value.unlocked, true);
});

test('game earnings unlock the badge and buying a frame preserves points and the achievement', async (t) => {
  const item = { key: 'club_frame', title: 'Cornice club', cost: 20, kind: 'frame', owned: false };
  const { state, scope } = fixture({ purchaseFanItem: async () => ({ ok: true, data: walletData({
    wallet: 10, points: 30, catalog: [{ ...item, owned: true }],
    collection: { equipped_frame: '', items: [{ ...item, owned: true }], badges: [], matches_followed: 1 },
  }) }) });
  t.after(() => scope.stop());
  for (const [game, coins] of [['reaction', 10], ['sponsor-rush', 12], ['memory-flash', 8]]) {
    state.openGame(game);
    await state.onGameClaim({ coins });
  }
  assert.equal(state.hasVoted.value, false);
  assert.equal(state.progress.value.earned, 30);
  assert.equal(state.progress.value.unlocked, true);
  state.applyWallet(walletData({ wallet: 30, points: 30, catalog: [item] }));
  await state.redeem(item);
  assert.equal(state.totalCoins.value, 10);
  assert.equal(state.participationPoints.value, 30);
  assert.equal(state.collection.value.items[0].key, 'club_frame');
  assert.equal(state.progress.value.current, 25);
  assert.equal(state.progress.value.unlocked, true);
});

test('game earnings advance progress only after saving succeeds, including a retry', async (t) => {
  let attempts = 0;
  const { state, scope } = fixture({ claimFanJourneyReward: async (_, key) => key === 'vote'
    ? { ok: true, data: { wallet: 10, coins_earned: 10, reward_amount: 10 } }
    : ++attempts === 1 ? { ok: false } : { ok: true, data: { wallet: 20, coins_earned: 10, reward_amount: 10 } } });
  t.after(() => scope.stop());
  await state.castVote(player);
  state.openGame('reaction');
  await state.onGameClaim({ coins: 10 });
  assert.equal(state.progress.value.current, 10);
  assert.equal(state.totalCoins.value, 10);
  assert.ok(state.coinError.value);
  await state.onGameClaim(state.pendingGameClaim.value);
  assert.equal(state.progress.value.current, 20);
  assert.equal(state.totalCoins.value, 20);
});

test('every confirmed basketball basket updates wallet and badge progress across repeated games', (t) => {
  const { state, scope } = fixture();
  t.after(() => scope.stop());
  state.openGame('basketball');
  state.onBasketballCredit({ event_id: 42, wallet: 1, basketball_earned: 1, coins_earned: 1 });
  assert.equal(state.progress.value.current, 1);
  assert.equal(state.activeGameId.value, 'basketball');
  assert.ok(state.doneKeys.value.includes('basketball'));
  state.closeGame();
  state.openGame('basketball');
  state.onBasketballCredit({ event_id: 42, wallet: 2, basketball_earned: 2, coins_earned: 1 });
  assert.equal(state.totalCoins.value, 2);
  assert.equal(state.progress.value.current, 2);
  state.onBasketballCredit({ event_id: 42, wallet: 2, basketball_earned: 2, coins_earned: 0 });
  assert.equal(state.progress.value.current, 2);
  state.onBasketballCredit({ event_id: 42, wallet: 25, basketball_earned: 25, coins_earned: 1 });
  assert.equal(state.progress.value.unlocked, true);
  const mission = state.earnMissions.value.find((item) => item.key === 'basketball');
  assert.equal(mission.repeatable, true);
  state.doneKeys.value = state.earnMissions.value.map((item) => item.key);
  assert.equal(state.featuredMission.value.key, 'basketball');
});

test('basketball receipts for another event or invalid balances do not change the home', (t) => {
  const { state, scope } = fixture();
  t.after(() => scope.stop());
  state.onBasketballCredit({ event_id: 43, wallet: 100, basketball_earned: 100, coins_earned: 1 });
  state.onBasketballCredit({ event_id: 42, wallet: -1, basketball_earned: 1, coins_earned: 1 });
  state.onBasketballCredit({ event_id: 42, wallet: 1, basketball_earned: NaN, coins_earned: 1 });
  assert.equal(state.totalCoins.value, 0);
  assert.equal(state.progress.value.current, 0);
});

test('Playmaker is repeatable and every server-confirmed assist advances wallet and the first goal', (t) => {
  const { state, scope } = fixture();
  t.after(() => scope.stop());
  const mission = state.earnMissions.value.find((item) => item.key === 'playmaker');
  assert.equal(mission.repeatable, true);
  assert.equal(mission.rewardUnit, 'assist');
  state.onMission(mission);
  assert.equal(state.activeGameId.value, 'playmaker');
  state.onPlaymakerCredit({ event_id: 42, wallet: 1, playmaker_earned: 1, coins_earned: 1 });
  state.onPlaymakerCredit({ event_id: 42, wallet: 2, playmaker_earned: 2, coins_earned: 1 });
  state.onPlaymakerCredit({ event_id: 42, wallet: 2, playmaker_earned: 2, coins_earned: 0 });
  assert.equal(state.totalCoins.value, 2);
  assert.equal(state.progress.value.earned, 2);
  assert.ok(state.doneKeys.value.includes('playmaker'));
  state.closeGame(); state.openGame('playmaker');
  state.onPlaymakerCredit({ event_id: 42, wallet: 25, playmaker_earned: 25, coins_earned: 1 });
  assert.equal(state.progress.value.unlocked, true);
  assert.equal(state.activeGameId.value, 'playmaker');
});

test('Playmaker rejects stale receipts and preserves confirmed progress when an older profile arrives', async (t) => {
  let finish;
  const { state, scope } = fixture({ fetchFanProfile: () => new Promise((resolve) => { finish = resolve; }) });
  t.after(() => scope.stop());
  state.onPlaymakerCredit({ event_id: 43, wallet: 100, playmaker_earned: 100 });
  state.onPlaymakerCredit({ event_id: 42, wallet: -1, playmaker_earned: 10 });
  assert.equal(state.totalCoins.value, 0);
  const loading = state.loadFanProfile();
  state.onPlaymakerCredit({ event_id: 42, wallet: 25, playmaker_earned: 25, coins_earned: 1 });
  finish({ ok: true, data: { registered: false, guest_coins: 1, journey_rewards: { playmaker: 1 } } });
  await loading;
  assert.equal(state.totalCoins.value, 25);
  assert.equal(state.progress.value.current, 25);
});

test('Playmaker saved progress restores independently from wallet spending and closes after saving', async (t) => {
  const { state, scope } = fixture({ fetchFanProfile: async () => ({ ok: true, data: {
    registered: true, wallet: 3, journey_rewards: { basketball: 10, playmaker: 15 },
  } }) });
  t.after(() => scope.stop());
  await state.loadFanProfile();
  assert.equal(state.progress.value.unlocked, true);
  assert.equal(state.totalCoins.value, 3);
  state.openGame('playmaker');
  state.isPlaymakerSaving.value = true; state.closeGame();
  assert.equal(state.activeGameId.value, 'playmaker');
  state.isPlaymakerSaving.value = false; state.closeGame();
  assert.equal(state.activeGameId.value, null);
  assert.equal(state.activeSheet.value, 'earn');
});

test('Crossover Rush is repeatable and confirmed duels update wallet and badge once', (t) => {
  const { state, scope } = fixture();
  t.after(() => scope.stop());
  const mission = state.earnMissions.value.find((item) => item.key === 'crossover');
  assert.equal(mission.repeatable, true);
  assert.equal(mission.rewardUnit, 'duello vinto');
  state.onMission(mission);
  assert.equal(state.activeGameId.value, 'crossover');
  state.onCrossoverCredit({ event_id: 42, wallet: 1, crossover_earned: 1, coins_earned: 1 });
  state.onCrossoverCredit({ event_id: 42, wallet: 1, crossover_earned: 1, coins_earned: 0 });
  assert.equal(state.progress.value.earned, 1);
  assert.equal(state.totalCoins.value, 1);
  state.closeGame(); state.openGame('crossover');
  state.onCrossoverCredit({ event_id: 42, wallet: 25, crossover_earned: 25, coins_earned: 1 });
  assert.equal(state.progress.value.unlocked, true);
  assert.ok(state.doneKeys.value.includes('crossover'));
});

test('Crossover ignores stale receipts and restores combined progress while saving blocks closing', async (t) => {
  const { state, scope } = fixture({ fetchFanProfile: async () => ({ ok: true, data: {
    registered: true, wallet: 2, journey_rewards: { basketball: 10, playmaker: 10, crossover: 5 },
  } }) });
  t.after(() => scope.stop());
  await state.loadFanProfile();
  state.onCrossoverCredit({ event_id: 43, wallet: 900, crossover_earned: 900 });
  state.onCrossoverCredit({ event_id: 42, wallet: -1, crossover_earned: 2 });
  assert.equal(state.totalCoins.value, 2);
  assert.equal(state.progress.value.earned, 25);
  state.openGame('crossover'); state.isCrossoverSaving.value = true; state.closeGame();
  assert.equal(state.activeGameId.value, 'crossover');
  state.isCrossoverSaving.value = false; state.closeGame();
  assert.equal(state.activeSheet.value, 'earn');
});

test('basketball progress is restored from the profile and saving finishes before closing the game', async (t) => {
  const { state, scope } = fixture({ fetchFanProfile: async () => ({ ok: true, data: {
    registered: true, wallet: 4, journey_rewards: { basketball: 28 },
  } }) });
  t.after(() => scope.stop());
  await state.loadFanProfile();
  assert.equal(state.totalCoins.value, 4);
  assert.equal(state.progress.value.unlocked, true);
  state.openGame('basketball');
  state.isBasketballSaving.value = true;
  state.closeGame();
  assert.equal(state.activeGameId.value, 'basketball');
  state.isBasketballSaving.value = false;
  state.closeGame();
  assert.equal(state.activeGameId.value, null);
  assert.equal(state.activeSheet.value, 'earn');
});

test('a delayed profile response cannot roll back basketball progress or wallet after a made basket', async (t) => {
  let finish;
  const { state, scope } = fixture({ fetchFanProfile: () => new Promise((resolve) => { finish = resolve; }) });
  t.after(() => scope.stop());
  const loading = state.loadFanProfile();
  state.onBasketballCredit({ event_id: 42, wallet: 25, basketball_earned: 25, coins_earned: 1 });
  finish({ ok: true, data: { registered: false, guest_coins: 1, journey_rewards: { basketball: 1 } } });
  await loading;
  assert.equal(state.totalCoins.value, 25);
  assert.equal(state.progress.value.current, 25);
  assert.equal(state.progress.value.unlocked, true);
});

test('wallet failure preserves the vote and retries the same bonus without inventing coins', async (t) => {
  const requested = [];
  const { state, scope } = fixture({ claimFanJourneyReward: async (_, key) => {
    const coins = key === 'vote' ? 10 : 15;
    requested.push(coins);
    return requested.length === 1 ? { ok: false } : { ok: true, data: { wallet: coins, coins_earned: coins, reward_amount: coins } };
  } });
  t.after(() => scope.stop());
  await state.castVote(player);
  assert.equal(state.hasVoted.value, true);
  assert.equal(state.totalCoins.value, 0);
  assert.equal(state.voteBonusCoins.value, 0);
  assert.equal(state.journey.value.pendingVoteBonus, true);
  await state.retryVoteBonus();
  await state.retryVoteBonus();
  assert.deepEqual(requested, [10, 10]);
  assert.equal(state.progress.value.earned, 10);
  assert.equal(state.totalCoins.value, 10);
  assert.equal(state.journey.value.pendingVoteBonus, false);
});

test('a wallet response for an old match cannot update the new match', async (t) => {
  let finish;
  const { state, scope, props } = fixture({ claimFanJourneyReward: () => new Promise((resolve) => { finish = resolve; }) });
  t.after(() => scope.stop());
  const action = state.castVote(player);
  while (!finish) await new Promise(setImmediate);
  props.eventId = 43;
  finish({ ok: true, data: { wallet: 10 } });
  await action;
  assert.equal(state.totalCoins.value, 0);
  assert.equal(state.progress.value.earned, 0);
});

test('completed badge survives reload and does not carry over to another match', (t) => {
  const { state, scope, props, window } = fixture();
  t.after(() => scope.stop());
  window.localStorage.setItem(journey.journeyStorageKey('club', 42), JSON.stringify({ rewards: { vote: 10, quiz: 15 } }));
  state.loadJourney();
  assert.equal(state.progress.value.unlocked, true);
  props.eventId = 43;
  state.loadJourney();
  assert.equal(state.progress.value.earned, 0);
});

test('an old match vote status cannot mark the next match as already voted', async (t) => {
  let finish;
  const { state, scope, props } = fixture({ fetchVoteStatus: () => new Promise((resolve) => { finish = resolve; }) });
  t.after(() => scope.stop());
  const action = state.loadVoteStatus();
  props.eventId = 43;
  finish({ ok: true, hasVoted: true, playerId: 7 });
  await action;
  assert.equal(state.hasVoted.value, false);
});

test('the saved MVP restores the complete player card from the roster after a reload', async (t) => {
  const { state, scope } = fixture({
    fetchVoteStatus: async () => ({ ok: true, hasVoted: true, playerId: 7 }),
    apiClient: { get: async (path) => ({ data: path === '/public/players'
      ? [{ id: 7, first_name: 'Mario', last_name: 'Rossi', jersey_number: 0, role: 'Guardia', image_url: '/rossi.png', is_called_up: true }]
      : {} }) },
  });
  t.after(() => scope.stop());
  await state.loadPlayers();
  await state.loadVoteStatus();
  assert.equal(state.votedPlayerCard.value.fullName, 'Mario Rossi');
  assert.equal(state.votedPlayerCard.value.number, '0');
  assert.equal(state.votedPlayerCard.value.subtitle, 'Guardia');
  assert.equal(state.displayVotedImage.value, '/rossi.png');
  assert.equal(state.displayVotedName.value, 'M. Rossi');
  await state.castVote(state.players.value[0]);
  assert.equal(state.votedPlayerCard.value.fullName, 'Mario Rossi');
  assert.equal(state.votedPlayerCard.value.subtitle, 'Guardia');
});

test('the player card keeps the restored parent details when the roster is unavailable', (t) => {
  const { state, scope } = fixture({}, { props: {
    votedPlayerName: 'A. Bianchi', votedPlayerLastName: 'Bianchi',
    votedPlayerNumber: 12, votedPlayerImageUrl: '/bianchi.png',
  } });
  t.after(() => scope.stop());
  assert.equal(state.votedPlayerCard.value.name, 'A. Bianchi');
  assert.equal(state.votedPlayerCard.value.number, 12);
  assert.equal(state.votedPlayerCard.value.image, '/bianchi.png');
  assert.equal(state.votedPlayerCard.value.badge, 'AB');
});

test('final MVP cards include every tied winner and stay hidden until the match is concluded', (t) => {
  const { state, scope } = fixture();
  t.after(() => scope.stop());
  state.rawPlayers.value = [{ id: 7, first_name: 'Mario', last_name: 'Rossi', jersey_number: 7, image_url: '/roster.png' }];
  state.liveEvent.value = { event_id: 42, is_concluded: false, mvp: [
    { player_id: 7, first_name: 'Mario', last_name: 'Rossi', image_url: '/winner.png', votes: 18 },
    { player_id: 8, first_name: 'Anna', last_name: 'Bianchi', image_url: '/bianchi.png', votes: 18 },
  ] };
  assert.equal(state.finalMvpPlayers.value.length, 0);
  state.liveEvent.value.is_concluded = true;
  assert.equal(state.finalMvpPlayers.value.length, 2);
  assert.equal(state.finalMvpPlayers.value[0].image, '/winner.png');
  assert.equal(state.finalMvpPlayers.value[0].number, '7');
  assert.equal(state.finalMvpPlayers.value[1].name, 'A. Bianchi');
  assert.equal(state.finalMvpPlayers.value[1].subtitle, '18 voti');
});

test('a saved vote with a lost response recovers its bonus from the server status', async (t) => {
  const { state, scope, saves } = fixture({ fetchVoteStatus: async () => ({ ok: true, hasVoted: true, playerId: 7 }) });
  t.after(() => scope.stop());
  await state.loadVoteStatus();
  assert.equal(state.hasVoted.value, true);
  assert.equal(state.totalCoins.value, 10);
  assert.equal(state.progress.value.current, 10);
  await state.loadVoteStatus();
  assert.equal(saves.length, 1);
});

test('featured mission follows quiz availability and completed activities', (t) => {
  const { state, scope } = fixture();
  t.after(() => scope.stop());
  state.quizAvailable.value = true;
  assert.equal(state.featuredMission.value.key, 'quiz');
  state.doneKeys.value = ['quiz'];
  assert.equal(state.featuredMission.value.key, 'reaction');
  state.liveEvent.value = { event_id: 42, is_active: true, is_concluded: true };
  assert.equal(state.featuredMission.value, null);
  assert.equal(state.canVote.value, false);
  assert.equal(state.showMatchSummary.value, true);
});

test('the leaderboard sends the fan session and normalizes legacy coin rows into points', async (t) => {
  let headers;
  const { state, scope } = fixture({ apiClient: { get: async (path, config) => {
    if (!path.includes('coins-leaderboard')) return { data: {} };
    headers = config.headers;
    return { data: { top3: [], userRank: { rank: 3, coins: 40 }, nearby: [{ rank: 2, coins: 40, name: 'Rivale' }, { rank: 3, coins: 40, is_me: true }] } };
  } } });
  t.after(() => scope.stop());
  await state.refreshLeaderboard();
  assert.equal(headers['X-Fan-Session'], 'test-session');
  assert.equal(state.leaderboardNearby.value.length, 2);
  assert.match(state.rankGapLabel.value, /Manca 1 punto/);
});

test('live status exposes only final MVPs in the end-of-match recap', async (t) => {
  const { state, scope } = fixture({ apiClient: { get: async () => ({ data: { event_id: 42, is_active: false, votes_closed: true, is_concluded: true, quiz: { available: false }, mvp: [{ first_name: 'Mario', last_name: 'Rossi', votes: 18 }] } }) } });
  t.after(() => scope.stop());
  await state.refreshLiveStatus();
  assert.equal(state.showMatchSummary.value, true);
  assert.equal(state.canVote.value, false);
  assert.equal(state.finalMvpLabel.value, 'Mario Rossi');
  state.openJourneyNext();
  assert.equal(state.activeSheet.value, 'match-summary');
});

test('server rewards restore the badge on a fresh browser profile', async (t) => {
  const { state, scope } = fixture({ fetchFanProfile: async () => ({ ok: true, data: { registered: true, wallet: 5, journey_rewards: { vote: 10, quiz: 15 } } }) });
  t.after(() => scope.stop());
  await state.loadFanProfile();
  assert.equal(state.progress.value.unlocked, true);
  assert.equal(state.totalCoins.value, 5);
  assert.ok(state.doneKeys.value.includes('quiz'));
});

test('the gap provided by the server respects tie breaking instead of overstating the target', async (t) => {
  const { state, scope } = fixture({ apiClient: { get: async () => ({ data: { top3: [], userRank: { rank: 3, coins: 42 }, next_rank_gap: 8, nearby: [{ rank: 2, coins: 50, name: 'Rivale' }] } }) } });
  t.after(() => scope.stop());
  await state.refreshLeaderboard();
  assert.match(state.rankGapLabel.value, /Mancano 8 punti/);
});

test('a retry after a lost response restores progress without re-awarding the vote bonus', async (t) => {
  let attempts = 0;
  const { state, scope } = fixture({ claimFanJourneyReward: async () => ++attempts === 1 ? { ok: false } : { ok: true, data: { wallet: 10, coins_earned: 0, reward_amount: 10 } } });
  t.after(() => scope.stop());
  await state.castVote(player);
  await state.retryVoteBonus();
  assert.equal(state.totalCoins.value, 10);
  assert.equal(state.progress.value.earned, 10);
  assert.equal(state.voteBonusCoins.value, 0);
});

test('the wallet sheet loads confirmed balance, points, movements, catalog and collection together', async (t) => {
  const movement = { id: 1, delta: -25, balance: 15, label: 'Cornice club', created_at: '2026-10-16 18:30:00' };
  const item = { key: 'club_frame', title: 'Cornice club', kind: 'frame', cost: 25, owned: true };
  const { state, scope } = fixture({ fetchFanWallet: async () => ({ ok: true, data: walletData({
    wallet: 15, points: 40, movements: [movement], catalog: [item],
    collection: { equipped_frame: 'club_frame', items: [item], badges: [{ title: 'Tifoso Attivo' }], matches_followed: 2 },
  }) }) });
  t.after(() => scope.stop());
  state.openSheet('wallet');
  await flush();
  assert.equal(state.activeSheet.value, 'wallet');
  assert.equal(state.totalCoins.value, 15);
  assert.equal(state.participationPoints.value, 40);
  assert.deepEqual(state.walletMovements.value, [movement]);
  assert.equal(state.collection.value.equipped_frame, 'club_frame');
  assert.equal(state.collection.value.matches_followed, 2);
  assert.equal(state.catalog.value[0].owned, true);
  assert.equal(state.walletError.value, '');
});

test('a failed or malformed wallet read preserves the last confirmed state and can be retried', async (t) => {
  let attempts = 0;
  const { state, scope } = fixture({ fetchFanWallet: async () => ++attempts === 1
    ? { ok: true, data: { wallet: 999, points: 999 } }
    : { ok: true, data: walletData({ wallet: 8, points: 30 }) } });
  t.after(() => scope.stop());
  state.applyWallet(walletData({ wallet: 12, points: 30 }));
  await state.loadWallet();
  assert.ok(state.walletError.value);
  assert.equal(state.totalCoins.value, 12);
  assert.equal(state.participationPoints.value, 30);
  await state.loadWallet();
  assert.equal(state.totalCoins.value, 8);
  assert.equal(state.participationPoints.value, 30);
  assert.equal(state.walletError.value, '');
});

test('an earlier wallet request cannot overwrite a newer wallet and collection response', async (t) => {
  const requests = [];
  const { state, scope } = fixture({ fetchFanWallet: () => {
    const request = deferred(); requests.push(request); return request.promise;
  } });
  t.after(() => scope.stop());
  const oldRead = state.loadWallet();
  const newRead = state.loadWallet();
  requests[1].resolve({ ok: true, data: walletData({ wallet: 25, points: 50 }) });
  await newRead;
  requests[0].resolve({ ok: true, data: walletData({ wallet: 2, points: 2 }) });
  await oldRead;
  assert.equal(state.totalCoins.value, 25);
  assert.equal(state.participationPoints.value, 50);
  assert.equal(state.isLoadingWallet.value, false);
});

test('wallet reads for a previous match or organization cannot enter the current home', async (t) => {
  for (const nextProps of [{ eventId: 43 }, { organizationSlug: 'other-club' }]) {
    const response = deferred();
    const { state, props, scope } = fixture({ fetchFanWallet: () => response.promise });
    t.after(() => scope.stop());
    const loading = state.loadWallet();
    Object.assign(props, nextProps);
    response.resolve({ ok: true, data: walletData({ wallet: 900, points: 900 }) });
    await loading;
    assert.equal(state.totalCoins.value, 0);
    assert.equal(state.participationPoints.value, 0);
    assert.deepEqual(state.collection.value.items, []);
  }
});

test('an old wallet read refreshes again instead of rolling back a newly credited basket', async (t) => {
  const old = deferred();
  let calls = 0;
  const { state, scope } = fixture({ fetchFanWallet: () => ++calls === 1
    ? old.promise : Promise.resolve({ ok: true, data: walletData({ wallet: 25, points: 25 }) }) });
  t.after(() => scope.stop());
  const loading = state.loadWallet();
  state.onBasketballCredit({ event_id: 42, wallet: 25, basketball_earned: 25, coins_earned: 1 });
  old.resolve({ ok: true, data: walletData({ wallet: 1, points: 1 }) });
  await loading;
  await flush();
  assert.equal(calls, 2);
  assert.equal(state.totalCoins.value, 25);
  assert.equal(state.participationPoints.value, 25);
  assert.equal(state.progress.value.unlocked, true);
});

test('shop purchases wait for acknowledgement, prevent duplicate clicks and block overlapping games', async (t) => {
  const purchase = deferred();
  const calls = [];
  const item = { key: 'club_frame', title: 'Cornice club', kind: 'frame', cost: 25, owned: false };
  const { state, scope } = fixture({ purchaseFanItem: (...args) => { calls.push(args); return purchase.promise; } });
  t.after(() => scope.stop());
  state.applyWallet(walletData({ wallet: 60, points: 90, catalog: [item] }));
  const buying = state.redeem(item);
  void state.redeem(item);
  state.openGame('basketball');
  assert.equal(state.shopBusy.value, true);
  assert.equal(state.activeGameId.value, null);
  assert.equal(state.totalCoins.value, 60);
  assert.deepEqual(state.collection.value.items, []);
  assert.deepEqual(calls, [[42, 'club_frame']]);
  purchase.resolve({ ok: true, data: walletData({ wallet: 35, points: 90, catalog: [{ ...item, owned: true }],
    collection: { equipped_frame: '', items: [item], badges: [], matches_followed: 1 },
  }) });
  await buying;
  assert.equal(state.totalCoins.value, 35);
  assert.equal(state.participationPoints.value, 90);
  assert.equal(state.collection.value.items[0].key, 'club_frame');
  assert.equal(state.shopBusy.value, false);
  assert.ok(state.shopMessage.value);
  await state.redeem(state.catalog.value[0]);
  assert.equal(calls.length, 1, 'an already owned item cannot trigger a second purchase');
});

test('a lost purchase response never subtracts locally and a wallet refresh recovers ownership', async (t) => {
  let purchases = 0;
  const item = { key: 'club_frame', title: 'Cornice club', kind: 'frame', cost: 25, owned: false };
  const { state, scope } = fixture({
    purchaseFanItem: async () => { purchases += 1; return { ok: false }; },
    fetchFanWallet: async () => ({ ok: true, data: walletData({ wallet: 25, points: 50, catalog: [{ ...item, owned: true }],
      collection: { equipped_frame: '', items: [item], badges: [], matches_followed: 1 },
    }) }),
  });
  t.after(() => scope.stop());
  state.applyWallet(walletData({ wallet: 50, points: 50, catalog: [item] }));
  await state.redeem(item);
  assert.equal(state.totalCoins.value, 50);
  assert.equal(state.participationPoints.value, 50);
  assert.deepEqual(state.collection.value.items, []);
  assert.ok(state.walletError.value);
  await state.loadWallet();
  assert.equal(state.totalCoins.value, 25);
  assert.equal(state.participationPoints.value, 50);
  assert.equal(state.collection.value.items[0].key, 'club_frame');
  await state.redeem(state.catalog.value[0]);
  assert.equal(purchases, 1);
});

test('a purchase completed for an earlier match cannot change the new match wallet or collection', async (t) => {
  const purchase = deferred();
  const item = { key: 'match_memory', title: 'Ricordo partita', kind: 'memory', cost: 25, owned: false };
  const { state, props, scope } = fixture({ purchaseFanItem: () => purchase.promise });
  t.after(() => scope.stop());
  state.applyWallet(walletData({ wallet: 50, points: 50, catalog: [item] }));
  const buying = state.redeem(item);
  props.eventId = 43;
  await state.bootstrap();
  purchase.resolve({ ok: true, data: walletData({ wallet: 25, points: 50,
    collection: { equipped_frame: '', items: [item], badges: [], matches_followed: 1 },
  }) });
  await buying;
  assert.equal(state.totalCoins.value, 0);
  assert.equal(state.participationPoints.value, 0);
  assert.deepEqual(state.collection.value.items, []);
  assert.equal(state.shopMessage.value, '');
  assert.equal(state.shopBusy.value, false);
});

test('equipping a frame applies only server confirmation and cannot mutate another organization', async (t) => {
  const requests = [];
  const { state, props, scope } = fixture({ equipFanItem: (...args) => {
    const response = deferred(); requests.push({ args, response }); return response.promise;
  } });
  t.after(() => scope.stop());
  state.applyWallet(walletData({ wallet: 4, points: 80, collection: { equipped_frame: '', items: [{ key: 'club_frame', kind: 'frame' }], badges: [], matches_followed: 1 } }));
  const equipping = state.equipCollectionItem('club_frame');
  void state.equipCollectionItem('club_frame');
  assert.equal(state.collection.value.equipped_frame, '');
  assert.equal(state.shopBusy.value, true);
  assert.deepEqual(requests[0].args, [42, 'club_frame']);
  assert.equal(requests.length, 1);
  requests[0].response.resolve({ ok: true, data: walletData({ wallet: 4, points: 80,
    collection: { equipped_frame: 'club_frame', items: [{ key: 'club_frame', kind: 'frame' }], badges: [], matches_followed: 1 },
  }) });
  await equipping;
  assert.equal(state.collection.value.equipped_frame, 'club_frame');
  assert.equal(state.participationPoints.value, 80);
  const removing = state.equipCollectionItem('');
  props.organizationSlug = 'different-club';
  await state.bootstrap();
  requests[1].response.resolve({ ok: true, data: walletData({ wallet: 900, points: 900 }) });
  await removing;
  assert.equal(state.collection.value.equipped_frame, '');
  assert.equal(state.totalCoins.value, 0);
});

test('game progress records the authoritative server reward instead of the emitted client amount', async (t) => {
  const { state, scope } = fixture();
  t.after(() => scope.stop());
  state.openGame('reaction');
  await state.onGameClaim({ coins: 999 });
  assert.equal(state.totalCoins.value, 10);
  assert.equal(state.progress.value.earned, 10);
  assert.equal(state.progress.value.unlocked, false);
  state.openGame('memory-flash');
  await state.onGameClaim({ coins: 999 });
  assert.equal(state.totalCoins.value, 18);
  assert.equal(state.progress.value.earned, 18);
  assert.equal(state.progress.value.unlocked, false);
});

test('points rankings prefer permanent points over a smaller spendable wallet balance', async (t) => {
  const { state, scope } = fixture({ apiClient: { get: async () => ({ data: {
    top3: [{ rank: 1, name: 'Capitano', points: 90, coins: 1 }],
    userRank: { rank: 2, points: 80, coins: 3 },
    nearby: [{ rank: 1, name: 'Capitano', points: 90, coins: 1 }, { rank: 2, points: 80, coins: 3, is_me: true }], next_rank_gap: 11,
  } }) } });
  t.after(() => scope.stop());
  await state.refreshLeaderboard();
  assert.equal(state.leaderboardTop3.value[0].points, 90);
  assert.equal(state.leaderboardUser.value.points, 80);
  assert.equal(state.leaderboardNearby.value[0].points, 90);
  assert.match(state.rankGapLabel.value, /Mancano 11 punti/);
  state.applyWallet(walletData({ wallet: 0, points: 80 }));
  assert.equal(state.leaderboardUser.value.rank, 2);
  assert.equal(state.leaderboardUser.value.points, 80);
});

test('feedback opens only when configured and submitting it prevents another opening for the match', (t) => {
  const { state, scope, window } = fixture();
  t.after(() => scope.stop());
  state.openFeedbackModal();
  assert.equal(state.isFeedbackModalOpen.value, false);
  state.liveEvent.value = { event_id: 42, is_active: true, feedback_survey: { questions: [{ id: 'atmosphere', text: 'Atmosfera?' }] } };
  assert.equal(state.hasFeedbackSurvey.value, true);
  state.openFeedbackModal();
  assert.equal(state.isFeedbackModalOpen.value, true);
  state.handleFeedbackSubmitted();
  assert.equal(window.localStorage.getItem('wcmvpvs-feedback:42'), '1');
  state.isFeedbackModalOpen.value = false;
  state.openFeedbackModal();
  assert.equal(state.isFeedbackModalOpen.value, false);
});

test('cached wallet balances are isolated by organization and match and ignore the old global cache', (t) => {
  const { state, props, scope, window } = fixture();
  t.after(() => scope.stop());
  window.localStorage.setItem('wallet:coins', '999');
  assert.equal(state.readStoredCoins(), 0);
  state.setCoins(27);
  assert.equal(window.localStorage.getItem('wallet:coins:club:42'), '27');
  props.organizationSlug = 'other-club';
  assert.equal(state.readStoredCoins(), 0);
  state.setCoins(4);
  props.organizationSlug = 'club';
  assert.equal(state.readStoredCoins(), 27);
  props.eventId = 43;
  assert.equal(state.readStoredCoins(), 0);
  state.setCoins(8);
  props.organizationSlug = 'other-club'; props.eventId = 42;
  assert.equal(state.readStoredCoins(), 4);
  assert.equal(window.localStorage.getItem('wallet:coins:club:43'), '8');
});

test('switching organizations clears the previous fan identity even while profile loading is unavailable', async (t) => {
  const { state, props, scope } = fixture({ fetchFanProfile: async () => ({ ok: false }) });
  t.after(() => scope.stop());
  await state.bootstrap();
  state.isRegistered.value = true;
  state.fanNickname.value = 'OldClubFan';
  state.nicknameDraft.value = 'OldClubDraft';
  state.fanLotteryTicket.value = { ticket_code: 'OLD-CLUB' };
  state.setCoins(80);
  props.organizationSlug = 'different-club';
  await state.bootstrap();
  assert.equal(state.totalCoins.value, 0);
  assert.equal(state.isRegistered.value, false);
  assert.equal(state.fanNickname.value, '');
  assert.equal(state.nicknameDraft.value, '');
  assert.equal(state.fanLotteryTicket.value, null);
});

test('an authoritative empty server reward list clears a stale local badge and completed claims', async (t) => {
  const { state, scope, window } = fixture({ fetchFanProfile: async () => ({ ok: true, data: {
    registered: true, wallet: 0, journey_rewards: {},
  } }) });
  t.after(() => scope.stop());
  window.localStorage.setItem(journey.journeyStorageKey('club', 42), JSON.stringify({ rewards: { vote: 10, quiz: 15, basketball: 5 } }));
  state.loadJourney();
  state.doneKeys.value = ['vote', 'quiz', 'basketball'];
  assert.equal(state.progress.value.unlocked, true);
  await state.loadFanProfile();
  assert.equal(state.progress.value.earned, 0);
  assert.equal(state.progress.value.unlocked, false);
  assert.deepEqual(state.journey.value.rewards, {});
  assert.equal(state.doneKeys.value.includes('quiz'), false);
  assert.equal(state.doneKeys.value.includes('basketball'), false);
  const stored = JSON.parse(window.localStorage.getItem(journey.journeyStorageKey('club', 42)));
  assert.deepEqual(stored.rewards, {});
});

test('Tap Challenge single-player claims its fixed eight coins through the journey API once', async (t) => {
  const claims = [];
  const { state, scope } = fixture({
    claimFanJourneyReward: async (...args) => {
      claims.push(args);
      return { ok: true, data: { wallet: 8, points: 8, coins_earned: 8, reward_amount: 8 } };
    },
    syncGuestCoins: async () => { throw new Error('Tap single-player must use the journey reward endpoint'); },
  });
  t.after(() => scope.stop());
  state.openGame('tap');
  await state.onGameClaim({ coins: 8 });
  assert.equal(state.totalCoins.value, 8);
  assert.equal(state.progress.value.earned, 8);
  assert.equal(state.journey.value.rewards.tap, 8);
  assert.equal(state.activeSheet.value, 'vote-result');
  state.openGame('tap');
  await state.onGameClaim({ coins: 8 });
  assert.deepEqual(claims, [[42, 'tap']]);
  assert.equal(state.totalCoins.value, 8);
  assert.equal(state.progress.value.earned, 8);
});

test('Tap live refreshes cumulative server rewards and never trusts or recredits the emitted amount', async (t) => {
  let profiles = 0;
  const { state, scope, saves } = fixture({ fetchFanProfile: async () => {
    const coins = ++profiles === 1 ? 6 : 9;
    return { ok: true, data: { registered: true, wallet: coins, journey_rewards: { tap: coins } } };
  } });
  t.after(() => scope.stop());
  state.openGame('tap');
  await state.onGameClaim({ coins: 999, source: 'tap_challenge_live', keepOpen: true });
  assert.equal(state.totalCoins.value, 6);
  assert.equal(state.progress.value.earned, 6);
  assert.equal(state.journey.value.rewards.tap, 6);
  assert.equal(state.activeGameId.value, 'tap');
  assert.equal(state.pendingGameClaim.value, null);
  assert.equal(saves.length, 0);
  await state.onGameClaim({ coins: 999, source: 'tap_challenge_live', keepOpen: true });
  assert.equal(state.totalCoins.value, 9);
  assert.equal(state.progress.value.earned, 9);
  assert.equal(state.journey.value.rewards.tap, 9);
  assert.equal(state.activeGameId.value, 'tap');
  assert.equal(profiles, 2);
  assert.equal(saves.length, 0);
});
