<template>
  <main class="min-h-screen bg-[radial-gradient(circle_at_top,_rgba(249,115,22,0.22),_rgba(2,6,23,1)_48%)] px-4 py-8 text-white">
    <div class="mx-auto flex w-full max-w-6xl flex-col gap-6">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p class="text-xs font-black uppercase tracking-[0.28em] text-orange-300">Dev Playground</p>
          <h1 class="mt-2 text-4xl font-black tracking-tight">Free Throw Challenge</h1>
          <p class="mt-3 max-w-2xl text-sm text-slate-300">
            Demo tecnica isolata del mini-game branded. Questa pagina bypassa il modal e monta direttamente il wrapper Phaser.
          </p>
        </div>
        <div class="rounded-3xl border border-white/10 bg-slate-950/60 px-5 py-4 backdrop-blur">
          <p class="text-[11px] font-black uppercase tracking-[0.24em] text-slate-400">Stato corrente</p>
          <p class="mt-2 text-sm text-slate-200">Fase C: timer globale, hold power, tilt/swipe fallback e scoring attivi.</p>
        </div>
      </div>

      <div class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_320px]">
        <section class="min-h-[78vh] rounded-[32px] border border-white/10 bg-slate-950/50 p-3 shadow-[0_30px_80px_rgba(0,0,0,0.35)]">
          <FreeThrowChallengeGame
            :config="demoConfig"
            :event-id="9999"
            :wallet-coins="128"
            @claim="handleClaim"
            @exit="handleExit"
          />
        </section>

        <aside class="space-y-4 rounded-[32px] border border-white/10 bg-slate-950/60 p-5 backdrop-blur">
          <div>
            <p class="text-[11px] font-black uppercase tracking-[0.24em] text-slate-400">Brand config</p>
            <h2 class="mt-2 text-xl font-black">Preset demo</h2>
          </div>

          <div class="rounded-2xl border border-white/10 bg-white/5 p-4">
            <p class="text-sm font-black text-white">{{ demoConfig.sponsor_name }}</p>
            <p class="mt-1 text-sm text-slate-300">Game type: {{ demoConfig.game_type }}</p>
            <p class="mt-1 text-sm text-slate-300">Duration: {{ demoConfig.free_throw_config.game_duration_seconds }}s</p>
            <p class="mt-1 text-sm text-slate-300">Difficulty: {{ demoConfig.free_throw_config.difficulty_curve }}</p>
          </div>

          <div class="rounded-2xl border border-white/10 bg-white/5 p-4">
            <p class="text-[11px] font-black uppercase tracking-[0.24em] text-slate-400">Controlli attivi</p>
            <ul class="mt-3 space-y-2 text-sm text-slate-300">
              <li>Hold per caricare la potenza con barra sweet-spot.</li>
              <li>Tilt device se disponibile, swipe orizzontale come fallback.</li>
              <li>Scoring 3/2/1/0 con timer continuo e respawn palla.</li>
            </ul>
          </div>

          <div class="rounded-2xl border border-dashed border-orange-300/30 bg-orange-400/5 p-4 text-sm text-orange-100">
            Progressione difficoltà, bonus round, results scene finale e claim completo entrano nelle fasi successive.
          </div>

          <p v-if="lastAction" class="text-sm text-emerald-300">{{ lastAction }}</p>
        </aside>
      </div>
    </div>
  </main>
</template>

<script setup>
import { ref } from 'vue';
import FreeThrowChallengeGame from '../components/minigames/FreeThrowChallengeGame.vue';

const lastAction = ref('');

const demoConfig = {
  sponsor_id: 'demo-sponsor',
  sponsor_name: 'NightQuest Arena',
  sponsor_logo_url: '',
  primary_color: '#f97316',
  secondary_color: '#fff7ed',
  game_type: 'free_throw_challenge',
  cta_label: 'Scopri il partner',
  cta_url: 'https://example.com',
  reward_type: 'coins',
  reward_coins: 50,
  max_plays_per_user: 3,
  free_throw_config: {
    backboard_logo_url: '',
    ball_logo_url: '',
    background_image_url: '',
    announcer_audio_urls: {
      good_shot: '',
      streak: '',
    },
    music_url: '',
    game_duration_seconds: 60,
    difficulty_curve: 'standard',
    bonus_round_enabled: true,
    reward_per_point: 0.5,
  },
};

function handleClaim(payload) {
  lastAction.value = `Claim ricevuto: ${JSON.stringify(payload)}`;
}

function handleExit() {
  lastAction.value = 'Exit richiesto dal wrapper.';
}
</script>
