<template>
  <div class="sponsor-rush flex h-full min-h-0 w-full flex-col">
    <main class="relative flex-1 overflow-hidden rounded-3xl border border-white/10 bg-slate-950/70">
      <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top,rgba(34,211,238,0.18),transparent_58%)]"></div>

      <section v-if="phase === 'ready'" class="overlay-panel">
        <p class="eyebrow">SPONSOR RUSH</p>
        <h3 class="title">Raccogli i loghi al volo</h3>
        <p class="subtitle">Muoviti a sinistra/destra, prendi almeno {{ targetLogos }} loghi in {{ roundDurationSeconds }} secondi.</p>
      </section>

      <section v-else-if="phase === 'countdown'" class="overlay-panel">
        <p class="eyebrow">PREPARATI</p>
        <div class="countdown">{{ countdown }}</div>
      </section>

      <section v-else-if="phase === 'result'" class="overlay-panel">
        <p class="eyebrow">RISULTATO</p>
        <h3 class="title">{{ won ? 'Missione completata! 🎉' : 'Quasi! Riprova subito 💪' }}</h3>
        <p class="subtitle">
          Loghi raccolti: <strong>{{ score }}</strong> / {{ targetLogos }}
        </p>
        <p class="subtitle">Ricompensa: <strong>+{{ rewardCoins }} coin</strong></p>
      </section>

      <div class="absolute left-3 right-3 top-3 z-20 flex items-center justify-between gap-2 rounded-2xl border border-white/15 bg-slate-900/80 px-3 py-2 text-xs text-white">
        <span>Tempo: <strong>{{ timeLeftLabel }}</strong></span>
        <span>Score: <strong>{{ score }}</strong></span>
        <span>Target: <strong>{{ targetLogos }}</strong></span>
      </div>

      <div class="absolute inset-x-0 bottom-0 top-0" ref="arenaRef">
        <div
          v-for="item in items"
          :key="item.id"
          class="falling-item"
          :class="item.type === 'obstacle' ? 'falling-item--obstacle' : 'falling-item--logo'"
          :style="{ transform: `translate3d(${item.x}px, ${item.y}px, 0)` }"
        >
          <template v-if="item.type === 'logo'">
            <img v-if="sponsorLogo" :src="sponsorLogo" alt="Logo sponsor" class="logo-image" />
            <span v-else class="logo-fallback">🏷️</span>
          </template>
          <span v-else class="obstacle">⛔</span>
        </div>

        <div class="player" :style="{ transform: `translate3d(${playerX}px, 0, 0)` }">
          🏐
        </div>
      </div>

      <transition name="pop-fade">
        <div v-if="feedback" class="feedback" :class="feedback.type === 'good' ? 'feedback--good' : 'feedback--bad'">
          {{ feedback.text }}
        </div>
      </transition>
    </main>

    <div class="mt-3 grid grid-cols-2 gap-2" aria-label="Controlli movimento sponsor rush">
      <button type="button" class="ctrl-btn" :disabled="phase !== 'playing'" @touchstart.prevent="moveBy(-40)" @click="moveBy(-40)">⬅️ Sinistra</button>
      <button type="button" class="ctrl-btn" :disabled="phase !== 'playing'" @touchstart.prevent="moveBy(40)" @click="moveBy(40)">Destra ➡️</button>
    </div>

    <footer class="mt-2 grid grid-cols-1 gap-2">
      <button type="button" class="cta-primary" :disabled="phase === 'countdown' || isClaiming" @click="onPrimaryAction">
        {{ primaryLabel }}
      </button>
      <button type="button" class="cta-secondary" :disabled="isClaiming" @click="emit('exit')">Esci</button>
    </footer>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { apiClient } from '../../api';

const emit = defineEmits(['claim', 'exit']);

const arenaRef = ref(null);
const sponsorLogo = ref('');
const phase = ref('ready');
const countdown = ref(3);
const score = ref(0);
const timeLeftMs = ref(0);
const feedback = ref(null);
const isClaiming = ref(false);

const items = ref([]);
const playerX = ref(130);
let arenaWidth = 320;
let arenaHeight = 420;
let gameRaf = 0;
let loopStart = 0;
let lastTick = 0;
let spawnAccumulator = 0;
let countdownTimer = 0;

