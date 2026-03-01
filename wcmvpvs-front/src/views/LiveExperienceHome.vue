<template>
  <div class="live-experience relative h-[100dvh] overflow-hidden text-white">
    <div class="arena-bg absolute inset-0" aria-hidden="true" />
    <div class="vignette absolute inset-0" aria-hidden="true" />

    <main class="relative z-10 flex h-full flex-col px-3 pb-3 pt-3 sm:px-4">
      <LiveHeader
        :team-name="teamName"
        :team-logo-url="teamLogoUrl"
        :is-live="isLive"
        :profile-avatar-url="profileAvatarUrl"
        :sponsor-line="sponsorLine"
        @profile-click="openProfileOverlay"
      >
        <StoriesBar
          v-if="activeStories.length"
          :stories="activeStories"
          :seen-ids="seenStoryIds"
          :loading-story-id="loadingStoryId"
          @open="openStory"
        />
        <template v-else>
          <p class="truncate text-center text-[clamp(0.86rem,2.8vw,1.16rem)] font-extrabold tracking-tight text-white">
            LIVE EXPERIENCE UFFICIALE
          </p>
          <p class="mt-1 truncate text-center text-[clamp(0.62rem,2.1vw,0.84rem)] text-slate-200/90">
            {{ sponsorLine }}
          </p>
        </template>
      </LiveHeader>

      <section class="hero animate-on-enter mt-[3.2vh] text-center">
        <h1 class="font-black uppercase leading-[0.92] tracking-tight drop-shadow-[0_4px_14px_rgba(0,0,0,0.85)]">
          <span class="block text-[clamp(2rem,10vw,4rem)]">ENTRA NELLA</span>
          <span class="block text-[clamp(2.7rem,12vw,4.8rem)] text-amber-400">PARTITA</span>
        </h1>
        <p class="mx-auto mt-2 max-w-[92%] border-t border-amber-300/50 pt-2 text-[clamp(0.9rem,3.8vw,1.4rem)] font-extrabold tracking-tight text-slate-100/95 drop-shadow-md">
          {{ matchLabel }}
        </p>
      </section>

      <section ref="topCardsRef" class="animate-on-enter mt-[3.2vh] grid grid-cols-3 gap-2.5">
        <FeatureCard
          v-bind="voteFeature"
          @select="onFeatureSelect"
        />

        <div class="flex min-h-[220px] flex-col gap-2">
          <article
            class="mini-feature mini-feature--earn"
            role="button"
            tabindex="0"
            aria-label="Apri guadagna monete"
            @click="onFeatureSelect('game-live')"
            @keydown.enter.prevent="onFeatureSelect('game-live')"
            @keydown.space.prevent="onFeatureSelect('game-live')"
          >
            <div class="mini-feature__content">
              <p id="wallet-coin-target" class="mini-feature__coins">🪙 {{ totalCoins }}</p>
            </div>
            <button type="button" class="mini-feature__cta mini-feature__cta--earn" @click.stop="onFeatureSelect('game-live')">
              GUADAGNA
            </button>
          </article>

          <article
            class="mini-feature mini-feature--spend"
            role="button"
            tabindex="0"
            aria-label="Apri premi e utilizza monete"
            @click="openSpendPreview"
            @keydown.enter.prevent="openSpendPreview"
            @keydown.space.prevent="openSpendPreview"
          >
            <div class="mini-feature__content">
              <p class="mini-feature__icons" aria-hidden="true">🎁 🏷️ ⚡</p>
            </div>
            <button type="button" class="mini-feature__cta mini-feature__cta--spend" @click.stop="openSpendPreview">
              SPENDI
            </button>
          </article>
        </div>

        <article
          class="leaderboard-preview"
          role="button"
          tabindex="0"
          aria-label="Apri classifica tifosi"
          @click="openLeaderboard"
          @keydown.enter.prevent="openLeaderboard"
          @keydown.space.prevent="openLeaderboard"
        >
          <p class="leaderboard-preview__title">CLASSIFICA TIFOSI</p>
          <ul class="leaderboard-preview__list">
            <li v-for="(entry, index) in leaderboardTop3" :key="`${entry.name}-${index}`" class="leaderboard-preview__item">
              <span>{{ medals[index] }} {{ entry.name }}</span>
              <strong>{{ entry.coins }} 🪙</strong>
            </li>
          </ul>
          <p v-if="isRegisteredFan && leaderboardUser" class="leaderboard-preview__you">Tu: #{{ leaderboardUser.rank }} • {{ leaderboardUser.coins }} 🪙</p>
          <button type="button" class="leaderboard-preview__cta" @click.stop="openLeaderboard">CLASSIFICA</button>
        </article>
      </section>

      <div ref="liveResultsRef" class="animate-on-enter mt-auto">
        <SponsorsMarquee
          v-if="showSponsorsBox"
          ref="sponsorBoxRef"
          :sponsors="sponsors"
          :height-px="sponsorHeight"
          :event-id="eventId"
          @image-loaded="queueSponsorGapMeasure"
          @sponsor-click="handleSponsorClick"
        />
      </div>
    </main>

    <StoryModal
      :open="isStoryModalOpen"
      :current-story="currentStory"
      :show-prev="activeStoryIndex > 0"
      @close="closeStoryModal"
      @next="goToNextStory"
      @prev="goToPrevStory"
    />

    <EarnCoinsModal
      v-model="isEarnModalOpen"
      :event-id="eventId"
      :wallet-target-el="walletTargetEl"
      @coins-earned="addCoins"
    />

    <FansLeaderboardModal
      v-model="isLeaderboardModalOpen"
      :top-list="leaderboardTop3"
      :user-rank="leaderboardUser"
    />

    <FanRegistrationPromptModal
      v-model="isRegistrationPromptOpen"
      :trigger="registrationTrigger"
      :earned-coins="lastEarnedCoins"
      :wallet-coins="totalCoins"
      :reward-label="selectedRewardLabel"
      :on-submit="handleRegistrationSubmit"
      :on-login="handleExistingFanLogin"
      @dismissed="markPromptDismissed"
    />



    <Teleport to="body">
      <Transition name="earn-modal-fade">
        <div
          v-if="isProfileOverlayOpen"
          class="fixed inset-0 z-[130] flex"
          role="dialog"
          aria-modal="true"
          aria-label="Profilo tifoso"
          @click.self="closeProfileOverlay"
        >
          <div class="absolute inset-0 bg-slate-950/95 backdrop-blur-md" aria-hidden="true" />

          <div class="relative flex h-full w-full flex-col overflow-hidden">
            <header class="sticky top-0 z-10 border-b border-white/10 bg-slate-950/85 px-4 py-4 backdrop-blur md:px-6">
              <div class="flex items-start justify-between gap-4">
                <div>
                  <p class="text-xs font-bold uppercase tracking-[0.2em] text-amber-300/90">Profilo utente</p>
                  <h2 class="mt-1 text-2xl font-black text-white md:text-3xl">Il tuo account</h2>
                </div>
                <button
                  type="button"
                  class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-white/20 bg-white/5 text-2xl leading-none text-white transition hover:bg-white/15"
                  aria-label="Chiudi profilo"
                  @click="closeProfileOverlay"
                >
                  ×
                </button>
              </div>
            </header>

            <div class="flex-1 overflow-y-auto px-4 pb-10 pt-5 md:px-6">
              <div class="mx-auto grid w-full max-w-5xl gap-4 lg:grid-cols-3">
                <section class="rounded-2xl border border-white/15 bg-white/10 p-5 shadow-[0_10px_30px_rgba(15,23,42,0.4)] lg:col-span-1">
                  <div class="flex items-center gap-3">
                    <div class="flex h-16 w-16 items-center justify-center overflow-hidden rounded-full border border-amber-300/60 bg-slate-800 text-2xl">
                      <img v-if="profileAvatarUrl" :src="profileAvatarUrl" alt="Avatar profilo" class="h-full w-full object-cover">
                      <span v-else aria-hidden="true">👤</span>
                    </div>
                    <div>
                      <p class="text-xs uppercase tracking-[0.18em] text-slate-300">Nickname</p>
                      <p class="text-xl font-extrabold text-white">{{ profileNickname }}</p>
                    </div>
                  </div>

                  <div class="mt-5 rounded-xl border border-emerald-300/25 bg-emerald-400/10 p-4">
                    <p class="text-xs uppercase tracking-[0.2em] text-emerald-200/90">Saldo monete</p>
                    <p class="mt-1 text-3xl font-black text-emerald-300">{{ totalCoins }} 🪙</p>
                  </div>
                </section>

                <section class="rounded-2xl border border-white/15 bg-white/10 p-5 shadow-[0_10px_30px_rgba(15,23,42,0.4)] lg:col-span-1">
                  <h3 class="text-lg font-extrabold text-white">Coupon acquistati</h3>
                  <ul v-if="accountRedemptions.length" class="mt-3 space-y-2">
                    <li v-for="entry in accountRedemptions" :key="`${entry.id}-${entry.createdAt}`" class="rounded-xl border border-white/10 bg-slate-900/50 px-3 py-2">
                      <p class="text-sm font-semibold text-white">{{ entry.label }}</p>
                      <p class="mt-0.5 text-xs text-slate-300">{{ entry.costCoins }} 🪙 • {{ entry.createdAt }}</p>
                    </li>
                  </ul>
                  <p v-else class="mt-3 rounded-xl border border-dashed border-white/20 bg-slate-900/35 px-3 py-4 text-sm text-slate-300">
                    Non hai ancora acquistato coupon.
                  </p>
                </section>

                <section class="rounded-2xl border border-white/15 bg-white/10 p-5 shadow-[0_10px_30px_rgba(15,23,42,0.4)] lg:col-span-1">
                  <h3 class="text-lg font-extrabold text-white">QR Lotteria MVP</h3>
                  <div v-if="displayLotteryCode" class="mt-3">
                    <p class="text-sm text-slate-300">Codice lotteria:</p>
                    <p class="text-base font-black tracking-wider text-amber-300">{{ displayLotteryCode }}</p>
                    <img
                      :src="displayLotteryQrUrl"
                      alt="QR lotteria utente"
                      class="mt-3 h-44 w-44 rounded-xl border border-white/20 bg-white p-2"
                    >
                    <p class="mt-3 rounded-xl border border-amber-300/25 bg-amber-300/10 px-3 py-2 text-xs font-semibold text-amber-100">
                      Resta fino a fine partita per ritirare il premio.
                    </p>
                  </div>
                  <p v-else class="mt-3 rounded-xl border border-dashed border-white/20 bg-slate-900/35 px-3 py-4 text-sm text-slate-300">
                    Vota l'MVP per ottenere il tuo QR lotteria personale.
                  </p>
                </section>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <Teleport to="body">
      <Transition name="earn-modal-fade">
        <div
          v-if="isSpendPreviewOpen"
          class="fixed inset-0 z-[120] flex"
          role="dialog"
          aria-modal="true"
          aria-label="Spendi monete"
          @click.self="closeSpendPreview"
        >
          <div class="absolute inset-0 bg-slate-950/90 backdrop-blur-sm" aria-hidden="true" />

          <div class="relative flex h-full w-full flex-col overflow-hidden">
            <header class="sticky top-0 z-10 border-b border-white/10 bg-slate-950/85 px-4 py-4 backdrop-blur md:px-6">
              <div class="flex items-start justify-between gap-4">
                <div>
                  <h2 class="text-2xl font-black text-white md:text-3xl">Spendi Monete</h2>
                  <p class="mt-1 text-sm text-slate-300 md:text-base">I guest possono vedere il catalogo, ma per riscattare serve il profilo tifoso.</p>
                </div>
                <button
                  type="button"
                  class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-white/20 bg-white/5 text-2xl leading-none text-white transition hover:bg-white/15"
                  aria-label="Chiudi modale Spendi Monete"
                  @click="closeSpendPreview"
                >
                  ×
                </button>
              </div>
            </header>

            <div class="flex-1 overflow-y-auto px-4 pb-8 pt-5 md:px-6">
              <div class="mx-auto grid max-w-6xl grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
                <button
                  v-for="coupon in spendCouponPreview"
                  :key="coupon.id"
                  type="button"
                  class="group rounded-2xl border border-white/15 bg-white/10 p-4 text-left shadow-[0_10px_28px_rgba(15,23,42,0.45)] backdrop-blur transition hover:-translate-y-0.5 hover:bg-white/15"
                  @click="attemptRedeem(coupon.id, coupon.cost, coupon.label)"
                >
                  <div class="flex items-start justify-between gap-2">
                    <span class="text-2xl" aria-hidden="true">🎟️</span>
                    <span class="rounded-full border border-amber-300/40 bg-amber-300/15 px-2 py-0.5 text-xs font-bold text-amber-200">
                      {{ coupon.cost }} 🪙
                    </span>
                  </div>
                  <h3 class="mt-3 text-lg font-extrabold text-white">{{ coupon.label }}</h3>
                  <p class="mt-1 text-sm text-slate-300">{{ coupon.description }}</p>
                  <p class="mt-4 text-xs font-semibold uppercase tracking-wide text-emerald-300">Riscatta</p>
                </button>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import EarnCoinsModal from '../components/EarnCoinsModal.vue';
