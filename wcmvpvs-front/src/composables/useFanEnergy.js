import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { apiClient, getDeviceHeaders } from "../api";

const ENERGY_CAP = 999;
const BASE_GAIN = 5;
const BOOST_GAIN = 20;
const TICK_MS = 5_000;
const BOOST_DURATION_MS = 30_000;
const BOOST_COOLDOWN_MS = 5 * 60 * 1_000;
const CLAIM_COST = 100;

function computeGain(startMs, endMs, boostActiveUntilMs) {
  if (!Number.isFinite(startMs) || !Number.isFinite(endMs) || endMs <= startMs) {
    return 0;
  }

  const boostStartMs = boostActiveUntilMs - BOOST_DURATION_MS;
  let total = 0;

  const applyBase = (from, to) => {
    if (to <= from) return;
    const ticks = Math.floor((to - from) / TICK_MS);
    total += ticks * BASE_GAIN;
  };

  const applyBoost = (from, to) => {
    if (to <= from) return;
    const ticks = Math.floor((to - from) / TICK_MS);
    total += ticks * BOOST_GAIN;
  };

  if (boostActiveUntilMs > startMs && boostActiveUntilMs > 0) {
    const windowStart = Math.max(startMs, boostStartMs);
    const windowEnd = Math.min(endMs, boostActiveUntilMs);

    if (windowStart > startMs) {
      applyBase(startMs, windowStart);
    }

    if (windowEnd > windowStart) {
      applyBoost(windowStart, windowEnd);
    }

    if (endMs > windowEnd) {
      applyBase(windowEnd, endMs);
    }
  } else {
    applyBase(startMs, endMs);
  }

  return total;
}

function formatSeconds(seconds) {
  const s = Math.max(0, Math.ceil(seconds));
  const minutes = Math.floor(s / 60);
  const remainingSeconds = s % 60;
  if (minutes <= 0) {
    return `${remainingSeconds}s`;
  }
  return `${minutes}:${remainingSeconds.toString().padStart(2, "0")}`;
}

function useFanEnergy() {
  const energy = ref(0);
  const tickets = ref(0);
  const boostReadyAt = ref(0);
  const boostActiveUntil = ref(0);
  const lastSyncTs = ref(Date.now());
  const nowTs = ref(Date.now());
  const claimPulse = ref(false);

  let tickTimer = 0;
  let refreshTimer = 0;
  let isFetching = false;

  const currentEnergy = computed(() => {
    const gained = computeGain(lastSyncTs.value, nowTs.value, boostActiveUntil.value);
    return Math.min(ENERGY_CAP, energy.value + gained);
  });

  const isBoostActive = computed(() => nowTs.value < boostActiveUntil.value);
  const isBoostReady = computed(() => !isBoostActive.value && nowTs.value >= boostReadyAt.value);

  const boostCooldownRemainingMs = computed(() => {
    if (isBoostActive.value) {
      return Math.max(0, boostActiveUntil.value - nowTs.value);
    }
    if (isBoostReady.value) {
      return 0;
    }
    return Math.max(0, boostReadyAt.value - nowTs.value);
  });

  const boostProgressPct = computed(() => {
    if (isBoostActive.value) {
      const elapsed = BOOST_DURATION_MS - boostCooldownRemainingMs.value;
      return Math.min(100, Math.max(0, (elapsed / BOOST_DURATION_MS) * 100));
    }
    if (isBoostReady.value) {
      return 100;
    }
    const remaining = boostCooldownRemainingMs.value;
    const done = BOOST_COOLDOWN_MS - remaining;
    return Math.min(100, Math.max(0, (done / BOOST_COOLDOWN_MS) * 100));
  });

  const boostLabel = computed(() => {
    if (isBoostActive.value) {
      return `BOOST ATTIVO (${formatSeconds(boostCooldownRemainingMs.value / 1000)})`;
    }
    if (isBoostReady.value) {
      return "Pronto";
    }
    return `Pronto tra ${formatSeconds(boostCooldownRemainingMs.value / 1000)}`;
  });

  const canCollect = computed(() => currentEnergy.value >= CLAIM_COST);
  const canBoost = computed(() => isBoostReady.value);

  const applyServerState = (payload = {}) => {
    const nowValue = Number(payload.now) || Date.now();
    energy.value = Number(payload.energy) || 0;
    tickets.value = Number(payload.tickets) || 0;
    boostReadyAt.value = Number(payload.boostReadyAt) || nowValue;
    boostActiveUntil.value = Number(payload.boostActiveUntil) || 0;
    lastSyncTs.value = nowValue;
    nowTs.value = nowValue;
  };

  const fetchStatus = async () => {
    if (isFetching) return;
    isFetching = true;
    try {
      const { data } = await apiClient.get("/fan-energy/status", {
        headers: getDeviceHeaders(),
      });
      applyServerState(data || {});
    } catch (error) {
      console.warn("fan energy status error", error);
    } finally {
      isFetching = false;
    }
  };

  const triggerBoost = async () => {
    if (!canBoost.value) {
      return false;
    }
    try {
      const { data } = await apiClient.post(
        "/fan-energy/boost",
        {},
        { headers: getDeviceHeaders() },
      );
      applyServerState(data || {});
      return true;
    } catch (error) {
      console.warn("fan energy boost error", error);
      return false;
    }
  };

  const claim = async () => {
    if (!canCollect.value) {
      return false;
    }
    try {
      const { data } = await apiClient.post(
        "/fan-energy/claim",
        { amount: CLAIM_COST },
        { headers: getDeviceHeaders() },
      );
      applyServerState(data || {});
      claimPulse.value = true;
      setTimeout(() => {
        claimPulse.value = false;
      }, 500);
      return true;
    } catch (error) {
      console.warn("fan energy claim error", error);
      return false;
    }
  };

  const startLoops = () => {
    tickTimer = window.setInterval(() => {
      nowTs.value = Date.now();
    }, 1000);
    refreshTimer = window.setInterval(fetchStatus, 5000);
  };

  const stopLoops = () => {
    if (tickTimer) {
      clearInterval(tickTimer);
      tickTimer = 0;
    }
    if (refreshTimer) {
      clearInterval(refreshTimer);
      refreshTimer = 0;
    }
  };

  const handleVisibility = () => {
    if (document.visibilityState === "visible") {
      fetchStatus();
    }
  };

  onMounted(() => {
    fetchStatus();
    startLoops();
    document.addEventListener("visibilitychange", handleVisibility);
  });

  onBeforeUnmount(() => {
    stopLoops();
    document.removeEventListener("visibilitychange", handleVisibility);
  });

  return {
    energy: currentEnergy,
    tickets,
    isBoostActive,
    isBoostReady,
    boostLabel,
    boostProgressPct,
    canCollect,
    canBoost,
    triggerBoost,
    claim,
    claimPulse,
    fetchStatus,
  };
}

export default useFanEnergy;
