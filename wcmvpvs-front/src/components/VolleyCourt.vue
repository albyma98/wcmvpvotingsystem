<script setup>
import { computed } from 'vue';
import PlayerCard from './PlayerCard.vue';
import teamLogo from '../assets/team-logo.svg';

const props = defineProps({
  players: {
    type: Array,
    default: () => [],
  },
  cardSize: {
    type: Number,
    default: 90,
  },
  selectedPlayerId: {
    type: Number,
    default: null,
  },
  disableVotes: {
    type: Boolean,
    default: false,
  },
  isVoting: {
    type: Boolean,
    default: false,
  },
  courtSponsors: {
    type: Array,
    default: () => [],
  },
  isPreMatch: {
    type: Boolean,
    default: false,
  },
  votingOpen: {
    type: Boolean,
    default: false,
  },
});

const emits = defineEmits(['select', 'sponsor-click']);

const sponsorDimensions = computed(() => {
  const cardBaseWidth = Number.isFinite(props.cardSize) ? props.cardSize : 90;
  return {
    width: `${cardBaseWidth * 1.75}px`,
    height: `${cardBaseWidth * 1.1}px`,
  };
});

const sponsorCardBaseClass =
  'pointer-events-auto group relative flex items-center justify-center overflow-hidden rounded-3xl border border-white/15 bg-black/35 shadow-[0_20px_42px_rgba(8,15,28,0.45)] backdrop-blur';

const positionStyle = computed(() => (player) => ({
  left: `${player.position.x}%`,
  top: `${player.position.y}%`,
  transform: 'translate(-50%, -50%)',
}));

const preMatchCardStyle = (index) => {
  if (!props.isPreMatch) {
    return {};
  }

  const baseDelayMs = 90;

  return {
    '--prematch-delay': `${index * baseDelayMs}ms`,
    '--prematch-duration': '260ms',
    '--prematch-ease': 'cubic-bezier(1, 1, 1, 1)',
  };
};

const sponsorList = computed(() =>
  Array.isArray(props.courtSponsors)
    ? props.courtSponsors.filter((sponsor) => Boolean(sponsor))
    : [],
);

const hasTwoSponsors = computed(() => sponsorList.value.length >= 2);
const centerSponsor = computed(() => (!hasTwoSponsors.value ? sponsorList.value[0] ?? null : null));
const leftSponsor = computed(() => (hasTwoSponsors.value ? sponsorList.value[0] ?? null : null));
const rightSponsor = computed(() => (hasTwoSponsors.value ? sponsorList.value[1] ?? null : null));

const emitSponsorClick = (sponsor) => {
  if (!sponsor) {
    return;
  }
  emits('sponsor-click', sponsor);
};

const findCentralReferencePlayer = () => {
  if (!Array.isArray(props.players) || !props.players.length) {
    return null;
  }

  const perfectCenter = props.players.find(
    (player) => Math.abs(player.position.x - 50) < 0.001 && Math.abs(player.position.y - 50) < 0.001,
  );
  if (perfectCenter) {
    return perfectCenter;
  }

  const sortedByDistance = [...props.players].sort((a, b) => {
    const distanceA = Math.hypot(a.position.x - 50, a.position.y - 50);
    const distanceB = Math.hypot(b.position.x - 50, b.position.y - 50);
    return distanceA - distanceB;
  });

  return sortedByDistance[0] ?? null;
};

const centralReferencePlayer = computed(() => findCentralReferencePlayer());

const sponsorVerticalCenter = computed(() => 50);

const resolveColumnReference = (direction = 'left') => {
  const center = centralReferencePlayer.value;
  if (!center) {
    return direction === 'left' ? 20 : 80;
  }

  const candidates = props.players
    .filter((player) => (direction === 'left' ? player.position.x < center.position.x : player.position.x > center.position.x))
    .sort((a, b) =>
      direction === 'left'
        ? b.position.x - a.position.x
        : a.position.x - b.position.x,
    );

  if (candidates.length) {
    return candidates[0].position.x;
  }

  return direction === 'left' ? Math.max(center.position.x - 30, 10) : Math.min(center.position.x + 30, 90);
};

const leftColumnReference = computed(() => resolveColumnReference('left'));
const rightColumnReference = computed(() => resolveColumnReference('right'));

