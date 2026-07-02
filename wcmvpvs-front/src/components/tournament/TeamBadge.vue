<script setup>
// v4: layout typography-led. Nel contesto torneo le squadre raramente hanno
// un logo: il nome È l'identità. Il logo resta supportato (modalità club):
// se team.logo è presente viene mostrato sopra il nome, altrimenti solo testo.
const props = defineProps({
  team: { type: Object, required: true },   // { name, logo?, sub? }
  size: { type: String, default: 'lg' }     // 'lg' | 'md'
})
</script>

<template>
  <div class="team">
    <div v-if="team.logo" class="team-logo" :class="`team-logo--${size}`">
      <img :src="team.logo" :alt="team.name" loading="lazy" />
    </div>
    <div class="team-name" :class="`team-name--${size}`">{{ team.name }}</div>
    <div class="team-sub" v-if="team.sub">{{ team.sub }}</div>
  </div>
</template>

<style scoped>
.team { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 2px; min-width: 0; }
.team-logo {
  border-radius: 50%; background: #0D0D12; border: 2px solid rgba(255,255,255,.14);
  display: grid; place-items: center; overflow: hidden; margin-bottom: 4px;
}
.team-logo--lg { width: clamp(38px,6.8dvh,58px); height: clamp(38px,6.8dvh,58px); }
.team-logo--md { width: clamp(36px,6.4dvh,54px); height: clamp(36px,6.4dvh,54px); }
.team-logo img { width: 100%; height: 100%; object-fit: cover; }
/* Senza logo il nome porta l'identità: corpo maggiore, max 2 righe */
.team-name {
  font-weight: 900; letter-spacing: .5px; text-transform: uppercase; text-align: center;
  line-height: 1.15; max-width: 100%;
  display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden;
}
.team-name--lg { font-size: clamp(12px,2dvh,16px); }
.team-name--md { font-size: clamp(11.5px,1.9dvh,15px); }
.team-sub { font-size: clamp(8px,1.2dvh,10px); color: var(--tm-text-faint); letter-spacing: 1.5px; }
</style>
