<template>
  <div class="cento-root" :class="{ 'reduce-motion': false }">
    <div class="phone">
      <!-- Top bar -->
      <header class="topbar">
        <div class="club">
          <div class="club-logo">
            <img v-if="teamLogoUrl" :src="teamLogoUrl" :alt="homeName" />
            <template v-else>{{ homeBadge }}</template>
          </div>
          <div class="club-name">
            <b>{{ homeName }} Live</b>
            <span>{{ venueLabel }}</span>
          </div>
        </div>
        <div class="top-right">
          <button class="wallet" type="button" aria-label="Saldo monete" @click="openSheet('earn')">
            <span class="coin"></span> <span>{{ totalCoins }}</span>
          </button>
          <button class="profile-btn" type="button" aria-label="Profilo" @click="openProfile">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round">
              <circle cx="12" cy="8" r="4" /><path d="M4 21c0-4 3.6-7 8-7s8 3 8 7" />
            </svg>
          </button>
        </div>
      </header>

      <main>
        <!-- Grid -->
        <div class="grid">
          <!-- MVP: sheet inline con giocatori reali + voto reale -->
          <button class="tile tile-mvp" type="button" @click="openSheet('mvp')">
            <b>{{ hasVoted ? 'Il tuo MVP' : "Vota l'MVP del pubblico" }}</b>
            <p>{{ hasVoted ? 'Grazie per il voto!' : 'Il tuo voto vale +10 monete' }}</p>
            <div class="mvp-photo">
              <span class="status">{{ hasVoted ? 'Voto registrato' : 'Votazioni aperte' }}</span>
              <img v-if="hasVoted && displayVotedImage" :src="displayVotedImage" :alt="displayVotedName" class="mvp-img" />
              <svg v-else width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#E4212E" stroke-width="1.5">
                <circle cx="12" cy="8" r="4" /><path d="M4 21c0-4 3.6-7 8-7s8 3 8 7" />
              </svg>
            </div>
            <span class="btn-vote">{{ hasVoted ? votedButtonLabel : 'Vota ora' }}</span>
          </button>

          <button class="tile" type="button" @click="openSheet('earn')">
            <div class="tile-icon">
              <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2 4 14h6l-1 8 9-12h-6l1-8z" /></svg>
            </div>
            <b>Guadagna</b>
            <p>Quiz e sfide sponsor</p>
            <div class="spacer"></div>
            <span class="tile-cta">Gioca →</span>
          </button>

          <button class="tile" type="button" @click="openSheet('spend')">
            <div class="tile-icon">
              <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 12v8H4v-8M2 7h20v5H2zM12 7v13" /></svg>
            </div>
            <b>Spendi</b>
            <p>Premi e merch del club</p>
            <div class="spacer"></div>
            <span class="tile-cta">Riscatta →</span>
          </button>
        </div>

        <!-- Leaderboard tile -->
        <button class="tile lb-tile" type="button" @click="openLeaderboard">
          <div class="lb-head">
            <b>Classifica tifosi</b>
            <span class="tile-cta">Vedi tutti →</span>
          </div>
          <div class="lb-mini">
            <div v-for="(row, i) in leaderboardTop3" :key="i" class="lb-mini-row">
              <span class="rank" :class="`r${i + 1}`">{{ i + 1 }}</span>
              <span class="lb-mini-name">{{ row.name }}</span>
              <span class="lb-mini-pts">{{ row.coins }}</span>
            </div>
            <div v-if="!leaderboardTop3.length" class="lb-empty">Gioca e vota per popolare la classifica</div>
          </div>
        </button>

        <!-- Sponsor rush → apre lo sheet Guadagna -->
        <button v-if="rushSponsor" class="rush" type="button" @click="openSheet('earn')">
          <span class="rush-icon">{{ rushSponsor.badge }}</span>
          <span class="rush-copy">
            <b>Sponsor Rush · {{ rushSponsor.name }}</b>
            <span>15 secondi, monete subito</span>
          </span>
          <span class="rush-reward">+50 <span class="coin"></span></span>
        </button>

        <!-- Sponsors (spazio ampliato al posto dello scoreboard) -->
        <section v-if="sponsors.length" class="sponsors">
          <div class="sponsors-head">
            <span class="label">PARTNER UFFICIALI</span>
          </div>
          <div class="sponsor-grid">
            <button
              v-for="sponsor in sponsors.slice(0, 6)"
              :key="sponsor.id"
              class="sponsor-chip"
              type="button"
              @click="onSponsorClick(sponsor)"
            >
              <img v-if="sponsor.imageUrl" :src="sponsor.imageUrl" :alt="sponsor.name || 'Sponsor'" />
              <span v-else>{{ sponsor.name }}</span>
            </button>
          </div>
        </section>
      </main>

      <!-- Overlay + spend sheet (inline, stile cento) -->
      <div class="overlay" :class="{ open: activeSheet }" @click="closeSheets"></div>

      <!-- MVP: lista giocatori reale -->
      <div class="sheet" :class="{ open: activeSheet === 'mvp' }" role="dialog" aria-label="Vota MVP">
        <div class="grabber"></div>
        <div class="sheet-head">
          <div>
            <div class="sheet-title">Vota l'MVP del pubblico</div>
            <div class="sheet-sub">Un voto per partita · +10 monete</div>
          </div>
          <button class="close" type="button" @click="closeSheets">✕</button>
        </div>
        <div class="sheet-body">
          <div v-if="isLoadingPlayers" class="sheet-empty">Caricamento giocatori…</div>
          <div v-else-if="playersError" class="sheet-empty error">{{ playersError }}</div>
          <div v-else-if="!players.length" class="sheet-empty">Nessun giocatore disponibile.</div>
          <button
            v-for="player in players"
            :key="player.id"
            class="row-item"
            type="button"
            :disabled="isVoting"
            @click="castVote(player)"
          >
            <span class="avatar">
              <img v-if="player.image" :src="player.image" :alt="player.name" />
              <template v-else>{{ player.number || player.badge }}</template>
            </span>
            <span class="row-copy"><b>{{ player.name }}</b><span>{{ player.subtitle }}</span></span>
            <span class="chip-mint">{{ votedPlayerId === player.id ? 'Votato' : 'Vota' }}</span>
          </button>
        </div>
      </div>

      <!-- Guadagna: missioni del match -->
      <div class="sheet" :class="{ open: activeSheet === 'earn' }" role="dialog" aria-label="Guadagna monete">
        <div class="grabber"></div>
        <div class="sheet-head">
          <div>
            <div class="sheet-title">Guadagna monete</div>
            <div class="sheet-sub">Completa le missioni del match</div>
          </div>
          <button class="close" type="button" @click="closeSheets">✕</button>
        </div>
        <div class="sheet-body">
          <button
            v-for="mission in earnMissions"
            :key="mission.key"
            class="row-item"
            type="button"
            @click="onMission(mission)"
          >
            <span class="avatar">{{ mission.emoji }}</span>
            <span class="row-copy"><b>{{ mission.title }}</b><span>{{ mission.subtitle }}</span></span>
            <span class="row-pts">
              <template v-if="mission.key === 'vote' && hasVoted">✓</template>
              <template v-else>+{{ mission.pts }} <span class="coin"></span></template>
            </span>
          </button>
        </div>
      </div>

      <div class="sheet" :class="{ open: activeSheet === 'spend' }" role="dialog" aria-label="Spendi monete">
        <div class="grabber"></div>
        <div class="sheet-head">
          <div>
            <div class="sheet-title">Spendi le tue monete</div>
            <div class="sheet-sub">Premi offerti dal club e dagli sponsor</div>
          </div>
          <button class="close" type="button" @click="closeSheets">✕</button>
        </div>
        <div class="sheet-body">
          <button
            v-for="reward in rewards"
            :key="reward.key"
            class="row-item"
            type="button"
            :disabled="redeeming === reward.key || totalCoins < reward.cost"
            @click="redeem(reward)"
          >
            <span class="avatar">{{ reward.emoji }}</span>
            <span class="row-copy"><b>{{ reward.title }}</b><span>{{ reward.subtitle }}</span></span>
            <span class="row-pts">{{ reward.cost }} <span class="coin"></span></span>
          </button>
        </div>
      </div>

      <!-- Profilo (come LiveExperienceHome: nickname + saldo + QR lotteria) -->
      <div class="sheet" :class="{ open: activeSheet === 'profile' }" role="dialog" aria-label="Profilo">
        <div class="grabber"></div>
        <div class="sheet-head">
          <div>
            <div class="sheet-title">Il tuo profilo</div>
            <div class="sheet-sub">{{ isRegistered ? 'Account tifoso' : 'Ospite · non registrato' }}</div>
          </div>
          <button class="close" type="button" @click="closeSheets">✕</button>
        </div>
        <div class="sheet-body">
          <div class="profile-hero">
            <div class="profile-avatar">
              <svg width="34" height="34" viewBox="0 0 24 24" fill="none" stroke="#E4212E" stroke-width="1.8" stroke-linecap="round">
                <circle cx="12" cy="8" r="4" /><path d="M4 21c0-4 3.6-7 8-7s8 3 8 7" />
              </svg>
            </div>
            <div>
              <div class="profile-label">Nickname</div>
              <div class="profile-nick">{{ profileNickname }}</div>
            </div>
          </div>

          <button class="row-item profile-toggle" type="button" @click="toggleNicknameEditor">
            <b>Modifica nickname</b>
          </button>
          <div v-if="isNicknameEditorOpen" class="nick-editor">
            <input
              v-model.trim="nicknameDraft"
              type="text"
              minlength="3"
              maxlength="24"
              placeholder="Inserisci nickname"
              class="nick-input"
              @keyup.enter="submitNickname"
            />
            <button
              class="chip-mint nick-save"
              type="button"
              :disabled="isSavingNickname || !canSubmitNickname"
              @click="submitNickname"
            >
              {{ isSavingNickname ? 'Salvataggio…' : 'Aggiorna' }}
            </button>
          </div>
          <p v-if="nicknameError" class="nick-msg error">{{ nicknameError }}</p>
          <p v-else-if="nicknameSuccess" class="nick-msg ok">{{ nicknameSuccess }}</p>

          <div class="coin-balance">
            <span class="coin-balance-label">Saldo monete</span>
            <span class="coin-balance-val">{{ totalCoins }} <span class="coin"></span></span>
          </div>

          <div class="lottery-box">
            <div class="lottery-title">QR Lotteria MVP</div>
            <template v-if="lotteryCode">
              <div class="lottery-code">{{ lotteryCode }}</div>
              <img v-if="lotteryQrUrl" :src="lotteryQrUrl" alt="QR lotteria" class="lottery-qr" />
              <p class="lottery-hint">Resta fino a fine partita per ritirare il premio.</p>
            </template>
            <p v-else class="lottery-empty">Vota l'MVP per ottenere il tuo QR lotteria personale.</p>
          </div>
        </div>
      </div>

      <!-- Overlay gioco a schermo pieno -->
      <div v-if="activeGameId" class="game-overlay">
        <header class="game-head">
          <button class="close" type="button" aria-label="Indietro" @click="closeGame">←</button>
          <div class="game-title">{{ games[activeGameId]?.title }}</div>
          <button class="close" type="button" aria-label="Chiudi" @click="closeGame">✕</button>
        </header>
        <div class="game-stage">
          <ReactionTestGame
            v-if="activeGameId === 'reaction'"
            class="game-fill"
            @claim="onGameClaim"
            @exit="closeGame"
          />
          <QuickQuizGame
            v-else-if="activeGameId === 'quiz'"
            class="game-fill"
            :event-id="eventId"
            @claim="onGameClaim"
            @exit="closeGame"
          />
          <TapChallenge
            v-else-if="activeGameId === 'tap'"
            class="game-fill"
            :event-id="eventId"
            @claim="onGameClaim"
            @exit="closeGame"
          />
          <SponsorRushGame
            v-else-if="activeGameId === 'sponsor-rush'"
            class="game-fill"
            @claim="onGameClaim"
            @exit="closeGame"
          />
          <MemoryFlashGame
            v-else-if="activeGameId === 'memory-flash'"
            class="game-fill"
            :wallet-coins="totalCoins"
            @claim="onGameClaim"
            @spend="onGameSpend"
            @exit="closeGame"
          />
        </div>
      </div>

    </div>

    <!-- Classifica tifosi (componente reale) -->
    <FansLeaderboardModal
      v-model="isLeaderboardOpen"
      :top-list="leaderboardTop3"
      :user-rank="leaderboardUser"
    />
  </div>
