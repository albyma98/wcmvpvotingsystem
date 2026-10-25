<template>
  <section class="volley-arena" :class="[gameId, { 'is-scored': celebration }]" :aria-label="`${mode.title} ${mode.emphasis}`">
    <header class="va-header">
      <div class="va-brand"><span>{{ clubName || 'VOLLEY CLUB' }}</span><b>{{ mode.title }} <em>{{ mode.emphasis }}</em></b><small>{{ mode.eyebrow }}</small></div>
      <div class="va-wallet" aria-label="Saldo monete"><i class="va-coin" aria-hidden="true"></i>{{ walletCoins }}</div>
      <button class="va-sound" type="button" :aria-pressed="soundEnabled" :aria-label="soundEnabled ? 'Disattiva audio' : 'Attiva audio'" @click="toggleSound"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true"><path d="M11 5 6 9H3v6h3l5 4Z"/><path v-if="soundEnabled" d="M15 8c3 2 3 6 0 8m3-11c5 4 5 10 0 14"/><path v-else d="m16 9 6 6m0-6-6 6"/></svg></button>
    </header>

    <div class="va-court">
      <div class="va-stats" aria-live="off"><span><small>{{ mode.noun }}</small><b>{{ wins }}</b></span><span><small>DI FILA</small><b :class="{ 'va-hot': streak > 2 }">{{ streak }}<svg v-if="streak > 2" viewBox="0 0 16 20" aria-hidden="true"><path d="M9 0C10 7 4 6 6 12c-3-1-3-4-3-4C-4 17 4 23 11 19c8-5 4-11 2-13 0 4-2 5-2 5C13 6 9 0 9 0Z" fill="currentColor"/></svg></b></span><span><small>MONETE PARTITA</small><b class="va-gold">{{ matchEarned }}<i class="va-coin" aria-hidden="true"></i></b></span></div>
      <svg ref="courtEl" class="va-svg" viewBox="0 0 360 560" preserveAspectRatio="xMidYMid meet" tabindex="0" role="application" :aria-label="courtLabel" @pointerdown="onPointerDown" @pointermove="onPointerMove" @pointerup="onPointerUp" @pointercancel="cancelAim" @lostpointercapture="onLostCapture" @keydown="onCourtKeydown">
        <defs>
          <radialGradient :id="`${uid}-night`" cx=".5" cy=".35" r=".75"><stop stop-color="#193a55"/><stop offset="1" stop-color="#060e21"/></radialGradient>
          <linearGradient :id="`${uid}-floor`" x2="0" y2="1"><stop stop-color="#c78354"/><stop offset="1" stop-color="#eead72"/></linearGradient>
          <linearGradient :id="`${uid}-zone`" x2="0" y2="1"><stop stop-color="#e38c62"/><stop offset="1" stop-color="#f8b080"/></linearGradient>
          <linearGradient :id="`${uid}-teal`" x2="0" y2="1"><stop stop-color="#123d50"/><stop offset="1" stop-color="#247984"/></linearGradient>
          <linearGradient :id="`${uid}-skin`" x2="1" y2="1"><stop stop-color="#f4d1ac"/><stop offset=".55" stop-color="#c88d69"/><stop offset="1" stop-color="#955f49"/></linearGradient>
          <linearGradient :id="`${uid}-kit`" x2="1" y2="1"><stop stop-color="#7df4da"/><stop offset="1" stop-color="#25938f"/></linearGradient>
          <radialGradient :id="`${uid}-ball`" cx=".28" cy=".2" r=".85"><stop stop-color="#fff4aa"/><stop offset=".45" stop-color="#ffd83e"/><stop offset="1" stop-color="#b88b11"/></radialGradient>
          <radialGradient :id="`${uid}-glow`"><stop stop-color="#a3ffe5" stop-opacity=".28"/><stop offset="1" stop-color="#80ffe1" stop-opacity="0"/></radialGradient>
          <pattern :id="`${uid}-mesh`" width="13" height="12" patternUnits="userSpaceOnUse"><path d="M0 0H13V12" fill="none" stroke="#d7edfa" stroke-width=".7" opacity=".5"/></pattern>
          <pattern :id="`${uid}-wood`" width="36" height="92" patternUnits="userSpaceOnUse"><path d="M0 0v92M0 46h36M9 0q-3 23 0 46t0 46m19-92q3 23 0 46t0 46" fill="none" stroke="#ffddaf" stroke-opacity=".1"/></pattern>
          <clipPath :id="`${uid}-ball-clip`"><circle r="24"/></clipPath>
          <g :id="`${uid}-volleyball`"><circle r="26" fill="#fff5bb" opacity=".09"/><circle r="24" :fill="paint('ball')"/><g :clip-path="`url(#${uid}-ball-clip)`"><path d="M-25-8Q-8-27 9-20q12 6 14 19Q8-12-8-4q-7 4-11 14L-27 2Z" fill="#17375a"/><path d="M21-15Q6-12 5 4q0 12 16 20L7 29Q-8 17-7 5q2-16 20-24Z" fill="#f1f6f4"/><path d="M-21 13Q-8 25 10 20q8-3 14-10l4 11Q12 37-7 28q-12-6-18-19Z" fill="#17375a"/><g fill="none" stroke="#102746" stroke-width=".65" stroke-opacity=".55"><path d="M-24-8Q-9-20 9-14M-22-3Q-7-15 10-9M17-18Q2-9 0 5q0 12 15 20M-21 18Q-4 33 19 15"/></g></g><circle r="23.5" fill="none" stroke="#fff5c3" stroke-opacity=".35"/><path d="M-13-14q6-6 13-5" fill="none" stroke="#fff" stroke-width="2.5" stroke-linecap="round" opacity=".55"/></g>
          <g :id="`${uid}-athlete`"><ellipse cy="62" rx="22" ry="5" fill="#071b2e" opacity=".22"/><path d="m-12 25-7 25 8 4 11-20 9 20 9-4-7-25" :fill="paint('skin')"/><path d="M-16 19-20 32H-4l4-6 4 6h16l-4-13Z" fill="#173348"/><path d="M-12-17-17 20h34l-5-37-8-5h-8Z" :fill="paint('kit')"/><path d="m-12-11-9-25 5-16m28 41 9-25-5-16" fill="none" :stroke="paint('skin')" stroke-width="7" stroke-linecap="round"/><path d="m-21-49-3-9m6 8-1-11m38 11 1-11m3 12 3-9" fill="none" :stroke="paint('skin')" stroke-width="3" stroke-linecap="round"/><circle cy="-34" r="10" :fill="paint('skin')"/><path d="M-10-35q-2-13 10-13t10 13l-5-5-11 1Z" fill="#172536"/><path d="m-18 49-8 7 15 2 2-6m18 0 2 6 15-2-8-7" fill="#e9f5f5"/><text y="9" text-anchor="middle" fill="#113c49" font-size="13" font-weight="900">1</text></g>
          <g :id="`${uid}-hand`"><path d="M-16 72-14 34q-12-13-11-22 0-5 4-3l10 9-3-33q0-7 5-6l5 31-1-39q1-6 5-4l4 40 4-37q1-6 5-3l-1 41 7-28q3-5 6-1L19 22q-1 12-6 16l3 34Z" :fill="paint('skin')"/><path d="m-16 55 32 1 1 21h-34Z" fill="#f6f7ec"/><path d="M-14 62h28" stroke="#bccac6" stroke-width="1.5"/></g>
        </defs>
        <rect width="360" height="560" :fill="paint('night')"/>
        <g opacity=".07" fill="#9ee6ff"><path d="M20 0H70L220 325H100ZM340 0H290L140 325H260Z"/></g>
        <g fill="#c5e8f8" opacity=".17"><circle v-for="(dot, i) in crowd" :key="i" :cx="dot.x" :cy="dot.y" :r="dot.r"/></g>
        <g stroke="#d2f0ff" stroke-linecap="round"><path d="M10 82H73M287 82H350" stroke-width="3" opacity=".75"/><path d="M10 82H73M287 82H350" stroke-width="9" opacity=".08"/></g>
        <path d="M0 115Q180 100 360 115v445H0Z" :fill="paint('teal')"/>
        <path d="M42 122H318L411 560H-51Z" :fill="paint('floor')"/>
        <path d="M60 133H300L373 537H-13Z" :fill="paint('zone')"/>
        <path d="M42 122H318L411 560H-51Z" :fill="paint('wood')"/>
        <g fill="none" stroke="#fff4dc" stroke-width="2" opacity=".7"><path d="M60 133H300L373 537H-13ZM23 336H337M40 244H320M8 421H352"/></g>
        <path d="M0 112H360v13H0Z" fill="#091b2c"/><text x="180" y="121" text-anchor="middle" fill="#a4dfda" font-size="6" font-weight="800" letter-spacing="3">{{ (clubName || 'THE HOME OF VOLLEY').toUpperCase() }}</text>
        <image v-if="clubLogoUrl" :href="clubLogoUrl" x="150" y="389" width="60" height="60" preserveAspectRatio="xMidYMid meet" opacity=".2"/>
        <text x="180" y="495" text-anchor="middle" fill="#744833" font-size="15" font-weight="900" letter-spacing="5" opacity=".2">{{ mode.title }} {{ mode.emphasis }}</text>

        <g v-if="gameId === 'setter-pro' && challenge" :transform="`translate(${target.x}, ${target.y + 88}) scale(.85)`" class="va-attacker"><use :href="`#${uid}-athlete`"/><path d="M-25 73h50" stroke="#baffea" stroke-width="2" stroke-linecap="round" opacity=".5"/></g>
        <g v-if="challenge && !['loading', 'error'].includes(state)" :transform="`translate(${target.x}, ${target.y})`" class="va-target" :class="{ 'va-target-high': gameId !== 'volley-rush' }" aria-hidden="true">
          <ellipse :rx="mode.rx + 20" :ry="mode.ry + 18" :fill="paint('glow')"/>
          <ellipse :rx="mode.rx" :ry="mode.ry" fill="#b4ffdd" fill-opacity=".09" stroke="#bcffe7" stroke-width="1.5" stroke-dasharray="5 5"/>
          <ellipse :rx="mode.rx * .43" :ry="mode.ry * .43" fill="#d0ffe6" fill-opacity=".14" stroke="#e6fff4" stroke-width="1.5"/>
          <path d="M-5 0H5M0-5V5" stroke="#fff" stroke-width="2" stroke-linecap="round"/>
          <rect x="-40" :y="-mode.ry - 19" width="80" height="13" rx="6.5" fill="#071c2a" fill-opacity=".85"/>
          <text :y="-mode.ry - 10" text-anchor="middle" fill="#baffea" font-size="6" font-weight="900" letter-spacing="1.2">{{ gameId === 'volley-rush' ? 'SPAZIO LIBERO' : gameId === 'setter-pro' ? 'IL TUO ATTACCANTE' : 'CHIUDI QUI' }}</text>
        </g>
        <g v-if="gameId === 'volley-rush' && challenge?.blocker" :transform="`translate(${challenge.blocker.x}, 300)`" aria-hidden="true"><ellipse cy="49" rx="27" ry="5" fill="#142d39" opacity=".2"/><path d="M-16-11-21 28H21L16-11 6-17H-6Z" fill="#29465c"/><path d="M-20 28-21 44H-5L0 34 5 44H21L20 28Z" fill="#1a2f46"/><circle cy="-29" r="11" :fill="paint('skin')"/><path d="M-11-29q-2-14 12-14 12 1 10 14l-5-6-11 1Z" fill="#1a2234"/><path d="m-14-7-13-26 3-22m38 48 13-26-3-22" fill="none" :stroke="paint('skin')" stroke-width="8" stroke-linecap="round"/><path d="m-27-53-3-9m6 9v-12m48 12v-12m3 12 3-9" :stroke="paint('skin')" stroke-width="3" stroke-linecap="round"/><text y="15" text-anchor="middle" fill="#b5d3e3" font-size="14" font-weight="900">4</text></g>

        <g v-if="aimVisible && gameId !== 'monster-block'" aria-hidden="true"><path :d="`M180 466 Q${aim.x} ${Math.max(70, aim.y - 90)} ${aim.x} ${aim.y}`" fill="none" stroke="#dcfff5" stroke-width="1.5" stroke-dasharray="3 9" opacity=".45"/><g :transform="`translate(${aim.x}, ${aim.y})`"><circle r="10" fill="#10263e" fill-opacity=".35" stroke="#fff" stroke-width="1.6"/><path d="M-16 0h6m20 0h6M0-16v6m0 20v6" stroke="#fff" stroke-width="1.6" stroke-linecap="round"/></g></g>
        <g v-if="ballBehindNet && ballPose.visible" :transform="ballTransform" aria-hidden="true"><use :href="`#${uid}-volleyball`"/></g>
        <g class="va-net" aria-hidden="true"><path d="M24 285H336v70H24Z" fill="#10283a" fill-opacity=".13"/><path d="M24 285H336v70H24Z" :fill="paint('mesh')"/><path d="M24 285H336" stroke="#f3f7e8" stroke-width="5"/><path d="M24 288H336" stroke="#cbd9de" stroke-width="1"/><path d="M24 356H336" stroke="#e4e8dd" stroke-width="2"/><path d="M22 276v118m316-118v118" stroke="#23374f" stroke-width="7"/><path d="M22 278v48m316-48v48" stroke="#d9e2dc" stroke-width="2"/><path d="M22 263v18m316-18v18" stroke="#f5f3e7" stroke-width="2"/><path d="M22 263v5m316-5v5" stroke="#f16d60" stroke-width="2"/><path d="m20 391-8 8m328-8 8 8" stroke="#143748" stroke-width="4"/></g>

        <g v-if="gameId === 'monster-block' && challenge" :transform="`translate(${aim.x}, ${handY})`" class="va-block-hands" aria-hidden="true"><ellipse cy="148" rx="45" ry="7" fill="#10293d" opacity=".12"/><use :href="`#${uid}-hand`" transform="translate(-25 0) rotate(-6)"/><use :href="`#${uid}-hand`" transform="translate(25 0) scale(-1 1) rotate(-6)"/></g>
        <g v-else-if="aimVisible" aria-hidden="true"><path d="m108 560 35-66 17-22q5-6 9-2t-1 11l-8 16 11-5q8-2 4 7l-27 38-5 23m109 0-35-66-17-22q-5-6-9-2t1 11l8 16-11-5q-8-2-4 7l27 38 5 23" :fill="paint('skin')"/><path d="m105 550 43 1-3 9h-45m155-10-43 1 3 9h45" fill="#eaf5e9"/></g>
        <ellipse v-if="!ballBehindNet && gameId !== 'monster-block'" :cx="ballPose.x" cy="512" :rx="24 * ballPose.scale" :ry="5 * ballPose.scale" fill="#153040" opacity=".18"/>
        <path v-if="state === 'flying' && gameId === 'volley-rush' && flightProgress > .48 && flightProgress < .95" :d="`M${ballPose.x - (aim.x - 180) * .09} ${ballPose.y - 38} ${ballPose.x} ${ballPose.y}`" stroke="#fff6c2" stroke-width="9" stroke-linecap="round" opacity=".28"/>
        <g v-if="!ballBehindNet && ballPose.visible" :transform="ballTransform" aria-hidden="true"><use :href="`#${uid}-volleyball`"/></g>
        <g v-if="state === 'ready' && !attempts" class="va-drag-hint" aria-hidden="true"><path v-if="gameId === 'monster-block'" d="m112 450-10 8 10 8m136-16 10 8-10 8m-113-8h90" fill="none" stroke="#e7fff1" stroke-width="2" stroke-linecap="round"/><path v-else d="M180 410v-43m-7 8 7-8 7 8" fill="none" stroke="#fffbe5" stroke-width="2" stroke-linecap="round"/><text x="180" :y="gameId === 'monster-block' ? 492 : 545" text-anchor="middle" fill="#fff5df" font-size="8" font-weight="900" letter-spacing="2">{{ gameId === 'monster-block' ? 'SPOSTA LE MANI' : 'TRASCINA E MIRA' }}</text></g>
        <g v-if="celebration && lastReceipt?.coins_earned > 0" class="va-reward" aria-hidden="true"><circle cx="180" cy="214" r="44" fill="#071a2c" fill-opacity=".65"/><text x="171" y="224" text-anchor="middle" fill="#ffe8a4" font-size="35" font-weight="900">+{{ lastReceipt.coins_earned }}</text><circle cx="208" cy="212" r="10" fill="#edbd43" stroke="#fff0bd" stroke-width="2"/><path d="M208 206v12" stroke="#b17a15" stroke-width="2"/><g transform="translate(180 214)" stroke="#d4ffed" stroke-width="2" stroke-linecap="round"><path v-for="angle in burstAngles" :key="angle" :transform="`rotate(${angle})`" d="M0-52v-8"/></g></g>
      </svg>

      <div v-if="state === 'loading'" class="va-cover"><i class="va-loader"></i><b>Il palazzetto è pronto.</b><span>Prepariamo il prossimo pallone…</span></div>
      <div v-if="state === 'timing'" class="va-timing"><div><b>{{ gameId === 'monster-block' ? 'SALTA AL MOMENTO GIUSTO' : 'TOCCA QUANDO SEI NEL VERDE' }}</b><button type="button" @click="resetAim">Cambia mira</button></div><div class="va-meter" role="meter" aria-label="Tempo di esecuzione" :aria-valuenow="Math.round(timing * 100)" aria-valuemin="0" aria-valuemax="100" :aria-valuetext="timingGood ? 'Momento giusto: tocca ora' : 'Attendi la zona verde'"><span class="va-safe" :style="{ left: `${safeStart * 100}%`, width: `${(safeEnd - safeStart) * 100}%` }"></span><span class="va-perfect" :style="{ left: `${(timingTarget - .055) * 100}%`, width: '11%' }"></span><i :style="{ left: `${timing * 100}%` }"></i></div><span>PREPARA <b>{{ timingGood ? 'ORA!' : 'ATTENDI…' }}</b> ESEGUI</span></div>
      <div v-if="state === 'result'" class="va-feedback" :class="{ 'va-feedback-miss': !celebration }" aria-hidden="true"><b>{{ celebration ? lastReceipt?.perfect ? 'CHE GIOCATA!' : mode.success : missTitle }}</b><span>{{ celebration ? `${streak > 1 ? `${streak} di fila · ` : ''}${lastReceipt?.coins_earned > 0 ? `${lastReceipt.coins_earned} moneta salvata` : 'Punto confermato · saldo aggiornato'}` : hint }}</span></div>
    </div>

    <footer class="va-footer"><div class="va-status" role="status" aria-live="polite"><b>{{ statusTitle }}</b><span>{{ statusDetail }}</span></div><button v-if="state === 'error'" class="va-action" type="button" @click="retry">{{ pendingAttempt ? 'SALVA IL PUNTO' : 'RIPROVA' }}</button><button v-else-if="state === 'ready' || state === 'aiming' || state === 'timing'" class="va-action" :class="{ 'va-action-now': state === 'timing' && timingGood }" type="button" @click="primaryAction">{{ state === 'timing' ? mode.action : 'PRONTO' }}<svg viewBox="0 0 20 20" aria-hidden="true"><path d="m7 4 6 6-6 6" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"/></svg></button><span v-else class="va-rule"><i class="va-coin" aria-hidden="true"></i>1 punto<br>1 moneta</span></footer>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { prepareVolleyballRound, submitVolleyballRound } from '../../api';