import FansLeaderboardModal from '../components/FansLeaderboardModal.vue';
import FeatureCard from '../components/FeatureCard.vue';
import FanRegistrationPromptModal from '../components/FanRegistrationPromptModal.vue';
import LiveHeader from '../components/LiveHeader.vue';
import SponsorsMarquee from '../components/SponsorsMarquee.vue';
import StoriesBar from '../components/StoriesBar.vue';
import StoryModal from '../components/StoryModal.vue';
import { apiClient, fetchFanProfile, fetchVoteStatus, redeemFanReward, registerFanProfile, syncGuestCoins } from '../api';
import { getOrCreateDeviceId } from '../deviceId';

const anonymousAvatarSvg = encodeURIComponent(
  `<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 320 220'>
    <defs>
      <linearGradient id='bg' x1='0' x2='1' y1='0' y2='1'>
        <stop offset='0%' stop-color='#1e293b'/>
        <stop offset='100%' stop-color='#0f172a'/>
      </linearGradient>
      <linearGradient id='ring' x1='0' x2='1' y1='0' y2='1'>
        <stop offset='0%' stop-color='#fde68a'/>
        <stop offset='100%' stop-color='#f97316'/>
      </linearGradient>
    </defs>
    <rect width='320' height='220' fill='url(#bg)'/>
    <circle cx='160' cy='88' r='44' fill='#334155' stroke='url(#ring)' stroke-width='6'/>
    <path d='M74 200c0-40 39-66 86-66s86 26 86 66' fill='#334155' stroke='url(#ring)' stroke-width='6' stroke-linecap='round'/>
  </svg>`,
);
const anonymousAvatarDataUrl = `data:image/svg+xml,${anonymousAvatarSvg}`;
const medals = ['🥇', '🥈', '🥉'];
const rewardLabelMap = {
  'coupon-match': 'Coupon Match Day',
  'coupon-merch': 'Sconto Merch 20%',
  'coupon-upgrade': 'Upgrade posto',
  'coupon-photo': 'Foto Team Edition',
};

