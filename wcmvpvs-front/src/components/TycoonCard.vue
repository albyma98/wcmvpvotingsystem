<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import useTycoonGame from "../composables/useTycoonGame";

const UPGRADE_META = {
  drum: { key: "drum", serverKey: "tamburello", title: "Tamburello", icon: "🥁", effect: "+1 per tick" },
  megaphone: { key: "megaphone", serverKey: "megafono", title: "Megafono", icon: "📣", effect: "+3 per tick" },
  choir: { key: "choir", serverKey: "coro", title: "Coro", icon: "🎶", effect: "+12% prod." },
};

const {
  serverPoints,
  formattedPoints,
  pointsPerTick,
  pointsPerSecond,
  upgradeViews,
  isClickCoolingDown,
  drumPulse,
  tickPulse,
  tickEventId,
  handleManualClick,
  buyUpgrade,
  couponViews,
  redeemCoupon,
} = useTycoonGame();

const normalizeUpgradeKey = (key) => {
  const lowered = (key || "").toString().toLowerCase();
  if (lowered === "tamburello") return "drum";
  if (lowered === "megafono") return "megaphone";
  if (lowered === "coro") return "choir";
  return lowered;
};

const upgradeCards = computed(() => {
  const byKey = new Map();
  upgradeViews.value.forEach((upgrade) => {
    const normalized = normalizeUpgradeKey(upgrade.key || upgrade.id);
    if (!normalized) return;
    byKey.set(normalized, { ...upgrade, normalizedKey: normalized });
  });

  return Object.values(UPGRADE_META).map((meta) => {
    const data = byKey.get(meta.key) || {};
    const rawCost = Number(data.nextCost ?? 0);
    const cost = Number.isFinite(rawCost) && rawCost > 0 ? rawCost : 0;
    const affordable = data.canAfford ?? (cost > 0 && serverPoints.value >= cost);
    return {
      key: meta.key,
      purchaseKey: data.key || data.id || meta.serverKey || meta.key,
      title: data.name || meta.title,
      icon: data.icon || meta.icon,
      effect: data.effectLabel || data.description || meta.effect,
      level: Number.isFinite(data.level) ? data.level : 0,
      cost,
      affordable,
      locked: cost <= 0 || !affordable,
    };
  });
});

const tappedUpgradeId = ref(null);
const floatingTexts = ref([]);
const cardRef = ref(null);
const drumRef = ref(null);
const prefersReducedMotion = ref(false);
const isCouponPanelOpen = ref(false);
const redeemedNoticeId = ref(null);
let motionMediaQuery;
let cleanupMotion;
let floatingId = 0;

const recommendedUpgradeId = computed(() => {
  const candidate = upgradeCards.value
    .filter((item) => item.affordable && !item.locked)
    .sort((a, b) => a.cost - b.cost)[0];
  return candidate?.key ?? null;
});

function handleUpgradeTap(upgrade) {
  const upgradeId = upgrade?.key;
  const purchaseKey = upgrade?.purchaseKey || upgradeId;
  tappedUpgradeId.value = upgradeId;
  setTimeout(() => {
    if (tappedUpgradeId.value === upgradeId) {
      tappedUpgradeId.value = null;
    }
  }, 160);
  buyUpgrade(purchaseKey);
}

function removeFloating(id) {
  floatingTexts.value = floatingTexts.value.filter((item) => item.id !== id);
}

function addFloatingText(payload) {
  if (prefersReducedMotion.value) {
    return;
  }
  const id = ++floatingId;
  floatingTexts.value.push({ ...payload, id });
  const duration = payload.type === "tick" ? 780 : 640;
  setTimeout(() => removeFloating(id), duration);
}

function getRelativeCoords(event) {
  const cardEl = cardRef.value;
  if (!cardEl) {
    return null;
  }
  const cardRect = cardEl.getBoundingClientRect();
  const x = event?.clientX != null ? event.clientX - cardRect.left : cardRect.width / 2;
  const y = event?.clientY != null ? event.clientY - cardRect.top : cardRect.height / 2;
  return { x, y };
}

