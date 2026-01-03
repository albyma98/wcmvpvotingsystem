import { computed, onMounted, onUnmounted, ref } from "vue";
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
  const points = ref(0);
  const pointsPerTick = ref(0);
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

  let syncTimer = null;
  let clickCooldownTimer = null;

  const upgradeViews = computed(() => {
    return upgrades.value.map((upgrade) => {
      const nextCost = Number(upgrade.nextCost || 0);
      return {
        ...upgrade,
        nextCost,
        canAfford: points.value >= nextCost,
      };
    });
  });

  const quickUpgrade = computed(() => {
    const sorted = [...upgradeViews.value].sort((a, b) => a.nextCost - b.nextCost);
    return sorted.find((item) => item.canAfford) || sorted[0];
  });

  const formattedPoints = computed(() => points.value.toLocaleString("it-IT"));

  const couponViews = computed(() => {
    return coupons.value.map((coupon) => {
      const redeemed = Boolean(coupon.redeemed);
      const canRedeem = !redeemed && points.value >= (coupon.cost || 0);
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

  function applyState(payload) {
    if (!payload || typeof payload !== "object") {
      return;
    }

    const previousPoints = points.value;
    points.value = Math.max(0, Math.floor(payload.points ?? 0));
    pointsPerTick.value = Math.max(0, Math.floor(payload.pointsPerTick ?? 0));

    const tickInterval = payload.tickIntervalMs || config.value.baseTickMs || defaultConfig.baseTickMs;
    if (typeof payload.pointsPerSecond === "number") {
      pointsPerSecond.value = payload.pointsPerSecond;
    } else if (tickInterval > 0) {
      pointsPerSecond.value = Number(((pointsPerTick.value * 1000) / tickInterval).toFixed(2));
    } else {
      pointsPerSecond.value = 0;
    }

    upgrades.value = Array.isArray(payload.upgrades) ? payload.upgrades : [];
    coupons.value = Array.isArray(payload.coupons) ? payload.coupons : [];

    if (payload.config && typeof payload.config === "object") {
      config.value = { ...config.value, ...payload.config };
    }

    if (points.value > previousPoints) {
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
      applyState(state);
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

  async function handleManualClick() {
    if (isClickCoolingDown.value) {
      return;
    }
    manualClicks.value += 1;
    drumPulse.value = true;
    setTimeout(() => (drumPulse.value = false), 180);
    scheduleClickCooldown();

    try {
      const state = await sendTycoonClick(deviceId);
      applyState(state);
      if (manualClicks.value % 10 === 0) {
        trackEvent("tycoon_manual_clicks", { count: manualClicks.value });
      }
    } catch (error) {
      const state = error?.response?.data?.state;
      if (state) {
        applyState(state);
      }
      if (error?.response?.status === 429) {
        scheduleClickCooldown();
      }
      console.warn("tycoon click failed", error);
    }
  }

  async function buyUpgrade(upgradeId) {
    if (!upgradeId) {
      return false;
    }
    try {
      const state = await sendTycoonBuyUpgrade(upgradeId, deviceId);
      applyState(state);
      trackEvent("tycoon_upgrade_purchase", {
        upgradeId,
        remainingPoints: state.points,
      });
      return true;
    } catch (error) {
      const state = error?.response?.data?.state;
      if (state) {
        applyState(state);
      }
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
      applyState(state);
      trackEvent("coupon_redeem", { couponId, remainingPoints: state.points });
      return true;
    } catch (error) {
      const state = error?.response?.data?.state;
      if (state) {
        applyState(state);
      }
      console.warn("coupon redeem failed", error);
      return false;
    }
  }

  onMounted(() => {
    syncState();
    startSyncLoop();
  });

  onUnmounted(() => {
    stopSyncLoop();
    clearClickCooldownTimer();
  });

  return {
    BASE_TICK_MS: computed(() => config.value.baseTickMs),
    OFFLINE_CAP_SECONDS: computed(() => config.value.offlineCapSeconds),
    COST_GROWTH_FACTOR: computed(() => config.value.costGrowthFactor),
    points,
    formattedPoints,
    pointsPerTick,
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
