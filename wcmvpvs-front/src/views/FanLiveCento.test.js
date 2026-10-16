import test from 'node:test';
import assert from 'node:assert/strict';
import * as journey from '../fanJourney.js';
import * as brand from '../organizationTheme.js';
import * as assets from '../fanHomeAssets.js';
import { setupSfc, testVue, testWindow } from '../../test/sfcSetup.js';

const player = { id: 7, name: 'Rossi', lastName: 'Rossi', number: '7', image: '' };

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
      const coins = serverAwards.has(key) ? 0 : key === 'vote' ? 10 : 15;
      serverWallet += coins; serverAwards.add(key); saves.push({ eventId, coins: serverWallet });
      return { ok: true, data: { wallet: serverWallet, coins_earned: coins } };
    },
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

test('game earnings unlock the badge without voting and spending does not undo the achievement', async (t) => {
  const { state, scope } = fixture();
  t.after(() => scope.stop());
  for (const [game, coins] of [['reaction', 10], ['sponsor-rush', 12], ['memory-flash', 8]]) {
    state.openGame(game);
    await state.onGameClaim({ coins });
  }
  assert.equal(state.hasVoted.value, false);
  assert.equal(state.progress.value.earned, 30);
  assert.equal(state.progress.value.unlocked, true);
  await state.onGameSpend({ coins: 20 });
  assert.equal(state.totalCoins.value, 10);
  assert.equal(state.progress.value.current, 25);
  assert.equal(state.progress.value.unlocked, true);
});

test('game earnings advance progress only after saving succeeds, including a retry', async (t) => {
  let attempts = 0;
  const { state, scope } = fixture({ syncGuestCoins: async (_, coins) => ++attempts === 1
    ? { ok: false } : { ok: true, data: { wallet: coins } } });
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
    return requested.length === 1 ? { ok: false } : { ok: true, data: { wallet: coins } };
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

test('the leaderboard sends the fan session and shows an honest gap when coins are tied', async (t) => {
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
  assert.match(state.rankGapLabel.value, /Manca 1 moneta/);
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
  assert.match(state.rankGapLabel.value, /Mancano 8 monete/);
});

test('a retry after a lost response restores progress without re-awarding the vote bonus', async (t) => {
  let attempts = 0;
  const { state, scope } = fixture({ claimFanJourneyReward: async () => ++attempts === 1 ? { ok: false } : { ok: true, data: { wallet: 10, coins_earned: 0 } } });
  t.after(() => scope.stop());
  await state.castVote(player);
  await state.retryVoteBonus();
  assert.equal(state.totalCoins.value, 10);
  assert.equal(state.progress.value.earned, 10);
  assert.equal(state.voteBonusCoins.value, 0);
});