function spawnTickText() {
  if (!drumRef.value || prefersReducedMotion.value) {
    return;
  }
  const drumRect = drumRef.value.getBoundingClientRect();
  const cardRect = cardRef.value?.getBoundingClientRect();
  if (!cardRect) {
    return;
  }
  const x = drumRect.left + drumRect.width / 2 - cardRect.left;
  const y = drumRect.top + drumRect.height / 2 - cardRect.top;
  addFloatingText({ x, y, value: `+${pointsPerTick.value}`, type: "tick" });
}

function spawnClickText(event) {
  const coords = getRelativeCoords(event);
  if (!coords || prefersReducedMotion.value) {
    return;
  }
  addFloatingText({ ...coords, value: "+1", type: "click" });
}

function handleDrumClick(event) {
  handleManualClick();
  spawnClickText(event);
}

function toggleCouponPanel() {
  isCouponPanelOpen.value = !isCouponPanelOpen.value;
}

function handleCouponRedeem(couponId) {
  const ok = redeemCoupon(couponId);
  if (ok) {
    redeemedNoticeId.value = couponId;
    setTimeout(() => {
      if (redeemedNoticeId.value === couponId) {
        redeemedNoticeId.value = null;
      }
    }, 2000);
  }
}

function setupMotionPreference() {
  if (typeof window === "undefined" || !window.matchMedia) {
    return;
  }
  motionMediaQuery = window.matchMedia("(prefers-reduced-motion: reduce)");
  prefersReducedMotion.value = motionMediaQuery.matches;
  const handleChange = (mediaEvent) => {
    prefersReducedMotion.value = mediaEvent.matches;
    if (mediaEvent.matches) {
      floatingTexts.value = [];
    }
  };
  motionMediaQuery.addEventListener("change", handleChange);
  return () => motionMediaQuery.removeEventListener("change", handleChange);
}

onMounted(() => {
  cleanupMotion = setupMotionPreference();
});

onBeforeUnmount(() => {
  if (cleanupMotion) {
    cleanupMotion();
  }
  floatingTexts.value = [];
});

watch(tickEventId, (current, previous) => {
  if (previous === undefined || current > previous) {
    spawnTickText();
  }
});
</script>

