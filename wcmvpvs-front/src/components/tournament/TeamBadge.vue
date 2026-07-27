<script setup>
// v4: layout typography-led. Nel contesto torneo le squadre raramente hanno
// un logo: il nome È l'identità. Se team.logo è presente viene mostrato come
// piccola icona accanto al nome, altrimenti resta il solo testo.
const props = defineProps({
  team: { type: Object, required: true },   // { name, logo?, sub? }
  size: { type: String, default: 'lg' }     // 'lg' | 'md'
})
</script>

<template>
  <div class="team">
    <div class="team-identity">
      <div v-if="team.logo" class="team-logo" :class="`team-logo--${size}`">
        <img :src="team.logo" :alt="team.name" loading="lazy" />
      </div>
      <div class="team-name" :class="`team-name--${size}`">{{ team.name }}</div>
    </div>
    <div class="team-sub" v-if="team.sub">{{ team.sub }}</div>
  </div>
</template>

<style scoped>
.team { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 2px; min-width: 0; }
.team-identity { display: flex; align-items: center; justify-content: center; gap: 7px; min-width: 0; max-width: 100%; }
.team-logo {
  border-radius: 50%; background: #0D0D12; border: 2px solid rgba(255,255,255,.14);
  display: grid; place-items: center; overflow: hidden; flex: none;
}
.team-logo--lg { width: clamp(28px,4.5dvh,38px); height: clamp(28px,4.5dvh,38px); }
.team-logo--md { width: clamp(25px,4dvh,34px); height: clamp(25px,4dvh,34px); }
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