const roundDurationSeconds = 8;
const targetLogos = 5;
const roundDurationMs = roundDurationSeconds * 1000;
const playerSize = 52;
const itemSize = 40;

const won = computed(() => score.value >= targetLogos);
const rewardCoins = computed(() => (won.value ? 12 : Math.min(3, score.value)));
const timeLeftLabel = computed(() => `${(Math.max(0, timeLeftMs.value) / 1000).toFixed(1)}s`);
const primaryLabel = computed(() => {
  if (phase.value === 'ready') return 'Avvia Sponsor Rush';
  if (phase.value === 'countdown') return 'Preparazione…';
  if (phase.value === 'playing') return 'Partita in corso…';
  if (phase.value === 'result') return isClaiming.value ? 'Accredito…' : `Riscatta +${rewardCoins.value} coin`;
  return 'Avvia Sponsor Rush';
});

onMounted(() => {
  loadSponsorLogo();
  syncArenaSize();
  if (typeof window !== 'undefined') {
    window.addEventListener('resize', syncArenaSize);
    window.addEventListener('keydown', onKeydown);
  }
});

onBeforeUnmount(() => {
  stopGameLoop();
  clearCountdownTimer();
  if (typeof window !== 'undefined') {
    window.removeEventListener('resize', syncArenaSize);
    window.removeEventListener('keydown', onKeydown);
  }
});

async function loadSponsorLogo() {
  try {
    const { data } = await apiClient.get('/sponsors');
    const first = Array.isArray(data) ? data.find((item) => typeof item?.logo_data === 'string' && item.logo_data.trim()) : null;
    sponsorLogo.value = first?.logo_data || '';
  } catch (error) {
    sponsorLogo.value = '';
  }
}

function syncArenaSize() {
  const el = arenaRef.value;
  if (!el) return;
  const rect = el.getBoundingClientRect();
  arenaWidth = Math.max(280, Math.floor(rect.width));
  arenaHeight = Math.max(320, Math.floor(rect.height));
  playerX.value = clamp(playerX.value, 0, arenaWidth - playerSize);
}

function onKeydown(event) {
  if (phase.value !== 'playing') return;
  if (event.key === 'ArrowLeft') moveBy(-24);
  if (event.key === 'ArrowRight') moveBy(24);
}

function moveBy(delta) {
  playerX.value = clamp(playerX.value + delta, 0, arenaWidth - playerSize);
}

function onPrimaryAction() {
  if (phase.value === 'ready') {
    startCountdown();
    return;
  }
  if (phase.value === 'result') {
    claimReward();
  }
}

function startCountdown() {
  resetRound();
  phase.value = 'countdown';
  countdown.value = 3;
  clearCountdownTimer();
  countdownTimer = window.setInterval(() => {
    countdown.value -= 1;
    if (countdown.value <= 0) {
      clearCountdownTimer();
      startGameplay();
    }
  }, 900);
}

function clearCountdownTimer() {
  if (countdownTimer) {
    window.clearInterval(countdownTimer);
    countdownTimer = 0;
  }
}

function resetRound() {
  score.value = 0;
  items.value = [];
  feedback.value = null;
  spawnAccumulator = 0;
  timeLeftMs.value = roundDurationMs;
}

function startGameplay() {
  syncArenaSize();
  phase.value = 'playing';
  loopStart = performance.now();
  lastTick = loopStart;
  stopGameLoop();
  gameRaf = requestAnimationFrame(gameLoop);
}

function stopGameLoop() {
  if (gameRaf) {
    cancelAnimationFrame(gameRaf);
    gameRaf = 0;
  }
}

function gameLoop(ts) {
  const dt = Math.min(40, ts - lastTick);
  lastTick = ts;
  const elapsed = ts - loopStart;
  timeLeftMs.value = Math.max(0, roundDurationMs - elapsed);

  spawnAccumulator += dt;
  if (spawnAccumulator > 320) {
    spawnAccumulator = 0;
    spawnItem();
  }

  updateItems(dt);

  if (elapsed >= roundDurationMs) {
    finishRound();
    return;
  }

  gameRaf = requestAnimationFrame(gameLoop);
}

function spawnItem() {
  const isObstacle = Math.random() < 0.24;
  items.value.push({
    id: `${Date.now()}-${Math.random()}`,
    type: isObstacle ? 'obstacle' : 'logo',
    x: Math.random() * Math.max(1, arenaWidth - itemSize),
    y: -itemSize,
    vy: isObstacle ? 220 + Math.random() * 110 : 180 + Math.random() * 90,
  });
}

