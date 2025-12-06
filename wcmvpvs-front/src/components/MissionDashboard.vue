<template>
  <section class="mission-pack">
    <header class="mission-pack__header">
      <div>
        <p class="mission-pack__eyebrow">🏅 Missioni tifoso</p>
        <h3 class="mission-pack__title">Completa le missioni per ottenere più chance e badge esclusivi.</h3>
      </div>
      <div class="mission-pack__progress">
        <div class="mission-pack__progress-bar" aria-hidden="true">
          <div
            class="mission-pack__progress-fill"
            :style="{ width: `${progressPercent}%` }"
          ></div>
          <span
            class="mission-pack__progress-label"
            :class="{ 'mission-pack__progress-label--pulse': progressLabelPulse }"
          >
            {{ completedMissions }} / {{ totalMissions }} completate
          </span>
        </div>
      </div>
    </header>

    <div class="mission-grid" role="navigation" aria-label="Mission Pack 2.0">
      <button
        v-for="mission in missions"
        :key="mission.id"
        type="button"
        class="mission-tile"
        :class="{
          'mission-tile--disabled': !isMissionEnabled(mission.id),
          'mission-tile--completed': mission.completed,
          'mission-tile--celebrate': mission.justCompleted,
        }"
        :disabled="!isMissionEnabled(mission.id)"
        @click="handleMissionClick(mission.id)"
      >
        <div class="mission-tile__main">
          <div class="mission-tile__icon" aria-hidden="true">{{ resolveIcon(mission.icon) }}</div>
          <div class="mission-tile__texts">
            <p class="mission-tile__label">{{ mission.label }}</p>
            <p v-if="mission.id === 'reaction_test' && mission.bestScoreMs" class="mission-tile__meta">
              Miglior tempo: {{ mission.bestScoreMs }} ms
            </p>
            <p class="mission-tile__reward">{{ mission.rewardText }}</p>
          </div>
        </div>
        <span v-if="mission.completed" class="mission-tile__badge">COMPLETATA ✅</span>
      </button>
    </div>
  </section>
</template>

<script setup>
import { computed, ref, watch } from 'vue';

const props = defineProps({
  missions: {
    type: Array,
    default: () => [],
  },
  completedMissions: {
    type: Number,
    default: 0,
  },
  totalMissions: {
    type: Number,
    default: 0,
  },
  isRegistered: {
    type: Boolean,
    default: false,
  },
  registrationThreshold: {
    type: Number,
    default: 2,
  },
  canOpenVoteTrend: {
    type: Boolean,
    default: true,
  },
  canOpenSelfie: {
    type: Boolean,
    default: true,
  },
  canOpenReactionTest: {
    type: Boolean,
    default: true,
  },
});

const emit = defineEmits([
  'open-vote-trend',
  'open-self-mvp',
  'open-reaction-test',
  'request-registration',
]);

const progressLabelPulse = ref(false);
const registrationRequested = ref(false);

const progressPercent = computed(() => {
  if (!props.totalMissions) {
    return 0;
  }
  return Math.min(100, Math.round((props.completedMissions / props.totalMissions) * 100));
});

watch(
  () => props.completedMissions,
  (current, previous) => {
    if (current !== previous) {
      progressLabelPulse.value = true;
      window.setTimeout(() => {
        progressLabelPulse.value = false;
      }, 280);
    }

    if (
      !registrationRequested.value &&
      !props.isRegistered &&
      props.registrationThreshold > 0 &&
      current >= props.registrationThreshold
    ) {
      registrationRequested.value = true;
      emit('request-registration');
    }
  },
  { immediate: true },
);

const missionIconMap = {
  chart: '📈',
  selfie: '🤳',
  flash: '⚡',
};

const resolveIcon = (icon) => missionIconMap[icon] || '🏅';

const handleMissionClick = (missionId) => {
  if (missionId === 'vote_trend') {
    emit('open-vote-trend');
  } else if (missionId === 'self_mvp') {
    emit('open-self-mvp');
  } else if (missionId === 'reaction_test') {
    emit('open-reaction-test');
  }
};

const isMissionEnabled = (missionId) => {
  if (missionId === 'vote_trend') {
    return props.canOpenVoteTrend;
  }
  if (missionId === 'self_mvp') {
    return props.canOpenSelfie;
  }
  if (missionId === 'reaction_test') {
    return props.canOpenReactionTest;
  }
  return true;
};
</script>

