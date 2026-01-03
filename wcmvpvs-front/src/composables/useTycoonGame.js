import { computed, onMounted, onUnmounted, ref, watch } from "vue";

// Configurazioni di base (MVP)
const BASE_POINTS_PER_TICK = 1; // punti generati ad ogni tick
const BASE_TICK_MS = 2000; // durata base del tick (ms)
const OFFLINE_CAP_MS = 2 * 60 * 60 * 1000; // progresso offline massimo (2 ore)
const COST_GROWTH_FACTOR = 1.6; // moltiplicatore di crescita dei costi
const CLICK_COOLDOWN_MS = 300; // cooldown manuale per il tamburello

const STORAGE_KEY = "tycoon_game_state_v1";

const UPGRADE_BLUEPRINTS = [
  {
    id: "tamburello",
    name: "Tamburello Pro",
    icon: "🥁",
    description: "+1 punto per tick",
    baseCost: 8,
    bonusType: "flat",
    bonusValue: 1,
  },
  {
    id: "megafono",
    name: "Megafono Curva",
    icon: "📣",
    description: "+3 punti per tick",
    baseCost: 18,
    bonusType: "flat",
    bonusValue: 3,
  },
  {
    id: "coro",
    name: "Coro Coordinato",
    icon: "🎶",
    description: "+12% produzione",
    baseCost: 36,
    bonusType: "multiplier",
    bonusValue: 0.12,
  },
  {
    id: "banda",
    name: "Banda Ritmo",
    icon: "🥁🥁",
    description: "Tick più rapido (-5%)",
    baseCost: 55,
    bonusType: "speed",
    bonusValue: 0.05,
  },
];

function getStorage() {
  if (typeof window === "undefined" || !window.localStorage) {
    return null;
  }
  return window.localStorage;
}

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