</template>

<script setup>
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import FansLeaderboardModal from '../components/FansLeaderboardModal.vue';

// Giochi reali riusati da EarnCoinsModal (lazy-load: pesano).
const ReactionTestGame = defineAsyncComponent(() => import('../components/ReactionTestGame.vue'));
const QuickQuizGame = defineAsyncComponent(() => import('../components/QuickQuizGame.vue'));
const TapChallenge = defineAsyncComponent(() => import('../components/minigames/TapChallenge.vue'));
const SponsorRushGame = defineAsyncComponent(() => import('../components/minigames/SponsorRushGame.vue'));
const MemoryFlashGame = defineAsyncComponent(() => import('../components/minigames/MemoryFlashGame.vue'));
import {
  apiClient,
  fetchFanProfile,
  fetchVoteStatus,
  vote,
  syncGuestCoins,
  redeemFanReward,
  updateFanNickname,
  resolveApiUrl,
  getOrganizationSlug,
} from '../api';

const FAN_NICKNAME_MIN_LEN = 3;
const FAN_NICKNAME_MAX_LEN = 24;
const FAN_NICKNAME_PATTERN = /^[A-Za-z0-9._ -]+$/;

const props = defineProps({
  eventId: { type: Number, default: 0 },
  teamName: { type: String, default: 'TEAM' },
  teamLogoUrl: { type: String, default: '' },
  matchLabel: { type: String, default: 'Vota • Gioca • Vinci • Partecipa' },
  activeEvent: { type: Object, default: null },
  votedPlayerImageUrl: { type: String, default: '' },
  votedPlayerName: { type: String, default: '' },
  votedPlayerLastName: { type: String, default: '' },
  votedPlayerNumber: { type: [String, Number], default: '' },
  // Segnale dal parent dopo il primo voto (registrazione). Accettato per compat; TODO: prompt registrazione.
  registrationPromptSignal: { type: Number, default: 0 },
});

