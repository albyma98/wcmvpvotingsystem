<template>
  <div
    class="si-root"
    ref="rootEl"
    tabindex="-1"
    :style="rootStyle"
    @keydown.space.prevent="onStack"
    @keydown.enter.prevent="onStack"
  >
    <!-- HUD -->
    <div class="si-hud" aria-live="polite" aria-atomic="true">
      <div class="si-hud__block">
        <span class="si-hud__val">{{ score }}</span>
        <span class="si-hud__lbl">score</span>
      </div>
      <div class="si-hud__sep" aria-hidden="true" />
      <div class="si-hud__block">
        <span class="si-hud__val">{{ bestScore }}</span>
        <span class="si-hud__lbl">best</span>
      </div>
    </div>

    <!-- Arena -->
    <div class="si-arena" ref="arenaEl">
      <!-- Active (oscillating) block -->
      <div
        v-if="phase === 'playing'"
        class="si-block si-block--active"
        :class="{ 'si-block--flash': perfectFlash }"
        :style="activeBlockStyle"
        aria-hidden="true"
      />

      <!-- World: translateY drives camera scroll -->
      <div
        class="si-world"
        ref="worldEl"
        :style="{ transform: `translateY(${cameraY}px)` }"
        aria-hidden="true"
      >
        <!-- Base platform -->
        <div class="si-base" :style="baseStyle" />
        <!-- Placed tower blocks -->
        <div
          v-for="blk in visibleTower"
          :key="blk.id"
          class="si-block"
          :class="{ 'si-block--flash': blk.flash }"
          :style="placedBlockStyle(blk)"
        />
      </div>

      <!-- Debris: CSS-animated cut pieces (in arena coords, not world) -->
      <div
        v-for="d in debris"
        :key="d.id"
        class="si-block si-block--debris"
        :style="debrisStyle(d)"
        aria-hidden="true"
      />
    </div>

    <!-- First-play hint -->
    <div v-if="phase === 'playing' && score === 0 && !stacking" class="si-hint" aria-hidden="true">
      Tocca STACK! per fermare il blocco
    </div>

    <!-- STACK! button -->
    <div class="si-btn-area">
      <button
        v-if="phase === 'playing'"
        class="si-btn"
        :style="btnStyle"
        :disabled="stacking"
        ref="stackBtnEl"
        @click="onStack"
      >
        STACK!
      </button>
    </div>

    <!-- Game Over overlay -->
    <Transition name="si-fade">
      <div
        v-if="phase === 'gameover'"
        class="si-gameover"
        role="alertdialog"
        aria-labelledby="si-go-title"
      >
        <div class="si-go-card" ref="gameoverEl">
          <p id="si-go-title" class="si-go__title">GAME OVER</p>
          <p class="si-go__score">{{ score }}</p>
          <p class="si-go__score-lbl">blocchi</p>

          <div class="si-go__stats">
            <span>Best: {{ bestScore }}</span>
            <span>Perfect: {{ perfectCount }}</span>
          </div>

          <!-- Leaderboard -->
          <div v-if="leaderboard.length" class="si-lb">
            <p class="si-lb__title">Classifica</p>
            <ol class="si-lb__list" aria-label="Top 10 punteggi">
              <li
                v-for="(e, i) in leaderboard"
                :key="i"
                class="si-lb__item"
                :class="{ 'si-lb__item--me': e.isMe }"
              >
                <span class="si-lb__rank">#{{ i + 1 }}</span>
                <span class="si-lb__score">{{ e.score }}</span>
              </li>
            </ol>
          </div>

          <div class="si-go__coins">🪙 +{{ earnedCoins }} coin</div>

          <a
            v-if="cfg.cta_url"
            :href="cfg.cta_url"
            target="_blank"
            rel="noopener noreferrer"
            class="si-go__cta"
            @click="trackCta"
          >
            {{ cfg.cta_label || 'Scopri di più' }}
          </a>

          <button class="si-go__claim" :style="btnStyle" @click="doClaim">
            Ritira i premi
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, nextTick } from 'vue';
import gsap from 'gsap';
import { apiClient } from '../../api';
import { trackAppEvent } from '../../eventTracking';

// ── Props / emits ─────────────────────────────────────────────────────────────
const props = defineProps({
  config: { type: Object, required: true },
  eventId: { type: Number, default: 0 },
  walletCoins: { type: Number, default: 0 },
});

const emit = defineEmits(['claim', 'exit']);

