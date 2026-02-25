<template>
  <div class="live-experience relative h-[100dvh] overflow-hidden text-white">
    <div class="arena-bg absolute inset-0" aria-hidden="true" />
    <div class="vignette absolute inset-0" aria-hidden="true" />

    <main class="relative z-10 flex h-full flex-col px-3 pb-3 pt-3 sm:px-4">
      <LiveHeader
        :team-name="teamName"
        :team-logo-url="teamLogoUrl"
        :is-live="isLive"
        :sponsor-line="sponsorLine"
      >
        <StoriesBar
          v-if="activeStories.length"
          :stories="activeStories"
          :seen-ids="seenStoryIds"
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

      <section class="animate-on-enter mt-[3.2vh] grid grid-cols-3 gap-2.5">
        <FeatureCard
          v-for="feature in decoratedFeatures"
          :key="feature.id"
          v-bind="feature"
          @select="onFeatureSelect"
        />
      </section>

      <LiveResultsBar class="animate-on-enter mt-auto" :results="results" />
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
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import EarnCoinsModal from '../components/EarnCoinsModal.vue';
import FeatureCard from '../components/FeatureCard.vue';
import LiveHeader from '../components/LiveHeader.vue';
import LiveResultsBar from '../components/LiveResultsBar.vue';
import StoriesBar from '../components/StoriesBar.vue';
import StoryModal from '../components/StoryModal.vue';
import { apiClient } from '../api';

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
  results: {
    type: Array,
    default: () => [
      { name: 'ROSSI', value: 42 },
      { name: 'BIANCHI', value: 37 },
      { name: 'VERDI', value: 21 },
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
});

const emit = defineEmits(['feature-select']);
const isEarnModalOpen = ref(false);
const totalCoins = ref(0);
const walletTargetEl = ref(null);

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

  loadEventStories();
});

async function addCoins(amount) {
  const parsed = Math.max(0, Number(amount) || 0);
  totalCoins.value += parsed;

  if (typeof window !== 'undefined') {
    window.localStorage.setItem('wallet:coins', String(totalCoins.value));
  }

  await nextTick();
  syncWalletTargetEl();
}


const eventStories = ref([]);
const seenStoryIds = ref([]);
const isStoryModalOpen = ref(false);
const activeStoryIndex = ref(0);

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
  activeStoryIndex.value = Math.max(0, Math.min(index, activeStories.value.length - 1));
  isStoryModalOpen.value = true;
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

const decoratedFeatures = computed(() =>
  props.features.map((feature) => {
    if (feature.id !== 'vote-mvp') {
      if (feature.id !== 'game-live') {
        return feature;
      }

      return {
        ...feature,
        centerBadge: `🪙 ${totalCoins.value}`,
        centerBadgeId: 'wallet-coin-target',
      };
    }

    const hasVotedPlayer = Boolean(props.votedPlayerImageUrl);
    const playerLastName = String(props.votedPlayerLastName || '').trim();
    const fallbackName = String(props.votedPlayerName || '').trim();
    const titleLabel = hasVotedPlayer ? (playerLastName || fallbackName || feature.title) : feature.title;
    const hasPlayerNumber = String(props.votedPlayerNumber || '').trim() !== '';
    const subtitleLabel = hasVotedPlayer
      ? (hasPlayerNumber ? `#${String(props.votedPlayerNumber).trim()}` : '')
      : feature.subtitle;

    return {
      ...feature,
      title: titleLabel,
      subtitle: subtitleLabel,
      actionLabel: hasVotedPlayer ? 'MODIFICA' : feature.actionLabel,
      previewImageUrl: hasVotedPlayer ? props.votedPlayerImageUrl : anonymousAvatarDataUrl,
      previewImageFit: hasVotedPlayer ? 'contain' : 'cover',
      previewAlt: props.votedPlayerName
        ? `MVP selezionato: ${props.votedPlayerName}`
        : 'Avatar anonimo MVP',
    };
  }),
);


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
    loadEventStories();
  },
);

watch(currentStory, (story) => {
  if (!story) {
    return;
  }
  markStorySeen(Number(story.id));
});

watch(isStoryModalOpen, (open) => {
  if (typeof document === 'undefined') {
    return;
  }
  document.body.style.overflow = open ? 'hidden' : '';
});

onBeforeUnmount(() => {
  if (typeof document !== 'undefined') {
    document.body.style.overflow = '';
  }
});

function onFeatureSelect(featureId) {
  if (featureId === 'game-live') {
    isEarnModalOpen.value = true;
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
  background: radial-gradient(circle at center, rgba(2, 6, 23, 0) 44%, rgba(2, 6, 23, 0.8) 100%);
}

.animate-on-enter {
  animation: fade-slide-up 0.6s ease both;
}

@keyframes fade-slide-up {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-height: 760px) {
  .hero {
    margin-top: 2vh;
  }

  :deep(article) {
    min-height: 196px;
    padding-top: 0.5rem;
    padding-bottom: 0.5rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .animate-on-enter {
    animation: none;
  }
}
</style>