const WALLET_STORAGE_KEY = 'wallet:coins';

/* ---------- Stato ---------- */
const totalCoins = ref(0);
const isRegistered = ref(false);
const fanNickname = ref('');
const fanLotteryTicket = ref(null);
const nicknameDraft = ref('');
const isNicknameEditorOpen = ref(false);
const isSavingNickname = ref(false);
const nicknameError = ref('');
const nicknameSuccess = ref('');
const hasVoted = ref(false);
const votedPlayerId = ref(null);
const localVoted = ref(null); // giocatore votato in questa sessione (feedback immediato)
const leaderboardTop3 = ref([]);
const leaderboardUser = ref(null);
const sponsors = ref([]);

const rawPlayers = ref([]);
const isLoadingPlayers = ref(false);
const playersError = ref('');
const isVoting = ref(false);

const doneKeys = ref([]); // missioni completate (persistite per evento)

const activeSheet = ref('');
const activeGameId = ref(null);
const isLeaderboardOpen = ref(false);

// Registry giochi reali (titolo + reward indicativa mostrata nella lista).
const games = {
  reaction: { title: 'Reaction Test', reward: 10 },
  quiz: { title: 'Quiz Lampo', reward: 15 },
  tap: { title: 'Tap Challenge', reward: 8 },
  'sponsor-rush': { title: 'Sponsor Rush', reward: 12 },
  'memory-flash': { title: 'Memory Flash', reward: 8 },
};
const redeeming = ref('');
let coinsStream = null;

/* ---------- Derivati header ---------- */
function initials(name) {
  const parts = String(name || '').trim().split(/\s+/).filter(Boolean);
  if (!parts.length) return '—';
  return parts.slice(0, 3).map((p) => p[0]).join('').toUpperCase();
}
const homeName = computed(() =>
  String(props.activeEvent?.team1_name || props.teamName || 'TEAM').trim(),
);
const homeBadge = computed(() => initials(homeName.value));
const venueLabel = computed(() =>
  String(props.activeEvent?.venue || 'Esperienza ufficiale').trim() || 'Esperienza ufficiale',
);

/* ---------- Giocatore votato (locale o da prop parent) ---------- */
const displayVotedImage = computed(() => localVoted.value?.image || props.votedPlayerImageUrl);
const displayVotedName = computed(() => localVoted.value?.name || props.votedPlayerName);
const votedButtonLabel = computed(() => {
  const number = localVoted.value?.number ?? props.votedPlayerNumber;
  const num = number === '' || number == null ? '' : `#${number} `;
  const label = localVoted.value?.lastName || props.votedPlayerLastName || localVoted.value?.name || props.votedPlayerName;
  return label ? `${num}${label}`.trim() : 'Modifica voto';
});

/* ---------- Profilo (come LiveExperienceHome) ---------- */
const profileNickname = computed(() => {
  if (fanNickname.value.trim()) return fanNickname.value.trim();
  return isRegistered.value ? 'Tifoso' : 'Ospite';
});
const canSubmitNickname = computed(
  () => !isSavingNickname.value && nicknameDraft.value.trim() !== fanNickname.value.trim(),
);
const lotteryCode = computed(() => String(fanLotteryTicket.value?.ticket_code || '').trim());
const lotteryQrUrl = computed(() => {
  const qr = String(fanLotteryTicket.value?.qr_data || '').trim();
  return qr ? `https://api.qrserver.com/v1/create-qr-code/?size=260x260&data=${encodeURIComponent(qr)}` : '';
});

/* ---------- Giocatori (lista MVP reale) ---------- */
const players = computed(() => {
  const list = Array.isArray(rawPlayers.value) ? rawPlayers.value : [];
  const calledUp = list.filter((p) => p?.is_called_up === true);
  const source = calledUp.length ? calledUp : list;
  return source.map((p) => {
    const first = String(p?.first_name || '').trim();
    const last = String(p?.last_name || '').trim();
    const name = last ? `${first ? `${first[0]}. ` : ''}${last}` : first || 'Giocatore';
    const role = String(p?.role || p?.position || '').trim();
    const number = p?.jersey_number == null ? '' : String(p.jersey_number);
    return {
      id: Number(p?.id),
      name,
      lastName: last,
      subtitle: role || (number ? `Maglia #${number}` : 'In campo'),
      number,
      badge: initials(`${first} ${last}`),
      image: String(p?.image_url || '').trim(),
    };
  });
});