const props = defineProps({
  eventId: {
    type: Number,
    default: 0,
  },
  teamName: {
    type: String,
    default: 'TEAM',
  },
  teamLogoUrl: {
    type: String,
    default: '',
  },
  isLive: {
    type: Boolean,
    default: true,
  },
  sponsorLine: {
    type: String,
    default: 'Powered by MVP System',
  },
  matchLabel: {
    type: String,
    default: 'Vota • Gioca • Vinci • Partecipa',
  },
  features: {
    type: Array,
    default: () => [
      {
        id: 'vote-mvp',
        title: 'VOTA L\'MVP',
        subtitle: 'del pubblico',
        description: 'Votazioni aperte',
        actionLabel: 'CLICCA ORA',
        icon: '◔',
        theme: 'orange',
      },
      {
        id: 'game-live',
        title: 'GUADAGNA MONETE',
        subtitle: '',
        description: 'Gioca per guadagnarle',
        actionLabel: 'GIOCA ORA',
        centerBadge: '🪙 0',
        icon: '⚡',
        theme: 'blue',
      },
      {
        id: 'lottery-live',
        title: 'PREMI',
        subtitle: 'utilizza le tue monete',
        description: 'X premi disponibili',
        actionLabel: 'SCOPRI',
        centerBadge: '🎁',
        icon: '🎁',
        theme: 'green',
      },
    ],
  },
  votedPlayerImageUrl: {
    type: String,
    default: '',
  },
  votedPlayerName: {
    type: String,
    default: '',
  },
  votedPlayerLastName: {
    type: String,
    default: '',
  },
  votedPlayerNumber: {
    type: [String, Number],
    default: '',
  },
  gainedCoins: {
    type: Number,
    default: 0,
  },
  registrationPromptSignal: {
    type: Number,
    default: 0,
  },
});