function updateItems(dt) {
  const playerY = arenaHeight - 72;
  const next = [];
  for (const item of items.value) {
    const y = item.y + (item.vy * dt) / 1000;
    if (y > arenaHeight + itemSize) continue;

    const collides = intersects(item.x, y, itemSize, itemSize, playerX.value, playerY, playerSize, playerSize);
    if (collides) {
      if (item.type === 'logo') {
        score.value += 1;
        showFeedback('+1 logo', 'good');
      } else {
        score.value = Math.max(0, score.value - 1);
        showFeedback('-1 ostacolo', 'bad');
      }
      continue;
    }

    next.push({ ...item, y });
  }
  items.value = next;
}

function finishRound() {
  stopGameLoop();
  phase.value = 'result';
}

async function claimReward() {
  if (isClaiming.value) return;
  isClaiming.value = true;
  try {
    emit('claim', {
      coins: rewardCoins.value,
      game: 'sponsor-rush',
      result: won.value ? 'win' : 'lose',
      score: score.value,
      target: targetLogos,
    });
  } finally {
    isClaiming.value = false;
  }
}

function showFeedback(text, type) {
  feedback.value = { text, type };
  window.setTimeout(() => {
    if (feedback.value?.text === text) {
      feedback.value = null;
    }
  }, 360);
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}

function intersects(ax, ay, aw, ah, bx, by, bw, bh) {
  return ax < bx + bw && ax + aw > bx && ay < by + bh && ay + ah > by;
}
</script>

<style scoped>
.overlay-panel { position: absolute; inset: 0; z-index: 30; display: grid; place-content: center; text-align: center; padding: 1.2rem; background: rgba(2, 6, 23, 0.72); }
.eyebrow { font-size: 11px; letter-spacing: .22em; text-transform: uppercase; color: #7dd3fc; font-weight: 700; }
.title { margin-top: .5rem; font-size: 1.6rem; line-height: 1.2; font-weight: 900; color: white; }
.subtitle { margin-top: .7rem; font-size: .92rem; color: #cbd5e1; max-width: 320px; }
.countdown { margin-top: .8rem; font-size: 4rem; font-weight: 900; color: #fff; }
.player { position: absolute; bottom: 12px; width: 52px; height: 52px; border-radius: 50%; display: grid; place-content: center; font-size: 1.7rem; background: linear-gradient(160deg,#f97316,#facc15); box-shadow: 0 12px 24px rgba(15,23,42,.45); }
.falling-item { position: absolute; width: 40px; height: 40px; border-radius: 12px; display: grid; place-content: center; }
.falling-item--logo { background: rgba(255,255,255,.95); border: 1px solid rgba(2,6,23,.2); overflow: hidden; }
.logo-image { width: 100%; height: 100%; object-fit: contain; }
.logo-fallback { font-size: 1.2rem; }
.falling-item--obstacle { background: rgba(127,29,29,.85); color: #fff; }
.obstacle { font-size: 1.2rem; }
.feedback { position: absolute; left: 50%; top: 76px; transform: translateX(-50%); z-index: 35; border-radius: 999px; padding: .3rem .7rem; font-weight: 700; font-size: .78rem; }
.feedback--good { background: rgba(16,185,129,.9); color: white; }
.feedback--bad { background: rgba(239,68,68,.9); color: white; }
.ctrl-btn { border-radius: 14px; border: 1px solid rgba(255,255,255,.2); background: rgba(15,23,42,.8); color: white; padding: .75rem .6rem; font-weight: 700; }
.cta-primary { border-radius: 14px; background: linear-gradient(135deg,#22c55e,#16a34a); color: white; padding: .8rem .9rem; font-weight: 800; }
.cta-secondary { border-radius: 14px; border: 1px solid rgba(255,255,255,.2); background: rgba(15,23,42,.75); color: #e2e8f0; padding: .7rem .9rem; font-weight: 700; }
.pop-fade-enter-active,.pop-fade-leave-active { transition: all .18s ease; }
.pop-fade-enter-from,.pop-fade-leave-to { opacity: 0; transform: translate(-50%, -10px); }
</style>