// ── Game config shorthand ──────────────────────────────────────────────────────
const cfg = computed(() => props.config?.stack_it_config ?? {});
const primaryColor = computed(() => props.config?.primary_color ?? '#3b82f6');
const secondaryColor = computed(() => props.config?.secondary_color ?? '#ffffff');

// ── Constants ─────────────────────────────────────────────────────────────────
const BLOCK_H = 44;           // px, height of every tower block
const BASE_H = 28;            // px, height of base platform
const BASE_W = 260;           // px, base platform width (= initial block width)
const INITIAL_W = 260;        // px, initial block width
const MIN_W_RATIO = 0.10;     // game over threshold
const PERFECT_THRESHOLD = 2;  // px, ±2px edge tolerance
const ACTIVE_TOP = 36;        // px, active block Y in arena
const LANDING_Y = ACTIVE_TOP + BLOCK_H + 4; // = 84px, where tower top appears
const STACK_DEBOUNCE = 300;   // ms

// Speed multipliers per 5 blocks
const SPEED_MULTIPLIERS = { gentle: 0.97, standard: 0.95, aggressive: 0.90 };

// Default colors used if none configured
const DEFAULT_COLORS = ['#3b82f6', '#8b5cf6', '#f59e0b', '#10b981'];

// ── Element refs ──────────────────────────────────────────────────────────────
const rootEl = ref(null);
const arenaEl = ref(null);
const stackBtnEl = ref(null);
const gameoverEl = ref(null);

// ── Game state ────────────────────────────────────────────────────────────────
const phase = ref('playing');  // 'playing' | 'gameover'
const score = ref(0);
const bestScore = ref(Number(localStorage.getItem('stackit_best') ?? 0) || 0);
const perfectCount = ref(0);
const tower = ref([]);         // array of { id, x, width, color, stackIndex, flash }
const debris = ref([]);        // array of { id, x, width, color, debrisDir }
const leaderboard = ref([]);

// Active block state (driven by GSAP)
const activeX = ref(0);
const activeW = ref(INITIAL_W);
const cameraY = ref(0);
const perfectFlash = ref(false);
const stacking = ref(false);

// Internal game tracking
let colorIndex = 0;
let blockIdCounter = 0;
let debrisIdCounter = 0;
let lastStackTime = 0;
let gameStartTime = 0;
let pendulumTween = null;
const pendulumProxy = { x: 0 };
const cameraProxy = { y: 0 };
const prefersReducedMotion = () =>
  typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;

// ── Derived helpers ───────────────────────────────────────────────────────────
function arenaWidth() {
  return arenaEl.value ? arenaEl.value.clientWidth : 320;
}

function blockColors() {
  const c = cfg.value.block_colors;
  return Array.isArray(c) && c.length >= 2 ? c : DEFAULT_COLORS;
}

function nextColor() {
  const colors = blockColors();
  colorIndex = (colorIndex + 1) % colors.length;
  return colors[colorIndex];
}

function activeColor() {
  const colors = blockColors();
  return colors[colorIndex % colors.length];
}

function getSpeedMs(blocksPlaced) {
  const initial = cfg.value.initial_pendulum_speed_ms ?? 1500;
  const key = cfg.value.speed_curve ?? 'standard';
  const mult = SPEED_MULTIPLIERS[key] ?? SPEED_MULTIPLIERS.standard;
  const speedups = Math.floor(blocksPlaced / 5);
  return Math.max(400, initial * Math.pow(mult, speedups));
}

const visibleTower = computed(() => {
  const t = tower.value;
  return t.length > 30 ? t.slice(t.length - 30) : t;
});

const earnedCoins = computed(() => {
  const perBlock = cfg.value.reward_per_block ?? 0.5;
  const perfBonus = cfg.value.perfect_bonus_coins ?? 2;
  return Math.floor(score.value * perBlock + perfectCount.value * perfBonus);
});

// Reference block (base or top of tower)
function refBlock() {
  if (tower.value.length === 0) {
    const aW = arenaWidth();
    const baseX = (aW - BASE_W) / 2;
    return { x: baseX, width: BASE_W };
  }
  const top = tower.value[tower.value.length - 1];
  return { x: top.x, width: top.width };
}

// ── Styles ────────────────────────────────────────────────────────────────────
const rootStyle = computed(() => ({
  '--si-primary': primaryColor.value,
  '--si-secondary': secondaryColor.value,
}));

const baseStyle = computed(() => {
  const aW = arenaWidth();
  const baseX = (aW - BASE_W) / 2;
  return {
    position: 'absolute',
    top: `${LANDING_Y}px`,
    left: `${baseX}px`,
    width: `${BASE_W}px`,
    height: `${BASE_H}px`,
  };
});