/* ---------- Missioni "Guadagna" (aprono i giochi reali) ---------- */
const earnMissions = computed(() => [
  { key: 'vote', emoji: '🏆', title: "Vota l'MVP", subtitle: 'Un voto per partita', pts: 10, opensMvp: true },
  { key: 'reaction', emoji: '⚡', title: 'Reaction Test', subtitle: 'Testa i riflessi', pts: 10, game: 'reaction' },
  { key: 'quiz', emoji: '🧠', title: 'Quiz Lampo', subtitle: 'Domande sul match', pts: 15, game: 'quiz' },
  { key: 'tap', emoji: '👆', title: 'Tap Challenge', subtitle: '10 secondi a martello', pts: 8, game: 'tap' },
  {
    key: 'sponsor-rush',
    emoji: '🏷️',
    title: rushSponsor.value ? `Sponsor Rush · ${rushSponsor.value.name}` : 'Sponsor Rush',
    subtitle: 'Prendi i loghi al volo',
    pts: 12,
    game: 'sponsor-rush',
  },
  { key: 'memory-flash', emoji: '🧩', title: 'Memory Flash', subtitle: 'Memorizza le coppie', pts: 8, game: 'memory-flash' },
]);

/* ---------- Sponsor rush / partner ---------- */
const rushSponsor = computed(() => {
  const first = sponsors.value[0];
  if (!first) return null;
  return { name: first.name, badge: initials(first.name).slice(0, 1) };
});

/* ---------- Premi (riscatto reale via /rewards/redeem) ---------- */
const rewards = ref([
  { key: 'sponsor_discount', emoji: '🎁', title: 'Sconto 10% · Partner', subtitle: 'Codice valido 7 giorni', cost: 80 },
  { key: 'signed_ball', emoji: '🏀', title: 'Pallone autografato', subtitle: 'Estrazione a fine partita', cost: 150 },
  { key: 'next_ticket', emoji: '🎟️', title: 'Biglietto prossimo match', subtitle: 'Settore tifosi', cost: 300 },
]);

/* ---------- Sheets ---------- */
function openSheet(id) {
  activeSheet.value = id;
}
function closeSheets() {
  activeSheet.value = '';
}

/* ---------- Wallet ---------- */
function readStoredCoins() {
  try {
    const raw = window.localStorage.getItem(WALLET_STORAGE_KEY);
    const parsed = Number.parseInt(raw ?? '', 10);
    return Number.isFinite(parsed) && parsed >= 0 ? parsed : 0;
  } catch (error) {
    return 0;
  }
}
function writeStoredCoins(value) {
  try {
    window.localStorage.setItem(WALLET_STORAGE_KEY, String(value));
  } catch (error) {
    /* localStorage non disponibile */
  }
}
function setCoins(value) {
  const next = Math.max(0, Math.round(Number(value) || 0));
  totalCoins.value = next;
  writeStoredCoins(next);
}

async function loadFanProfile() {
  if (!props.eventId) return;
  const { ok, data } = await fetchFanProfile(props.eventId);
  if (!ok || !data) return;
  isRegistered.value = Boolean(data.registered);
  if (data.registered) {
    setCoins(Number(data.wallet) || 0);
  } else {
    setCoins(Math.max(totalCoins.value, Number(data.guest_coins) || 0));
  }
  fanNickname.value = String(data.user?.nickname || '').trim();
  if (!isNicknameEditorOpen.value) {
    nicknameDraft.value = fanNickname.value;
  }
  fanLotteryTicket.value = data.lottery_ticket || null;
  if (data.user_rank) {
    leaderboardUser.value = data.user_rank;
  }
}

// Accredita monete sul wallet reale (localStorage + sync guest).
async function grantCoins(delta) {
  const amount = Number(delta) || 0;
  if (!amount) return;
  setCoins(totalCoins.value + amount);
  if (props.eventId) {
    await syncGuestCoins(props.eventId, totalCoins.value);
  }
  refreshLeaderboard();
}

/* ---------- Missioni completate (persistite per evento) ---------- */
function doneStorageKey() {
  return `cento:done:${props.eventId || 'anon'}`;
}
function loadDoneKeys() {
  try {
    const raw = window.localStorage.getItem(doneStorageKey());
    const parsed = raw ? JSON.parse(raw) : [];
    doneKeys.value = Array.isArray(parsed) ? parsed : [];
  } catch (error) {
    doneKeys.value = [];
  }
}
function markDone(key) {
  if (doneKeys.value.includes(key)) return;
  doneKeys.value = [...doneKeys.value, key];
  try {
    window.localStorage.setItem(doneStorageKey(), JSON.stringify(doneKeys.value));
  } catch (error) {
    /* localStorage non disponibile */
  }
}

function onMission(mission) {
  // "Vota l'MVP" apre la lista giocatori; le monete arrivano col voto reale.
  if (mission.opensMvp) {
    openSheet('mvp');
    return;
  }
  if (mission.game) {
    openGame(mission.game);
  }
}

/* ---------- Giochi reali (overlay a schermo pieno) ---------- */
function openGame(id) {
  if (!games[id]) return;
  activeSheet.value = '';
  activeGameId.value = id;
}
function closeGame() {
  const wasGame = Boolean(activeGameId.value);
  activeGameId.value = null;
  if (wasGame) {
    openSheet('earn'); // torna alla lista giochi
  }
}
// Ogni gioco emette @claim { coins, keepOpen } al termine.
async function onGameClaim(payload) {
  const coins = Math.max(0, Number(payload?.coins) || 0);
  if (coins) {
    await grantCoins(coins);
  }
  if (!payload?.keepOpen) {
    closeGame();
  }
}
// Memory Flash può spendere monete (@spend).
async function onGameSpend(payload) {
  const coins = Math.max(0, Number(payload?.coins) || 0);
  if (!coins) return;
  setCoins(totalCoins.value - coins);
  if (props.eventId) {
    await syncGuestCoins(props.eventId, totalCoins.value);
  }
}