const emit = defineEmits(['feature-select']);
const isEarnModalOpen = ref(false);
const isLeaderboardModalOpen = ref(false);
const isRegistrationPromptOpen = ref(false);
const registrationTrigger = ref('after_vote');
const isRegisteredFan = ref(false);
const fanSessionToken = ref('');
const fanNickname = ref('');
const fanId = ref(0);
const isProfileOverlayOpen = ref(false);
const fanRewardRedemptions = ref([]);
const fanLotteryTicket = ref(null);
const hasVotedMvp = ref(false);
const lastEarnedCoins = ref(0);
const isSpendPreviewOpen = ref(false);
const selectedRewardLabel = ref('Coupon Match Day · 30 🪙');
const spendCouponPreview = [
  { id: 'coupon-match', label: 'Coupon Match Day', description: 'Bibita + snack al bar partner.', cost: 30 },
  { id: 'coupon-merch', label: 'Sconto Merch 20%', description: 'Sconto valido nello store ufficiale.', cost: 45 },
  { id: 'coupon-upgrade', label: 'Upgrade posto', description: 'Prova a passare al settore premium.', cost: 60 },
  { id: 'coupon-photo', label: 'Foto Team Edition', description: 'Scatto ricordo con layout personalizzato.', cost: 20 },
];
const totalCoins = ref(0);
const walletTargetEl = ref(null);
const topCardsRef = ref(null);
const sponsorBoxRef = ref(null);
const liveResultsRef = ref(null);
const sponsorHeight = ref(0);
const sponsors = ref([]);
const leaderboardTop3 = ref([
  { name: 'TIFO1', coins: 320 },
  { name: 'TIFO2', coins: 275 },
  { name: 'TIFO3', coins: 249 },
]);
const leaderboardUser = ref(null);
let sponsorMeasureRaf = 0;

const MIN_SPONSOR_HEIGHT = 48;
const HARD_HIDE_THRESHOLD = 24;
const GAP_BUFFER_PX = 8;
const showSponsorsBox = computed(() => sponsors.value.length > 0 && sponsorHeight.value >= HARD_HIDE_THRESHOLD);

const profileAvatarUrl = computed(() => {
  if (props.votedPlayerImageUrl) {
    return props.votedPlayerImageUrl;
  }
  return '';
});

const profileNickname = computed(() => {
  if (fanNickname.value.trim()) {
    return fanNickname.value.trim();
  }
  return isRegisteredFan.value ? 'Tifoso' : 'Guest';
});

const accountRedemptions = computed(() =>
  fanRewardRedemptions.value.map((entry, index) => ({
    id: Number(entry?.id) || index + 1,
    label: rewardLabelMap[String(entry?.reward_key || '').trim()] || String(entry?.reward_key || 'Reward').replace(/-/g, ' '),
    costCoins: Math.max(0, Number(entry?.cost_coins) || 0),
    createdAt: String(entry?.created_at || '').replace('T', ' ').slice(0, 16) || 'Data non disponibile',
  })),
);

const lotteryTicketCode = computed(() => String(fanLotteryTicket.value?.ticket_code || '').trim());
const fallbackLotteryCode = computed(() => {
  if (!hasVotedMvp.value || !props.eventId) {
    return '';
  }

  const fanSegment = fanId.value ? String(fanId.value).padStart(5, '0') : 'GUEST';
  const deviceSegment = getOrCreateDeviceId().replace(/[^a-zA-Z0-9]/g, '').slice(-6).toUpperCase() || 'DEVICE';
  return `MVP-${props.eventId}-${fanSegment}-${deviceSegment}`;
});
const displayLotteryCode = computed(() => lotteryTicketCode.value || fallbackLotteryCode.value);
const displayLotteryQrUrl = computed(() =>
  displayLotteryCode.value
    ? `https://api.qrserver.com/v1/create-qr-code/?size=260x260&data=${encodeURIComponent(displayLotteryCode.value)}`
    : '',
);