<template>
  <section ref="cardRef" class="tycoon-card" aria-label="Idle Tycoon">
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
        ref="drumRef"
        class="tycoon-drum"
        :class="{
          'tycoon-drum--cooldown': isClickCoolingDown,
          'tycoon-drum--tick': tickPulse && !prefersReducedMotion,
          'tycoon-drum--click': drumPulse && !tickPulse && !prefersReducedMotion,
        }"
        :disabled="isClickCoolingDown"
        @click="handleDrumClick"
      >
        <span class="tycoon-drum__emoji" aria-hidden="true">🥁</span>
        <span class="tycoon-drum__cta">Tocca</span>
      </button>

      <div class="tycoon-upgrade-grid" role="list">
        <button
          v-for="upgrade in upgradeCards"
          :key="upgrade.key"
          type="button"
          class="tycoon-upgrade-box"
          :class="{
            'tycoon-upgrade-box--disabled': upgrade.locked || !upgrade.affordable,
            'tycoon-upgrade-box--tapped': tappedUpgradeId === upgrade.key,
            'tycoon-upgrade-box--recommended': recommendedUpgradeId === upgrade.key,
            'tycoon-upgrade-box--active': upgrade.affordable && !upgrade.locked,
          }"
          :disabled="upgrade.locked || !upgrade.affordable"
          @click="handleUpgradeTap(upgrade)"
        >
          <span v-if="upgrade.locked || !upgrade.affordable" class="tycoon-lock" aria-hidden="true"
            >🔒</span
          >
          <span class="tycoon-upgrade__icon" aria-hidden="true">{{ upgrade.icon }}</span>
          <div class="tycoon-upgrade__text">
            <p class="tycoon-upgrade__title">{{ upgrade.title }}</p>
            <p class="tycoon-upgrade__desc">{{ upgrade.effect }}</p>
          </div>
          <div class="tycoon-upgrade__meta">
            <span class="tycoon-cost" :class="{ 'tycoon-cost--disabled': !upgrade.affordable }">
              {{ upgrade.cost.toLocaleString("it-IT") }} pts
            </span>
            <span class="tycoon-level">Lv. {{ upgrade.level }}</span>
          </div>
        </button>
      </div>

      <button
        type="button"
        class="tycoon-coupon-toggle"
        :class="{ 'tycoon-coupon-toggle--open': isCouponPanelOpen }"
        @click="toggleCouponPanel"
      >
        <span class="tycoon-coupon-toggle__icon" aria-hidden="true">🎟️</span>
        <span class="tycoon-coupon-toggle__text">Coupon rapidi</span>
        <span class="tycoon-coupon-toggle__badge">{{ couponViews.length }}</span>
        <span class="tycoon-coupon-toggle__chevron" aria-hidden="true">
          {{ isCouponPanelOpen ? "▲" : "▼" }}
        </span>
      </button>

      <div
        v-if="isCouponPanelOpen"
        class="tycoon-coupon-panel"
        role="region"
        aria-live="polite"
      >
        <div
          v-for="coupon in couponViews"
          :key="coupon.id"
          class="tycoon-coupon-row"
          :class="{ 'tycoon-coupon-row--redeemed': coupon.redeemed }"
        >
          <div class="tycoon-coupon__info">
            <p class="tycoon-coupon__name">{{ coupon.name }}</p>
            <p class="tycoon-coupon__cost">{{ coupon.cost.toLocaleString("it-IT") }} pts</p>
          </div>
          <div class="tycoon-coupon__actions">
            <span v-if="coupon.redeemed" class="tycoon-coupon__status">Coupon riscattato</span>
            <button
              v-else
              type="button"
              class="tycoon-coupon__btn"
              :disabled="!coupon.canRedeem"
              @click.stop="handleCouponRedeem(coupon.id)"
            >
              {{ coupon.canRedeem ? "Riscatta" : "Punti insufficienti" }}
            </button>
          </div>
        </div>
        <p v-if="redeemedNoticeId" class="tycoon-coupon__feedback">Coupon riscattato</p>
      </div>
    </div>

    <div class="tycoon-float-layer" aria-hidden="true">
      <span
        v-for="float in floatingTexts"
        :key="float.id"
        class="tycoon-float"
        :class="{
          'tycoon-float--tick': float.type === 'tick',
          'tycoon-float--click': float.type === 'click',
        }"
        :style="{
          left: `${float.x}px`,
          top: `${float.y}px`,
        }"
      >
        {{ float.value }}
      </span>
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
  position: relative;
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

.tycoon-drum--tick {
  animation: tickAnim 320ms ease;
}

.tycoon-drum--click {
  animation: clickAnim 200ms ease;
}

@keyframes tickAnim {
  0% {
    transform: scale(1) rotate(0deg);
  }
  25% {
    transform: scale(1.12) rotate(-10deg);
  }
  60% {
    transform: scale(0.95) rotate(8deg);
  }
  100% {
    transform: scale(1) rotate(0deg);
  }
}

@keyframes clickAnim {
  0% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.06);
  }
  100% {
    transform: scale(1);
  }
}

.tycoon-drum--cooldown {
  border-color: rgba(148, 163, 184, 0.4);
  color: #cbd5e1;
  cursor: not-allowed;
  opacity: 0.7;
}


.tycoon-cost {
  font-size: 11px;
  color: #0b1220;
  background: rgba(251, 191, 36, 0.35);
  padding: 2px 6px;
  border-radius: 8px;
  font-weight: 700;
}

.tycoon-float-layer {
  pointer-events: none;
  position: absolute;
  inset: 0;
  overflow: hidden;
}