/* ---------- Voto MVP (lista reale + voto reale) ---------- */
async function loadPlayers() {
  isLoadingPlayers.value = true;
  playersError.value = '';
  try {
    const { data } = await apiClient.get('/public/players');
    const payload = Array.isArray(data?.players) ? data.players : Array.isArray(data) ? data : [];
    rawPlayers.value = Array.isArray(payload) ? payload : [];
  } catch (error) {
    playersError.value = 'Impossibile caricare i giocatori. Riprova.';
    rawPlayers.value = [];
  } finally {
    isLoadingPlayers.value = false;
  }
}
async function loadVoteStatus() {
  if (!props.eventId) return;
  const status = await fetchVoteStatus(props.eventId);
  hasVoted.value = Boolean(status?.hasVoted);
  votedPlayerId.value = status?.playerId ?? null;
}
async function castVote(player) {
  if (isVoting.value || !player?.id || !props.eventId) return;
  isVoting.value = true;
  try {
    const response = await vote({ eventId: props.eventId, playerId: player.id });
    if (!response?.ok) {
      return;
    }
    hasVoted.value = true;
    votedPlayerId.value = player.id;
    localVoted.value = { name: player.name, lastName: player.lastName, number: player.number, image: player.image };
    closeSheets();
    // +10 monete solo al primo voto del match.
    if (!doneKeys.value.includes('vote')) {
      markDone('vote');
      await grantCoins(10);
    }
  } catch (error) {
    // voto non riuscito: nessuna azione
  } finally {
    isVoting.value = false;
  }
}

/* ---------- Leaderboard ---------- */
async function refreshLeaderboard() {
  if (!props.eventId) return;
  try {
    const { data } = await apiClient.get(`/events/${props.eventId}/coins-leaderboard`);
    const top = Array.isArray(data?.top3) ? data.top3 : Array.isArray(data?.top) ? data.top : [];
    leaderboardTop3.value = top.slice(0, 3).map((row) => ({
      name: String(row?.name || row?.nickname || 'Tifoso'),
      coins: Number(row?.coins ?? row?.points ?? 0),
    }));
    const rank = data?.userRank || data?.user_rank;
    if (rank) {
      leaderboardUser.value = rank;
    }
  } catch (error) {
    /* placeholder mostrato in assenza di dati */
  }
}
function openLeaderboard() {
  refreshLeaderboard();
  isLeaderboardOpen.value = true;
}

/* ---------- Profilo ---------- */
function openProfile() {
  nicknameDraft.value = fanNickname.value.trim();
  isNicknameEditorOpen.value = false;
  nicknameError.value = '';
  nicknameSuccess.value = '';
  loadFanProfile();
  openSheet('profile');
}
function toggleNicknameEditor() {
  isNicknameEditorOpen.value = !isNicknameEditorOpen.value;
  if (!isNicknameEditorOpen.value) {
    nicknameError.value = '';
    nicknameSuccess.value = '';
  }
}
function validateNickname(raw) {
  const n = String(raw || '').trim();
  if (!n) return { valid: false, message: 'Il nickname non può essere vuoto.' };
  if (n.length < FAN_NICKNAME_MIN_LEN) return { valid: false, message: 'Il nickname deve avere almeno 3 caratteri.' };
  if (n.length > FAN_NICKNAME_MAX_LEN) return { valid: false, message: 'Il nickname può avere massimo 24 caratteri.' };
  if (!FAN_NICKNAME_PATTERN.test(n)) return { valid: false, message: 'Usa solo lettere, numeri, spazio, punto, trattino o underscore.' };
  return { valid: true, nickname: n };
}
async function submitNickname() {
  nicknameError.value = '';
  nicknameSuccess.value = '';
  const check = validateNickname(nicknameDraft.value);
  if (!check.valid) {
    nicknameError.value = check.message;
    return;
  }
  if (check.nickname === fanNickname.value.trim()) {
    nicknameSuccess.value = 'Nessuna modifica da salvare.';
    return;
  }
  isSavingNickname.value = true;
  const response = await updateFanNickname(check.nickname);
  isSavingNickname.value = false;
  if (!response?.ok) {
    nicknameError.value = response?.message || 'Impossibile aggiornare il nickname.';
    return;
  }
  fanNickname.value = String(response.data?.user?.nickname || check.nickname).trim();
  nicknameDraft.value = fanNickname.value;
  nicknameSuccess.value = response.data?.message || 'Nickname aggiornato';
  isNicknameEditorOpen.value = false;
  refreshLeaderboard();
}

/* ---------- Sponsors ---------- */
// Stessa normalizzazione di LiveExperienceHome: sponsor = logo (logo_data/image_url).
function normalizeSponsor(item, index) {
  const imageUrl = String(item?.logo_data || item?.image_url || item?.imageUrl || '').trim();
  if (!imageUrl) return null;
  const priorityRaw = Number(item?.priority ?? item?.order_index ?? item?.order ?? item?.display_order);
  return {
    id: Number(item?.id) || index + 1,
    name: String(item?.name || '').trim(),
    imageUrl,
    linkUrl: String(item?.link_url || item?.linkUrl || '').trim(),
    priority: Number.isFinite(priorityRaw) ? priorityRaw : Number.POSITIVE_INFINITY,
    insertedIndex: index,
  };
}
async function loadSponsors() {
  try {
    const { data } = await apiClient.get('/sponsors');
    const list = Array.isArray(data) ? data : Array.isArray(data?.sponsors) ? data.sponsors : [];
    sponsors.value = list
      .map((item, index) => normalizeSponsor(item, index))
      .filter(Boolean)
      .sort((a, b) => (a.priority !== b.priority ? a.priority - b.priority : a.insertedIndex - b.insertedIndex));
  } catch (error) {
    sponsors.value = [];
  }
}
function onSponsorClick(sponsor) {
  if (props.eventId && sponsor?.id) {
    apiClient
      .post(`/events/${props.eventId}/sponsors/${sponsor.id}/click`, { at: new Date().toISOString() })
      .catch(() => {});
  }
  if (sponsor?.linkUrl) {
    window.open(sponsor.linkUrl, '_blank', 'noopener');
  }
}

/* ---------- Riscatto premi ---------- */
async function redeem(reward) {
  if (redeeming.value) return;
  if (totalCoins.value < reward.cost) {
    return;
  }
  redeeming.value = reward.key;
  try {
    const { ok, data } = await redeemFanReward(props.eventId, reward.key, reward.cost);
    if (ok && data && data.wallet != null) {
      setCoins(Number(data.wallet));
    } else if (ok) {
      setCoins(totalCoins.value - reward.cost);
    } else {
      return;
    }
    closeSheets();
  } catch (error) {
    // riscatto non riuscito: nessuna azione
  } finally {
    redeeming.value = '';
  }
}

