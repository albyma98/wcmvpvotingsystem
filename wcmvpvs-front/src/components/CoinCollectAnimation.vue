<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="coin-overlay"
      aria-hidden="true"
    >
      <div
        v-for="coin in coins"
        :key="coin.id"
        class="coin"
        :class="coin.phase"
        :style="coin.style"
      >
        🪙
      </div>
      <div
        v-if="amountLabel"
        class="amount-label"
        :style="amountStyle"
      >
        +{{ amountLabel }} 🪙
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref } from 'vue';

const visible = ref(false);
const coins = ref([]);
const amountLabel = ref(0);
const amountStyle = ref({});

let coinId = 0;

function wait(ms) {
  return new Promise((resolve) => {
    window.setTimeout(resolve, ms);
  });
}

function toCenterRect(el) {
  const rect = el.getBoundingClientRect();
  return {
    x: rect.left + (rect.width / 2),
    y: rect.top + (rect.height / 2),
  };
}

async function play({ fromEl, toEl, count = 18, amount = 0 }) {
  if (typeof window === 'undefined' || !fromEl || !toEl) {
    return;
  }

  const from = toCenterRect(fromEl);
  const to = toCenterRect(toEl);
  const coinsCount = Math.max(14, Math.min(22, Math.round(count || 18)));

  coins.value = Array.from({ length: coinsCount }, () => {
    const angle = Math.random() * Math.PI * 2;
    const radius = 30 + Math.random() * 64;
    const burstX = Math.cos(angle) * radius;
    const burstY = Math.sin(angle) * radius;
    const driftX = (Math.random() - 0.5) * 26;
    const driftY = (Math.random() - 0.5) * 20;
    const rotate = Math.round((Math.random() - 0.5) * 90);
    return {
      id: coinId += 1,
      phase: 'burst',
      style: {
        left: `${from.x}px`,
        top: `${from.y}px`,
        '--burst-x': `${burstX}px`,
        '--burst-y': `${burstY}px`,
        '--fly-x': `${to.x - from.x + driftX}px`,
        '--fly-y': `${to.y - from.y + driftY}px`,
        '--rot': `${rotate}deg`,
        '--delay': `${Math.random() * 220}ms`,
        '--dur': `${900 + Math.random() * 320}ms`,
      },
    };
  });

  amountLabel.value = amount;
  amountStyle.value = {
    left: `${to.x}px`,
    top: `${to.y}px`,
  };

  visible.value = true;

  await wait(320);

  coins.value = coins.value.map((coin) => ({
    ...coin,
    phase: 'fly',
  }));

  await wait(1400);

  amountLabel.value = 0;
  coins.value = [];
  visible.value = false;
}

defineExpose({ play });
</script>

<style scoped>
.coin-overlay {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 9999;
}

.coin {
  position: fixed;
  transform: translate3d(-50%, -50%, 0);
  font-size: 1.35rem;
  opacity: 0;
  will-change: transform, opacity;
}

.coin.burst {
  animation: coin-burst 320ms ease-out forwards;
}

.coin.fly {
  animation: coin-fly var(--dur) cubic-bezier(0.2, 0.72, 0.2, 1) var(--delay) forwards;
}

.amount-label {
  position: fixed;
  transform: translate3d(-50%, -50%, 0);
  font-weight: 800;
  color: #fde047;
  text-shadow: 0 2px 8px rgba(2, 6, 23, 0.42);
  animation: reward-float 980ms ease-out forwards;
}

@keyframes coin-burst {
  0% {
    opacity: 0;
    transform: translate3d(-50%, -50%, 0) scale(0.4);
  }
  100% {
    opacity: 1;
    transform: translate3d(calc(-50% + var(--burst-x)), calc(-50% + var(--burst-y)), 0) scale(1);
  }
}

@keyframes coin-fly {
  0% {
    opacity: 1;
    transform: translate3d(calc(-50% + var(--burst-x)), calc(-50% + var(--burst-y)), 0) scale(1) rotate(0deg);
  }
  100% {
    opacity: 0;
    transform: translate3d(calc(-50% + var(--fly-x)), calc(-50% + var(--fly-y)), 0) scale(0.52) rotate(var(--rot));
  }
}

@keyframes reward-float {
  0% {
    opacity: 0;
    transform: translate3d(-50%, -35%, 0) scale(0.8);
  }
  20% {
    opacity: 1;
    transform: translate3d(-50%, -55%, 0) scale(1);
  }
  100% {
    opacity: 0;
    transform: translate3d(-50%, -120%, 0) scale(1.02);
  }
}
</style>
