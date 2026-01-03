<script setup>
import { computed, ref } from "vue";
import useTycoonGame from "../composables/useTycoonGame";

const {
  points,
  formattedPoints,
  pointsPerTick,
  pointsPerSecond,
  upgradeViews,
  quickUpgrade,
  isClickCoolingDown,
  drumPulse,
  tickPulse,
  handleManualClick,
  buyUpgrade,
} = useTycoonGame();

const nextQuickCost = computed(() => quickUpgrade.value?.nextCost ?? 0);
const quickLabel = computed(() => quickUpgrade.value?.name || "Upgrade");
const primaryUpgrades = computed(() =>
  upgradeViews.value.filter((upgrade) =>
    ["tamburello", "megafono", "coro"].includes(upgrade.id),
  ),
);
const tappedUpgradeId = ref(null);

function handleUpgradeTap(upgradeId) {
  tappedUpgradeId.value = upgradeId;
  setTimeout(() => {
    if (tappedUpgradeId.value === upgradeId) {
      tappedUpgradeId.value = null;
    }
  }, 160);
  buyUpgrade(upgradeId);
}
</script>

<template>
  <section class="tycoon-card" aria-label="Idle Tycoon">
    <div class="tycoon-header">
      <div>
        <p class="tycoon-eyebrow">Mini-gioco</p>
        <h3 class="tycoon-title">Idle Tycoon curva</h3>
      </div>
      <div class="tycoon-rate" aria-live="polite">
        <span class="tycoon-dot"></span>
        <span>+{{ pointsPerSecond }} /s</span>
      </div>
    </div>

    <div class="tycoon-body">
      <div class="tycoon-stats">
        <p class="tycoon-label">Punti</p>
        <p class="tycoon-points">{{ formattedPoints }}</p>
        <p class="tycoon-hint">+{{ pointsPerTick }} per tick</p>
      </div>

      <button
        type="button"
        class="tycoon-drum"
        :class="{
          'tycoon-drum--cooldown': isClickCoolingDown,
          'tycoon-drum--pulse': drumPulse || tickPulse,
        }"
        :disabled="isClickCoolingDown"
        @click="handleManualClick"
      >
        <span class="tycoon-drum__emoji" aria-hidden="true">🥁</span>
        <span class="tycoon-drum__cta">Tocca</span>
      </button>

      <div class="tycoon-quick">
        <div>
          <p class="tycoon-label">Upgrade rapido</p>
          <p class="tycoon-hint">{{ quickLabel }}</p>
        </div>
        <button
          type="button"
          class="tycoon-upgrade-btn"
          :disabled="points < nextQuickCost"
          @click="buyUpgrade(quickUpgrade?.id)"
        >
          <span>{{ quickLabel }}</span>
          <span class="tycoon-cost">{{ nextQuickCost.toLocaleString("it-IT") }} pts</span>
        </button>
      </div>

      <div class="tycoon-upgrade-grid" role="list">
        <button
          v-for="upgrade in primaryUpgrades"
          :key="upgrade.id"
          type="button"
          class="tycoon-upgrade-box"
          :class="{
            'tycoon-upgrade-box--disabled': !upgrade.canAfford,
            'tycoon-upgrade-box--tapped': tappedUpgradeId === upgrade.id,
          }"
          :disabled="!upgrade.canAfford"
          @click="handleUpgradeTap(upgrade.id)"
        >
          <span class="tycoon-upgrade__icon" aria-hidden="true">{{ upgrade.icon }}</span>
          <div class="tycoon-upgrade__text">
            <p class="tycoon-upgrade__title">{{ upgrade.name }}</p>
            <p class="tycoon-upgrade__desc">{{ upgrade.description }}</p>
          </div>
          <div class="tycoon-upgrade__meta">
            <span class="tycoon-cost">{{ upgrade.nextCost.toLocaleString("it-IT") }} pts</span>
            <span class="tycoon-level">Lv. {{ upgrade.level }}</span>
          </div>
        </button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.tycoon-card {
  background: radial-gradient(circle at 10% 10%, rgba(251, 191, 36, 0.08), transparent 45%),
    radial-gradient(circle at 90% 0%, rgba(248, 113, 113, 0.06), transparent 40%),
    linear-gradient(135deg, #0f172a, #0b1220 70%);
  border: 1px solid rgba(248, 180, 0, 0.18);
  border-radius: 18px;
  padding: 14px 14px 12px;
  box-shadow: 0 12px 28px rgba(0, 0, 0, 0.32);
  color: #f8fafc;
  touch-action: manipulation;
  user-select: none;
}

.tycoon-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.tycoon-eyebrow {
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: #cbd5e1;
  margin: 0;
}

.tycoon-title {
  margin: 2px 0 0;
  font-weight: 700;
  font-size: 16px;
  color: #fbbf24;
}

.tycoon-rate {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  border-radius: 12px;
  background: rgba(234, 179, 8, 0.12);
  color: #f8fafc;
  font-weight: 700;
  font-size: 13px;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.05);
}

.tycoon-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: radial-gradient(circle, #fbbf24 0%, #f59e0b 80%);
  box-shadow: 0 0 8px rgba(251, 191, 36, 0.8);
}

.tycoon-body {
  display: grid;
  grid-template-columns: 1fr auto;
  grid-template-rows: auto auto auto auto;
  gap: 10px;
  margin-top: 12px;
}