const activeBlockStyle = computed(() => {
  const texUrl = cfg.value.block_texture_url;
  return {
    position: 'absolute',
    top: `${ACTIVE_TOP}px`,
    left: `${Math.round(activeX.value)}px`,
    width: `${activeW.value}px`,
    height: `${BLOCK_H}px`,
    background: texUrl
      ? `url(${texUrl}) center/cover no-repeat`
      : activeColor(),
    borderRadius: '6px',
  };
});

function placedBlockStyle(blk) {
  const texUrl = cfg.value.block_texture_url;
  return {
    position: 'absolute',
    top: `${LANDING_Y - (blk.stackIndex + 1) * BLOCK_H}px`,
    left: `${Math.round(blk.x)}px`,
    width: `${blk.width}px`,
    height: `${BLOCK_H}px`,
    background: texUrl
      ? `url(${texUrl}) center/cover no-repeat`
      : blk.color,
    borderRadius: '4px',
  };
}

function debrisStyle(d) {
  const texUrl = cfg.value.block_texture_url;
  return {
    position: 'absolute',
    top: `${ACTIVE_TOP}px`,
    left: `${Math.round(d.x)}px`,
    width: `${d.width}px`,
    height: `${BLOCK_H}px`,
    background: texUrl
      ? `url(${texUrl}) center/cover no-repeat`
      : d.color,
    borderRadius: '4px',
    '--si-debris-rot': `${d.debrisDir * (25 + Math.random() * 20)}deg`,
    animation: 'si-debris-fall 0.7s ease-in forwards',
  };
}

const btnStyle = computed(() => ({
  background: primaryColor.value,
  color: secondaryColor.value,
}));

// ── GSAP pendulum ─────────────────────────────────────────────────────────────
function startPendulum(speedMs) {
  const aW = arenaWidth();
  const maxX = Math.max(0, aW - activeW.value);
  const startX = pendulumProxy.x;

  gsap.killTweensOf(pendulumProxy);

  // Determine direction: go to whichever side the block isn't near
  const targetX = startX < maxX / 2 ? maxX : 0;
  const remainingFraction = targetX === maxX
    ? (maxX - startX) / (maxX || 1)
    : startX / (maxX || 1);
  const firstDuration = (speedMs / 1000) * remainingFraction;

  pendulumTween = gsap.to(pendulumProxy, {
    x: targetX,
    duration: firstDuration || speedMs / 1000,
    ease: 'sine.inOut',
    onUpdate() { activeX.value = pendulumProxy.x; },
    onComplete() {
      // Continue full oscillation
      pendulumTween = gsap.to(pendulumProxy, {
        x: pendulumProxy.x > maxX / 2 ? 0 : maxX,
        duration: speedMs / 1000,
        ease: 'sine.inOut',
        yoyo: true,
        repeat: -1,
        onUpdate() { activeX.value = pendulumProxy.x; },
      });
    },
  });
}

function stopPendulum() {
  gsap.killTweensOf(pendulumProxy);
  if (pendulumTween) { pendulumTween.kill(); pendulumTween = null; }
}

// ── Camera scroll ─────────────────────────────────────────────────────────────
function scrollCamera(targetScore) {
  const targetY = targetScore * BLOCK_H;
  const dur = prefersReducedMotion() ? 0 : 0.3;
  gsap.to(cameraProxy, {
    y: targetY,
    duration: dur,
    ease: 'power2.out',
    onUpdate() { cameraY.value = cameraProxy.y; },
  });
}

// ── Perfect stack flash ───────────────────────────────────────────────────────
function flashPerfect() {
  const dur = prefersReducedMotion() ? 0.05 : 0.15;
  perfectFlash.value = true;

  if (worldEl.value) {
    gsap.fromTo(
      worldEl.value,
      { scale: 1 },
      {
        scale: 1.03,
        duration: dur,
        ease: 'power2.out',
        yoyo: true,
        repeat: 1,
        onComplete() { perfectFlash.value = false; },
      },
    );
  } else {
    setTimeout(() => { perfectFlash.value = false; }, dur * 1000 * 2 + 50);
  }
}

const worldEl = ref(null);

// ── Core stack logic ──────────────────────────────────────────────────────────
function onStack() {
  const now = Date.now();
  if (now - lastStackTime < STACK_DEBOUNCE) return;
  if (stacking.value || phase.value !== 'playing') return;
  lastStackTime = now;
  doStack();
}