import { VOLLEYBALL_MODES, clamp, volleyballTiming, previewVolleyballAttempt, sampleVolleyballFlight } from '../../games/volleyball';

const props = defineProps({ gameId: { type: String, required: true }, eventId: { type: Number, required: true }, walletCoins: { type: Number, default: 0 }, earnedCoins: { type: Number, default: 0 }, clubName: { type: String, default: '' }, clubLogoUrl: { type: String, default: '' } });
const emit = defineEmits(['credited', 'saving-change', 'round-started', 'round-completed']);
const gameEventId = props.eventId;
const gameId = props.gameId;
const mode = VOLLEYBALL_MODES[gameId] || VOLLEYBALL_MODES['volley-rush'];
const uid = `va-${gameId}`;
const courtEl = ref(null);
const state = ref('loading');
const challenge = ref(null);
const aim = ref({ ...mode.target });
const timing = ref(0);
const flightProgress = ref(0);
const pendingAttempt = ref(null);
const lastReceipt = ref(null);
const wins = ref(0);
const attempts = ref(0);
const streak = ref(0);
const confirmedEarned = ref(0);
const celebration = ref(false);
const hint = ref('');
const errorMessage = ref('');
const soundEnabled = ref(false);
const crowd = Array.from({ length: 90 }, (_, i) => ({ x: i % 18 * 23 - 16, y: 61 + Math.floor(i / 18) * 12, r: .9 + i % 3 * .35 }));
const burstAngles = [0, 45, 90, 135, 180, 225, 270, 315];
let pointerId = null;
let pointerStart = null;
let pointerAimStart = 180;
let timingStartedAt = 0;
let preparedAt = 0;
let frameId;
let resultTimer;
let saveDelayTimer;
let resolveSaveDelay;
let resolveFlight;
let audioContext;
let prepareBusy = false;
let receiptBusy = false;
let disposed = false;
const paint = (name) => `url(#${uid}-${name})`;
const now = () => window.performance?.now?.() ?? Date.now();
const target = computed(() => challenge.value?.target || mode.target);
const timingTarget = computed(() => clamp(challenge.value?.timing_target ?? .7, 0, 1));
const safeStart = computed(() => Math.max(0, timingTarget.value - mode.timingWindow));
const safeEnd = computed(() => Math.min(1, timingTarget.value + mode.timingWindow));
const timingGood = computed(() => timing.value >= safeStart.value && timing.value <= safeEnd.value);
const matchEarned = computed(() => Math.max(Number(props.earnedCoins) || 0, confirmedEarned.value));
const aimVisible = computed(() => ['ready', 'aiming', 'timing'].includes(state.value));
const activeAttempt = computed(() => pendingAttempt.value?.input || lastReceipt.value?.input);
const ballPose = computed(() => {
  if (activeAttempt.value && ['flying', 'saving', 'result', 'error'].includes(state.value)) return sampleVolleyballFlight(gameId, activeAttempt.value, flightProgress.value, challenge.value);
  if (gameId === 'monster-block') {
    const progress = state.value === 'timing' ? timing.value / timingTarget.value : 0;
    return { x: target.value.x, y: 110 + (target.value.y - 110) * progress, scale: .4 + Math.min(1.4, progress) * .4, rotation: timing.value * 240, visible: Boolean(challenge.value) };
  }
  return { x: 180, y: 466, scale: 1, rotation: 0, visible: true };
});
const ballTransform = computed(() => `translate(${ballPose.value.x}, ${ballPose.value.y}) scale(${ballPose.value.scale}) rotate(${ballPose.value.rotation})`);
const ballBehindNet = computed(() => gameId !== 'monster-block' && activeAttempt.value && ['flying', 'saving', 'result', 'error'].includes(state.value) && flightProgress.value >= .45);
const handY = computed(() => state.value === 'flying' ? 370 - Math.sin(Math.PI * Math.min(1, flightProgress.value * 1.2)) * 102 : 370);
const courtLabel = computed(() => `${mode.title} ${mode.emphasis}. ${gameId === 'monster-block' ? 'Trascina le mani in orizzontale sotto la palla, poi tocca quando il cursore è verde.' : 'Trascina la palla o tocca il bersaglio per mirare, poi tocca quando il cursore è verde.'} Tastiera: frecce per mirare, Invio per preparare e di nuovo Invio per eseguire. Esc per cambiare mira.`);
const missTitle = computed(() => {
  const prediction = activeAttempt.value ? previewVolleyballAttempt(gameId, challenge.value, activeAttempt.value) : null;
  return prediction?.blocked ? 'IL MURO TI HA LETTO' : prediction && !prediction.inTime ? 'QUESTIONE DI TEMPO' : 'QUASI! RIPROVA';
});
const statusTitle = computed(() => errorMessage.value ? pendingAttempt.value ? 'Il punto resta da verificare' : 'Rientra in campo' : state.value === 'loading' ? 'Il prossimo punto è tuo' : state.value === 'timing' ? `Tocca ${mode.action.toLowerCase()} nel verde` : state.value === 'flying' ? gameId === 'monster-block' ? 'Su le mani!' : 'Palla in gioco…' : state.value === 'saving' ? 'Verifichiamo il punto…' : celebration.value ? lastReceipt.value?.coins_earned > 0 ? 'Una moneta nel tuo portafoglio!' : 'Punto confermato!' : gameId === 'monster-block' ? 'Sposta le mani sotto la palla' : gameId === 'setter-pro' ? 'Servi il tuo attaccante' : 'Mira allo spazio libero');
const statusDetail = computed(() => errorMessage.value || (state.value === 'timing' ? 'Il verde è il tuo momento. Tocca il campo o il pulsante.' : state.value === 'saving' ? 'Attendi la conferma delle monete.' : hint.value || (gameId === 'monster-block' ? 'Trascina a destra o sinistra, poi rilascia per preparare il muro.' : 'Trascina la palla sul bersaglio, poi rilascia per prepararti.')));