/* ---------- SSE classifica live ---------- */
function startCoinsStream() {
  if (!props.eventId || typeof window === 'undefined' || typeof EventSource === 'undefined') return;
  try {
    const slug = getOrganizationSlug();
    const base = resolveApiUrl(`/events/${props.eventId}/coins/stream`);
    const url = slug ? `${base}${base.includes('?') ? '&' : '?'}organization_slug=${encodeURIComponent(slug)}` : base;
    coinsStream = new EventSource(url);
    coinsStream.onmessage = () => refreshLeaderboard();
    coinsStream.onerror = () => {
      /* il browser ritenta da solo */
    };
  } catch (error) {
    coinsStream = null;
  }
}
function stopCoinsStream() {
  if (coinsStream) {
    coinsStream.close();
    coinsStream = null;
  }
}

/* ---------- Esc per chiudere ---------- */
function onKeydown(e) {
  if (e.key !== 'Escape') return;
  if (activeGameId.value) {
    closeGame();
    return;
  }
  closeSheets();
}

/* ---------- Lifecycle ---------- */
function bootstrap() {
  totalCoins.value = readStoredCoins();
  loadDoneKeys();
  loadFanProfile();
  loadVoteStatus();
  loadPlayers();
  refreshLeaderboard();
  loadSponsors();
  stopCoinsStream();
  startCoinsStream();
}

function ensureCentoFonts() {
  if (typeof document === 'undefined') return;
  const id = 'cento-fonts';
  if (document.getElementById(id)) return;
  const link = document.createElement('link');
  link.id = id;
  link.rel = 'stylesheet';
  link.href =
    'https://fonts.googleapis.com/css2?family=Fraunces:opsz,wght@9..144,600;9..144,750;9..144,900&family=Inter:wght@400;500;600;700;800&display=swap';
  document.head.appendChild(link);
}

onMounted(() => {
  ensureCentoFonts();
  window.addEventListener('keydown', onKeydown);
  bootstrap();
});

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown);
  stopCoinsStream();
});

watch(
  () => props.eventId,
  () => bootstrap(),
);

// Il parent aggiorna le prop del giocatore votato dopo il voto reale.
watch(
  () => props.votedPlayerName,
  (name) => {
    if (name) hasVoted.value = true;
  },
);
</script>

<style scoped>
.cento-root {
  position: fixed;
  inset: 0;
  z-index: 1;
  display: flex;
  justify-content: center;
  overflow: hidden;
  background: #f3e9e4;
  --bg: #fff7f2;
  --card: #ffffff;
  --ink: #23161a;
  --muted: #8c7a80;
  --red: #e4212e;
  --red-deep: #c0121f;
  --red-soft: #ffe8e6;
  --red-line: #ffd2ce;
  --gold: #f5b321;
  --gold-deep: #d18e00;
  --gold-soft: #fff3d6;
  --line: #f1e4e0;
  --shadow: 0 3px 0 rgba(35, 22, 26, 0.1);
  --shadow-red: 0 4px 0 var(--red-deep);
  color: var(--ink);
  font-family: 'Inter', system-ui, sans-serif;
}
.cento-root * {
  box-sizing: border-box;
}

.phone {
  width: 100%;
  max-width: 430px;
  height: 100dvh;
  background:
    radial-gradient(90% 40% at 50% -6%, #ffe3de, transparent 62%),
    var(--bg);
  display: flex;
  flex-direction: column;
  position: relative;
  overflow: hidden;
}

/* Top bar */
.topbar {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
}
.club {
  display: flex;
  align-items: center;
  gap: 9px;
}
.club-logo {
  width: 36px;
  height: 36px;
  border-radius: 12px;
  background: var(--red);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Fraunces', serif;
  font-weight: 900;
  font-size: 14px;
  color: #fff;
  box-shadow: var(--shadow-red);
  transform: rotate(-4deg);
  overflow: hidden;
}
.club-logo img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.club-name {
  line-height: 1.15;
}
.club-name b {
  font-size: 13px;
  font-weight: 800;
  display: block;
  letter-spacing: -0.01em;
}
.club-name span {
  font-size: 10px;
  color: var(--muted);
  font-weight: 600;
}
.wallet {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--gold-soft);
  border: 2px solid var(--gold);
  border-radius: 999px;
  padding: 6px 13px 6px 8px;
  font-weight: 800;
  font-size: 14px;
  font-family: inherit;
  font-variant-numeric: tabular-nums;
  color: var(--gold-deep);
  cursor: pointer;
  box-shadow: 0 3px 0 rgba(209, 142, 0, 0.25);
}
.coin {
  width: 17px;
  height: 17px;
  border-radius: 50%;
  flex: none;
  background: radial-gradient(circle at 34% 30%, #ffe293, #f0a81c 68%, #c67f00);
  box-shadow: inset 0 -1.5px 2px rgba(0, 0, 0, 0.22), 0 0 0 2px #fff;
}
.top-right {
  display: flex;
  align-items: center;
  gap: 8px;
}
.profile-btn {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  flex: none;
  background: #fff;
  border: 2px solid var(--red-line);
  color: var(--red);
  cursor: pointer;
  font-family: inherit;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: var(--shadow);
  transition: transform 0.12s ease, border-color 0.15s ease;
}
.profile-btn:active {
  transform: translateY(2px);
  box-shadow: none;
}
.profile-btn:hover {
  border-color: var(--red);
}

main {
  flex: 1;
  min-height: 0;
  padding: 6px 12px 10px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.4; transform: scale(0.75); }
}

