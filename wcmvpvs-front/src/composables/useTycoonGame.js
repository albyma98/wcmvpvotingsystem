import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import {
  fetchTycoonState,
  sendTycoonBuyUpgrade,
  sendTycoonClick,
  sendTycoonRedeemCoupon,
} from "../api";
import { getOrCreateDeviceId } from "../deviceId";

export function trackEvent(name, payload) {
  try {
    if (typeof window !== "undefined" && typeof window.trackEvent === "function") {
      window.trackEvent(name, payload);
      return;
    }
    console.debug("[tycoon] trackEvent", name, payload);
  } catch (error) {
    console.warn("trackEvent stub failed", error);
  }
}

const SYNC_INTERVAL_MS = 5000;

const defaultConfig = {
  baseTickMs: 2000,
  offlineCapSeconds: 7200,
  clickCooldownMs: 300,
  costGrowthFactor: 1.6,
  basePointsPerTick: 1,
};

export function useTycoonGame() {
  const deviceId = getOrCreateDeviceId();
  const serverPoints = ref(0);
  const displayPoints = ref(0);
  const pointsPerTick = ref(1);
  const tickIntervalMs = ref(2000);
  const pointsPerSecond = ref(0);
  const upgrades = ref([]);
  const coupons = ref([]);
  const config = ref({ ...defaultConfig });
  const isClickCoolingDown = ref(false);
  const drumPulse = ref(false);
  const tickPulse = ref(false);
  const tickEventId = ref(0);
  const manualClicks = ref(0);
  const isSyncing = ref(false);
  const affordableUpgradeKeys = ref(new Set());

  let syncTimer = null;
  let clickCooldownTimer = null;
  let clickFlushTimer = null;
  let pendingClickCount = 0;
  let isSendingClicks = false;
  let trackedClickMilestone = 0;
  let tickFrameId = null;
  let lastTickTimestamp = 0;
  let thresholdSyncTimer = null;

  const upgradeViews = computed(() => {
    return upgrades.value.map((upgrade) => {
      const key = upgrade.key || upgrade.id;
      const nextCost = Number(upgrade.nextCost || 0);
      return {
        ...upgrade,
        id: key,
        key,
        nextCost,
        canAfford: displayPoints.value >= nextCost,
      };
    });
  });

  const quickUpgrade = computed(() => {
    const sorted = [...upgradeViews.value].sort((a, b) => a.nextCost - b.nextCost);
    return sorted.find((item) => item.canAfford) || sorted[0];
  });

  const formattedPoints = computed(() => displayPoints.value.toLocaleString("it-IT"));

  const couponViews = computed(() => {
    return coupons.value.map((coupon) => {
      const redeemed = Boolean(coupon.redeemed);
      const canRedeem = !redeemed && serverPoints.value >= (coupon.cost || 0);
      return {
        ...coupon,
        redeemed,
        canRedeem,
      };
    });
  });

  const pointsPerSecondDisplay = computed(() => Number((pointsPerSecond.value || 0).toFixed(2)));

  function clearClickCooldownTimer() {
    if (clickCooldownTimer) {
      clearTimeout(clickCooldownTimer);
      clickCooldownTimer = null;
    }
  }

  function clearThresholdSyncTimer() {
    if (thresholdSyncTimer) {
      clearTimeout(thresholdSyncTimer);
      thresholdSyncTimer = null;
    }
  }

  function scheduleThresholdSync() {
    clearThresholdSyncTimer();
    thresholdSyncTimer = setTimeout(async () => {
      thresholdSyncTimer = null;
      await syncState();
    }, 500);
  }

  function clearClickFlushTimer() {
    if (clickFlushTimer) {
      clearTimeout(clickFlushTimer);
      clickFlushTimer = null;
    }
  }

  function scheduleClickFlush() {
    if (clickFlushTimer) {
      return;
    }
    clickFlushTimer = setTimeout(() => {
      clickFlushTimer = null;
      void flushPendingClicks();
    }, 240);
  }

  async function flushPendingClicks() {
    if (isSendingClicks || pendingClickCount <= 0) {
      return;
    }
    isSendingClicks = true;
    try {
      while (pendingClickCount > 0) {
        pendingClickCount -= 1;
        const state = await sendTycoonClick(deviceId);
        applyServerState(state, { silent: true });
        displayPoints.value = serverPoints.value;
        const milestone = Math.floor(manualClicks.value / 10) * 10;
        if (milestone > 0 && milestone > trackedClickMilestone) {
          trackedClickMilestone = milestone;
          trackEvent("tycoon_manual_clicks", { count: milestone });
        }
      }
    } catch (error) {
      const state = error?.response?.data?.state;
      if (state) {
        applyServerState(state, { silent: true });
        displayPoints.value = serverPoints.value;
      } else {
        displayPoints.value = serverPoints.value;
      }
      if (error?.response?.status === 429) {
        scheduleClickCooldown();
      }
      console.warn("tycoon click failed", error);
    } finally {
      isSendingClicks = false;
      if (pendingClickCount > 0) {
        scheduleClickFlush();
      }
    }
  }

  function scheduleClickCooldown(lastClickAt) {
    clearClickCooldownTimer();
    const cooldownMs = config.value.clickCooldownMs || defaultConfig.clickCooldownMs;
    let remaining = cooldownMs;

    if (lastClickAt) {
      const parsed = Date.parse(lastClickAt);
      if (!Number.isNaN(parsed)) {
        const elapsed = Date.now() - parsed;
        remaining = Math.max(0, cooldownMs - elapsed);
      }
    }

    if (remaining <= 0) {
      isClickCoolingDown.value = false;
      return;
    }

    isClickCoolingDown.value = true;
    clickCooldownTimer = setTimeout(() => {
      isClickCoolingDown.value = false;
    }, remaining);
  }

  function runTick() {
    const increment = pointsPerTick.value || 0;
    if (increment <= 0) {
      return;
    }
    displayPoints.value = Math.max(0, Math.floor(displayPoints.value + increment));
    tickPulse.value = true;
    tickEventId.value += 1;
    setTimeout(() => (tickPulse.value = false), 220);
  }

  function tickerStep(timestamp) {
    const interval = tickIntervalMs.value || config.value.baseTickMs || defaultConfig.baseTickMs;
    if (!lastTickTimestamp) {
      lastTickTimestamp = timestamp;
    }
    if (interval && interval > 0 && timestamp - lastTickTimestamp >= interval) {
      runTick();
      lastTickTimestamp = timestamp;
    }
    tickFrameId = requestAnimationFrame(tickerStep);
  }

  function stopTicker() {
    if (tickFrameId) {
      cancelAnimationFrame(tickFrameId);
      tickFrameId = null;
    }
    lastTickTimestamp = 0;
  }

  function ensureTickerRunning() {
    if (typeof window === "undefined" || tickFrameId) {
      return;
    }
    tickFrameId = requestAnimationFrame(tickerStep);
  }

  function applyServerState(payload, options = {}) {
    const { silent = false } = options;
    if (!payload || typeof payload !== "object") {
      return;
    }

    const previousDisplay = displayPoints.value;
    serverPoints.value = Math.max(0, Math.floor(payload.points ?? 0));
    pointsPerTick.value = Math.max(0, Math.floor(payload.pointsPerTick ?? 0));

    const incomingTickInterval =
      payload.tickIntervalMs ?? config.value.baseTickMs ?? defaultConfig.baseTickMs;
    tickIntervalMs.value = Math.max(0, incomingTickInterval);

    if (typeof payload.pointsPerSecond === "number") {
      pointsPerSecond.value = payload.pointsPerSecond;
    } else if (tickIntervalMs.value > 0) {
      pointsPerSecond.value = Number(
        ((pointsPerTick.value * 1000) / tickIntervalMs.value).toFixed(2),
      );
    } else {
      pointsPerSecond.value = 0;
    }

    upgrades.value = Array.isArray(payload.upgrades) ? payload.upgrades : [];
    coupons.value = Array.isArray(payload.coupons) ? payload.coupons : [];

    if (payload.config && typeof payload.config === "object") {
      config.value = { ...config.value, ...payload.config };
    }

    const divergence = Math.abs(displayPoints.value - serverPoints.value);
    const threshold = (pointsPerTick.value || 0) * 3;
    if (divergence > threshold) {
      displayPoints.value = serverPoints.value;
    } else {
      displayPoints.value = serverPoints.value;
    }

    if (!silent && displayPoints.value > previousDisplay) {
      tickPulse.value = true;
      tickEventId.value += 1;
      setTimeout(() => (tickPulse.value = false), 220);
    }

    if (payload.lastClickAt) {
      scheduleClickCooldown(payload.lastClickAt);
    }
  }

  async function syncState() {
    if (isSyncing.value) {
      return;
    }
    isSyncing.value = true;
    try {
      const state = await fetchTycoonState(deviceId);
      applyServerState(state);
    } catch (error) {
      console.warn("tycoon sync failed", error);
    } finally {
      isSyncing.value = false;
    }
  }

  function startSyncLoop() {
    if (syncTimer) {
      clearInterval(syncTimer);
    }
    syncTimer = setInterval(syncState, SYNC_INTERVAL_MS);
  }

  function stopSyncLoop() {
    if (syncTimer) {
      clearInterval(syncTimer);
      syncTimer = null;
    }
  }

  watch(
    () => upgradeViews.value.map((upgrade) => ({ key: upgrade.key || upgrade.id, affordable: upgrade.canAfford })),
    (current) => {
      const nextAffordable = new Set();
      current.forEach((item) => {
        if (!item.key) return;
        if (item.affordable) {
          nextAffordable.add(item.key);
          if (!affordableUpgradeKeys.value.has(item.key)) {
            scheduleThresholdSync();
          }
        }
      });
      affordableUpgradeKeys.value = nextAffordable;
    },
    { deep: true }
  );

  function handleManualClick() {
    if (isClickCoolingDown.value) {
      return;
    }
    displayPoints.value = Math.max(0, Math.floor(displayPoints.value + 1));
    manualClicks.value += 1;
    drumPulse.value = true;
    setTimeout(() => (drumPulse.value = false), 180);
    scheduleClickCooldown();
    pendingClickCount += 1;
    scheduleClickFlush();
  }

  async function buyUpgrade(upgradeId) {
    if (!upgradeId) {
      return false;
    }
    try {
      const state = await sendTycoonBuyUpgrade(upgradeId, deviceId);
      applyServerState(state);
      trackEvent("tycoon_upgrade_purchase", {
        upgradeId,
        remainingPoints: state.points,
      });
      return true;
    } catch (error) {
      const state = error?.response?.data?.state;
      if (state) {
        applyServerState(state);
      }
      displayPoints.value = serverPoints.value;
      console.warn("tycoon upgrade failed", error);
      return false;
    }
  }

  async function redeemCoupon(couponId) {
    if (!couponId) {
      return false;
    }
    try {
      const state = await sendTycoonRedeemCoupon(couponId, deviceId);
      applyServerState(state);
      trackEvent("coupon_redeem", { couponId, remainingPoints: state.points });
      return true;
    } catch (error) {
      const state = error?.response?.data?.state;
      if (state) {
        applyServerState(state);
      }
      console.warn("coupon redeem failed", error);
      return false;
    }
  }

  onMounted(() => {
    ensureTickerRunning();
    syncState();
    startSyncLoop();
  });

  onUnmounted(() => {
    stopSyncLoop();
    clearClickCooldownTimer();
    clearClickFlushTimer();
    stopTicker();
    clearThresholdSyncTimer();
  });

  return {
    BASE_TICK_MS: computed(() => config.value.baseTickMs),
    OFFLINE_CAP_SECONDS: computed(() => config.value.offlineCapSeconds),
    COST_GROWTH_FACTOR: computed(() => config.value.costGrowthFactor),
    serverPoints,
    displayPoints,
    formattedPoints,
    pointsPerTick,
    tickIntervalMs,
    pointsPerSecond: pointsPerSecondDisplay,
    upgradeViews,
    quickUpgrade,
    couponViews,
    isClickCoolingDown,
    drumPulse,
    tickPulse,
    tickEventId,
    handleManualClick,
    buyUpgrade,
    redeemCoupon,
  };
}

export default useTycoonGame;