.tycoon-float {
  position: absolute;
  transform: translate(-50%, -50%);
  font-weight: 800;
  white-space: nowrap;
  text-shadow: 0 6px 20px rgba(0, 0, 0, 0.5);
}

.tycoon-float--tick {
  color: #fbbf24;
  font-size: 18px;
  animation: floatTick 760ms ease-out forwards;
}

.tycoon-float--click {
  color: #cbd5e1;
  font-size: 14px;
  animation: floatClick 620ms ease-out forwards;
}

@keyframes floatTick {
  0% {
    opacity: 1;
    transform: translate(-50%, -50%) scale(1);
  }
  60% {
    opacity: 1;
    transform: translate(-50%, -80%) scale(1.06);
  }
  100% {
    opacity: 0;
    transform: translate(-50%, -115%) scale(1.12);
  }
}

@keyframes floatClick {
  0% {
    opacity: 1;
    transform: translate(-50%, -50%) scale(1);
  }
  60% {
    opacity: 0.9;
    transform: translate(-50%, -70%) scale(1.02);
  }
  100% {
    opacity: 0;
    transform: translate(-50%, -95%) scale(1.05);
  }
}

.tycoon-upgrade-grid {
  grid-column: 1 / 3;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  align-items: stretch;
}


.tycoon-upgrade-box {
  position: relative;
  min-height: 128px;
  height: 100%;
  padding: 10px 8px 12px;
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.25);
  background: linear-gradient(165deg, rgba(15, 23, 42, 0.9), rgba(17, 24, 39, 0.94));
  color: #e2e8f0;
  display: grid;
  grid-template-rows: auto 1fr auto;
  gap: 4px;
  align-items: start;
  text-align: left;
  transition:
    transform 120ms ease,
    box-shadow 120ms ease,
    border-color 120ms ease,
    background 120ms ease;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03), 0 8px 18px rgba(0, 0, 0, 0.3);
  isolation: isolate;
}

.tycoon-upgrade-box:not(:disabled):active {
  transform: scale(0.97);
}


.tycoon-upgrade-box:hover:not(:disabled) {
  transform: translateY(-1px);
  border-color: rgba(251, 191, 36, 0.45);
  background: linear-gradient(165deg, rgba(248, 180, 0, 0.12), rgba(17, 24, 39, 0.95));
  box-shadow: 0 10px 20px rgba(251, 191, 36, 0.16);
}

.tycoon-upgrade-box--active:not(:disabled) {
  border-color: rgba(251, 191, 36, 0.6);
  box-shadow: 0 10px 24px rgba(251, 191, 36, 0.16);
}


.tycoon-upgrade-box--disabled {
  opacity: 0.45;
  cursor: not-allowed;
  filter: grayscale(0.6);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.02);
  pointer-events: none;
}

.tycoon-upgrade-box--recommended:not(.tycoon-upgrade-box--disabled) {
  border-color: rgba(251, 191, 36, 0.6);
  box-shadow: 0 0 0 1px rgba(251, 191, 36, 0.35), 0 12px 22px rgba(251, 191, 36, 0.18);
}

.tycoon-upgrade-box--coupon {
  border-color: rgba(94, 234, 212, 0.4);
  background: linear-gradient(165deg, rgba(15, 23, 42, 0.94), rgba(34, 197, 94, 0.06));
}

.tycoon-upgrade-box--coupon:hover {
  border-color: rgba(94, 234, 212, 0.7);
  box-shadow: 0 10px 20px rgba(45, 212, 191, 0.2);
}

.tycoon-upgrade-box--tapped {
  box-shadow: 0 0 0 8px rgba(251, 191, 36, 0.08);
}


.tycoon-upgrade__icon {
  font-size: 22px;
  margin: 0 auto 2px;
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
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tycoon-upgrade__desc {
  margin: 0;
  font-size: 11px;
  color: #cbd5e1;
  line-height: 1.3;
}

.tycoon-upgrade__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 4px;
  font-weight: 700;
}

