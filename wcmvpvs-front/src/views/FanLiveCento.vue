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
        <button ref="walletEl" class="wallet" type="button" aria-label="Saldo monete" @click="openEarn">
          <span class="coin"></span> <span>{{ totalCoins }}</span>
        </button>
      </header>

      <main>
        <!-- Scoreboard -->
        <section class="scoreboard" aria-label="Punteggio live">
          <div class="live-row">
            <span class="live-pill"><span class="live-dot"></span>LIVE</span>
            <span v-if="competitionLabel" class="comp">{{ competitionLabel }}</span>
          </div>
          <div class="teams">
            <div class="team">
              <div class="team-badge">{{ homeBadge }}</div>
              <div class="team-name">{{ homeName }}</div>
            </div>
            <div class="setscore">
              <span class="sep">vs</span>
            </div>
            <div class="team right">
              <div class="team-badge">{{ awayBadge }}</div>
              <div class="team-name">{{ awayName }}</div>
            </div>
          </div>
          <div class="set-label">{{ matchLabel }}</div>
        </section>

        <!-- Grid -->
        <div class="grid">
          <!-- MVP: delega il voto reale al parent (feature-select 'vote-mvp' → NewUiVoteModal) -->
          <button class="tile tile-mvp" type="button" @click="onVoteMvp">
            <b>{{ hasVoted ? 'Il tuo MVP' : "Vota l'MVP del pubblico" }}</b>
            <p>{{ hasVoted ? 'Grazie per il voto!' : 'Il tuo voto vale +10 monete' }}</p>
            <div class="mvp-photo">
              <span class="status">{{ hasVoted ? 'Voto registrato' : 'Votazioni aperte' }}</span>
              <img v-if="hasVoted && votedPlayerImageUrl" :src="votedPlayerImageUrl" :alt="votedPlayerName" class="mvp-img" />
              <svg v-else width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#8B95AB" stroke-width="1.5">
                <circle cx="12" cy="8" r="4" /><path d="M4 21c0-4 3.6-7 8-7s8 3 8 7" />
              </svg>
            </div>
            <span class="btn-vote">{{ hasVoted ? votedButtonLabel : 'Vota ora' }}</span>
          </button>

          <button class="tile" type="button" @click="openEarn">
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

        <!-- Sponsor rush → riusa la stessa superficie earn (minigiochi sponsor) -->
        <button v-if="rushSponsor" class="rush" type="button" @click="openEarn">
          <span class="rush-icon">{{ rushSponsor.badge }}</span>
          <span class="rush-copy">
            <b>Sponsor Rush · {{ rushSponsor.name }}</b>
            <span>15 secondi, monete subito</span>
          </span>
          <span class="rush-reward">+50 <span class="coin"></span></span>
        </button>

        <!-- Sponsors -->
        <div v-if="sponsors.length" class="sponsor-strip">
          <span class="label">PARTNER</span>
          <button
            v-for="sponsor in sponsors.slice(0, 3)"
            :key="sponsor.id"
            class="sponsor-chip"
            type="button"
            @click="onSponsorClick(sponsor)"
          >{{ sponsor.name }}</button>
        </div>
      </main>

      <div class="powered">Powered by <b>ArenaBoostX</b> · MVP System</div>

      <!-- Overlay + spend sheet (inline, stile cento) -->
      <div class="overlay" :class="{ open: activeSheet }" @click="closeSheets"></div>

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

      <!-- Toast -->
      <div class="toast" :class="{ show: toastMsg }">{{ toastMsg }}</div>
    </div>

    <!-- Superfici reali riusate da LiveExperienceHome -->
    <EarnCoinsModal
      v-model="isEarnOpen"
      :event-id="eventId"
      :wallet-coins="totalCoins"
      :wallet-target-el="walletEl"
      @coins-earned="handleCoinsEarned"
    />
    <FansLeaderboardModal
      v-model="isLeaderboardOpen"
      :top-list="leaderboardTop3"
      :user-rank="leaderboardUser"
    />
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import EarnCoinsModal from '../components/EarnCoinsModal.vue';
import FansLeaderboardModal from '../components/FansLeaderboardModal.vue';
import {
  apiClient,
  fetchFanProfile,
  fetchVoteStatus,
  syncGuestCoins,
  redeemFanReward,
  resolveApiUrl,
  getOrganizationSlug,
} from '../api';

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