const voteFeature = computed(() => {
  const baseFeature = props.features.find((feature) => feature.id === 'vote-mvp') || props.features[0];
  const hasVotedPlayer = Boolean(props.votedPlayerImageUrl);
  const playerLastName = String(props.votedPlayerLastName || '').trim();
  const fallbackName = String(props.votedPlayerName || '').trim();
  const titleLabel = hasVotedPlayer ? (playerLastName || fallbackName || baseFeature.title) : baseFeature.title;
  const hasPlayerNumber = String(props.votedPlayerNumber || '').trim() !== '';
  const subtitleLabel = hasVotedPlayer
    ? (hasPlayerNumber ? `#${String(props.votedPlayerNumber).trim()}` : '')
    : baseFeature.subtitle;

  return {
    ...baseFeature,
    title: titleLabel,
    subtitle: subtitleLabel,
    actionLabel: hasVotedPlayer ? 'MODIFICA' : baseFeature.actionLabel,
    previewImageUrl: hasVotedPlayer ? props.votedPlayerImageUrl : anonymousAvatarDataUrl,
    previewImageFit: hasVotedPlayer ? 'contain' : 'cover',
    previewAlt: props.votedPlayerName
      ? `MVP selezionato: ${props.votedPlayerName}`
      : 'Avatar anonimo MVP',
  };
});

function openLeaderboard() {
  isLeaderboardModalOpen.value = true;
  if (!isRegisteredFan.value) {
    openRegistrationPrompt('leaderboard');
  }
}

function syncWalletTargetEl() {
  if (typeof document === 'undefined') {
    return;
  }

  walletTargetEl.value = document.getElementById('wallet-coin-target');
}

onMounted(async () => {
  if (typeof window === 'undefined') {
    return;
  }

  const stored = Number.parseInt(window.localStorage.getItem('wallet:coins') || '0', 10);
  totalCoins.value = Number.isFinite(stored) && stored > 0 ? stored : 0;
  await nextTick();
  syncWalletTargetEl();

  try {
    const parsed = JSON.parse(window.localStorage.getItem(storyStorageKey.value) || '[]');
    seenStoryIds.value = Array.isArray(parsed) ? parsed.filter((id) => Number.isFinite(Number(id))).map((id) => Number(id)) : [];
  } catch (error) {
    seenStoryIds.value = [];
  }

  await loadFanProfile();
  loadEventStories();
  loadSponsors();
  loadLeaderboardPreview();
  queueSponsorGapMeasure();

  window.addEventListener('resize', queueSponsorGapMeasure, { passive: true });
  window.addEventListener('orientationchange', queueSponsorGapMeasure, { passive: true });
});

async function addCoins(amount) {
  const parsed = Math.max(0, Number(amount) || 0);
  totalCoins.value += parsed;

  if (typeof window !== 'undefined') {
    window.localStorage.setItem('wallet:coins', String(totalCoins.value));
  }

  await nextTick();
  syncWalletTargetEl();

  if (props.eventId) {
    await syncGuestCoins(props.eventId, totalCoins.value);
  }

  if (!isRegisteredFan.value && props.eventId) {
    lastEarnedCoins.value = parsed;
    openRegistrationPrompt('after_earn');
  }
}

async function loadLeaderboardPreview() {
  if (!props.eventId) {
    return;
  }

  try {
    // TODO: sostituire endpoint placeholder con leaderboard ufficiale quando disponibile.
    const { data } = await apiClient.get(`/events/${props.eventId}/coins-leaderboard`);
    const top = Array.isArray(data?.top3) ? data.top3 : Array.isArray(data?.top) ? data.top : [];
    leaderboardTop3.value = top.slice(0, 3).map((entry, index) => ({
      name: String(entry?.name || `TIFO${index + 1}`).slice(0, 10).toUpperCase(),
      coins: Math.max(0, Number(entry?.coins) || 0),
    }));

    if (isRegisteredFan.value && Number.isFinite(Number(data?.userRank?.rank))) {
      leaderboardUser.value = {
        rank: Number(data.userRank.rank),
        coins: Math.max(0, Number(data.userRank.coins) || 0),
      };
    }
  } catch (error) {
    // placeholder fallback, keep static UI preview
  }
}

function queueSponsorGapMeasure() {
  if (typeof window === 'undefined') {
    return;
  }
  if (sponsorMeasureRaf) {
    window.cancelAnimationFrame(sponsorMeasureRaf);
  }
  sponsorMeasureRaf = window.requestAnimationFrame(() => {
    sponsorMeasureRaf = 0;
    measureSponsorGap();
  });
}

function measureSponsorGap() {
  const topCardsEl = topCardsRef.value;
  const liveResultsEl = liveResultsRef.value;
  if (!topCardsEl || !liveResultsEl) {
    sponsorHeight.value = 0;
    return;
  }

  const topCardsBottom = topCardsEl.getBoundingClientRect().bottom;
  const liveResultsTop = liveResultsEl.getBoundingClientRect().top;
  const availableGapPx = Math.floor(liveResultsTop - topCardsBottom);

  if (availableGapPx <= 0) {
    sponsorHeight.value = 0;
    return;
  }

  const targetHeight = Math.floor(availableGapPx * 0.5);
  const maxSafeHeight = Math.max(0, availableGapPx - GAP_BUFFER_PX);
  const resolvedHeight = Math.min(Math.max(targetHeight, MIN_SPONSOR_HEIGHT), maxSafeHeight);
  sponsorHeight.value = resolvedHeight < HARD_HIDE_THRESHOLD ? 0 : resolvedHeight;
}