.tycoon-level {
  padding: 2px 7px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.22);
  color: #cbd5e1;
  font-size: 11px;
}

.tycoon-lock {
  position: absolute;
  top: 6px;
  right: 8px;
  font-size: 13px;
  opacity: 0.85;
}

.tycoon-cost--disabled {
  background: rgba(148, 163, 184, 0.28);
  color: #0f172a;
}

.tycoon-coupon-toggle {
  grid-column: 1 / 3;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  margin-top: 4px;
  border-radius: 10px;
  border: 1px solid rgba(94, 234, 212, 0.4);
  background: linear-gradient(165deg, rgba(15, 23, 42, 0.94), rgba(34, 197, 94, 0.06));
  color: #e2e8f0;
  font-weight: 800;
  cursor: pointer;
  transition: border-color 120ms ease, transform 120ms ease, box-shadow 120ms ease;
}

.tycoon-coupon-toggle:hover {
  transform: translateY(-1px);
  border-color: rgba(94, 234, 212, 0.7);
  box-shadow: 0 10px 20px rgba(45, 212, 191, 0.15);
}

.tycoon-coupon-toggle--open {
  border-color: rgba(45, 212, 191, 0.75);
  box-shadow: inset 0 0 0 1px rgba(45, 212, 191, 0.25);
}

.tycoon-coupon-toggle__icon {
  font-size: 18px;
}

.tycoon-coupon-toggle__text {
  font-size: 13px;
}

.tycoon-coupon-toggle__badge {
  background: rgba(45, 212, 191, 0.2);
  border-radius: 999px;
  padding: 2px 8px;
  font-size: 12px;
  color: #a5f3fc;
}

.tycoon-coupon-toggle__chevron {
  margin-left: auto;
  opacity: 0.8;
}

.tycoon-coupon-panel {
  grid-column: 1 / 3;
  background: rgba(15, 23, 42, 0.8);
  border: 1px solid rgba(94, 234, 212, 0.25);
  border-radius: 12px;
  margin-top: 4px;
  padding: 8px;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
  display: grid;
  gap: 8px;
  max-height: 170px;
  overflow: hidden;
}

.tycoon-coupon-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 10px;
  border-radius: 10px;
  background: rgba(15, 23, 42, 0.8);
  border: 1px solid rgba(148, 163, 184, 0.15);
}

.tycoon-coupon-row--redeemed {
  border-color: rgba(94, 234, 212, 0.35);
  background: rgba(34, 197, 94, 0.07);
}

.tycoon-coupon__info {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.tycoon-coupon__name {
  margin: 0;
  color: #e2e8f0;
  font-weight: 700;
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tycoon-coupon__cost {
  margin: 0;
  color: #94a3b8;
  font-size: 12px;
}

.tycoon-coupon__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.tycoon-coupon__btn {
  background: linear-gradient(135deg, #0ea5e9, #22d3ee);
  color: #0b1220;
  font-weight: 800;
  border: none;
  padding: 8px 10px;
  border-radius: 10px;
  cursor: pointer;
  transition: transform 120ms ease, box-shadow 120ms ease, filter 120ms ease;
  min-width: 118px;
}

.tycoon-coupon__btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 10px 22px rgba(34, 211, 238, 0.25);
}

.tycoon-coupon__btn:disabled {
  cursor: not-allowed;
  background: linear-gradient(135deg, #475569, #1e293b);
  color: #cbd5e1;
  box-shadow: none;
}

.tycoon-coupon__status {
  color: #34d399;
  font-weight: 700;
  font-size: 12px;
}

.tycoon-coupon__feedback {
  margin: 0;
  text-align: center;
  color: #34d399;
  font-weight: 800;
  letter-spacing: 0.01em;
}

@media (prefers-reduced-motion: reduce) {
  .tycoon-drum--tick,
  .tycoon-drum--click,
  .tycoon-float--tick,
  .tycoon-float--click {
    animation: none;
  }
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

  .tycoon-coupon-panel {
    max-height: 150px;
    overflow: auto;
  }
}
</style>