const emit = defineEmits(['feature-select']);

const WALLET_STORAGE_KEY = 'wallet:coins';

/* ---------- Stato ---------- */
const totalCoins = ref(0);
const hasVoted = ref(false);
const leaderboardTop3 = ref([]);
const leaderboardUser = ref(null);
const sponsors = ref([]);
const walletEl = ref(null);

const activeSheet = ref('');
const isEarnOpen = ref(false);
const isLeaderboardOpen = ref(false);
const redeeming = ref('');
const toastMsg = ref('');
let toastTimer = null;
let coinsStream = null;

/* ---------- Derivati scoreboard/header ---------- */
function initials(name) {
  const parts = String(name || '').trim().split(/\s+/).filter(Boolean);
  if (!parts.length) return '—';
  return parts.slice(0, 3).map((p) => p[0]).join('').toUpperCase();
}
const homeName = computed(() =>
  String(props.activeEvent?.team1_name || props.teamName || 'TEAM').trim(),
);
const awayName = computed(() => String(props.activeEvent?.team2_name || 'Ospiti').trim());
const homeBadge = computed(() => initials(homeName.value));
const awayBadge = computed(() => initials(awayName.value));
const venueLabel = computed(() =>
  String(props.activeEvent?.venue || 'Esperienza ufficiale').trim() || 'Esperienza ufficiale',
);
// TODO(backend): nessun campo giornata/competizione nell'evento; mostrato solo se disponibile.
const competitionLabel = computed(() => String(props.activeEvent?.competition || '').trim());

const votedButtonLabel = computed(() => {
  const num = props.votedPlayerNumber === '' || props.votedPlayerNumber == null ? '' : `#${props.votedPlayerNumber} `;
  const label = props.votedPlayerLastName || props.votedPlayerName;
  return label ? `${num}${label}`.trim() : 'Modifica voto';
});

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

/* ---------- Sheets / toast ---------- */
function openSheet(id) {
  activeSheet.value = id;
}
function closeSheets() {
  activeSheet.value = '';
}
function toast(msg) {
  toastMsg.value = msg;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => {
    toastMsg.value = '';
  }, 2200);
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
  if (data.registered) {
    setCoins(Number(data.wallet) || 0);
  } else {
    setCoins(Math.max(totalCoins.value, Number(data.guest_coins) || 0));
  }
  if (data.user_rank) {
    leaderboardUser.value = data.user_rank;
  }
}

// EarnCoinsModal emette il totale di monete guadagnate; applichiamo il pattern addCoins.
async function handleCoinsEarned(coins) {
  const delta = Number(coins) || 0;
  if (!delta) return;
  setCoins(totalCoins.value + delta);
  if (props.eventId) {
    await syncGuestCoins(props.eventId, totalCoins.value);
  }
  if (delta > 0) {
    toast(`+${delta} monete`);
  }
  refreshLeaderboard();
}

