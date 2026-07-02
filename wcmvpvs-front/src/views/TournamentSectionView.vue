<script setup>
// Adattato per ArenaBoostX: niente vue-router. slug/section arrivano come prop
// da App.vue; il "back" emette 'navigate' verso l'host.
import '@/assets/tournament-tokens.css'

const props = defineProps({
  slug: { type: String, required: true },
  section: { type: String, required: true },
})
const emit = defineEmits(['navigate'])

const titles = {
  calendar: 'Calendario', standings: 'Classifiche', bracket: 'Tabellone',
  mvp: 'Vota MVP', prizes: 'Premi', gallery: 'Gallery',
  rules: 'Regolamento', event: 'Info Evento'
}
</script>

<template>
  <!-- Placeholder: sostituiremo con le view reali sezione per sezione.
       Il tabellone (bracket) è quello a più alto valore per gli organizzatori. -->
  <div class="section-page">
    <header class="section-head">
      <button class="back" @click="emit('navigate', `/t/${props.slug}`)" aria-label="Indietro">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>
      </button>
      <h1>{{ titles[props.section] || props.section }}</h1>
    </header>
    <div class="section-body">
      <p>Sezione «{{ titles[props.section] || props.section }}» in costruzione.</p>
    </div>
  </div>
</template>

<style scoped>
.section-page {
  width: 100%; max-width: 430px; margin: 0 auto; height: 100dvh;
  background: var(--tm-bg); color: var(--tm-text);
  display: flex; flex-direction: column;
}
.section-head {
  flex: none; display: flex; align-items: center; gap: 12px;
  padding: calc(10px + env(safe-area-inset-top)) 16px 12px;
  border-bottom: 1px solid var(--tm-border);
}
.back {
  width: 40px; height: 40px; display: grid; place-items: center;
  border: none; background: transparent; color: #fff; cursor: pointer; border-radius: 12px;
}
.back:focus-visible { outline: 2px solid var(--tm-gold); outline-offset: 2px; }
.section-head h1 { font-size: 18px; font-weight: 900; letter-spacing: .5px; }
.section-body { flex: 1; display: grid; place-items: center; color: var(--tm-text-dim); }
</style>
