<script setup>
import TeamBadge from './TeamBadge.vue'

defineProps({
  match: { type: Object, required: true } // { court, time, teamA, teamB }
})
</script>

<template>
  <article class="match-card">
    <div class="match-head">
      <span class="label">PROSSIMA PARTITA</span>
      <span class="court">{{ match.court }} &nbsp;|&nbsp; <b>ORE {{ match.time }}</b></span>
    </div>
    <div class="match-body">
      <TeamBadge :team="match.teamA" size="md" />
      <div class="vs" aria-hidden="true">VS</div>
      <TeamBadge :team="match.teamB" size="md" />
    </div>
  </article>
</template>

<style scoped>
.match-card {
  flex: none; border-radius: var(--tm-radius);
  background: var(--tm-surface); border: 1px solid var(--tm-border); overflow: hidden;
}
.match-head { display: flex; align-items: center; justify-content: space-between; padding: clamp(7px,1.2dvh,11px) 12px 2px; }
.match-head .label { font-size: clamp(9px,1.35dvh,11px); font-weight: 800; letter-spacing: 1px; color: var(--tm-gold); }
.match-head .court { font-size: clamp(9px,1.35dvh,11px); font-weight: 700; letter-spacing: 1px; color: var(--tm-text-dim); }
.match-head .court b { color: var(--tm-text); font-weight: 800; }
.match-body { display: grid; grid-template-columns: 1fr auto 1fr; align-items: center; padding: clamp(3px,.6dvh,6px) 12px clamp(7px,1.3dvh,12px); }
.vs { font-size: clamp(13px,2.1dvh,18px); font-weight: 900; font-style: italic; color: var(--tm-text-dim); letter-spacing: 1px; text-align: center; }
</style>