<style scoped>
.mission-pack {
  margin-top: 1rem;
  padding: 1rem 1.25rem;
  border-radius: 1.75rem;
  border: 1px solid rgba(148, 163, 184, 0.2);
  background: linear-gradient(135deg, rgba(30, 41, 59, 0.85), rgba(15, 23, 42, 0.9));
  box-shadow: 0 24px 54px rgba(8, 15, 28, 0.55);
}

.mission-pack__header {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin-bottom: 1rem;
}

.mission-pack__eyebrow {
  font-size: 0.95rem;
  font-weight: 800;
  letter-spacing: 0.05em;
  color: #f8fafc;
}

.mission-pack__title {
  font-size: 1rem;
  color: #cbd5e1;
  line-height: 1.4;
}

.mission-pack__progress {
  margin-top: 0.4rem;
}

.mission-pack__progress-bar {
  position: relative;
  width: 100%;
  height: 16px;
  border-radius: 999px;
  overflow: hidden;
  background: linear-gradient(90deg, rgba(15, 23, 42, 0.9), rgba(15, 23, 42, 0.85));
  border: 1px solid rgba(148, 163, 184, 0.3);
}

.mission-pack__progress-fill {
  position: absolute;
  inset: 0;
  width: 0;
  background: linear-gradient(90deg, #22d3ee, #8b5cf6, #f97316);
  transition: width 320ms ease;
}

.mission-pack__progress-label {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.85rem;
  font-weight: 700;
  color: #e2e8f0;
  text-shadow: 0 1px 8px rgba(0, 0, 0, 0.35);
  transition: transform 200ms ease, opacity 200ms ease;
}

.mission-pack__progress-label--pulse {
  transform: scale(1.05);
  opacity: 0.95;
}

.mission-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 0.75rem;
}

.mission-tile {
  position: relative;
  width: 100%;
  border-radius: 1.25rem;
  padding: 0.9rem 1rem;
  border: 1px solid rgba(148, 163, 184, 0.25);
  background: linear-gradient(145deg, rgba(30, 41, 59, 0.8), rgba(15, 23, 42, 0.9));
  color: #e2e8f0;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  box-shadow: 0 16px 36px rgba(8, 15, 28, 0.5);
  transition: transform 160ms ease, box-shadow 200ms ease, border-color 200ms ease;
}

.mission-tile:hover:not(:disabled),
.mission-tile:active:not(:disabled) {
  transform: scale(1.03);
  box-shadow: 0 20px 48px rgba(56, 189, 248, 0.15);
  border-color: rgba(94, 234, 212, 0.6);
}

.mission-tile--disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.mission-tile--completed {
  border-color: rgba(74, 222, 128, 0.6);
  box-shadow: 0 18px 42px rgba(74, 222, 128, 0.18);
}

.mission-tile--celebrate {
  animation: mission-pop 420ms ease;
}

.mission-tile__main {
  display: flex;
  gap: 0.65rem;
  align-items: center;
  width: 100%;
}

.mission-tile__icon {
  width: 48px;
  height: 48px;
  border-radius: 14px;
  background: linear-gradient(135deg, rgba(14, 165, 233, 0.24), rgba(59, 130, 246, 0.2));
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 1.35rem;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.08);
}

.mission-tile__texts {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  align-items: flex-start;
}

.mission-tile__label {
  font-size: 1rem;
  font-weight: 700;
}

.mission-tile__meta {
  font-size: 0.85rem;
  color: #a5b4fc;
}

.mission-tile__reward {
  font-size: 0.9rem;
  color: #22d3ee;
  font-weight: 600;
}

.mission-tile__badge {
  position: absolute;
  top: 10px;
  right: 10px;
  background: rgba(34, 197, 94, 0.18);
  color: #bbf7d0;
  font-size: 0.75rem;
  font-weight: 800;
  padding: 0.4rem 0.6rem;
  border-radius: 999px;
  border: 1px solid rgba(34, 197, 94, 0.45);
  box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.05);
  animation: badge-fade 260ms ease;
}

@keyframes mission-pop {
  0% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.05);
    box-shadow: 0 22px 54px rgba(34, 211, 238, 0.28);
  }
  100% {
    transform: scale(1);
  }
}

@keyframes badge-fade {
  from {
    opacity: 0;
    transform: translateY(-4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