.tycoon-stats {
  grid-column: 1 / 2;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.tycoon-label {
  margin: 0;
  color: #cbd5e1;
  font-size: 12px;
  letter-spacing: 0.04em;
}

.tycoon-points {
  margin: 0;
  font-size: 28px;
  font-weight: 800;
  color: #f8fafc;
  text-shadow: 0 2px 10px rgba(0, 0, 0, 0.35);
}

.tycoon-hint {
  margin: 0;
  font-size: 12px;
  color: #e2e8f0;
  opacity: 0.8;
}

.tycoon-drum {
  grid-column: 2 / 3;
  grid-row: 1 / 4;
  align-self: stretch;
  justify-self: end;
  width: 96px;
  height: 96px;
  border-radius: 18px;
  border: 1px solid rgba(251, 191, 36, 0.18);
  background: radial-gradient(circle at 30% 30%, rgba(251, 191, 36, 0.16), transparent 60%),
    linear-gradient(160deg, #1f2937, #0f172a);
  color: #fbbf24;
  font-weight: 800;
  position: relative;
  overflow: hidden;
  transition: transform 140ms ease, box-shadow 140ms ease, border-color 140ms ease;
  box-shadow: 0 10px 26px rgba(0, 0, 0, 0.3);
}

.tycoon-drum__emoji {
  font-size: 34px;
  display: block;
  line-height: 1;
}

.tycoon-drum__cta {
  display: block;
  font-size: 12px;
  margin-top: 4px;
  color: #f8fafc;
}

.tycoon-drum:hover:not(:disabled) {
  transform: scale(1.03);
  box-shadow: 0 12px 28px rgba(251, 191, 36, 0.2);
}

.tycoon-drum:active:not(:disabled) {
  transform: scale(0.97);
}

.tycoon-drum--pulse {
  animation: drumPulse 260ms ease;
}

@keyframes drumPulse {
  0% {
    transform: scale(1) rotate(0deg);
  }
  40% {
    transform: scale(1.05) rotate(-6deg);
  }
  70% {
    transform: scale(0.98) rotate(6deg);
  }
  100% {
    transform: scale(1) rotate(0deg);
  }
}

.tycoon-drum--cooldown {
  border-color: rgba(148, 163, 184, 0.4);
  color: #cbd5e1;
  cursor: not-allowed;
  opacity: 0.7;
}

.tycoon-quick {
  grid-column: 1 / 3;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: rgba(248, 180, 0, 0.08);
  border: 1px solid rgba(248, 180, 0, 0.25);
  border-radius: 14px;
  padding: 10px 12px;
  gap: 10px;
}

.tycoon-upgrade-btn {
  border: 1px solid rgba(251, 191, 36, 0.35);
  background: linear-gradient(135deg, #f59e0b, #d97706);
  color: #0f172a;
  padding: 8px 12px;
  border-radius: 12px;
  font-weight: 800;
  display: inline-flex;
  gap: 8px;
  align-items: center;
  transition: transform 140ms ease, box-shadow 140ms ease, filter 140ms ease;
  box-shadow: 0 8px 18px rgba(234, 179, 8, 0.25);
}

.tycoon-upgrade-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  box-shadow: none;
}

.tycoon-upgrade-btn:not(:disabled):active {
  transform: translateY(1px);
}

.tycoon-cost {
  font-size: 12px;
  color: #0b1220;
  background: rgba(251, 191, 36, 0.35);
  padding: 3px 6px;
  border-radius: 8px;
  font-weight: 700;
}

.tycoon-upgrade-grid {
  grid-column: 1 / 3;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}

.tycoon-upgrade-box {
  position: relative;
  aspect-ratio: 1 / 1;
  padding: 12px 10px;
  border-radius: 14px;
  border: 1px solid rgba(148, 163, 184, 0.3);
  background: linear-gradient(165deg, rgba(15, 23, 42, 0.85), rgba(17, 24, 39, 0.9));
  color: #e2e8f0;
  display: grid;
  grid-template-rows: auto 1fr auto;
  gap: 6px;
  align-items: start;
  text-align: left;
  transition:
    transform 120ms ease,
    box-shadow 120ms ease,
    border-color 120ms ease,
    background 120ms ease;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03), 0 10px 22px rgba(0, 0, 0, 0.32);
}

.tycoon-upgrade-box:not(:disabled):active {
  transform: scale(0.97);
}

.tycoon-upgrade-box:hover:not(:disabled) {
  transform: translateY(-1px);
  border-color: rgba(251, 191, 36, 0.5);
  background: linear-gradient(165deg, rgba(248, 180, 0, 0.12), rgba(17, 24, 39, 0.95));
  box-shadow: 0 12px 26px rgba(251, 191, 36, 0.18);
}

.tycoon-upgrade-box--disabled {
  opacity: 0.55;
  cursor: not-allowed;
  filter: grayscale(0.6);
}

.tycoon-upgrade-box--tapped {
  box-shadow: 0 0 0 8px rgba(251, 191, 36, 0.08);
}

.tycoon-upgrade__icon {
  font-size: 26px;
  margin: 0 auto;
  display: block;
}

.tycoon-upgrade__text {
  display: grid;
  gap: 2px;
}

.tycoon-upgrade__title {
  margin: 0;
  font-weight: 800;
  color: #fbbf24;
}

.tycoon-upgrade__desc {
  margin: 0;
  font-size: 12px;
  color: #cbd5e1;
}

.tycoon-upgrade__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  font-weight: 700;
}

.tycoon-level {
  padding: 3px 8px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.18);
  color: #cbd5e1;
  font-size: 12px;
}

@media (max-width: 430px) {
  .tycoon-card {
    max-height: none;
    overflow: visible;
  }

  .tycoon-body {
    grid-template-columns: 1fr auto;
    gap: 8px;
  }

  .tycoon-upgrade-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 8px;
  }
}
</style>