function courtPoint(event) {
  const rect = courtEl.value?.getBoundingClientRect();
  if (!rect?.width || !rect.height) return null;
  const scale = Math.min(rect.width / 360, rect.height / 560);
  return { x: (event.clientX - rect.left - (rect.width - 360 * scale) / 2) / scale, y: (event.clientY - rect.top - (rect.height - 560 * scale) / 2) / scale };
}
function moveAim(point) {
  aim.value = { x: clamp(point.x, 40, 320), y: gameId === 'monster-block' ? target.value.y : clamp(point.y, 85, 280) };
}
function isExpired() {
  const expiresAt = Date.parse(challenge.value?.expires_at || '');
  return Number.isFinite(expiresAt) && expiresAt <= Date.now() + 2500;
}
function onPointerDown(event) {
  if (event.isPrimary === false || (event.button != null && event.button !== 0) || !['ready', 'timing'].includes(state.value) || !challenge.value) return;
  if (isExpired()) { hint.value = 'Nuovo pallone pronto: scegli di nuovo la mira.'; void prepare(); return; }
  const point = courtPoint(event);
  if (!point || point.x < 0 || point.x > 360 || point.y < 80 || point.y > 560) return;
  event.preventDefault();
  courtEl.value?.focus({ preventScroll: true });
  if (state.value === 'timing' && gameId !== 'monster-block') { void launch(); return; }
  pointerId = event.pointerId;
  pointerStart = point;
  pointerAimStart = aim.value.x;
  hint.value = '';
  if (state.value !== 'timing' && (gameId === 'monster-block' || point.y < 300)) moveAim(point);
  if (state.value === 'ready') state.value = 'aiming';
  try { courtEl.value?.setPointerCapture(pointerId); } catch { /* pointer capture is optional */ }
}
function onPointerMove(event) {
  if (pointerId == null || event.pointerId !== pointerId || !['aiming', 'timing'].includes(state.value)) return;
  const point = courtPoint(event);
  if (!point) return;
  if (state.value === 'timing' && gameId === 'monster-block') {
    if (!pointerStart || Math.abs(point.x - pointerStart.x) < 8) return;
    moveAim({ x: pointerAimStart + point.x - pointerStart.x, y: target.value.y });
  } else moveAim(point);
}
function releasePointer() {
  const previous = pointerId;
  pointerId = null;
  pointerStart = null;
  if (previous != null) { try { courtEl.value?.releasePointerCapture(previous); } catch { /* already released */ } }
}
function onPointerUp(event) {
  if (event.pointerId !== pointerId || pointerId == null) return;
  const point = courtPoint(event);
  const moved = point && pointerStart ? Math.hypot(point.x - pointerStart.x, point.y - pointerStart.y) : 0;
  const wasTiming = state.value === 'timing';
  // A tap on the held ball is a prompt, never an unintended shot.
  const heldBallTap = gameId !== 'monster-block' && !wasTiming && point?.y > 300 && moved < 8;
  if (!heldBallTap) onPointerMove(event);
  releasePointer();
  if (heldBallTap) { state.value = 'ready'; hint.value = 'Accompagna la palla fino al bersaglio verde.'; return; }
  if (wasTiming) void launch(); else startTiming();
}
function cancelAim(event) {
  if (pointerId != null && event?.pointerId != null && event.pointerId !== pointerId) return;
  releasePointer();
  if (state.value === 'aiming') state.value = 'ready';
}
function onLostCapture(event) { if (event.pointerId === pointerId) cancelAim(event); }
function onCourtKeydown(event) {
  if (!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Enter', ' ', 'Escape'].includes(event.key) || pointerId != null) return;
  if (event.repeat && (event.key === 'Enter' || event.key === ' ')) { event.preventDefault(); return; }
  if (event.key === 'Escape') {
    if (['aiming', 'timing'].includes(state.value)) { event.preventDefault(); event.stopPropagation(); resetAim(); }
    return;
  }
  if (!['ready', 'aiming', 'timing'].includes(state.value)) return;
  event.preventDefault();
  if (event.key === 'Enter' || event.key === ' ') { primaryAction(); return; }
  const offsets = { ArrowLeft: [-8, 0], ArrowRight: [8, 0], ArrowUp: [0, -6], ArrowDown: [0, 6] };
  const [dx, dy] = offsets[event.key];
  moveAim({ x: aim.value.x + dx, y: aim.value.y + dy });
  hint.value = '';
}
function resetAim() {
  if (!['ready', 'aiming', 'timing'].includes(state.value)) return;
  window.cancelAnimationFrame(frameId);
  releasePointer();
  timing.value = 0;
  state.value = 'ready';
}
function primaryAction() { if (state.value === 'timing') void launch(); else startTiming(); }
function startTiming() {
  if (!['ready', 'aiming'].includes(state.value) || !challenge.value || pendingAttempt.value) return;
  releasePointer();
  if (isExpired()) { void prepare(); return; }
  timing.value = 0;
  timingStartedAt = now();
  state.value = 'timing';
  const tick = () => {
    if (disposed || state.value !== 'timing') return;
    timing.value = volleyballTiming(now() - timingStartedAt, mode.cycleMs);
    frameId = window.requestAnimationFrame(tick);
  };
  frameId = window.requestAnimationFrame(tick);
}

async function prepare() {
  if (disposed || prepareBusy || pendingAttempt.value || receiptBusy) return;
  prepareBusy = true;
  window.cancelAnimationFrame(frameId);
  releasePointer();
  state.value = 'loading';
  errorMessage.value = '';
  celebration.value = false;
  try {
    const response = await prepareVolleyballRound(gameEventId, gameId);
    if (disposed) return;
    if (!response.ok || !response.data?.round_id || !Number.isFinite(response.data.target?.x) || !Number.isFinite(response.data.target?.y)) {
      state.value = 'error'; errorMessage.value = response.message || 'Campo non disponibile. Controlla la connessione e riprova.'; return;
    }
    challenge.value = response.data;
    preparedAt = now();
    aim.value = { x: 180, y: gameId === 'monster-block' ? response.data.target.y : mode.target.y };
    timing.value = 0;
    flightProgress.value = 0;
    lastReceipt.value = null;
    state.value = 'ready';
  } catch {
    if (!disposed) { state.value = 'error'; errorMessage.value = 'Non riusciamo a preparare il pallone. Riprova.'; }
  } finally { prepareBusy = false; }
}
function animateFlight() {
  return new Promise((resolve) => {
    resolveFlight = resolve;
    const reducedMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
    const duration = reducedMotion ? 180 : mode.flightMs;
    let startedAt;
    const frame = (timestamp) => {
      if (disposed) { resolveFlight = null; resolve(); return; }
      startedAt ??= timestamp;
      flightProgress.value = Math.min(1, Math.max(0, timestamp - startedAt) / duration);
      if (flightProgress.value < 1) frameId = window.requestAnimationFrame(frame);
      else { resolveFlight = null; resolve(); }
    };
    frameId = window.requestAnimationFrame(frame);
  });
}
async function launch() {
  if (state.value !== 'timing' || !challenge.value || pendingAttempt.value) return;
  if (isExpired()) { resetAim(); hint.value = 'La palla è cambiata: prepara una nuova giocata.'; await prepare(); return; }
  timing.value = volleyballTiming(now() - timingStartedAt, mode.cycleMs);
  window.cancelAnimationFrame(frameId);
  releasePointer();
  pendingAttempt.value = { roundId: challenge.value.round_id, input: { aim_x: aim.value.x, aim_y: aim.value.y, timing: timing.value } };
  emit('round-started', { roundId: pendingAttempt.value.roundId });
  state.value = 'flying';
  hint.value = '';
  flightProgress.value = 0;
  await animateFlight();
  if (!disposed) await saveAttempt();
}
async function saveAttempt() {
  if (disposed || receiptBusy || !pendingAttempt.value) return;
  receiptBusy = true;
  state.value = 'saving';
  errorMessage.value = '';
  emit('saving-change', true);
  const attempt = pendingAttempt.value;
  try {
    // Reduced motion and background tabs can finish the visual before the server's 600 ms round minimum.
    const waitMs = Math.max(0, 600 - (now() - preparedAt));
    if (waitMs > 0) await new Promise((resolve) => { resolveSaveDelay = resolve; saveDelayTimer = window.setTimeout(() => { resolveSaveDelay = null; resolve(); }, waitMs); });
    if (disposed) return;
    const response = await submitVolleyballRound(gameEventId, gameId, attempt.roundId, { ...attempt.input });
    if (disposed) return;
    if (!response.ok) {
      state.value = 'error';
      if ([404, 410].includes(response.status)) {
        pendingAttempt.value = null; challenge.value = null;
        errorMessage.value = response.message || 'Questo pallone non è più disponibile. Rientra in campo per riprovare.';
      } else {
        errorMessage.value = response.message || 'Connessione interrotta. Salva di nuovo questa giocata: il punto sarà conteggiato una sola volta.';
      }
      return;
    }
    const receipt = response.data;
    if (receipt?.round_id !== attempt.roundId || receipt.game_id !== gameId || typeof receipt.success !== 'boolean' || typeof receipt.perfect !== 'boolean' || !Number.isInteger(receipt.wallet) || receipt.wallet < 0 || !Number.isInteger(receipt.game_earned) || receipt.game_earned < 0 || !Number.isInteger(receipt.coins_earned) || receipt.coins_earned < 0 || receipt.coins_earned > 1 || (!receipt.success && (receipt.perfect || receipt.coins_earned > 0))) throw new Error('Invalid volleyball receipt');
    lastReceipt.value = { ...receipt, input: attempt.input };
    confirmedEarned.value = Math.max(confirmedEarned.value, Number(receipt.game_earned) || 0);
    attempts.value += 1;
    celebration.value = receipt.success === true;
    if (celebration.value) { wins.value += 1; streak.value += 1; playPoint(); }
    else {
      streak.value = 0;
      const prediction = previewVolleyballAttempt(gameId, challenge.value, attempt.input);
      hint.value = prediction.blocked ? 'Cerca lo spazio lontano dalle mani del muro.' : !prediction.inTime ? 'Aspetta il verde, poi tocca per eseguire.' : gameId === 'monster-block' ? 'Porta le mani al centro del bersaglio.' : 'Porta la mira al centro del bersaglio verde.';
    }
    emit('credited', { ...receipt, event_id: gameEventId });
    emit('round-completed', { roundId: attempt.roundId });
    pendingAttempt.value = null;
    state.value = 'result';
    resultTimer = window.setTimeout(() => { void prepare(); }, celebration.value ? 1050 : 850);
  } catch {
    if (!disposed) { state.value = 'error'; errorMessage.value = 'Il risultato resta in attesa. Riprova a salvare questa giocata.'; }
  } finally { receiptBusy = false; if (!disposed) emit('saving-change', false); }
}
function retry() { if (pendingAttempt.value) void saveAttempt(); else void prepare(); }
function onVisibilityChange() {
  if (typeof document === 'undefined' || !document.hidden) return;
  if (['aiming', 'timing'].includes(state.value)) { resetAim(); hint.value = 'Riprendi la mira: il pallone ti aspetta.'; }
  if (state.value === 'flying') { window.cancelAnimationFrame(frameId); flightProgress.value = 1; const finish = resolveFlight; resolveFlight = null; finish?.(); }
}
function toggleSound() {
  soundEnabled.value = !soundEnabled.value;
  if (!soundEnabled.value) return;
  try { const Audio = window.AudioContext || window.webkitAudioContext; if (Audio) { audioContext ||= new Audio(); void audioContext.resume(); } } catch { soundEnabled.value = false; }
}
function playPoint() {
  if (!soundEnabled.value || !audioContext) return;
  try {
    [523, 784, 1047].forEach((frequency, index) => {
      const oscillator = audioContext.createOscillator();
      const gain = audioContext.createGain();
      const at = audioContext.currentTime + index * .065;
      oscillator.type = 'sine'; oscillator.frequency.value = frequency;
      gain.gain.setValueAtTime(0, at); gain.gain.linearRampToValueAtTime(.035, at + .01); gain.gain.exponentialRampToValueAtTime(.001, at + .2);
      oscillator.connect(gain); gain.connect(audioContext.destination); oscillator.start(at); oscillator.stop(at + .21);
    });
  } catch { /* sound is optional */ }
}
onMounted(() => { if (typeof document !== 'undefined') document.addEventListener('visibilitychange', onVisibilityChange); void prepare(); });
onBeforeUnmount(() => {
  disposed = true;
  window.cancelAnimationFrame(frameId);
  window.clearTimeout(resultTimer);
  window.clearTimeout(saveDelayTimer);
  releasePointer();
  resolveFlight?.();
  resolveSaveDelay?.();
  if (typeof document !== 'undefined') document.removeEventListener('visibilitychange', onVisibilityChange);
  emit('saving-change', false);
  if (audioContext) void audioContext.close();
});
</script>

<style scoped>
.volley-arena{--va-accent:#9bffdf;display:flex;flex-direction:column;width:100%;height:100%;min-height:0;overflow:hidden;border-radius:24px;background:#0a1528;color:#f3fcff;font-family:Inter,system-ui,sans-serif;box-shadow:0 15px 50px #0005}.volley-arena *{box-sizing:border-box}.volley-arena.setter-pro{--va-accent:#b5c5ff}.volley-arena.monster-block{--va-accent:#ffc393}.va-header{display:flex;align-items:center;gap:8px;padding:13px 14px 11px;background:linear-gradient(115deg,#1c3046,#0d192d);border-bottom:1px solid #d2f8ff14;flex:none}.va-brand{flex:1;min-width:0}.va-brand>span{display:block;max-width:160px;overflow:hidden;white-space:nowrap;text-overflow:ellipsis;color:#b4c9d4;font-size:7px;font-weight:800;letter-spacing:.18em;text-transform:uppercase}.va-brand b{display:block;margin:4px 0 5px;font-size:20px;font-weight:950;line-height:1;letter-spacing:-.045em;font-style:italic}.va-brand em{color:var(--va-accent);font-style:inherit}.va-brand>small{display:block;color:#829caa;font-size:5.5px;font-weight:800;letter-spacing:.1em}.va-wallet{display:flex;align-items:center;gap:6px;padding:8px 9px;border:1px solid #fed88a25;border-radius:999px;background:#ffd47210;color:#ffe5a5;font-size:12px;font-weight:900;font-variant-numeric:tabular-nums}.va-coin{display:inline-block;flex:none;width:14px;height:14px;border-radius:50%;background:radial-gradient(circle at 30% 25%,#fff2be,#f3c34a 55%,#a56d13);box-shadow:inset 0 0 0 2px #f8d47b,0 1px 4px #0004}.va-sound{display:grid;place-items:center;width:28px;height:28px;padding:6px;border:1px solid #c0deef25;border-radius:50%;background:#fff05;color:#bfd6e4;cursor:pointer}.va-sound svg{width:100%;height:100%}.va-court{position:relative;display:flex;flex:1;min-height:0;overflow:hidden;background:radial-gradient(ellipse at center,#1b3343,#081223)}.va-svg{display:block;width:100%;height:100%;touch-action:none;user-select:none;-webkit-user-select:none;-webkit-tap-highlight-color:transparent;outline-offset:-3px}.va-svg:focus-visible{outline:2px solid var(--va-accent)}.va-stats{position:absolute;z-index:2;top:13px;left:15px;right:15px;display:flex;justify-content:space-between;gap:8px;pointer-events:none}.va-stats>span{display:flex;flex-direction:column;align-items:center;gap:4px}.va-stats small{color:#98b2c1;font-size:6px;font-weight:850;letter-spacing:.12em}.va-stats b{display:flex;align-items:center;gap:5px;font-size:23px;line-height:1;font-weight:900;text-shadow:0 2px 10px #0007;font-variant-numeric:tabular-nums}.va-stats .va-gold{color:#ffe2a1}.va-stats .va-hot{color:#ffba7c}.va-hot svg{width:12px;height:16px}.va-stats .va-coin{width:12px;height:12px}.va-target{pointer-events:none}.va-target-high>ellipse:first-child{opacity:.8}.va-attacker{filter:drop-shadow(0 0 8px #b5ffee20)}.va-block-hands{filter:drop-shadow(0 4px 4px #0c233535)}.va-drag-hint{animation:va-hint 1.8s ease-in-out infinite;pointer-events:none}.va-cover{position:absolute;inset:0;display:flex;align-items:center;justify-content:center;flex-direction:column;gap:9px;background:#09192acc;backdrop-filter:blur(4px);font-size:13px}.va-cover>span{color:#9ab2c1;font-size:10px}.va-loader{width:29px;height:29px;margin-bottom:8px;border:2px solid #b8ffeb22;border-top-color:var(--va-accent);border-radius:50%;animation:va-spin .8s linear infinite}.va-timing{position:absolute;z-index:3;bottom:11px;left:15px;right:15px;padding:11px 12px 8px;border:1px solid #ccedff24;border-radius:14px;background:#091827e8;box-shadow:0 6px 22px #0003}.va-timing>div:first-child{display:flex;justify-content:space-between;align-items:center;gap:5px;margin-bottom:10px}.va-timing>div>b{font-size:7px;letter-spacing:.07em;color:#e5f2f8}.va-timing button{padding:2px 0;border:0;background:none;color:#91b1c0;font:inherit;font-size:8px;text-decoration:underline;text-underline-offset:2px;cursor:pointer}.va-meter{position:relative;height:13px;border-radius:7px;background:linear-gradient(90deg,#52718b,#64829c);box-shadow:inset 0 1px 4px #0006}.va-safe{position:absolute;top:0;bottom:0;background:linear-gradient(90deg,#46b797,#84f6c8);border-radius:4px}.va-perfect{position:absolute;top:0;bottom:0;background:#d2ffdc;border-left:1px solid #ffffff60;border-right:1px solid #ffffff60}.va-meter>i{position:absolute;top:-5px;bottom:-5px;width:4px;border:1px solid #fff;border-radius:4px;background:#12243a;translate:-50% 0;box-shadow:0 0 9px #0009}.va-timing>span{display:flex;justify-content:space-between;margin-top:8px;color:#8ea6b8;font-size:5.5px;font-weight:800;letter-spacing:.13em}.va-timing>span>b{font-size:7px;color:#baffdf}.va-reward{animation:va-reward 1.05s ease-out both;pointer-events:none}.va-feedback{position:absolute;z-index:3;top:48%;left:6px;right:6px;display:flex;align-items:center;flex-direction:column;gap:6px;text-align:center;pointer-events:none;text-shadow:0 3px 15px #03192b,0 1px 4px #03192b;animation:va-feedback .2s ease-out}.va-feedback>b{color:#e1fff1;font-size:25px;line-height:1.05;font-weight:950;letter-spacing:-.045em;font-style:italic}.va-feedback>span{padding:4px 8px;border-radius:8px;background:#0b203bc4;color:#e2f5f5;font-size:10px;font-weight:700}.va-feedback-miss>b{color:#fff0cb;font-size:23px}.va-feedback-miss>span{max-width:265px;font-size:9px;line-height:1.4}.va-footer{display:flex;align-items:center;justify-content:space-between;gap:10px;min-height:75px;padding:12px 14px;background:linear-gradient(115deg,#1a2d42,#0d192b);border-top:1px solid #c0e7ff18;flex:none}.va-status{flex:1;min-width:0}.va-status>b{display:block;color:#f0f9ff;font-size:11px;line-height:1.3;font-weight:850}.va-status>span{display:block;margin-top:4px;color:#9cb6c5;font-size:9px;line-height:1.45}.va-action{display:flex;align-items:center;justify-content:center;gap:6px;flex:none;min-width:92px;max-width:123px;min-height:40px;padding:10px 11px;border:1px solid #e2fff66b;border-radius:11px;background:var(--va-accent);box-shadow:0 3px 0 #04121c44;color:#0c2938;font:inherit;font-size:10px;font-weight:950;letter-spacing:.045em;cursor:pointer;transition:box-shadow .12s,background .12s}.va-action svg{width:14px;height:14px}.va-action-now{background:#c5ffe6;box-shadow:0 0 20px #92ffd440,0 3px 0 #04121c44}.va-action:active{transform:translateY(1px)}.va-action:focus-visible,.va-sound:focus-visible{outline:2px solid #fff;outline-offset:3px}.va-rule{display:flex;align-items:center;gap:7px;color:#e6ce8e;font-size:8px;font-weight:750;line-height:1.4}.is-scored .va-wallet{box-shadow:0 0 20px #fbd37225}@keyframes va-hint{50%{opacity:.4;translate:0 -3px}}@keyframes va-spin{to{rotate:360deg}}@keyframes va-reward{from{opacity:0;translate:0 12px;scale:.85}20%{opacity:1;scale:1}to{opacity:0;translate:0 -18px}}@keyframes va-feedback{from{opacity:0;translate:0 6px;scale:.95}to{opacity:1;translate:0 0;scale:1}}@media(max-width:340px){.va-header{padding-inline:11px}.va-brand b{font-size:18px}.va-brand>small{font-size:5px}.va-footer{gap:8px;padding-inline:11px}.va-action{min-width:83px;font-size:9px}.va-status>span{font-size:8px}.va-timing{left:11px;right:11px}}@media(prefers-reduced-motion:reduce){.va-drag-hint,.va-reward,.va-feedback{animation:none}.va-action{transition:none}}
</style>