function normalizeSponsor(item, index) {
  const imageUrl = String(item?.logo_data || item?.image_url || item?.imageUrl || '').trim();
  if (!imageUrl) {
    return null;
  }

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
    sponsors.value = Array.isArray(data)
      ? data
          .map((item, index) => normalizeSponsor(item, index))
          .filter(Boolean)
          .sort((a, b) => {
            if (a.priority !== b.priority) {
              return a.priority - b.priority;
            }
            return a.insertedIndex - b.insertedIndex;
          })
      : [];
  } catch (error) {
    sponsors.value = [];
  } finally {
    nextTick(() => {
      queueSponsorGapMeasure();
    });
  }
}

function handleSponsorClick(sponsor) {
  const eventId = Number(props.eventId) || 0;
  const sponsorId = Number(sponsor?.id) || 0;
  if (!eventId || !sponsorId) {
    return;
  }

  apiClient.post(`/events/${eventId}/sponsors/${sponsorId}/click`, {
    device_id: getOrCreateDeviceId(),
    at: new Date().toISOString(),
  }).catch(() => {});
}


const eventStories = ref([]);
const seenStoryIds = ref([]);
const isStoryModalOpen = ref(false);
const activeStoryIndex = ref(0);
const loadingStoryId = ref(0);

const preloadedStoryUrls = new Set();
const preloadPromises = new Map();

const activeStories = computed(() =>
  eventStories.value
    .filter((story) => story && story.is_active !== false)
    .sort((a, b) => (Number(a.order_index) || 0) - (Number(b.order_index) || 0)),
);

const currentStory = computed(() => activeStories.value[activeStoryIndex.value] || null);

const storyStorageKey = computed(() => `mvp:stories:seen:event:${props.eventId || 0}`);

function persistSeenStories() {
  if (typeof window === 'undefined') {
    return;
  }
  window.localStorage.setItem(storyStorageKey.value, JSON.stringify(Array.from(new Set(seenStoryIds.value))));
}

function markStorySeen(storyId) {
  if (!storyId || seenStoryIds.value.includes(storyId)) {
    return;
  }
  seenStoryIds.value = [...seenStoryIds.value, storyId];
  persistSeenStories();
}

async function loadEventStories() {
  if (!props.eventId) {
    eventStories.value = [];
    return;
  }
  try {
    const { data } = await apiClient.get(`/events/${props.eventId}/stories`);
    eventStories.value = Array.isArray(data) ? data : [];
  } catch (error) {
    eventStories.value = [];
  }
}

function openStory(index) {
  preloadAndOpenStory(index);
}

function preloadStoryVideo(url) {
  const targetUrl = String(url || '').trim();
  if (!targetUrl) {
    return Promise.resolve();
  }

  if (preloadedStoryUrls.has(targetUrl)) {
    return Promise.resolve();
  }

  if (preloadPromises.has(targetUrl)) {
    return preloadPromises.get(targetUrl);
  }

  const preloadPromise = new Promise((resolve) => {
    if (typeof document === 'undefined') {
      preloadedStoryUrls.add(targetUrl);
      resolve();
      return;
    }

    const preloader = document.createElement('video');
    preloader.preload = 'auto';
    preloader.src = targetUrl;

    const cleanup = () => {
      preloader.removeEventListener('canplaythrough', onReady);
      preloader.removeEventListener('loadeddata', onReady);
      preloader.removeEventListener('error', onReady);
      preloader.removeAttribute('src');
      preloader.load();
    };

    const onReady = () => {
      preloadedStoryUrls.add(targetUrl);
      preloadPromises.delete(targetUrl);
      cleanup();
      resolve();
    };

    preloader.addEventListener('canplaythrough', onReady, { once: true });
    preloader.addEventListener('loadeddata', onReady, { once: true });
    preloader.addEventListener('error', onReady, { once: true });
    preloader.load();
  });

  preloadPromises.set(targetUrl, preloadPromise);
  return preloadPromise;
}

function preloadOtherStories(excludeIndex) {
  activeStories.value.forEach((story, index) => {
    if (!story || index === excludeIndex) {
      return;
    }
    preloadStoryVideo(story.video_url);
  });
}

async function preloadAndOpenStory(index) {
  const safeIndex = Math.max(0, Math.min(index, activeStories.value.length - 1));
  const selectedStory = activeStories.value[safeIndex];
  if (!selectedStory) {
    return;
  }

  loadingStoryId.value = Number(selectedStory.id) || 0;
  await preloadStoryVideo(selectedStory.video_url);
  activeStoryIndex.value = safeIndex;
  isStoryModalOpen.value = true;
  loadingStoryId.value = 0;
  preloadOtherStories(safeIndex);
}

function closeStoryModal() {
  isStoryModalOpen.value = false;
}

function goToNextStory() {
  if (activeStoryIndex.value >= activeStories.value.length - 1) {
    closeStoryModal();
    return;
  }
  activeStoryIndex.value += 1;
}

function goToPrevStory() {
  if (activeStoryIndex.value <= 0) {
    return;
  }
  activeStoryIndex.value -= 1;
}

