<template>
  <main class="min-h-screen bg-[radial-gradient(circle_at_top,_rgba(59,130,246,0.2),_rgba(2,6,23,1)_50%)] px-4 py-8 text-white">
    <div class="mx-auto flex w-full max-w-5xl flex-col gap-6">
      <!-- Header -->
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p class="text-xs font-black uppercase tracking-[0.28em] text-blue-300">Dev Playground</p>
          <h1 class="mt-2 text-4xl font-black tracking-tight">Stack It</h1>
          <p class="mt-3 max-w-2xl text-sm text-slate-300">
            Demo isolata del mini-game branded Stack It. Bypassa il modal e monta il componente direttamente.
          </p>
        </div>
        <div class="rounded-3xl border border-white/10 bg-slate-950/60 px-5 py-4 backdrop-blur">
          <p class="text-[11px] font-black uppercase tracking-[0.24em] text-slate-400">Stato</p>
          <p class="mt-2 text-sm text-slate-200">GSAP pendolo + camera scroll + debris CSS</p>
        </div>
      </div>

      <div class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_340px]">
        <!-- Game area -->
        <section class="min-h-[78vh] rounded-[32px] border border-white/10 bg-slate-950/50 p-3 shadow-[0_30px_80px_rgba(0,0,0,0.35)]">
          <StackItGame
            :config="demoConfig"
            :event-id="9999"
            :wallet-coins="128"
            @claim="handleClaim"
            @exit="handleExit"
          />
        </section>

        <!-- Config sidebar -->
        <aside class="space-y-4 rounded-[32px] border border-white/10 bg-slate-950/60 p-5 backdrop-blur">
          <div>
            <p class="text-[11px] font-black uppercase tracking-[0.24em] text-slate-400">Brand config</p>
            <h2 class="mt-2 text-xl font-black">Preset demo</h2>
          </div>

          <div class="rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-slate-300 space-y-1">
            <p><span class="font-bold text-white">Sponsor:</span> {{ demoConfig.sponsor_name }}</p>
            <p><span class="font-bold text-white">Game type:</span> {{ demoConfig.game_type }}</p>
            <p><span class="font-bold text-white">Velocità iniziale:</span> {{ demoConfig.stack_it_config.initial_pendulum_speed_ms }}ms</p>
            <p><span class="font-bold text-white">Curva:</span> {{ demoConfig.stack_it_config.speed_curve }}</p>
            <p><span class="font-bold text-white">Reward:</span> {{ demoConfig.stack_it_config.reward_per_block }} coin/blocco + {{ demoConfig.stack_it_config.perfect_bonus_coins }} perfect</p>
          </div>

          <div class="rounded-2xl border border-white/10 bg-white/5 p-4">
            <p class="text-[11px] font-black uppercase tracking-[0.24em] text-slate-400">Meccaniche</p>
            <ul class="mt-3 space-y-2 text-sm text-slate-300">
              <li>Blocco oscilla orizzontalmente con GSAP sine.inOut</li>
              <li>Tap STACK! (o Spazio/Enter) per fermare</li>
              <li>±2px = perfect stack → nessuna riduzione + bonus coin</li>
              <li>La parte tagliata cade (CSS animation)</li>
              <li>Camera segue la cima della torre (GSAP power2.out)</li>
              <li>Velocità aumenta del 5% ogni 5 blocchi</li>
              <li>Game over se &lt;10% larghezza iniziale</li>
            </ul>
          </div>

          <div class="rounded-2xl border border-dashed border-blue-300/30 bg-blue-400/5 p-4 text-sm text-blue-100">
            Testa su mobile per la gestione del touch e 60fps. DevTools Memory tab dopo 10+ partite per leak check.
          </div>

          <div v-if="lastAction" class="rounded-2xl border border-emerald-500/30 bg-emerald-400/10 p-4">
            <p class="text-[11px] font-black uppercase tracking-[0.24em] text-emerald-400">Ultimo evento</p>
            <p class="mt-2 text-sm text-emerald-200 font-mono break-all">{{ lastAction }}</p>
          </div>
        </aside>
      </div>
    </div>
  </main>
</template>

<script setup>
import { ref } from 'vue';
import StackItGame from '../components/minigames/StackItGame.vue';

const lastAction = ref('');

const demoConfig = {
  sponsor_id: 'demo-sponsor',
  sponsor_name: 'DemoCorp Arena',
  sponsor_logo_url: '',
  primary_color: '#3b82f6',
  secondary_color: '#ffffff',
  game_type: 'stack_it',
  cta_label: 'Scopri il partner',
  cta_url: 'https://example.com',
  reward_type: 'coins',
  reward_coins: 0,
  max_plays_per_user: 5,
  stack_it_config: {
    block_texture_url: '',
    block_colors: ['#3b82f6', '#8b5cf6', '#f59e0b', '#10b981'],
    background_image_url: '',
    perfect_stack_sound_url: '',
    game_over_sound_url: '',
    reward_per_block: 0.5,
    perfect_bonus_coins: 2,
    initial_pendulum_speed_ms: 1500,
    speed_curve: 'standard',
    cta_label: 'Scopri il partner',
    cta_url: 'https://example.com',
  },
};

function handleClaim(payload) {
  lastAction.value = `@claim → ${JSON.stringify(payload)}`;
}

function handleExit() {
  lastAction.value = '@exit ricevuto';
}
</script>