function doStack() {
  stacking.value = true;
  stopPendulum();

  const ax = pendulumProxy.x;
  const aw = activeW.value;
  const ref = refBlock();

  const overlapLeft = Math.max(ax, ref.x);
  const overlapRight = Math.min(ax + aw, ref.x + ref.width);
  const overlap = overlapRight - overlapLeft;

  // No overlap → game over
  if (overlap <= 0) {
    spawnDebris(ax, aw, -1);
    triggerGameOver();
    return;
  }

  // Too small → game over
  if (aw - overlap >= aw * (1 - MIN_W_RATIO) && overlap < INITIAL_W * MIN_W_RATIO) {
    spawnDebris(ax, aw, ax < ref.x ? 1 : -1);
    triggerGameOver();
    return;
  }

  const isPerfect = overlap >= aw - PERFECT_THRESHOLD * 2;

  let newX = overlapLeft;
  let newWidth = isPerfect ? aw : overlap;

  if (!isPerfect) {
    // Spawn debris for the cut portion
    if (ax < ref.x) {
      // Active block overhangs on the left
      spawnDebris(ax, ref.x - ax, -1);
    } else {
      // Active block overhangs on the right
      spawnDebris(ref.x + ref.width, ax + aw - (ref.x + ref.width), 1);
    }
  }

  // Block too thin after cut
  if (newWidth < INITIAL_W * MIN_W_RATIO) {
    triggerGameOver();
    return;
  }

  const newScore = score.value + 1;
  const color = nextColor();

  tower.value.push({
    id: ++blockIdCounter,
    x: newX,
    width: newWidth,
    color,
    stackIndex: tower.value.length,
    flash: false,
  });

  score.value = newScore;
  activeW.value = isPerfect ? aw : newWidth;
  activeX.value = newX; // centering is handled by camera; align active to new block
  pendulumProxy.x = newX;

  if (isPerfect) {
    perfectCount.value++;
    trackAppEvent(
      'branded_game.stackit.block_placed',
      { score_so_far: newScore, is_perfect: true, event_id: props.eventId },
      'branded_game.stackit',
    );
    flashPerfect();
    if (newScore === 3 || newScore === 5 || newScore === 10) {
      trackAppEvent(
        'branded_game.stackit.perfect_streak_reached',
        { streak: newScore, event_id: props.eventId },
        'branded_game.stackit',
      );
    }
    playSound(cfg.value.perfect_stack_sound_url);
  } else {
    trackAppEvent(
      'branded_game.stackit.block_placed',
      { score_so_far: newScore, is_perfect: false, event_id: props.eventId },
      'branded_game.stackit',
    );
  }

  scrollCamera(newScore);

  const speedMs = getSpeedMs(newScore);
  setTimeout(() => {
    if (phase.value === 'playing') {
      startPendulum(speedMs);
      stacking.value = false;
    }
  }, prefersReducedMotion() ? 50 : 200);
}

function spawnDebris(x, width, dir) {
  if (width <= 0) return;
  const d = {
    id: ++debrisIdCounter,
    x,
    width,
    color: activeColor(),
    debrisDir: dir,
  };
  debris.value.push(d);
  setTimeout(() => {
    const idx = debris.value.findIndex((item) => item.id === d.id);
    if (idx !== -1) debris.value.splice(idx, 1);
  }, 900);
}

function triggerGameOver() {
  phase.value = 'gameover';
  stopPendulum();

  if (score.value > bestScore.value) {
    bestScore.value = score.value;
    localStorage.setItem('stackit_best', String(score.value));
  }

  const durationS = Math.round((Date.now() - gameStartTime) / 1000);
  trackAppEvent(
    'branded_game.stackit.game_over',
    {
      final_score: score.value,
      perfects: perfectCount.value,
      duration_s: durationS,
      event_id: props.eventId,
    },
    'branded_game.stackit',
  );

  playSound(cfg.value.game_over_sound_url);
  fetchLeaderboard();

  nextTick(() => {
    if (gameoverEl.value && !prefersReducedMotion()) {
      gsap.fromTo(
        gameoverEl.value,
        { opacity: 0, y: 20, scale: 0.9 },
        { opacity: 1, y: 0, scale: 1, duration: 0.35, ease: 'back.out(1.5)' },
      );
    }
  });
}

