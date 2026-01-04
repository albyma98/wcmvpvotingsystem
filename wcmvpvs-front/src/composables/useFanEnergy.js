import { computed, onBeforeUnmount, onMounted, ref } from "vue";

const STORAGE_KEY = "fan_energy_state";
const ENERGY_CAP = 999;
const ENERGY_PER_TICK = 1;
const BASE_TICK_MS = 8000;
const BOOST_TICK_MS = 2000;
const BOOST_DURATION_MS = 30_000;
const BOOST_COOLDOWN_MS = 5 * 60 * 1000;

function parseNumber(value, fallback = 0) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function loadState() {
  if (typeof window === "undefined" || !window.localStorage) {
    const now = Date.now();
    return {
      energy: 0,
      lastTs: now,
      boostReadyAt: 0,
      boostActiveUntil: 0,
    };
  }

  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      const now = Date.now();
      return {
        energy: 0,
        lastTs: now,
        boostReadyAt: 0,
        boostActiveUntil: 0,
      };
    }

    const parsed = JSON.parse(raw);
    const now = Date.now();
    return {
      energy: Math.min(ENERGY_CAP, Math.max(0, parseNumber(parsed.energy, 0))),
      lastTs: parseNumber(parsed.lastTs, now),
      boostReadyAt: parseNumber(parsed.boostReadyAt, 0),
      boostActiveUntil: parseNumber(parsed.boostActiveUntil, 0),
    };
  } catch (error) {
    console.warn("fan energy load failed", error);
    const now = Date.now();
    return {
      energy: 0,
      lastTs: now,
      boostReadyAt: 0,
      boostActiveUntil: 0,
    };
  }
}

function persistState(state) {
  if (typeof window === "undefined" || !window.localStorage) {
    return;
  }

  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
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

export function useFanEnergy() {
  const state = loadState();
  const energy = ref(state.energy);
  const lastTs = ref(state.lastTs);
  const boostReadyAt = ref(state.boostReadyAt);
  const boostActiveUntil = ref(state.boostActiveUntil);
  const nowTs = ref(Date.now());
  const claimPulse = ref(false);
  let timerId = 0;

  const isBoostActive = computed(() => nowTs.value < boostActiveUntil.value);
  const isBoostReady = computed(
    () => !isBoostActive.value && nowTs.value >= boostReadyAt.value,
  );

  const boostCooldownRemainingMs = computed(() => {
    if (isBoostReady.value) return 0;
    if (isBoostActive.value) {
      return Math.max(0, boostActiveUntil.value - nowTs.value);
    }
    return Math.max(0, boostReadyAt.value - nowTs.value);
  });

  const boostProgressPct = computed(() => {
    if (isBoostActive.value) {
      const elapsed = BOOST_DURATION_MS - boostCooldownRemainingMs.value;
      return Math.min(100, Math.max(0, (elapsed / BOOST_DURATION_MS) * 100));
    }
    const remaining = boostCooldownRemainingMs.value;
    const done = BOOST_COOLDOWN_MS - remaining;
    return Math.min(100, Math.max(0, (done / BOOST_COOLDOWN_MS) * 100));
  });

  const boostLabel = computed(() => {
    if (isBoostActive.value) {
      return `Attivo: ${formatSeconds(boostCooldownRemainingMs.value / 1000)}`;
    }
    if (isBoostReady.value) {
      return "Pronto";
    }
    return `Pronto tra ${formatSeconds(boostCooldownRemainingMs.value / 1000)}`;
  });

  function persist() {
    persistState({
      energy: energy.value,
      lastTs: lastTs.value,
      boostReadyAt: boostReadyAt.value,
      boostActiveUntil: boostActiveUntil.value,
    });
  }

  function computeGain(start, end) {
    if (end <= start) return 0;
    let gained = 0;
    let cursor = start;

    if (boostActiveUntil.value > cursor) {
      const boostWindowEnd = Math.min(boostActiveUntil.value, end);
      const boostElapsed = boostWindowEnd - cursor;
      gained += Math.floor(boostElapsed / BOOST_TICK_MS) * ENERGY_PER_TICK;
      cursor = boostWindowEnd;
    }

    if (end > cursor) {
      const baseElapsed = end - cursor;
      gained += Math.floor(baseElapsed / BASE_TICK_MS) * ENERGY_PER_TICK;
    }

    return gained;
  }

  function applyAccrual(now) {
    nowTs.value = now;
    if (energy.value >= ENERGY_CAP) {
      lastTs.value = now;
      persist();
      return;
    }

    const gained = computeGain(lastTs.value, now);
    if (gained <= 0) {
      lastTs.value = now;
      persist();
      return;
    }

    energy.value = Math.min(ENERGY_CAP, energy.value + gained);
    lastTs.value = now;
    persist();
  }

  function startLoop() {
    timerId = window.setInterval(() => {
      applyAccrual(Date.now());
    }, 1000);
  }

  function stopLoop() {
    if (timerId) {
      clearInterval(timerId);
      timerId = 0;
    }
  }

  function triggerBoost() {
    if (!isBoostReady.value) {
      return false;
    }
    const now = Date.now();
    applyAccrual(now);
    boostActiveUntil.value = now + BOOST_DURATION_MS;
    boostReadyAt.value = now + BOOST_COOLDOWN_MS;
    lastTs.value = now;
    persist();
    return true;
  }

  function claim() {
    if (energy.value < 100) {
      return false;
    }
    energy.value = Math.max(0, energy.value - 100);
    claimPulse.value = true;
    setTimeout(() => {
      claimPulse.value = false;
    }, 500);
    persist();
    return true;
  }

  onMounted(() => {
    applyAccrual(Date.now());
    startLoop();
  });

  onBeforeUnmount(() => {
    stopLoop();
  });

  return {
    energy,
    isBoostActive,
    isBoostReady,
    boostLabel,
    boostProgressPct,
    boostCooldownRemainingMs,
    triggerBoost,
    claim,
    claimPulse,
  };
}

export default useFanEnergy;