watch(
  () => props.eventId,
  () => {
    if (typeof window === 'undefined') {
      return;
    }
    try {
      const parsed = JSON.parse(window.localStorage.getItem(storyStorageKey.value) || '[]');
      seenStoryIds.value = Array.isArray(parsed) ? parsed.filter((id) => Number.isFinite(Number(id))).map((id) => Number(id)) : [];
    } catch (error) {
      seenStoryIds.value = [];
    }
    loadFanProfile();
    loadEventStories();
    loadSponsors();
    loadLeaderboardPreview();
    nextTick(() => {
      queueSponsorGapMeasure();
    });
  },
);


watch(() => props.registrationPromptSignal, (value, previous) => {
  if (value !== previous) {
    openRegistrationPrompt('after_vote');
  }
});

watch(showSponsorsBox, () => {
  nextTick(() => {
    queueSponsorGapMeasure();
  });
});

watch([currentStory, isStoryModalOpen], ([story, isOpen]) => {
  if (!isOpen || !story) {
    return;
  }
  markStorySeen(Number(story.id));
});

watch([isStoryModalOpen, isProfileOverlayOpen], ([storyOpen, profileOpen]) => {
  if (typeof document === 'undefined') {
    return;
  }
  document.body.style.overflow = storyOpen || profileOpen ? 'hidden' : '';
});

onBeforeUnmount(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('resize', queueSponsorGapMeasure);
    window.removeEventListener('orientationchange', queueSponsorGapMeasure);
    if (sponsorMeasureRaf) {
      window.cancelAnimationFrame(sponsorMeasureRaf);
    }
  }

  if (typeof document !== 'undefined') {
    document.body.style.overflow = '';
  }
  isProfileOverlayOpen.value = false;
});



function markPromptDismissed(trigger) {
  if (typeof window === 'undefined') return;
  window.sessionStorage.setItem(`fan:prompt:${trigger}`, '1');
}

function openRegistrationPrompt(trigger) {
  if (isRegisteredFan.value || typeof window === 'undefined') return;
  if (trigger === 'spend_redeem') {
    registrationTrigger.value = trigger;
    isRegistrationPromptOpen.value = true;
    return;
  }
  const key = `fan:prompt:${trigger}`;
  if (window.sessionStorage.getItem(key) === '1') return;
  registrationTrigger.value = trigger;
  isRegistrationPromptOpen.value = true;
  window.sessionStorage.setItem(key, '1');
}

async function loadFanProfile() {
  if (!props.eventId) return;

  const voteStatus = await fetchVoteStatus(props.eventId);
  hasVotedMvp.value = Boolean(voteStatus?.ok && voteStatus.hasVoted);

  const response = await fetchFanProfile(props.eventId);
  if (!response?.ok) return;
  const data = response.data || {};
  isRegisteredFan.value = Boolean(data.registered);
  if (data.session_token) {
    fanSessionToken.value = data.session_token;
  }
  if (data.registered) {
    fanId.value = Number(data.user?.id) || 0;
    fanNickname.value = data.user?.nickname || '';
    totalCoins.value = Math.max(0, Number(data.wallet) || 0);
    leaderboardUser.value = data.user_rank || null;
    fanRewardRedemptions.value = Array.isArray(data.reward_redemptions) ? data.reward_redemptions : [];
    fanLotteryTicket.value = data.lottery_ticket || null;
  } else if (Number.isFinite(Number(data.guest_coins))) {
    totalCoins.value = Math.max(totalCoins.value, Number(data.guest_coins) || 0);
    fanRewardRedemptions.value = [];
    fanLotteryTicket.value = null;
  }
}


async function handleExistingFanLogin() {
  await loadFanProfile();
  if (!isRegisteredFan.value) {
    return { ok: false, message: 'Impossibile trovare un profilo associato a questo numero.' };
  }
  return { ok: true };
}

async function handleRegistrationSubmit(form) {
  const response = await registerFanProfile({
    event_id: props.eventId,
    nickname: form.nickname,
    gender: form.gender,
    phone: form.phone,
    accepted_terms: form.acceptedTerms,
    guest_coins: totalCoins.value,
    enter_lottery: form.trigger === 'after_vote',
  });
  if (!response?.ok) {
    return { ok: false, message: response.message };
  }
  isRegisteredFan.value = true;
  fanId.value = Number(response.data?.user?.id) || 0;
  fanNickname.value = response.data?.user?.nickname || '';
  totalCoins.value = Math.max(0, Number(response.data?.wallet) || totalCoins.value);
  isRegistrationPromptOpen.value = false;
  await loadLeaderboardPreview();
  await loadFanProfile();
  return { ok: true, wallet: totalCoins.value };
}

function openProfileOverlay() {
  if (!isRegisteredFan.value) {
    openRegistrationPrompt('profile_overlay');
    return;
  }
  isProfileOverlayOpen.value = true;
}

function closeProfileOverlay() {
  isProfileOverlayOpen.value = false;
}

function openSpendPreview() {
  isSpendPreviewOpen.value = true;
}

function closeSpendPreview() {
  isSpendPreviewOpen.value = false;
}

async function attemptRedeem(rewardKey, costCoins, rewardLabel) {
  if (!isRegisteredFan.value) {
    selectedRewardLabel.value = rewardLabel || `${String(rewardKey).replace('-', ' ').toUpperCase()} · ${costCoins} 🪙`;
    openRegistrationPrompt('spend_redeem');
    return;
  }
  const response = await redeemFanReward(props.eventId, rewardKey, costCoins);
  if (response?.ok) {
    totalCoins.value = Number(response.data?.wallet) || totalCoins.value;
  }
}