// ── Leaderboard ───────────────────────────────────────────────────────────────
async function fetchLeaderboard() {
  if (!props.eventId) return;
  try {
    const { data } = await apiClient.get(
      `/events/${props.eventId}/branded-game/stackit-leaderboard`,
    );
    leaderboard.value = (data?.entries ?? []).map((e) => ({
      score: e.score,
      isMe: false, // device matching is server-side; flag comes from response
    }));
    trackAppEvent(
      'branded_game.stackit.leaderboard_shown',
      { event_id: props.eventId },
      'branded_game.stackit',
    );
  } catch {
    // silent - leaderboard is optional
  }
}

// ── Audio ─────────────────────────────────────────────────────────────────────
function playSound(url) {
  if (!url) return;
  try {
    const a = new Audio(url);
    a.volume = 0.5;
    a.play().catch(() => {});
  } catch {}
}

// ── Tracking ──────────────────────────────────────────────────────────────────
function trackCta() {
  trackAppEvent(
    'branded_game.stackit.cta_clicked',
    { event_id: props.eventId },
    'branded_game.stackit',
  );
}

// ── Claim ─────────────────────────────────────────────────────────────────────
function doClaim() {
  const durationMs = Date.now() - gameStartTime;
  emit('claim', {
    coins: earnedCoins.value,
    duration_ms: durationMs,
    meta: {
      game_score: score.value,        // raw block count → backend usa questo per ricalcolare
      perfect_stacks: perfectCount.value,
      max_height: score.value,
    },
    keepOpen: false,
  });
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────
onMounted(() => {
  trackAppEvent(
    'branded_game.stackit.started',
    { event_id: props.eventId },
    'branded_game.stackit',
  );

  gameStartTime = Date.now();

  // Pre-load block texture
  if (cfg.value.block_texture_url) {
    const img = new Image();
    img.src = cfg.value.block_texture_url;
  }

  // Initialise pendulum at center of arena
  nextTick(() => {
    const aW = arenaWidth();
    const startX = Math.max(0, (aW - activeW.value) / 2);
    pendulumProxy.x = startX;
    activeX.value = startX;
    startPendulum(cfg.value.initial_pendulum_speed_ms ?? 1500);
    stackBtnEl.value?.focus();
  });
});

onUnmounted(() => {
  stopPendulum();
  gsap.killTweensOf(cameraProxy);
  if (worldEl.value) gsap.killTweensOf(worldEl.value);
});
</script>

<style scoped>
/* ── Root ──────────────────────────────────────────────────────────────────── */
.si-root {
  display: flex;
  flex-direction: column;
  min-height: 400px;
  height: 100%;
  user-select: none;
  touch-action: manipulation;
  outline: none;
  font-family: inherit;
  background: #0a0a1a;
  color: #fff;
  border-radius: 16px;
  overflow: hidden;
  position: relative;
}

/* ── HUD ───────────────────────────────────────────────────────────────────── */
.si-hud {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 24px;
  height: 52px;
  padding: 0 16px;
  background: rgba(0, 0, 0, 0.35);
  flex-shrink: 0;
}

.si-hud__block {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1px;
}

.si-hud__val {
  font-size: 1.4rem;
  font-weight: 900;
  line-height: 1;
  letter-spacing: -0.01em;
}

.si-hud__lbl {
  font-size: 0.6rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  opacity: 0.55;
}

.si-hud__sep {
  width: 1px;
  height: 28px;
  background: rgba(255, 255, 255, 0.15);
}

/* ── Arena ─────────────────────────────────────────────────────────────────── */
.si-arena {
  flex: 1 1 auto;
  position: relative;
  overflow: hidden;
  min-height: 280px;
}

/* ── World (scrolling tower container) ────────────────────────────────────── */
.si-world {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 5000px;
  will-change: transform;
}

/* ── Blocks ─────────────────────────────────────────────────────────────────── */
.si-block {
  position: absolute;
  border-radius: 6px;
  will-change: transform;
  transition: none;
}

.si-block--active {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.5);
  z-index: 2;
}

.si-block--flash {
  animation: si-flash 0.2s ease-out;
}

.si-block--debris {
  z-index: 3;
  pointer-events: none;
  animation: si-debris-fall 0.7s ease-in forwards;
}

/* ── Base platform ──────────────────────────────────────────────────────────── */
.si-base {
  position: absolute;
  border-radius: 8px;
  background: var(--si-primary, #3b82f6);
  opacity: 0.85;
  box-shadow: 0 0 20px rgba(0, 0, 0, 0.6);
}

/* ── Button area ─────────────────────────────────────────────────────────────── */
.si-btn-area {
  flex-shrink: 0;
  padding: 12px 20px 16px;
}

.si-btn {
  width: 100%;
  min-height: 60px;
  border: none;
  border-radius: 16px;
  font-size: 1.25rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  cursor: pointer;
  transition: transform 0.08s, opacity 0.15s;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.35);
}