export function useTycoonGame() {
  const points = ref(0);
  const lastTickAt = ref(Date.now());
  const upgrades = ref(
    UPGRADE_BLUEPRINTS.map((upgrade) => ({ ...upgrade, level: 0 })),
  );
  const isClickCoolingDown = ref(false);
  const drumPulse = ref(false);
  const tickPulse = ref(false);
  const manualClicks = ref(0);

  let tickTimer = null;
  let clickCooldownTimer = null;

  const effectiveTickMs = computed(() => {
    const speedUpgrade = upgrades.value.find((item) => item.id === "banda");
    const reduction = speedUpgrade ? speedUpgrade.level * (speedUpgrade.bonusValue || 0) : 0;
    const cappedReduction = Math.min(reduction, 0.5); // minimo 50% del tempo base
    const tickMs = BASE_TICK_MS * (1 - cappedReduction);
    return Math.max(600, tickMs);
  });

  const baseFlatBonus = computed(() => {
    return upgrades.value
      .filter((item) => item.bonusType === "flat")
      .reduce((total, item) => total + item.level * (item.bonusValue || 0), 0);
  });

  const multiplierBonus = computed(() => {
    return upgrades.value
      .filter((item) => item.bonusType === "multiplier")
      .reduce((total, item) => total + item.level * (item.bonusValue || 0), 0);
  });

  const pointsPerTick = computed(() => {
    const flat = BASE_POINTS_PER_TICK + baseFlatBonus.value;
    const multiplier = 1 + multiplierBonus.value;
    const value = flat * multiplier;
    return Math.max(1, Math.round(value));
  });

  const pointsPerSecond = computed(() => {
    return Number(((pointsPerTick.value * 1000) / effectiveTickMs.value).toFixed(2));
  });

  const upgradeViews = computed(() => {
    return upgrades.value.map((upgrade) => {
      const nextCost = Math.round(upgrade.baseCost * Math.pow(COST_GROWTH_FACTOR, upgrade.level));
      const canAfford = points.value >= nextCost;
      return {
        ...upgrade,
        nextCost,
        canAfford,
      };
    });
  });

  const quickUpgrade = computed(() => {
    const sorted = [...upgradeViews.value].sort((a, b) => a.nextCost - b.nextCost);
    return sorted.find((item) => item.canAfford) || sorted[0];
  });

  const formattedPoints = computed(() => points.value.toLocaleString("it-IT"));

  function persistState() {
    const storage = getStorage();
    if (!storage) {
      return;
    }
    try {
      storage.setItem(
        STORAGE_KEY,
        JSON.stringify({
          points: points.value,
          upgrades: upgrades.value.map(({ id, level }) => ({ id, level })),
          lastTickAt: lastTickAt.value,
        }),
      );
    } catch (error) {
      console.warn("persist tycoon failed", error);
    }
  }

  function hydrateState() {
    const storage = getStorage();
    if (!storage) {
      return;
    }
    try {
      const raw = storage.getItem(STORAGE_KEY);
      if (!raw) {
        return;
      }
      const parsed = JSON.parse(raw);
      if (Number.isFinite(parsed?.points)) {
        points.value = Math.max(0, Math.floor(parsed.points));
      }
      if (Array.isArray(parsed?.upgrades)) {
        upgrades.value = UPGRADE_BLUEPRINTS.map((blueprint) => {
          const savedLevel = parsed.upgrades.find((item) => item.id === blueprint.id)?.level ?? 0;
          return {
            ...blueprint,
            level: Math.max(0, Number(savedLevel) || 0),
          };
        });
      }
      if (Number.isFinite(parsed?.lastTickAt)) {
        lastTickAt.value = parsed.lastTickAt;
      }
    } catch (error) {
      console.warn("hydrate tycoon failed", error);
    }
  }

  function applyOfflineProgress() {
    const now = Date.now();
    const elapsed = Math.max(0, now - lastTickAt.value);
    const cappedElapsed = Math.min(elapsed, OFFLINE_CAP_MS);
    const ticks = Math.floor(cappedElapsed / effectiveTickMs.value);
    if (ticks > 0) {
      const gained = ticks * pointsPerTick.value;
      points.value += gained;
      tickPulse.value = true;
      setTimeout(() => (tickPulse.value = false), 280);
    }
    lastTickAt.value = now;
  }

  function runTick() {
    points.value += pointsPerTick.value;
    lastTickAt.value = Date.now();
    tickPulse.value = true;
    setTimeout(() => (tickPulse.value = false), 200);
  }

  function clearTimers() {
    if (tickTimer) {
      clearInterval(tickTimer);
      tickTimer = null;
    }
    if (clickCooldownTimer) {
      clearTimeout(clickCooldownTimer);
      clickCooldownTimer = null;
    }
  }

  function scheduleTick() {
    clearTimers();
    tickTimer = setInterval(runTick, effectiveTickMs.value);
  }

  function handleManualClick() {
    if (isClickCoolingDown.value) {
      return;
    }
    points.value += 1;
    manualClicks.value += 1;
    drumPulse.value = true;
    setTimeout(() => (drumPulse.value = false), 180);

    if (manualClicks.value % 10 === 0) {
      trackEvent("tycoon_manual_clicks", { count: manualClicks.value });
    }

    isClickCoolingDown.value = true;
    clickCooldownTimer = setTimeout(() => {
      isClickCoolingDown.value = false;
    }, CLICK_COOLDOWN_MS);
  }

  function buyUpgrade(upgradeId) {
    const upgrade = upgradeViews.value.find((item) => item.id === upgradeId);
    if (!upgrade) {
      return false;
    }
    if (points.value < upgrade.nextCost) {
      return false;
    }
    points.value -= upgrade.nextCost;
    const target = upgrades.value.find((item) => item.id === upgradeId);
    if (target) {
      target.level += 1;
      trackEvent("tycoon_upgrade_purchase", {
        upgradeId,
        level: target.level,
        remainingPoints: points.value,
      });
    }
    tickPulse.value = true;
    setTimeout(() => (tickPulse.value = false), 200);
    return true;
  }

  onMounted(() => {
    hydrateState();
    applyOfflineProgress();
    scheduleTick();
  });

  onUnmounted(() => {
    clearTimers();
  });

  watch(points, persistState);
  watch(
    () => upgrades.value.map((item) => item.level),
    () => {
      persistState();
      scheduleTick();
    },
  );

  watch(effectiveTickMs, scheduleTick);

  return {
    BASE_TICK_MS,
    OFFLINE_CAP_MS,
    COST_GROWTH_FACTOR,
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
  };
}

export default useTycoonGame;