/* Grid */
.grid {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 1.15fr 1fr;
  grid-template-rows: 1fr 1fr;
  gap: 10px;
}
.tile {
  background: var(--card);
  border: 2px solid var(--line);
  border-radius: 20px;
  padding: 13px;
  display: flex;
  flex-direction: column;
  text-align: left;
  color: var(--ink);
  font-family: inherit;
  cursor: pointer;
  transition: transform 0.12s ease, border-color 0.15s ease;
  min-height: 0;
  overflow: hidden;
  box-shadow: var(--shadow);
}
.tile:active {
  transform: scale(0.98) translateY(2px);
  box-shadow: none;
}
.tile:hover {
  border-color: var(--red-line);
}
.tile-mvp {
  grid-row: 1 / 3;
  border-color: var(--red-line);
  background: linear-gradient(175deg, #fff 55%, var(--red-soft));
  position: relative;
}
.tile-icon {
  width: 34px;
  height: 34px;
  border-radius: 11px;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--red-soft);
  color: var(--red);
  margin-bottom: 9px;
  transform: rotate(-4deg);
}
.tile b {
  font-size: 14px;
  font-weight: 800;
  letter-spacing: -0.01em;
}
.tile p {
  font-size: 11px;
  color: var(--muted);
  line-height: 1.4;
  margin-top: 3px;
  font-weight: 500;
}
.tile .spacer {
  flex: 1;
  min-height: 4px;
}
.tile-cta {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 800;
  color: var(--red);
}
.mvp-photo {
  flex: 1;
  min-height: 0;
  margin: 9px 0;
  border-radius: 14px;
  background:
    radial-gradient(80% 80% at 50% 20%, #ffdcd8, #ffc7c1),
    var(--red-soft);
  border: 2px solid var(--red-line);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}
.mvp-photo svg {
  opacity: 0.55;
}
.mvp-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.status {
  position: absolute;
  top: 8px;
  left: 8px;
  font-size: 9.5px;
  font-weight: 800;
  color: #fff;
  background: var(--red);
  padding: 4px 9px;
  border-radius: 999px;
  display: flex;
  align-items: center;
  gap: 5px;
}
.status::before {
  content: '';
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #fff;
  animation: pulse 1.5s ease-in-out infinite;
}
.btn-vote {
  border: none;
  border-radius: 13px;
  padding: 12px;
  width: 100%;
  text-align: center;
  background: var(--red);
  color: #fff;
  font-family: inherit;
  font-size: 13.5px;
  font-weight: 800;
  box-shadow: var(--shadow-red);
  letter-spacing: 0.01em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Leaderboard tile */
.lb-tile {
  flex: none;
}
.lb-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.lb-head b {
  font-size: 14px;
  font-weight: 800;
}
.lb-mini {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 5px;
  min-height: 0;
  overflow: hidden;
}
.lb-mini-row {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 11.5px;
  font-weight: 700;
}
.lb-empty {
  font-size: 11px;
  color: var(--muted);
  font-weight: 500;
}
.rank {
  width: 20px;
  height: 20px;
  border-radius: 7px;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 9.5px;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  background: var(--bg);
  color: var(--muted);
  border: 2px solid var(--line);
}
.rank.r1 {
  background: var(--gold);
  color: #5a3c00;
  border: none;
  transform: rotate(-5deg);
}
.rank.r2 {
  background: #d7dee8;
  color: #3a4557;
  border: none;
  transform: rotate(3deg);
}
.rank.r3 {
  background: #e8a25f;
  color: #59300b;
  border: none;
  transform: rotate(-3deg);
}
.lb-mini-name {
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.lb-mini-pts {
  color: var(--gold-deep);
  font-variant-numeric: tabular-nums;
  font-weight: 800;
  font-size: 11px;
}

/* Sponsor rush */
.rush {
  flex: none;
  display: flex;
  align-items: center;
  gap: 11px;
  background: var(--ink);
  border: none;
  border-radius: 20px;
  padding: 12px 14px;
  cursor: pointer;
  position: relative;
  overflow: hidden;
  font-family: inherit;
  color: #fff;
  text-align: left;
  transition: transform 0.12s ease;
  box-shadow: 0 4px 0 #0d0608;
}
.rush:active {
  transform: scale(0.98) translateY(2px);
  box-shadow: none;
}
.rush::before {
  content: '';
  position: absolute;
  right: -18px;
  top: -22px;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: rgba(228, 33, 46, 0.35);
}
.rush-icon {
  width: 36px;
  height: 36px;
  border-radius: 11px;
  flex: none;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Fraunces', serif;
  font-weight: 900;
  font-size: 16px;
  color: var(--ink);
  transform: rotate(-4deg);
}
.rush-copy {
  flex: 1;
  min-width: 0;
  position: relative;
}
.rush-copy b {
  font-size: 12.5px;
  font-weight: 800;
  display: block;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.rush-copy span {
  font-size: 10.5px;
  color: rgba(255, 255, 255, 0.65);
  font-weight: 500;
}
.rush-reward {
  display: flex;
  align-items: center;
  gap: 5px;
  flex: none;
  background: var(--gold);
  padding: 6px 11px;
  border-radius: 999px;
  font-weight: 800;
  font-size: 12.5px;
  color: #4a3000;
  font-variant-numeric: tabular-nums;
  animation: reward 2.4s ease-in-out infinite;
  position: relative;
}
.rush-reward .coin {
  width: 13px;
  height: 13px;
  box-shadow: inset 0 -1.5px 2px rgba(0, 0, 0, 0.22);
}
@keyframes reward {
  0%, 100% { transform: scale(1) rotate(0deg); }
  50% { transform: scale(1.07) rotate(-2deg); }
}

/* Sponsors (blocco ampliato) */
.sponsors {
  flex: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 2px 2px 0;
}
.sponsors-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.sponsors-head .label {
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.12em;
  color: var(--muted);
}
.sponsor-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}
.sponsor-chip {
  min-width: 0;
  height: 44px;
  border-radius: 11px;
  background: var(--card);
  border: 2px solid var(--line);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Fraunces', serif;
  font-size: 12.5px;
  font-weight: 750;
  color: var(--ink);
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 0 8px;
  box-shadow: var(--shadow);
  transition: transform 0.12s ease, border-color 0.15s ease;
}
.sponsor-chip:active {
  transform: translateY(2px);
  box-shadow: none;
}
.sponsor-chip:hover {
  border-color: var(--red-line);
}
.sponsor-chip img {
  max-width: 100%;
  max-height: 30px;
  object-fit: contain;
}
.powered {
  flex: none;
  text-align: center;
  font-size: 9.5px;
  color: var(--muted);
  padding: 6px 0 8px;
  letter-spacing: 0.04em;
  font-weight: 600;
}
.powered b {
  color: var(--red);
  font-weight: 800;
}

/* Sheets */
.overlay {
  position: absolute;
  inset: 0;
  z-index: 100;
  background: rgba(35, 22, 26, 0.42);
  backdrop-filter: blur(3px);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.22s ease;
}
.overlay.open {
  opacity: 1;
  pointer-events: auto;
}
.sheet {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 101;
  background: var(--bg);
  border-radius: 26px 26px 0 0;
  padding: 10px 18px calc(18px + env(safe-area-inset-bottom));
  max-height: 82%;
  display: flex;
  flex-direction: column;
  transform: translateY(105%);
  transition: transform 0.28s cubic-bezier(0.32, 0.72, 0.28, 1);
  box-shadow: 0 -8px 30px rgba(35, 22, 26, 0.18);
}
.sheet.open {
  transform: translateY(0);
}
.grabber {
  width: 38px;
  height: 5px;
  border-radius: 99px;
  background: var(--red-line);
  margin: 0 auto 12px;
  flex: none;
}
.sheet-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  flex: none;
}
.sheet-title {
  font-family: 'Fraunces', serif;
  font-size: 19px;
  font-weight: 750;
  letter-spacing: -0.01em;
}
.sheet-sub {
  font-size: 11.5px;
  color: var(--muted);
  margin-top: 2px;
  font-weight: 500;
}
.close {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 2px solid var(--line);
  background: #fff;
  color: var(--muted);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  font-family: inherit;
  font-weight: 700;
}
.sheet-body {
  overflow-y: auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 9px;
  padding-bottom: 4px;
}
.sheet-empty {
  padding: 18px 4px;
  text-align: center;
  font-size: 12.5px;
  color: var(--muted);
  font-weight: 500;
}
.sheet-empty.error {
  color: var(--red);
}
.row-item {
  display: flex;
  align-items: center;
  gap: 11px;
  background: #fff;
  border: 2px solid var(--line);
  border-radius: 15px;
  padding: 11px 13px;
  font-family: inherit;
  color: var(--ink);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s ease, transform 0.1s ease;
  width: 100%;
  box-shadow: var(--shadow);
}
.row-item:active {
  transform: translateY(2px);
  box-shadow: none;
}
.row-item:hover {
  border-color: var(--red-line);
}
.row-item:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  box-shadow: none;
}
.avatar {
  width: 38px;
  height: 38px;
  border-radius: 12px;
  flex: none;
  background: var(--red-soft);
  border: 2px solid var(--red-line);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Fraunces', serif;
  font-weight: 750;
  font-size: 14px;
  color: var(--red);
  transform: rotate(-3deg);
  overflow: hidden;
}
.avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.row-copy {
  flex: 1;
  min-width: 0;
}
.row-copy b {
  font-size: 13.5px;
  font-weight: 800;
  display: block;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.row-copy span {
  font-size: 11px;
  color: var(--muted);
  font-weight: 500;
}
.row-pts {
  display: flex;
  align-items: center;
  gap: 5px;
  flex: none;
  font-weight: 800;
  font-size: 13px;
  color: var(--gold-deep);
  font-variant-numeric: tabular-nums;
}
.row-pts .coin {
  width: 14px;
  height: 14px;
}
.chip-mint {
  flex: none;
  font-size: 11px;
  font-weight: 800;
  color: #fff;
  background: var(--red);
  padding: 6px 12px;
  border-radius: 999px;
  box-shadow: 0 2px 0 var(--red-deep);
}
.chip-mint:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  box-shadow: none;
}

/* Profilo */
.profile-hero {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 4px 0 6px;
}
.profile-avatar {
  width: 66px;
  height: 66px;
  border-radius: 50%;
  flex: none;
  background: var(--red-soft);
  border: 2px solid var(--red-line);
  display: flex;
  align-items: center;
  justify-content: center;
  transform: rotate(-3deg);
}
.profile-label {
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--muted);
}
.profile-nick {
  font-family: 'Fraunces', serif;
  font-size: 22px;
  font-weight: 750;
  color: var(--ink);
  line-height: 1.1;
}
.profile-toggle {
  justify-content: center;
}
.nick-editor {
  display: flex;
  gap: 8px;
  align-items: center;
}
.nick-input {
  flex: 1;
  min-width: 0;
  border-radius: 13px;
  border: 2px solid var(--line);
  background: #fff;
  padding: 11px 13px;
  font-family: inherit;
  font-size: 13.5px;
  font-weight: 700;
  color: var(--ink);
  outline: none;
  transition: border-color 0.15s ease;
}
.nick-input:focus {
  border-color: var(--red-line);
}
.nick-save {
  border: none;
  cursor: pointer;
  font-family: inherit;
}
.nick-msg {
  font-size: 11.5px;
  font-weight: 700;
  padding: 0 2px;
}
.nick-msg.error {
  color: var(--red);
}
.nick-msg.ok {
  color: var(--gold-deep);
}
.coin-balance {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--gold-soft);
  border: 2px solid var(--gold);
  border-radius: 15px;
  padding: 12px 15px;
  box-shadow: 0 3px 0 rgba(209, 142, 0, 0.2);
}
.coin-balance-label {
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--gold-deep);
}
.coin-balance-val {
  display: flex;
  align-items: center;
  gap: 6px;
  font-family: 'Fraunces', serif;
  font-size: 24px;
  font-weight: 900;
  color: var(--gold-deep);
  font-variant-numeric: tabular-nums;
}
.lottery-box {
  background: #fff;
  border: 2px solid var(--line);
  border-radius: 15px;
  padding: 14px 15px;
  box-shadow: var(--shadow);
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
}
.lottery-title {
  font-family: 'Fraunces', serif;
  font-size: 15px;
  font-weight: 750;
  color: var(--ink);
}
.lottery-code {
  font-family: 'Fraunces', serif;
  font-size: 18px;
  font-weight: 900;
  letter-spacing: 0.06em;
  color: var(--red);
}
.lottery-qr {
  width: 168px;
  height: 168px;
  border-radius: 14px;
  border: 2px solid var(--line);
  background: #fff;
  padding: 8px;
  align-self: center;
}
.lottery-hint {
  font-size: 11px;
  font-weight: 700;
  color: var(--gold-deep);
  background: var(--gold-soft);
  border-radius: 10px;
  padding: 8px 11px;
}
.lottery-empty {
  font-size: 12px;
  color: var(--muted);
  font-weight: 500;
  line-height: 1.5;
}

/* Overlay gioco a schermo pieno */
.game-overlay {
  position: absolute;
  inset: 0;
  z-index: 150;
  background: var(--bg);
  display: flex;
  flex-direction: column;
}
.game-head {
  flex: none;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border-bottom: 2px solid var(--line);
}
.game-title {
  flex: 1;
  min-width: 0;
  text-align: center;
  font-family: 'Fraunces', serif;
  font-size: 15px;
  font-weight: 750;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.game-stage {
  flex: 1;
  min-height: 0;
  display: flex;
  padding: 12px;
  overflow: hidden;
}
.game-fill {
  width: 100%;
  height: 100%;
}

@media (prefers-reduced-motion: reduce) {
  .status::before,
  .rush-reward {
    animation: none;
  }
  .sheet,
  .overlay,
  .tile,
  .rush,
  .row-item {
    transition: none;
  }
}
</style>