/* ---------- Voto MVP (delegato al parent) ---------- */
function onVoteMvp() {
  emit('feature-select', 'vote-mvp');
}
async function loadVoteStatus() {
  if (!props.eventId) return;
  const status = await fetchVoteStatus(props.eventId);
  hasVoted.value = Boolean(status?.hasVoted);
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

/* ---------- Earn ---------- */
function openEarn() {
  closeSheets();
  isEarnOpen.value = true;
}

/* ---------- Sponsors ---------- */
async function loadSponsors() {
  try {
    const { data } = await apiClient.get('/sponsors');
    const list = Array.isArray(data?.sponsors) ? data.sponsors : Array.isArray(data) ? data : [];
    sponsors.value = list
      .map((s) => ({
        id: Number(s?.id) || s?.name,
        name: String(s?.name || s?.sponsor_name || '').trim(),
        priority: Number(s?.priority) || 0,
        url: String(s?.website_url || s?.url || '').trim(),
      }))
      .filter((s) => s.name)
      .sort((a, b) => b.priority - a.priority);
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
  if (sponsor?.url) {
    window.open(sponsor.url, '_blank', 'noopener');
  }
}

/* ---------- Riscatto premi ---------- */
async function redeem(reward) {
  if (redeeming.value) return;
  if (totalCoins.value < reward.cost) {
    toast('Monete insufficienti');
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
      toast('Riscatto non riuscito');
      return;
    }
    closeSheets();
    toast(`Riscattato: ${reward.title}`);
  } catch (error) {
    toast('Riscatto non riuscito');
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
  if (e.key === 'Escape') closeSheets();
}

/* ---------- Lifecycle ---------- */
function bootstrap() {
  totalCoins.value = readStoredCoins();
  loadFanProfile();
  loadVoteStatus();
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
    'https://fonts.googleapis.com/css2?family=Fraunces:opsz,wght@9..144,500;9..144,650;9..144,750&family=Inter:wght@400;500;600;700&display=swap';
  document.head.appendChild(link);
}

onMounted(() => {
  ensureCentoFonts();
  window.addEventListener('keydown', onKeydown);
  bootstrap();
});

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown);
  clearTimeout(toastTimer);
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
  background: #05080f;
  overflow: hidden;
  --bg: #120d0f;
  --card: #1d1518;
  --card-2: #271b1f;
  --line: rgba(255, 255, 255, 0.07);
  --line-strong: rgba(255, 255, 255, 0.13);
  --text: #f2f5fa;
  --muted: #8b95ab;
  --mint: #ff4d57;
  --mint-dark: #d91e2e;
  --mint-soft: rgba(255, 77, 87, 0.12);
  --gold: #f0c24b;
  --gold-soft: rgba(240, 194, 75, 0.14);
  color: var(--text);
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
    radial-gradient(120% 50% at 50% -8%, rgba(255, 77, 87, 0.09), transparent 60%),
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
  border-bottom: 1px solid var(--line);
}
.club {
  display: flex;
  align-items: center;
  gap: 9px;
}
.club-logo {
  width: 34px;
  height: 34px;
  border-radius: 10px;
  background: linear-gradient(140deg, #ffffff, #f0e4e4);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Fraunces', serif;
  font-weight: 750;
  font-size: 15px;
  color: #c41722;
  letter-spacing: -0.5px;
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
  font-weight: 700;
  display: block;
}
.club-name span {
  font-size: 10px;
  color: var(--muted);
  font-weight: 500;
}
.wallet {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--gold-soft);
  border: 1px solid rgba(240, 194, 75, 0.3);
  border-radius: 999px;
  padding: 6px 12px 6px 8px;
  font-weight: 700;
  font-size: 14px;
  font-variant-numeric: tabular-nums;
  color: var(--gold);
  font-family: inherit;
  cursor: pointer;
}
.coin {
  width: 17px;
  height: 17px;
  border-radius: 50%;
  flex: none;
  background: radial-gradient(circle at 34% 30%, #fbe29a, #e5ae33 68%, #b8841f);
  box-shadow: inset 0 -1.5px 2px rgba(0, 0, 0, 0.3), 0 0 0 1px rgba(240, 194, 75, 0.4);
}

main {
  flex: 1;
  min-height: 0;
  padding: 12px 12px 10px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

/* Scoreboard */
.scoreboard {
  flex: none;
  background: linear-gradient(165deg, var(--card-2), var(--card) 70%);
  border: 1px solid var(--line-strong);
  border-radius: 18px;
  padding: 12px 14px;
  position: relative;
  overflow: hidden;
}
.scoreboard::after {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(90% 70% at 50% 0%, rgba(255, 77, 87, 0.08), transparent 65%);
  pointer-events: none;
}
.live-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  position: relative;
  min-height: 20px;
}
.live-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.14em;
  color: var(--mint);
  background: var(--mint-soft);
  border: 1px solid rgba(255, 77, 87, 0.32);
  padding: 4px 9px;
  border-radius: 999px;
}
.live-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--mint);
  animation: pulse 1.7s ease-in-out infinite;
}
@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.45; transform: scale(0.8); }
}
.comp {
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.1em;
  color: var(--muted);
}
.teams {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 10px;
  position: relative;
}
.team {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.team.right {
  flex-direction: row-reverse;
  text-align: right;
}
.team-badge {
  width: 34px;
  height: 34px;
  border-radius: 10px;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Fraunces', serif;
  font-weight: 650;
  font-size: 13px;
  background: var(--card-2);
  border: 1px solid var(--line-strong);
}
.team-name {
  font-size: 11.5px;
  font-weight: 600;
  line-height: 1.2;
}
.setscore {
  font-family: 'Fraunces', serif;
  font-weight: 750;
  font-size: 28px;
  display: flex;
  align-items: baseline;
  gap: 6px;
  font-variant-numeric: tabular-nums;
}
.setscore .sep {
  color: var(--muted);
  font-size: 18px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}
.set-label {
  text-align: center;
  margin-top: 6px;
  font-size: 10.5px;
  color: var(--muted);
  position: relative;
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
  border: 1px solid var(--line);
  border-radius: 16px;
  padding: 13px;
  display: flex;
  flex-direction: column;
  text-align: left;
  color: var(--text);
  font-family: inherit;
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease, transform 0.1s ease;
  min-height: 0;
  overflow: hidden;
}
.tile:active {
  transform: scale(0.985);
}
.tile:hover {
  border-color: rgba(255, 77, 87, 0.38);
  background: var(--card-2);
}
.tile-mvp {
  grid-row: 1 / 3;
  border-color: rgba(255, 77, 87, 0.32);
  position: relative;
}
.tile-mvp::after {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: radial-gradient(120% 60% at 50% 110%, rgba(255, 77, 87, 0.09), transparent 60%);
}
.tile-icon {
  width: 32px;
  height: 32px;
  border-radius: 10px;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--mint-soft);
  color: var(--mint);
  margin-bottom: 9px;
}
.tile b {
  font-size: 14px;
  font-weight: 700;
  letter-spacing: -0.01em;
}
.tile p {
  font-size: 11px;
  color: var(--muted);
  line-height: 1.4;
  margin-top: 3px;
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
  font-weight: 700;
  color: var(--mint);
}
.mvp-photo {
  flex: 1;
  min-height: 0;
  margin: 9px 0;
  border-radius: 12px;
  background: linear-gradient(160deg, #3a252b, #221619);
  border: 1px solid var(--line-strong);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}
.mvp-photo svg {
  opacity: 0.45;
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
  font-weight: 700;
  color: var(--mint);
  background: rgba(11, 17, 32, 0.7);
  border: 1px solid rgba(255, 77, 87, 0.32);
  padding: 3px 8px;
  border-radius: 999px;
  display: flex;
  align-items: center;
  gap: 5px;
  backdrop-filter: blur(4px);
}
.status::before {
  content: '';
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--mint);
}
.btn-vote {
  border: none;
  border-radius: 11px;
  padding: 11px;
  width: 100%;
  text-align: center;
  background: linear-gradient(180deg, var(--mint), var(--mint-dark));
  color: #fff6f6;
  font-family: inherit;
  font-size: 13.5px;
  font-weight: 700;
  box-shadow: 0 5px 16px rgba(217, 30, 46, 0.3);
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
  font-weight: 600;
}
.lb-empty {
  font-size: 11px;
  color: var(--muted);
}
.rank {
  width: 19px;
  height: 19px;
  border-radius: 6px;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 9.5px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  background: var(--card-2);
  color: var(--muted);
  border: 1px solid var(--line);
}
.rank.r1 {
  background: linear-gradient(160deg, #f5d77a, #c89a2e);
  color: #231a05;
  border: none;
}
.rank.r2 {
  background: linear-gradient(160deg, #dde4ee, #9aa6ba);
  color: #1b2230;
  border: none;
}
.rank.r3 {
  background: linear-gradient(160deg, #e0a164, #a5652c);
  color: #2a1608;
  border: none;
}
.lb-mini-name {
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.lb-mini-pts {
  color: var(--gold);
  font-variant-numeric: tabular-nums;
  font-weight: 700;
  font-size: 11px;
}

/* Sponsor rush */
.rush {
  flex: none;
  display: flex;
  align-items: center;
  gap: 11px;
  background: linear-gradient(120deg, var(--card-2), var(--card));
  border: 1px solid rgba(255, 77, 87, 0.32);
  border-radius: 16px;
  padding: 11px 13px;
  cursor: pointer;
  position: relative;
  overflow: hidden;
  font-family: inherit;
  color: var(--text);
  text-align: left;
  transition: transform 0.1s ease;
}
.rush:active {
  transform: scale(0.985);
}
.rush::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  background: var(--mint);
}
.rush-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  flex: none;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Fraunces', serif;
  font-weight: 750;
  font-size: 16px;
  color: #c41722;
}
.rush-copy {
  flex: 1;
  min-width: 0;
}
.rush-copy b {
  font-size: 12.5px;
  font-weight: 700;
  display: block;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.rush-copy span {
  font-size: 10.5px;
  color: var(--muted);
}
.rush-reward {
  display: flex;
  align-items: center;
  gap: 5px;
  flex: none;
  background: var(--gold-soft);
  border: 1px solid rgba(240, 194, 75, 0.3);
  padding: 5px 10px;
  border-radius: 999px;
  font-weight: 700;
  font-size: 12.5px;
  color: var(--gold);
  font-variant-numeric: tabular-nums;
  animation: reward 2.6s ease-in-out infinite;
}
.rush-reward .coin {
  width: 13px;
  height: 13px;
}
@keyframes reward {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.05); }
}

/* Sponsor strip */
.sponsor-strip {
  flex: none;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 2px 2px 4px;
}
.sponsor-strip .label {
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.12em;
  color: var(--muted);
  flex: none;
}
.sponsor-chip {
  flex: 1;
  min-width: 0;
  height: 34px;
  border-radius: 10px;
  background: var(--card);
  border: 1px solid var(--line);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Fraunces', serif;
  font-size: 12.5px;
  font-weight: 650;
  color: rgba(242, 245, 250, 0.7);
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 0 8px;
}
.powered {
  flex: none;
  text-align: center;
  font-size: 9.5px;
  color: var(--muted);
  padding-bottom: 8px;
  letter-spacing: 0.04em;
}
.powered b {
  color: var(--mint);
  font-weight: 600;
}

/* Modals */
.overlay {
  position: absolute;
  inset: 0;
  z-index: 100;
  background: rgba(4, 7, 14, 0.6);
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
  background: var(--card);
  border: 1px solid var(--line-strong);
  border-bottom: none;
  border-radius: 22px 22px 0 0;
  padding: 10px 18px calc(18px + env(safe-area-inset-bottom));
  max-height: 82%;
  display: flex;
  flex-direction: column;
  transform: translateY(105%);
  transition: transform 0.28s cubic-bezier(0.32, 0.72, 0.28, 1);
}
.sheet.open {
  transform: translateY(0);
}
.grabber {
  width: 38px;
  height: 4px;
  border-radius: 99px;
  background: var(--line-strong);
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
  font-size: 16px;
  font-weight: 700;
  letter-spacing: -0.01em;
}
.sheet-sub {
  font-size: 11.5px;
  color: var(--muted);
  margin-top: 2px;
}
.close {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  border: 1px solid var(--line-strong);
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  font-family: inherit;
}
.sheet-body {
  overflow-y: auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 9px;
  padding-bottom: 4px;
}
.row-item {
  display: flex;
  align-items: center;
  gap: 11px;
  background: var(--card-2);
  border: 1px solid var(--line);
  border-radius: 13px;
  padding: 11px 13px;
  font-family: inherit;
  color: var(--text);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s ease, opacity 0.15s ease;
  width: 100%;
}
.row-item:hover {
  border-color: rgba(255, 77, 87, 0.42);
}
.row-item:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.avatar {
  width: 38px;
  height: 38px;
  border-radius: 11px;
  flex: none;
  background: linear-gradient(160deg, #3a252b, #221619);
  border: 1px solid var(--line-strong);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Fraunces', serif;
  font-weight: 650;
  font-size: 16px;
}
.row-copy {
  flex: 1;
  min-width: 0;
}
.row-copy b {
  font-size: 13.5px;
  font-weight: 700;
  display: block;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.row-copy span {
  font-size: 11px;
  color: var(--muted);
}
.row-pts {
  display: flex;
  align-items: center;
  gap: 5px;
  flex: none;
  font-weight: 700;
  font-size: 13px;
  color: var(--gold);
  font-variant-numeric: tabular-nums;
}
.row-pts .coin {
  width: 14px;
  height: 14px;
}
.toast {
  position: absolute;
  left: 50%;
  bottom: 24px;
  transform: translate(-50%, 20px);
  z-index: 200;
  background: var(--mint);
  color: #fff6f6;
  font-size: 13px;
  font-weight: 700;
  padding: 11px 18px;
  border-radius: 999px;
  box-shadow: 0 8px 24px rgba(255, 77, 87, 0.38);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.25s ease, transform 0.25s ease;
  white-space: nowrap;
}
.toast.show {
  opacity: 1;
  transform: translate(-50%, 0);
}

@media (prefers-reduced-motion: reduce) {
  .live-dot,
  .rush-reward {
    animation: none;
  }
  .sheet,
  .overlay,
  .toast {
    transition: none;
  }
}
</style>