.si-btn:active:not(:disabled) { transform: scale(0.96); }
.si-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.si-btn:focus-visible { outline: 3px solid #fff; outline-offset: 2px; }

/* ── Hint ────────────────────────────────────────────────────────────────────── */
.si-hint {
  position: absolute;
  bottom: 80px;
  left: 0;
  right: 0;
  text-align: center;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: rgba(255, 255, 255, 0.5);
  animation: si-blink 1.6s ease-in-out infinite;
  pointer-events: none;
}

/* ── Game Over ───────────────────────────────────────────────────────────────── */
.si-gameover {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.82);
  backdrop-filter: blur(4px);
  z-index: 20;
}

.si-go-card {
  width: 100%;
  max-width: 320px;
  background: #111827;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 20px;
  padding: 24px 20px;
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 80vh;
  overflow-y: auto;
}

.si-go__title {
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.2em;
  opacity: 0.6;
  text-transform: uppercase;
  margin: 0;
}

.si-go__score {
  font-size: 3.5rem;
  font-weight: 900;
  line-height: 1;
  margin: 4px 0 0;
}

.si-go__score-lbl {
  font-size: 0.7rem;
  font-weight: 700;
  letter-spacing: 0.15em;
  opacity: 0.5;
  text-transform: uppercase;
  margin: 0 0 4px;
}

.si-go__stats {
  display: flex;
  justify-content: center;
  gap: 20px;
  font-size: 0.8rem;
  font-weight: 700;
  opacity: 0.7;
}

.si-go__coins {
  font-size: 1.1rem;
  font-weight: 900;
  color: #fbbf24;
  padding: 8px;
  background: rgba(251, 191, 36, 0.12);
  border-radius: 10px;
}

.si-go__cta {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 20px;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  color: rgba(255, 255, 255, 0.75);
  font-size: 0.8rem;
  font-weight: 700;
  text-decoration: none;
  transition: background 0.15s;
}
.si-go__cta:hover { background: rgba(255, 255, 255, 0.08); }

.si-go__claim {
  width: 100%;
  padding: 14px;
  border: none;
  border-radius: 14px;
  font-size: 1rem;
  font-weight: 900;
  cursor: pointer;
  transition: transform 0.08s;
}
.si-go__claim:active { transform: scale(0.97); }

/* ── Leaderboard ─────────────────────────────────────────────────────────────── */
.si-lb { text-align: left; }
.si-lb__title {
  font-size: 0.65rem;
  font-weight: 900;
  letter-spacing: 0.15em;
  text-transform: uppercase;
  opacity: 0.5;
  margin-bottom: 6px;
}
.si-lb__list { list-style: none; padding: 0; margin: 0; display: flex; flex-direction: column; gap: 3px; }
.si-lb__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 0.78rem;
  font-weight: 700;
  background: rgba(255, 255, 255, 0.04);
}
.si-lb__item--me { background: rgba(251, 191, 36, 0.15); color: #fbbf24; }
.si-lb__rank { opacity: 0.5; font-size: 0.7rem; }
.si-lb__score { font-weight: 900; }

/* ── Animations ──────────────────────────────────────────────────────────────── */
@keyframes si-debris-fall {
  from { transform: translateY(0) rotate(0deg); opacity: 1; }
  to   { transform: translateY(420px) rotate(var(--si-debris-rot, 30deg)); opacity: 0; }
}

@keyframes si-flash {
  0%   { filter: brightness(1); }
  30%  { filter: brightness(2.5); }
  100% { filter: brightness(1); }
}

@keyframes si-blink {
  0%, 100% { opacity: 0.5; }
  50%       { opacity: 1; }
}

.si-fade-enter-active { transition: opacity 0.25s ease; }
.si-fade-leave-active { transition: opacity 0.15s ease; }
.si-fade-enter-from, .si-fade-leave-to { opacity: 0; }

/* ── Reduced motion ──────────────────────────────────────────────────────────── */
@media (prefers-reduced-motion: reduce) {
  .si-block--debris { animation-duration: 0.1s !important; }
  .si-block--flash  { animation-duration: 0.05s !important; }
  .si-hint          { animation: none; }
  .si-fade-enter-active,
  .si-fade-leave-active { transition-duration: 0.05s !important; }
}
</style>