const leftSponsorStyle = computed(() => ({
  top: `${sponsorVerticalCenter.value}%`,
  left: `${leftColumnReference.value}%`,
  transform: 'translate(-50%, -50%)',
}));

const rightSponsorStyle = computed(() => ({
  top: `${sponsorVerticalCenter.value}%`,
  left: `${rightColumnReference.value}%`,
  transform: 'translate(-50%, -50%)',
}));

const centerSponsorStyle = computed(() => ({
  top: `${sponsorVerticalCenter.value}%`,
  left: '50%',
  transform: 'translate(-50%, -50%)',
}));
</script>

<template>
  <section class="relative mx-auto h-full w-full overflow-hidden">
    <div
      class="absolute inset-0 overflow-hidden rounded-[2.75rem] border-4 border-[rgba(64,34,10,0.35)] bg-gradient-to-b from-court-light via-court-base to-court-dark shadow-court"
    >
      <div class="absolute inset-0 opacity-50 mix-blend-soft-light bg-court-wood-planks"></div>
      <div class="absolute inset-0 opacity-70 mix-blend-overlay bg-court-wood-grain"></div>
      <div class="absolute inset-0 flex items-center justify-center pointer-events-none">
        <img
          :src="teamLogo"
          alt="Team crest"
          class="h-[42%] opacity-[0.14] drop-shadow-[0_20px_35px_rgba(65,34,9,0.35)]"
        />
      </div>
      <div class="absolute inset-0 pointer-events-none opacity-[0.65] bg-[radial-gradient(circle_at_center,_rgba(0,0,0,0)_38%,_rgba(31,20,9,0.28)_100%)]"></div>
      <div class="absolute inset-x-[12%] top-1/2 border-t border-b border-white/25"></div>
      <div class="absolute inset-x-0 top-1/2 h-[18px] -translate-y-1/2">
        <div class="absolute inset-x-[8%] top-0 h-[4px] rounded-full bg-white/80 shadow-lg shadow-white/30"></div>
        <div class="absolute inset-x-[6%] top-1/2 h-[2px] -translate-y-1/2 bg-white/90"></div>
        <div class="absolute inset-x-[8%] bottom-0 h-[1px] bg-white/70"></div>
      </div>
      <div class="absolute inset-x-[12%] top-[25%] h-[2px] bg-white/30"></div>
      <div class="absolute inset-x-[12%] top-[75%] h-[2px] bg-white/30"></div>
    </div>

    <div class="absolute inset-0">
      <div v-if="centerSponsor" class="absolute z-10" :style="centerSponsorStyle">
        <div class="flex flex-col items-center gap-2 text-center">
          <a
            v-if="centerSponsor.link"
            :class="sponsorCardBaseClass"
            :style="sponsorDimensions"
            :href="centerSponsor.link"
            target="_blank"
            rel="noopener noreferrer"
            :aria-label="centerSponsor.name || 'Sponsor'"
            @click="emitSponsorClick(centerSponsor)"
          >
            <div
              class="absolute inset-0 bg-gradient-to-br from-white/10 via-transparent to-white/5 opacity-0 transition-opacity duration-300 group-hover:opacity-100"
            ></div>
            <img
              v-if="centerSponsor.image"
              :src="centerSponsor.image"
              :alt="centerSponsor.name || 'Sponsor'"
              class="relative h-full w-full object-cover"
            />
          </a>
          <div
            v-else
            :class="sponsorCardBaseClass"
            :style="sponsorDimensions"
            :aria-label="centerSponsor.name || 'Sponsor'"
            role="group"
            @click="emitSponsorClick(centerSponsor)"
          >
            <div
              class="absolute inset-0 bg-gradient-to-br from-white/10 via-transparent to-white/5 opacity-0 transition-opacity duration-300 group-hover:opacity-100"
            ></div>
            <img
              v-if="centerSponsor.image"
              :src="centerSponsor.image"
              :alt="centerSponsor.name || 'Sponsor'"
              class="relative h-full w-full object-cover"
            />
          </div>
        </div>
      </div>

      <div v-if="leftSponsor" class="absolute z-10" :style="leftSponsorStyle">
        <div class="flex flex-col items-center gap-2 text-center">
          <a
            v-if="leftSponsor.link"
            :class="sponsorCardBaseClass"
            :style="sponsorDimensions"
            :href="leftSponsor.link"
            target="_blank"
            rel="noopener noreferrer"
            :aria-label="leftSponsor.name || 'Sponsor'"
            @click="emitSponsorClick(leftSponsor)"
          >
            <div class="absolute inset-0 bg-gradient-to-br from-white/10 via-transparent to-white/5 opacity-0 transition-opacity duration-300 group-hover:opacity-100"></div>
            <img
              v-if="leftSponsor.image"
              :src="leftSponsor.image"
              :alt="leftSponsor.name || 'Sponsor'"
              class="relative h-full w-full object-cover"
            />
          </a>
          <div
            v-else
            :class="sponsorCardBaseClass"
            :style="sponsorDimensions"
            :aria-label="leftSponsor.name || 'Sponsor'"
            role="group"
            @click="emitSponsorClick(leftSponsor)"
          >
            <div class="absolute inset-0 bg-gradient-to-br from-white/10 via-transparent to-white/5 opacity-0 transition-opacity duration-300 group-hover:opacity-100"></div>
            <img
              v-if="leftSponsor.image"
              :src="leftSponsor.image"
              :alt="leftSponsor.name || 'Sponsor'"
              class="relative h-full w-full object-cover"
            />
          </div>
        </div>
      </div>

      <div v-if="rightSponsor" class="absolute z-10" :style="rightSponsorStyle">
        <div class="flex flex-col items-center gap-2 text-center">
          <a
            v-if="rightSponsor.link"
            :class="sponsorCardBaseClass"
            :style="sponsorDimensions"
            :href="rightSponsor.link"
            target="_blank"
            rel="noopener noreferrer"
            :aria-label="rightSponsor.name || 'Sponsor'"
            @click="emitSponsorClick(rightSponsor)"
          >
            <div class="absolute inset-0 bg-gradient-to-br from-white/10 via-transparent to-white/5 opacity-0 transition-opacity duration-300 group-hover:opacity-100"></div>
            <img
              v-if="rightSponsor.image"
              :src="rightSponsor.image"
              :alt="rightSponsor.name || 'Sponsor'"
              class="relative h-full w-full object-cover"
            />
          </a>
          <div
            v-else
            :class="sponsorCardBaseClass"
            :style="sponsorDimensions"
            :aria-label="rightSponsor.name || 'Sponsor'"
            role="group"
            @click="emitSponsorClick(rightSponsor)"
          >
            <div class="absolute inset-0 bg-gradient-to-br from-white/10 via-transparent to-white/5 opacity-0 transition-opacity duration-300 group-hover:opacity-100"></div>
            <img
              v-if="rightSponsor.image"
              :src="rightSponsor.image"
              :alt="rightSponsor.name || 'Sponsor'"
              class="relative h-full w-full object-cover"
            />
          </div>
        </div>
      </div>

      <TransitionGroup
        name="prematch-card"
        tag="div"
        class="absolute inset-0"
        :class="{ 'is-pre-match': isPreMatch }"
      >
        <div
          v-for="(player, index) in players"
          :key="player.id"
          class="absolute prematch-card-item"
          :style="[positionStyle(player), preMatchCardStyle(index)]"
        >
          <PlayerCard
            :player="player"
            :card-size="cardSize"
            :is-selected="selectedPlayerId === player.id"
            :disabled="disableVotes && selectedPlayerId !== player.id"
            :is-voting="isVoting"
            :is-pre-match="isPreMatch"
            :voting-open="votingOpen"
            @select="() => emits('select', player)"
          />
        </div>
      </TransitionGroup>
    </div>
  </section>
</template>

<style scoped>
.prematch-card-enter-active,
.prematch-card-leave-active {
  transition-property: opacity, transform;
  transition-duration: var(--prematch-duration, 240ms);
  transition-timing-function: var(--prematch-ease, ease-out);
  transition-delay: var(--prematch-delay, 0s);
}

.prematch-card-leave-active {
  transition-delay: 0s;
}

.prematch-card-enter-from {
  opacity: 0;
  transform: translateY(10px) scale(0.98);
}

.prematch-card-enter-to {
  opacity: 1;
  transform: translateY(0) scale(1);
}

.prematch-card-leave-from {
  opacity: 1;
  transform: translateY(0) scale(1);
}

.prematch-card-leave-to {
  opacity: 0;
  transform: translateY(6px) scale(0.98);
}

.prematch-card-item {
  will-change: transform, opacity;
}
</style>