function onFeatureSelect(featureId) {
  if (featureId === 'game-live') {
    isEarnModalOpen.value = true;
    return;
  }

  if (featureId === 'leaderboard-live') {
    openLeaderboard();
    return;
  }

  emit('feature-select', featureId);
}
</script>

<style scoped>
.live-experience {
  background:
    radial-gradient(circle at 50% -15%, rgba(59, 130, 246, 0.35), transparent 55%),
    radial-gradient(circle at 85% 26%, rgba(251, 191, 36, 0.32), transparent 38%),
    radial-gradient(circle at 14% 28%, rgba(255, 255, 255, 0.21), transparent 28%),
    linear-gradient(180deg, #030712 0%, #0f172a 45%, #030712 100%);
}

.arena-bg {
  background:
    radial-gradient(circle at 20% 24%, rgba(255, 255, 255, 0.46), transparent 13%),
    radial-gradient(circle at 83% 27%, rgba(255, 180, 92, 0.5), transparent 16%),
    radial-gradient(circle at 50% 68%, rgba(255, 255, 255, 0.16), transparent 35%),
    linear-gradient(180deg, rgba(15, 23, 42, 0.72) 0%, rgba(2, 6, 23, 0.85) 100%);
  filter: blur(1.8px);
}

.vignette {
  background: radial-gradient(circle at center, transparent 45%, rgba(2, 6, 23, 0.8) 100%);
}

.mini-feature {
  position: relative;
  display: flex;
  min-height: 106px;
  flex: 1;
  cursor: pointer;
  flex-direction: column;
  justify-content: space-between;
  overflow: hidden;
  border-radius: 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.35);
  padding: 0.5rem;
  transition: transform 0.15s ease;
}

.mini-feature:active {
  transform: scale(0.99);
}

.mini-feature--earn {
  background: linear-gradient(180deg, rgba(96, 165, 250, 0.26), rgba(23, 37, 84, 0.9));
  box-shadow: 0 0 22px rgba(59, 130, 246, 0.44);
}

.mini-feature--spend {
  background: linear-gradient(180deg, rgba(110, 231, 183, 0.24), rgba(20, 83, 45, 0.9));
  box-shadow: 0 0 22px rgba(34, 197, 94, 0.35);
}

.mini-feature__content {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.5rem;
  border: 1px solid rgba(255, 255, 255, 0.25);
  background: rgba(0, 0, 0, 0.22);
  margin-bottom: 0.45rem;
}

.mini-feature__coins {
  font-size: clamp(1.3rem, 5vw, 2rem);
  font-weight: 900;
  letter-spacing: -0.02em;
}

.mini-feature__icons {
  font-size: clamp(1.2rem, 4.5vw, 1.7rem);
  letter-spacing: 0.2rem;
}

.mini-feature__cta {
  width: 100%;
  border-radius: 0.5rem;
  border: 1px solid rgba(255, 255, 255, 0.25);
  padding: 0.38rem 0.5rem;
  font-size: clamp(0.72rem, 2.2vw, 0.88rem);
  font-weight: 900;
  letter-spacing: 0.04em;
  color: #fff;
}

.mini-feature__cta--earn {
  background: linear-gradient(180deg, #60a5fa, #1d4ed8);
}

.mini-feature__cta--spend {
  background: linear-gradient(180deg, #a3e635, #15803d);
}

.leaderboard-preview {
  display: flex;
  min-height: 220px;
  cursor: pointer;
  flex-direction: column;
  border-radius: 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.35);
  background: linear-gradient(180deg, rgba(250, 204, 21, 0.18), rgba(120, 53, 15, 0.84));
  padding: 0.55rem;
  box-shadow: 0 0 22px rgba(245, 158, 11, 0.35);
}

.leaderboard-preview__title {
  text-align: center;
  font-size: clamp(0.7rem, 2vw, 0.9rem);
  font-weight: 900;
}

.leaderboard-preview__list {
  margin-top: 0.4rem;
  display: flex;
  flex-direction: column;
  gap: 0.33rem;
}

.leaderboard-preview__item {
  display: flex;
  justify-content: space-between;
  border-radius: 0.4rem;
  background: rgba(2, 6, 23, 0.4);
  padding: 0.24rem 0.34rem;
  font-size: clamp(0.65rem, 1.95vw, 0.8rem);
  font-weight: 700;
}

.leaderboard-preview__you {
  margin-top: auto;
  border-radius: 0.45rem;
  border: 1px solid rgba(255, 255, 255, 0.2);
  background: rgba(2, 6, 23, 0.35);
  padding: 0.28rem;
  text-align: center;
  font-size: clamp(0.62rem, 1.8vw, 0.74rem);
  font-weight: 700;
}

.leaderboard-preview__cta {
  margin-top: 0.4rem;
  width: 100%;
  border-radius: 0.45rem;
  border: 1px solid rgba(255, 255, 255, 0.25);
  background: linear-gradient(180deg, #f59e0b, #b45309);
  padding: 0.36rem 0.45rem;
  font-size: clamp(0.66rem, 1.9vw, 0.78rem);
  font-weight: 900;
}
</style>
